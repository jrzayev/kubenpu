//
// Created by Javid Rzayev on 12.08.26.
//

package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	ReasonDeviceUnknown     = "device_unknown"
	ReasonNotPod            = "not_pod"
	ReasonNotIndexed        = "not_indexed"
	ReasonContainerNotFound = "container_not_found"
	ReasonQueueFull         = "queue_full"
	ReasonRingbufFull       = "ringbuf_full"
)

var dropReasons = []string{
	ReasonDeviceUnknown,
	ReasonNotPod,
	ReasonNotIndexed,
	ReasonContainerNotFound,
	ReasonQueueFull,
}

type key struct {
	namespace string
	pod       string
	container string
	device    string
	vendor    string
	kind      string
}

type deviceKey struct {
	device string
	vendor string
	driver string
	pciID  string
}

type Collector struct {
	mu sync.Mutex

	ioctlTotal         map[key]uint64
	eventsDroppedTotal map[string]uint64
	deviceInfo         map[deviceKey]struct{}

	cgroupIndexRebuildsSource func() uint64
	kernelDroppedSource       func() (uint64, error)

	ioctlDesc          *prometheus.Desc
	eventsDroppedDesc  *prometheus.Desc
	cgroupRebuildsDesc *prometheus.Desc
	deviceInfoDesc     *prometheus.Desc
}

func NewCollector() *Collector {
	dropped := make(map[string]uint64, len(dropReasons))
	for _, reason := range dropReasons {
		dropped[reason] = 0
	}

	return &Collector{
		ioctlTotal:         make(map[key]uint64),
		eventsDroppedTotal: dropped,
		deviceInfo:         make(map[deviceKey]struct{}),

		ioctlDesc: prometheus.NewDesc(
			"kubenpu_ioctl_total",
			"Total number of ioctls",
			[]string{"namespace", "pod", "container", "device", "vendor", "kind"},
			nil,
		),
		eventsDroppedDesc: prometheus.NewDesc(
			"kubenpu_events_dropped_total",
			"Total number of dropped events",
			[]string{"reason"},
			nil,
		),
		cgroupRebuildsDesc: prometheus.NewDesc(
			"kubenpu_cgroup_index_rebuilds_total",
			"Total number of cgroup index rebuilds",
			nil,
			nil,
		),
		deviceInfoDesc: prometheus.NewDesc(
			"kubenpu_device_info",
			"Device information",
			[]string{"device", "vendor", "driver", "pci_id"},
			nil,
		),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.ioctlDesc
	ch <- c.eventsDroppedDesc
	ch <- c.cgroupRebuildsDesc
	ch <- c.deviceInfoDesc
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for k, value := range c.ioctlTotal {
		ch <- prometheus.MustNewConstMetric(
			c.ioctlDesc,
			prometheus.CounterValue,
			float64(value),
			k.namespace,
			k.pod,
			k.container,
			k.device,
			k.vendor,
			k.kind,
		)
	}

	for reason, value := range c.eventsDroppedTotal {
		ch <- prometheus.MustNewConstMetric(
			c.eventsDroppedDesc,
			prometheus.CounterValue,
			float64(value),
			reason,
		)
	}

	if c.kernelDroppedSource != nil {
		lost, err := c.kernelDroppedSource()
		if err == nil {
			ch <- prometheus.MustNewConstMetric(
				c.eventsDroppedDesc,
				prometheus.CounterValue,
				float64(lost),
				ReasonRingbufFull,
			)
		}
	}

	var cgroupRebuilds uint64
	if c.cgroupIndexRebuildsSource != nil {
		cgroupRebuilds = c.cgroupIndexRebuildsSource()
	}

	ch <- prometheus.MustNewConstMetric(
		c.cgroupRebuildsDesc,
		prometheus.CounterValue,
		float64(cgroupRebuilds),
	)

	for k := range c.deviceInfo {
		ch <- prometheus.MustNewConstMetric(
			c.deviceInfoDesc,
			prometheus.GaugeValue,
			1,
			k.device,
			k.vendor,
			k.driver,
			k.pciID,
		)
	}
}

func (c *Collector) IncIoctl(namespace string, pod string, container string,
	device string, vendor string, kind string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ioctlTotal[key{
		namespace: namespace,
		pod:       pod,
		container: container,
		device:    device,
		vendor:    vendor,
		kind:      kind,
	}]++
}

func (c *Collector) IncEventsDropped(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.eventsDroppedTotal[reason]++
}

func (c *Collector) SetCgroupIndexRebuildsSource(source func() uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cgroupIndexRebuildsSource = source
}

func (c *Collector) SetKernelDroppedSource(source func() (uint64, error)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.kernelDroppedSource = source
}

func (c *Collector) SetDeviceInfo(device string, vendor string, driver string, pciID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.deviceInfo[deviceKey{
		device: device,
		vendor: vendor,
		driver: driver,
		pciID:  pciID,
	}] = struct{}{}
}
