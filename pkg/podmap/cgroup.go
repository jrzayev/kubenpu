package podmap

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var ErrNotPod = errors.New("cgroup is not a kubernetes pod")
var ErrNotIndexed = errors.New("cgroup not found in index")

type ToIDs struct {
	PodUID      string
	ContainerID string
}

type Index struct {
	byInode     map[uint64]string
	root        string
	lastUpdated time.Time
	interval    time.Duration
}

func parseCgroupPath(path string) (*ToIDs, error) {
	if !strings.Contains(path, "kubepods") {
		return nil, ErrNotPod
	}

	parts := strings.Split(filepath.ToSlash(path), "/")

	var podUID string
	var containerID string

	for _, part := range parts {
		if strings.HasPrefix(part, "kubepods-") &&
			strings.HasSuffix(part, ".slice") {

			const marker = "-pod"

			if _, after, ok := strings.Cut(part, marker); ok {
				pod := strings.TrimSuffix(
					after,
					".slice",
				)

				podUID = strings.ReplaceAll(pod, "_", "-")
			}
		}

		if !strings.HasSuffix(part, ".scope") {
			continue
		}

		part = strings.TrimSuffix(part, ".scope")

		idx := strings.LastIndex(part, "-")
		if idx == -1 {
			return nil, fmt.Errorf("invalid container scope: %q", part)
		}
		containerID = part[idx+1:]
	}

	if podUID == "" {
		return nil, fmt.Errorf("pod UID not found in %q", path)
	}

	if containerID == "" {
		return nil, fmt.Errorf("container ID not found in cgroup path %q", path)
	}

	return &ToIDs{
		PodUID:      podUID,
		ContainerID: containerID,
	}, nil
}

func buildCgroupInodeMap(root string) (map[uint64]string, error) {
	byInode := make(map[uint64]string)

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}

		byInode[stat.Ino] = path
		return nil
	})
	if err != nil {
		return nil, err
	}

	return byInode, nil
}

func BuildCgroupIndex(root string, interval time.Duration) (*Index, error) {
	byInode, err := buildCgroupInodeMap(root)
	if err != nil {
		return nil, err
	}

	return &Index{
		byInode:     byInode,
		root:        root,
		interval:    interval,
		lastUpdated: time.Now(),
	}, nil
}

func (idx *Index) RebuildCgroupIndex() (bool, error) {
	if time.Since(idx.lastUpdated) < idx.interval {
		return false, nil
	}

	byInode, err := buildCgroupInodeMap(idx.root)
	if err != nil {
		return false, err
	}
	idx.byInode = byInode
	idx.lastUpdated = time.Now()
	return true, nil
}

func (idx *Index) Lookup(inode uint64) (string, error) {
	path, ok := idx.byInode[inode]
	if !ok {
		return "", ErrNotIndexed
	}

	return path, nil
}

func (idx *Index) ParseCgroupToIDs(cgroupID uint64) (*ToIDs, error) {
	path, err := idx.Lookup(cgroupID)
	if err != nil {
		return nil, err
	}

	return parseCgroupPath(path)
}
