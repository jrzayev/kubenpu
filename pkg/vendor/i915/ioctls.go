package i915

import "github.com/jrzayev/kubenpu/pkg/vendor"

func (v *Vendor) Ioctls() map[uint32]vendor.Kind {
	return map[uint32]vendor.Kind{
		0x69: vendor.KindSubmit, // DRM_I915_GEM_EXECBUFFER2
		0x5b: vendor.KindAlloc,  // DRM_I915_GEM_CREATE
		0x6c: vendor.KindWait,   // DRM_I915_GEM_WAIT
	}
}
