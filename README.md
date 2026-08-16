# KubeNPU

**Per-pod visibility into NPU and accelerator usage on Kubernetes.**

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
