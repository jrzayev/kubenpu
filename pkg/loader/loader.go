//
// Created by Javid Rzayev on 10.08.26.
//

package loader

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
)

type Loader struct {
	bpfObjects bpfObjects
	hook       link.Link
	rb         *ringbuf.Reader
}

type Event struct {
	Timestamp uint64
	CgroupID  uint64
	TGID      uint32
	Cmd       uint32
	Minor     uint32
	Major     uint16
	Kind      uint8
}

func NewLoader() (*Loader, error) {
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, fmt.Errorf("failed to remove memlock rlimit: %w", err)
	}

	l := &Loader{}
	if err := loadBpfObjects(&l.bpfObjects, nil); err != nil {
		return nil, fmt.Errorf("failed to load bpf objects: %w", err)
	}

	return l, nil
}

func (l *Loader) AddDevice(major, minor, vendorID uint32) error {
	key := bpfDeviceKey{
		Major: major,
		Minor: minor,
	}

	err := l.bpfObjects.Devices.Put(&key, &vendorID)
	if err != nil {
		return fmt.Errorf("failed to add device: %w", err)
	}

	return nil
}

func (l *Loader) AddIoctl(vendorID, nr uint32, kind uint8) error {
	key := bpfIoctlKey{
		VendorId: vendorID,
		Nr:       nr,
	}
	err := l.bpfObjects.Ioctls.Put(&key, &kind)
	if err != nil {
		return fmt.Errorf("failed to add ioctl: %w", err)
	}
	return nil
}

func (l *Loader) Attach() error {
	if l.hook != nil {
		if err := l.hook.Close(); err != nil {
			return fmt.Errorf("failed to close previous hook: %w", err)
		}
		l.hook = nil
	}

	hook, err := link.AttachTracing(link.TracingOptions{
		Program: l.bpfObjects.KubenpuIoctl,
	})
	if err != nil {
		return fmt.Errorf("failed to attach kubenpu_ioctl: %w", err)
	}

	l.hook = hook
	return nil
}

func (l *Loader) CreateReader() error {
	rb, err := ringbuf.NewReader(l.bpfObjects.Events)
	if err != nil {
		return fmt.Errorf("failed to create ring buffer reader: %w", err)
	}

	l.rb = rb
	return nil
}

func (l *Loader) ReadEvent() (Event, error) {
	record, err := l.rb.Read()
	if err != nil {
		if errors.Is(err, ringbuf.ErrClosed) {
			return Event{}, err
		}

		return Event{}, fmt.Errorf("failed to read ring buffer: %w", err)
	}

	if len(record.RawSample) < int(unsafe.Sizeof(bpfEvent{})) {
		return Event{}, fmt.Errorf(
			"invalid event size: got %d bytes, want at least %d",
			len(record.RawSample),
			unsafe.Sizeof(bpfEvent{}),
		)
	}

	raw := (*bpfEvent)(unsafe.Pointer(&record.RawSample[0]))

	return Event{
		Timestamp: raw.Timestamp,
		CgroupID:  raw.CgroupId,
		TGID:      raw.Tgid,
		Cmd:       raw.Cmd,
		Minor:     raw.Minor,
		Major:     raw.Major,
		Kind:      raw.Kind,
	}, nil
}

func (l *Loader) Close() error {
	var errs []error

	if l.rb != nil {
		if err := l.rb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close ring buffer reader: %w", err))
		}
		l.rb = nil
	}

	if l.hook != nil {
		if err := l.hook.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close hook: %w", err))
		}
		l.hook = nil
	}

	if err := l.bpfObjects.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close bpf objects: %w", err))
	}

	return errors.Join(errs...)
}
