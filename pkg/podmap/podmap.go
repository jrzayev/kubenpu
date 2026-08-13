//
// Created by Javid Rzayev on 12.08.26.
//

package podmap

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type PodMap struct {
	index *Index
	cri   *CRIClient

	mu    sync.RWMutex
	cache map[uint64]ContainerInfo
}

func NewPodMap(cgroupRoot string, criSocket string, interval time.Duration) (*PodMap, error) {
	if cgroupRoot == "" {
		return nil, errors.New("cgroup root is empty")
	}

	if criSocket == "" {
		return nil, errors.New("CRI socket is empty")
	}

	if interval <= 0 {
		return nil, errors.New("interval must be positive")
	}

	index, err := BuildCgroupIndex(cgroupRoot, interval)
	if err != nil {
		return nil, fmt.Errorf("build cgroup index: %w", err)
	}

	cri, err := NewCRIClient(criSocket)
	if err != nil {
		return nil, fmt.Errorf("create CRI client: %w", err)
	}

	return &PodMap{
		index: index,
		cri:   cri,
		cache: make(map[uint64]ContainerInfo),
	}, nil
}

func (p *PodMap) ContainerInfo(cgroupID uint64) (ContainerInfo, error) {
	if cgroupID == 0 {
		return ContainerInfo{}, errors.New("cgroup ID is empty")
	}

	p.mu.RLock()
	info, ok := p.cache[cgroupID]
	p.mu.RUnlock()

	if ok {
		return info, nil
	}

	ids, err := p.index.ParseCgroupToIDs(cgroupID)
	if errors.Is(err, ErrNotIndexed) {
		rebuilt, rerr := p.index.RebuildCgroupIndex()
		if rerr != nil {
			return ContainerInfo{}, fmt.Errorf("rebuild cgroup index: %w", rerr)
		}
		if !rebuilt {
			return ContainerInfo{}, err
		}

		p.mu.Lock()
		clear(p.cache)
		p.mu.Unlock()

		ids, err = p.index.ParseCgroupToIDs(cgroupID)
		if err != nil {
			return ContainerInfo{}, fmt.Errorf("parse cgroup %d after rebuild: %w", cgroupID, err)
		}
	} else if err != nil {
		return ContainerInfo{}, fmt.Errorf("parse cgroup %d: %w", cgroupID, err)
	}

	info, err = p.cri.ContainerNames(ids.ContainerID)
	if err != nil {
		return ContainerInfo{}, fmt.Errorf("get container info for cgroup %d: %w", cgroupID, err)
	}

	p.mu.Lock()
	p.cache[cgroupID] = info
	p.mu.Unlock()

	return info, nil
}

func (p *PodMap) Close() error {
	if p == nil || p.cri == nil {
		return nil
	}

	return p.cri.Close()
}
