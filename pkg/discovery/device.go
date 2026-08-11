//
// Created by Javid Rzayev on 11.08.26.
//

package discovery

type Node struct {
	Minor uint32
	Major uint32
	Path  string
}

type Device struct {
	DriverName string
	SysfsPath  string
	PciID      string
	PciAddress string
	Nodes      []Node
}
