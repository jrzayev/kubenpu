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
	isKubepods := false
	for i, part := range parts {
		if !isKubepods {
			if !strings.HasPrefix(part, "kubepods") {
				continue
			}
			isKubepods = true
		}
		podUID, ok := podUIDFromPart(part)
		if !ok {
			continue
		}
		if i+1 >= len(parts) || parts[i+1] == "" {
			return nil, fmt.Errorf("container ID not found in cgroup path %q", path)
		}
		containerID, err := containerIDFromPart(parts[i+1])
		if err != nil {
			return nil, fmt.Errorf("%w in cgroup path %q", err, path)
		}
		return &ToIDs{PodUID: podUID, ContainerID: containerID}, nil
	}
	return nil, fmt.Errorf("pod UID not found in %q", path)
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

func podUIDFromPart(part string) (string, bool) {
	if strings.HasPrefix(part, "kubepods-") && strings.HasSuffix(part, ".slice") {
		_, uid, ok := strings.Cut(strings.TrimSuffix(part, ".slice"), "-pod")
		if !ok || uid == "" {
			return "", false
		}
		return strings.ReplaceAll(uid, "_", "-"), true
	}

	if uid, ok := strings.CutPrefix(part, "pod"); ok && uid != "" {
		return uid, true
	}

	return "", false
}

func containerIDFromPart(part string) (string, error) {
	id := strings.TrimSuffix(part, ".scope")
	if idx := strings.LastIndex(id, "-"); idx != -1 {
		id = id[idx+1:]
	}

	if !isHex(id) {
		return "", fmt.Errorf("invalid container cgroup %q", part)
	}

	return id, nil
}

func isHex(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}

	return true
}
