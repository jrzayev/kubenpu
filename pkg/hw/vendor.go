//
// Created by Javid Rzayev on 11.08.26.
//

package hw

import (
	"errors"

	"github.com/jrzayev/kubenpu/pkg/discovery"
)

type Stats struct {
	BusyNs uint64
}

var ErrNotSupported = errors.New("driver does not expose device stats")

type Vendor interface {
	Name() string
	Match(d discovery.Device) bool
	Ioctls() map[uint32]Kind
	Weight(cmd uint32, payload []byte) uint64
	DeviceStats(d discovery.Device) (Stats, error)
}
