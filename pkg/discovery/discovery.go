//
// Created by Javid Rzayev on 11.08.26.
//

package discovery

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type Paths struct {
	DriPath        string
	AccelPath      string
	SysfsDriPath   string
	SysfsAccelPath string
}

func Discover(paths Paths) ([]Device, error) {
	devices := make(map[string]*Device)

	if err := discoverDevices(paths.DriPath, paths.SysfsDriPath, devices); err != nil {
		return nil, err
	}

	if err := discoverDevices(paths.AccelPath, paths.SysfsAccelPath, devices); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	result := make([]Device, 0, len(devices))
	for _, device := range devices {
		result = append(result, *device)
	}

	return result, nil
}

func discoverDevices(devPath, sysfsPath string, devices map[string]*Device) error {
	files, err := os.ReadDir(devPath)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileType := file.Type()
		if fileType&os.ModeCharDevice == 0 {
			continue
		}

		fileFullPath := filepath.Join(devPath, file.Name())
		sysfsFullPath := filepath.Join(sysfsPath, file.Name())

		var stat unix.Stat_t
		if err := unix.Stat(fileFullPath, &stat); err != nil {
			log.Printf("Failed to get stat for file %v: %v\n", fileFullPath, err)
			continue
		}

		rdev := uint64(stat.Rdev)
		major := unix.Major(rdev)
		minor := unix.Minor(rdev)

		uevent, err := ReadUevent(sysfsFullPath)
		if err != nil {
			log.Printf("Failed to read uevent for file %v: %v\n", sysfsFullPath, err)
			continue
		}

		deviceAddress, deviceSysfsPath, err := GetDeviceAddress(sysfsFullPath)
		if err != nil {
			log.Printf("Failed to get device address for file %v: %v\n", sysfsFullPath, err)
			continue
		}

		n := Node{
			Minor: minor,
			Major: major,
			Path:  fileFullPath,
		}

		device, ok := devices[deviceAddress]
		if !ok {
			device = &Device{
				DriverName: uevent["DRIVER"],
				SysfsPath:  deviceSysfsPath,
				PciID:      uevent["PCI_ID"],
				PciAddress: deviceAddress,
				Nodes:      []Node{n},
			}

			devices[deviceAddress] = device
			continue
		}

		device.Nodes = append(device.Nodes, n)
	}

	return nil
}
