# KubeNPU

**Per-pod visibility into NPU and accelerator usage on Kubernetes.**

KubeNPU counts accelerator calls per pod with eBPF and exports them to Prometheus.

## How it works

Every DRM and accel driver routes userspace through one kernel function,
`drm_ioctl`. The agent attaches a fentry program there and filters in-kernel:

```
ffmpeg in a pod
  └─ ioctl(fd, DRM_IOCTL_I915_GEM_EXECBUFFER2)
       └─ drm_ioctl()
            └─ fentry program
                 ├─ filp->f_inode->i_rdev  →  226:128
                 ├─ devices[226:128]       →  vendor id, or drop
                 ├─ cmd & 0xff             →  0x69
                 ├─ ioctls[vendor, 0x69]   →  kind=submit, or drop
                 └─ ringbuf ← {timestamp, cgroup_id, tgid, cmd, major, minor, kind}
```

Both maps are filled by Go before the program is attached: device discovery
walks `/dev/dri` and sysfs, the vendor is matched by driver name, and its ioctl
table is uploaded. More than half the calls never reach userspace.

Userspace then turns `cgroup_id` into a pod name:

```
cgroup_id 117316
  → /sys/fs/cgroup/kubepods.slice/.../cri-containerd-079adc….scope
    → CRI ContainerStatus
      → default / ffmpeg-vaapi-test / ffmpeg
```

The cgroup index is rebuilt on a miss, not on a timer a pod created after
startup is picked up within one event.

This works identically across vendors because ioctl **numbers** are frozen uAPI.
Unlike kernel function names, they do not change between releases.

## Metrics

```
kubenpu_ioctl_total{pod,namespace,container,device,vendor,kind="submit|alloc|wait"}
kubenpu_device_info{device,vendor,driver,pci_id}
kubenpu_events_dropped_total{reason}
kubenpu_cgroup_index_rebuilds_total
```

`device` is the PCI address, which survives reboots. `card1` does not, because
its minor number can change. The node name is in `kubenpu_device_info`.

Counts are exact, never estimated. They are call counts, not utilization: on
Intel UHD 620 a workload with 4× more submissions used 7× less device time.

Counting starts with the agent, not with the pod. Anything a pod did before
that is missing, mostly `kind="alloc"`, which happens once at startup. The same
applies after every DaemonSet rollout.

## Quick test

### Docker, single host

```shell
docker build -t kubenpu-local .

docker run --rm --name kubenpu --privileged \
  -p 127.0.0.1:8080:8080 \
  -v /sys/fs/cgroup:/sys/fs/cgroup:ro \
  -v /sys:/sys:ro -v /dev:/dev:ro \
  -v /run/k3s/containerd:/run/k3s/containerd:ro \
  kubenpu-local
```

```shell
curl -s localhost:8080/metrics | grep kubenpu_
```

`--privileged` is only for the quick test. In Kubernetes the agent runs with
`CAP_BPF` and `CAP_PERFMON`.

### Locally, without a container

```shell
make build
sudo ./bin/agent -debug
```

`-debug` turns on the full verifier log - without it a rejected program prints
two useless lines.

To check what the agent sees before touching the kernel:

```shell
make build-kubenpuctl
./bin/kubenpuctl devices
```

```
DRIVER  PCI ID     ADDRESS       NODES          VENDOR
i915    8086:5917  0000:00:02.0  226:1,226:128  i915
```

This reads sysfs only and needs no privileges. If your device shows up with
`-` in the VENDOR column, KubeNPU found the hardware but has no implementation
for it yet.

### Using your own vmlinux.h

`bpf/vmlinux.h` is checked in and works on any kernel with BTF, since the
program is CO-RE and relocates field offsets at load time. To regenerate it
from your own kernel:

```shell
bpftool btf dump file /sys/kernel/btf/vmlinux format c > bpf/vmlinux.h
make generate
make build
```

### Kubernetes

```shell
helm install kubenpu deploy/helm/kubenpu -n kubenpu --create-namespace
```

```shell
kubectl -n kubenpu port-forward daemonset/kubenpu 8080:8080
curl -s localhost:8080/metrics | grep kubenpu_ioctl_total
```

Set `serviceMonitor.enabled=true` if you run the Prometheus Operator.

## Requirements

- Linux 6.1+ with BTF (`CONFIG_DEBUG_INFO_BTF=y`)
- cgroup v2 with the systemd driver - `cgroupfs` layouts are not parsed yet
- a CRI v1 runtime, socket reachable by the agent

Check both cgroup settings before installing, a mismatch on the second one is
silent:

```shell
ls /sys/fs/cgroup/cgroup.controllers
ls /sys/fs/cgroup/ | grep kubepods
```

`kubepods.slice` means the systemd driver and is supported. Plain `kubepods`
means `cgroupfs`: the agent will start, find your devices and report zero pods.

## Tested Hardware

| driver | device                | ioctls              | status    | hardware                   | kernel      | cluster                                    | workload               |
|--------|-----------------------|---------------------|-----------|----------------------------|-------------|--------------------------------------------|------------------------|
| `i915` | Intel iGPU            | submit, alloc, wait | validated | Intel UHD 620, `8086:5917` | 7.0, Ubuntu | k3s, containerd, cgroup v2, systemd driver | `ffmpeg`, `h264_vaapi` |
| `ivpu` | Intel NPU, Core Ultra | submit, alloc, wait | untested  | -                          | -           | -                                          | -                      |


## NVIDIA is not supported

CUDA does not go through `drm_ioctl`, so nothing KubeNPU does applies to it.
Use [DCGM exporter](https://github.com/NVIDIA/dcgm-exporter).

## Adding your hardware

Copy `pkg/hw/ivpu/` and fill in the numbers from the kernel's uapi header.

## License

Apache 2.0. The eBPF program under `bpf/` is GPL the helpers it uses are
GPL-only, and the program will not load otherwise.
