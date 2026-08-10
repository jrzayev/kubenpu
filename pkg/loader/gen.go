//
// Created by Javid Rzayev on 10.08.26.
//

package loader

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -type event -type device_key -type ioctl_key -target amd64 -cflags "-I../../bpf" bpf ../../bpf/drm_common.bpf.c
