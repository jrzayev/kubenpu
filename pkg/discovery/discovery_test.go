//
// Created by Javid Rzayev on 11.08.26.
//

package discovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadUevent(t *testing.T) {
	t.Run("reads i915 uevent", func(t *testing.T) {
		uevent, err := ReadUevent("testdata/i915")

		if err != nil {
			t.Fatalf("ReadUevent() error = %v, want nil", err)
		}

		if got, want := uevent["DRIVER"], "i915"; got != want {
			t.Errorf("DRIVER = %q, want %q", got, want)
		}

		if got, want := uevent["PCI_ID"], "8086:5917"; got != want {
			t.Errorf("PCI_ID = %q, want %q", got, want)
		}
	})

	t.Run("missing DRIVER returns error", func(t *testing.T) {
		uevent, err := ReadUevent("testdata/missing-driver")

		if err == nil {
			t.Fatalf("ReadUevent() error = nil, want error")
		}

		if uevent != nil {
			t.Errorf("ReadUevent() result = %#v, want nil", uevent)
		}
	})

	t.Run("missing PCI_ID is allowed", func(t *testing.T) {
		uevent, err := ReadUevent("testdata/no-pci-id")

		if err != nil {
			t.Fatalf("ReadUevent() error = %v, want nil", err)
		}

		if got, want := uevent["DRIVER"], "panthor"; got != want {
			t.Errorf("DRIVER = %q, want %q", got, want)
		}

		if _, ok := uevent["PCI_ID"]; ok {
			t.Errorf("PCI_ID is present, want it to be absent")
		}
	})

	t.Run("value containing equals is preserved", func(t *testing.T) {
		uevent, err := ReadUevent("testdata/equals-in-value")

		if err != nil {
			t.Fatalf("ReadUevent() error = %v, want nil", err)
		}

		if got, want := uevent["MODALIAS"], "pci:v00008086d00005917=extra"; got != want {
			t.Errorf("MODALIAS = %q, want %q", got, want)
		}
	})
}

func TestGetDeviceAddress(t *testing.T) {
	t.Run("resolves device symlink", func(t *testing.T) {
		root := t.TempDir()

		devicePath := filepath.Join(root, "i915")
		pciPath := filepath.Join(root, "0000:02:00.0")

		if err := os.MkdirAll(devicePath, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		if err := os.MkdirAll(pciPath, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		if err := os.Symlink(pciPath, filepath.Join(devicePath, "device")); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}

		address, sysfsPath, err := GetDeviceAddress(devicePath)
		if err != nil {
			t.Fatalf("GetDeviceAddress() error = %v, want nil", err)
		}

		if got, want := address, "0000:02:00.0"; got != want {
			t.Errorf("address = %q, want %q", got, want)
		}

		if got, want := sysfsPath, pciPath; got != want {
			t.Errorf("sysfsPath = %q, want %q", got, want)
		}
	})

	t.Run("missing device symlink returns error", func(t *testing.T) {
		root := t.TempDir()

		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		_, _, err := GetDeviceAddress(root)
		if err == nil {
			t.Fatalf("GetDeviceAddress() error = nil, want error")
		}
	})
}

func TestDiscover(t *testing.T) {
	t.Run("missing dri path returns error", func(t *testing.T) {
		paths := Paths{
			DriPath:        filepath.Join(t.TempDir(), "dri"),
			AccelPath:      filepath.Join(t.TempDir(), "accel"),
			SysfsDriPath:   filepath.Join(t.TempDir(), "sysfs-dri"),
			SysfsAccelPath: filepath.Join(t.TempDir(), "sysfs-accel"),
		}

		devices, err := Discover(paths)

		if err == nil {
			t.Fatalf("Discover() error = nil, want error")
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Discover() error = %v, want os.ErrNotExist", err)
		}

		if devices != nil {
			t.Errorf("Discover() devices = %#v, want nil", devices)
		}
	})

	t.Run("empty dri and missing accel returns empty result", func(t *testing.T) {
		root := t.TempDir()

		driPath := filepath.Join(root, "dri")
		sysfsDriPath := filepath.Join(root, "sysfs-dri")

		if err := os.MkdirAll(driPath, 0o755); err != nil {
			t.Fatalf("MkdirAll(dri) error = %v", err)
		}

		if err := os.MkdirAll(sysfsDriPath, 0o755); err != nil {
			t.Fatalf("MkdirAll(sysfs-dri) error = %v", err)
		}

		devices, err := Discover(Paths{
			DriPath:        driPath,
			AccelPath:      filepath.Join(root, "missing-accel"),
			SysfsDriPath:   sysfsDriPath,
			SysfsAccelPath: filepath.Join(root, "missing-sysfs-accel"),
		})

		if err != nil {
			t.Fatalf("Discover() error = %v, want nil", err)
		}

		if len(devices) != 0 {
			t.Errorf("Discover() returned %d devices, want 0", len(devices))
		}
	})
}
