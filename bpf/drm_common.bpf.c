//
// Created by Javid Rzayev on 10.08.26.
//

#include "vmlinux.h"
#include "common.h"
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#define MAX_ENTRIES_DEVICES	32 // Avg 8 or 16 devices in single node so 32 will be enough
#define MAX_ENTRIES_IOCTLS	640 // Avg 10 or 15 vendors so 640 (20 * 32) will be enough
#define MAX_ENTRIES_EVENTS	(256 * 1024)

char LICENSE[] SEC("license") = "GPL";

const struct event *unused_event __attribute__((unused));

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, MAX_ENTRIES_DEVICES);
  __type(key, struct device_key);
  __type(value, __u32);
} devices SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_HASH);
  __uint(max_entries, MAX_ENTRIES_IOCTLS);
  __type(key, struct ioctl_key);
  __type(value, __u8);
} ioctls SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_RINGBUF);
  __uint(max_entries, MAX_ENTRIES_EVENTS);
} events SEC(".maps");

struct {
  __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
  __uint(max_entries, 1);
  __type(key, __u32);
  __type(value, __u64);
} dropped SEC(".maps");


SEC("fentry/drm_ioctl")
int BPF_PROG(kubenpu_ioctl, struct file *filp, unsigned int cmd, unsigned long arg) {
  __u64 rdev = filp->f_inode->i_rdev;
  __u32 minor = rdev & 0xFFFFF;
  __u32 major = rdev >> 20;

  struct device_key dkey = {.major = major, .minor = minor};

  __u32 *devices_data = bpf_map_lookup_elem(&devices, &dkey);
  if (!devices_data) {
    return 0;
  }

  __u32 nr = cmd & 0xFF;

  struct ioctl_key ikey = {.vendor_id = *devices_data, .nr = nr};
  __u8 *ioctls_data = bpf_map_lookup_elem(&ioctls, &ikey);
  if (!ioctls_data) {
    return 0;
  }

  struct event* poll_event = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
  if (!poll_event) {
    __u32 zero = 0;
    __u64 *lost = bpf_map_lookup_elem(&dropped, &zero);
    if (lost) {
      *lost += 1;
    }
    return 0;
  }

  poll_event->timestamp = bpf_ktime_get_tai_ns();
  poll_event->cgroup_id = bpf_get_current_cgroup_id();
  poll_event->tgid = bpf_get_current_pid_tgid() >> 32;
  poll_event->cmd = cmd;
  poll_event->minor = minor;
  poll_event->major = major;
  poll_event->kind = *ioctls_data;
  poll_event->_pad[0] = 0;
  bpf_ringbuf_submit(poll_event, 0);
  return 0;
}
