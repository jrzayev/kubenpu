//
// Created by Javid Rzayev on 11.08.26.
//

package hw

import (
	"hash/fnv"

	"github.com/jrzayev/kubenpu/pkg/discovery"
)

const fallbackVendorID uint32 = 1

var registeredVendors []Vendor

func RegisterVendor(vendor Vendor) {
	registeredVendors = append(registeredVendors, vendor)
}

func GetVendor(d discovery.Device) Vendor {
	for _, vendor := range registeredVendors {
		if vendor.Match(d) {
			return vendor
		}
	}
	return nil
}

func GetVendorID(vendorName string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(vendorName))

	id := h.Sum32()
	if id == 0 {
		return fallbackVendorID
	}

	return id
}
