//
// Created by Javid Rzayev on 10.08.26.
//

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf/ringbuf"
	"github.com/jrzayev/kubenpu/pkg/discovery"
	"github.com/jrzayev/kubenpu/pkg/hw/i915"
	"github.com/jrzayev/kubenpu/pkg/loader"
)

func printVersion() {
	fmt.Println("Version: 0.0.1")
}

func main() {
	versionFlag := flag.Bool("version", false, "print version information")
	debugFlag := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	if *debugFlag {
		fmt.Println("Debug mode enabled")
	}

	l, err := loader.NewLoader()
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
		DriPath:        "/dev/dri",
		AccelPath:      "/dev/accel",
		SysfsDriPath:   "/sys/class/drm",
		SysfsAccelPath: "/sys/class/accel",
	}

	devices, err := discovery.Discover(paths)
	if err != nil {
		log.Fatal(err)
	}
	for _, device := range devices {
		for _, node := range device.Nodes {
			err = l.AddDevice(node.Major, node.Minor, 85)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	m := &i915.Vendor{}
	kinds := m.Ioctls()
	for key := range kinds {
		err = l.AddIoctl(85, key, uint8(kinds[key]))
		if err != nil {
			log.Fatal(err)
		}
	}

	err = l.Attach()
	if err != nil {
		log.Fatal(err)
	}

	err = l.CreateReader()
	if err != nil {
		log.Fatal(err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println("\nReceived signal:", sig)

		if err := l.Close(); err != nil {
			fmt.Println("Error closing listener:", err)
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
		fmt.Println("Event received", event.Kind, event.CgroupID, event.TGID)
	}
}
