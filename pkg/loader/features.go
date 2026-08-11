//
// Created by Javid Rzayev on 10.08.26.
//

package loader

import (
	"errors"
	"fmt"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/btf"
	"github.com/cilium/ebpf/features"
)

func CheckKernel() error {
	_, err := btf.LoadKernelSpec()
	if err != nil {
		return fmt.Errorf("kernel BTF is not available; required for CO-RE and fentry: %w", err)
	}

	err = features.HaveProgramType(ebpf.Tracing)
	if err != nil {
		if errors.Is(err, ebpf.ErrNotSupported) {
			return errors.New("fentry is not supported; required kernel version: 5.5+")
		}
		return fmt.Errorf("failed to check kernel support for fentry: %w", err)
	}

	err = features.HaveMapType(ebpf.RingBuf)
	if err != nil {
		if errors.Is(err, ebpf.ErrNotSupported) {
			return errors.New("ring buffer is not supported; required kernel version: 5.8+")
		}
		return fmt.Errorf("failed to check kernel support for ring buffer: %w", err)
	}

	err = features.HaveProgramHelper(ebpf.Kprobe, asm.FnKtimeGetTaiNs)
	if err != nil {
		if errors.Is(err, ebpf.ErrNotSupported) {
			return errors.New("bpf_ktime_get_tai_ns is not supported; required kernel version: 6.1+")
		}
		return fmt.Errorf("failed to check kernel support for bpf_ktime_get_tai_ns: %w", err)
	}
	return nil
}
