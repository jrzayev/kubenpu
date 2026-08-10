//
// Created by Javid Rzayev on 10.08.26.
//

package i915

import "github.com/jrzayev/kubenpu/pkg/hw"

func (v *Vendor) Ioctls() map[uint32]hw.Kind {
	return map[uint32]hw.Kind{
		0x69: hw.KindSubmit, // DRM_I915_GEM_EXECBUFFER2
		0x5b: hw.KindAlloc,  // DRM_I915_GEM_CREATE
		0x6c: hw.KindWait,   // DRM_I915_GEM_WAIT
	}
}
