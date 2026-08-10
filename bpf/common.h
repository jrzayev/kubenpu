//
// Created by Javid Rzayev on 10.08.26.
//

#ifndef KUBENPU_COMMON_H
#define KUBENPU_COMMON_H
#include "vmlinux.h"

#define KindUnknown 0
#define KindSubmit 1
#define KindAlloc 2
#define KindWait 3

struct device_key {
  __u32  major;
  __u32  minor;
};

struct ioctl_key {
  __u32  vendor_id;
  __u32  nr;
};

struct event {
  __u64  timestamp;
  __u64  cgroup_id;
  __u32  tgid;
  __u32  cmd;
  __u32  minor;
  __u16  major;
  __u8   kind;
  __u8   _pad[1];
};

#endif // KUBENPU_COMMON_H
