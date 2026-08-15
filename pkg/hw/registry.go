//
// Created by Javid Rzayev on 11.08.26.
//

package hw

import (
	"github.com/jrzayev/kubenpu/pkg/discovery"
)

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
	for i, vendor := range registeredVendors {
		if vendor.Name() == vendorName {
			return uint32(i) + 1
		}
	}

	return 0
}
