//
// Created by Javid Rzayev on 10.08.26.
//

package ivpu

import "github.com/jrzayev/kubenpu/pkg/hw"

func (v *Vendor) Ioctls() map[uint32]hw.Kind {
	return map[uint32]hw.Kind{
		// For this vendor we use both DRM_IVPU_SUBMIT and DRM_IVPU_CMDQ_SUBMIT because kernel still supports both
		// and userspace will choose based on DRM_IVPU_CAP_MANAGE_CMDQ
		0x4d: hw.KindSubmit, // DRM_IVPU_CMDQ_SUBMIT
		0x45: hw.KindSubmit, // DRM_IVPU_SUBMIT
		0x42: hw.KindAlloc,  // DRM_IVPU_BO_CREATE
		0x46: hw.KindWait,   // DRM_IVPU_BO_WAIT
	}
}
