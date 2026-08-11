//
// Created by Javid Rzayev on 10.08.26.
//

package ivpu

import (
	"github.com/jrzayev/kubenpu/pkg/discovery"
	"github.com/jrzayev/kubenpu/pkg/hw"
)

type Vendor struct{}

func init() {
	hw.RegisterVendor(&Vendor{})
}

func (v *Vendor) Name() string {
	return "ivpu"
}

func (v *Vendor) Match(d discovery.Device) bool {
	return d.DriverName == "ivpu"
}

func (v *Vendor) Weight(cmd uint32, payload []byte) uint64 {
	return 0
}

func (v *Vendor) DeviceStats(d discovery.Device) (hw.Stats, error) {
	return hw.Stats{}, hw.ErrNotSupported
}

var _ hw.Vendor = (*Vendor)(nil)
