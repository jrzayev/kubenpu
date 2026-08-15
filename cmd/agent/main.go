//
// Created by Javid Rzayev on 10.08.26.
//

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/jrzayev/kubenpu/pkg/config"
	"github.com/jrzayev/kubenpu/pkg/discovery"
	"github.com/jrzayev/kubenpu/pkg/hw"
	_ "github.com/jrzayev/kubenpu/pkg/hw/i915"
	_ "github.com/jrzayev/kubenpu/pkg/hw/ivpu"
	"github.com/jrzayev/kubenpu/pkg/loader"
	"github.com/jrzayev/kubenpu/pkg/metrics"
	"github.com/jrzayev/kubenpu/pkg/podmap"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type deviceNode struct {
	major uint32
	minor uint32
}

type deviceMeta struct {
	pciAddress string
	vendor     string
}

func main() {
	appConfig := config.Load()
	register := prometheus.NewRegistry()
	collector := metrics.NewCollector()
	register.MustRegister(collector)
	register.MustRegister(collectors.NewGoCollector())
	register.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	versionFlag := flag.Bool("version", false, "print version information")
	debugFlag := flag.Bool("debug", appConfig.Debug, "enable debug logging")
	flag.Parse()

	if *versionFlag {
		fmt.Println("Version: ", appConfig.AppVersion)
		return
	}

	if *debugFlag {
		fmt.Println("Debug mode enabled")
	}

	l, err := loader.NewLoader(debugFlag != nil && *debugFlag)
	if err != nil {
		log.Fatal(err)
	}

	defer func(loader *loader.Loader) {
		err := loader.Close()
		if err != nil {
			log.Print(err)
		}
	}(l)

	paths := discovery.Paths{
		DriPath:        appConfig.DriPath,
		AccelPath:      appConfig.AccelPath,
		SysfsDriPath:   appConfig.SysfsDriPath,
		SysfsAccelPath: appConfig.SysfsAccelPath,
	}

	devices, err := discovery.Discover(paths)
	if err != nil {
		log.Fatal(err)
	}

	deviceMap := make(map[deviceNode]deviceMeta)

	for _, device := range devices {
		vendor := hw.GetVendor(device)
		if vendor == nil {
			log.Printf("Found unknown device: %v", device)
			continue
		}

		vendorID := hw.GetVendorID(vendor.Name())

		collector.SetDeviceInfo(
			device.PciAddress,
			vendor.Name(),
			device.DriverName,
			device.PciID,
		)

		for _, node := range device.Nodes {
			deviceMap[deviceNode{
				major: node.Major,
				minor: node.Minor,
			}] = deviceMeta{
				pciAddress: device.PciAddress,
				vendor:     vendor.Name(),
			}

			err = l.AddDevice(node.Major, node.Minor, vendorID)
			if err != nil {
				log.Fatal(err)
			}
		}

		kinds := vendor.Ioctls()
		for key, val := range kinds {
			err = l.AddIoctl(vendorID, key, uint8(val))
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	err = l.Attach()
	if err != nil {
		log.Fatal(err)
	}

	p, err := podmap.NewPodMap(appConfig.CgroupRootPath, appConfig.CriSocketPath,
		appConfig.Interval, appConfig.CriTimeout, appConfig.CacheTTL)
	if err != nil {
		log.Fatal(err)
	}

	collector.SetCgroupIndexRebuildsSource(p.CgroupIndexRebuilds)
	collector.SetKernelDroppedSource(l.DroppedEvents)

	defer func(podMap *podmap.PodMap) {
		err := podMap.Close()
		if err != nil {
			log.Print(err)
		}
	}(p)

	var ready atomic.Bool

	srv := metrics.NewServer(
		net.JoinHostPort(
			appConfig.Host,
			strconv.Itoa(appConfig.Port),
		),
		ready.Load,
		register,
	)

	go func() {
		if err := srv.Run(); err != nil {
			log.Printf("Metrics server error: %v", err)
		}
	}()

	err = l.CreateReader()
	if err != nil {
		log.Fatal(err)
	}

	ready.Store(true)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println("\nReceived signal:", sig)

		ready.Store(false)

		ctx, cancel := context.WithTimeout(context.Background(), appConfig.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down metrics server: %v", err)
		}

		if err := l.CloseReader(); err != nil {
			log.Printf("Error closing ring buffer reader: %v", err)
		}
	}()

	queue := make(chan loader.Event, appConfig.QueueSize)
	var workers sync.WaitGroup

	workers.Add(1)
	go func() {
		defer workers.Done()

		for event := range queue {
			device, ok := deviceMap[deviceNode{
				major: uint32(event.Major),
				minor: event.Minor,
			}]
			if !ok {
				collector.IncEventsDropped(metrics.ReasonDeviceUnknown)
				continue
			}

			cI, err := p.ContainerInfo(event.CgroupID)
			if errors.Is(err, podmap.ErrNotPod) {
				collector.IncEventsDropped(metrics.ReasonNotPod)
				continue
			}

			if errors.Is(err, podmap.ErrNotIndexed) {
				collector.IncEventsDropped(metrics.ReasonNotIndexed)
				continue
			}

			if errors.Is(err, podmap.ErrContainerNotFound) {
				collector.IncEventsDropped(metrics.ReasonContainerNotFound)
				continue
			}

			if err != nil {
				log.Print(err)
				continue
			}

			collector.IncIoctl(
				cI.Namespace,
				cI.PodName,
				cI.ContainerName,
				device.pciAddress,
				device.vendor,
				hw.Kind(event.Kind).String(),
			)
		}
	}()

	fmt.Println("Starting agent")

	for {
		event, err := l.ReadEvent()
		if errors.Is(err, ringbuf.ErrClosed) {
			break
		}
		if err != nil {
			log.Print(err)
			continue
		}

		select {
		case queue <- event:
		default:
			collector.IncEventsDropped(metrics.ReasonQueueFull)
		}
	}

	close(queue)
	workers.Wait()
}
