//
// Created by Javid Rzayev on 11.08.26.
//

package discovery

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func GetDeviceAddress(path string) (string, string, error) {
	devicePath := filepath.Join(path, "device")

	retDevicePath, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve %s: %w", devicePath, err)
	}

	deviceAddress := filepath.Base(retDevicePath)

	return deviceAddress, retDevicePath, nil
}

func ReadUevent(drmPath string) (map[string]string, error) {
	devicePath := filepath.Join(drmPath, "device")
	ueventPath := filepath.Join(devicePath, "uevent")

	uevent := make(map[string]string)

	file, err := os.Open(ueventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", ueventPath, err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("failed to close %s: %v\n", ueventPath, err)
		}
	}(file)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}

		uevent[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", ueventPath, err)
	}

	if uevent["DRIVER"] == "" {
		return nil, fmt.Errorf("%s does not contain required field: DRIVER", ueventPath)
	}

	return uevent, nil
}
