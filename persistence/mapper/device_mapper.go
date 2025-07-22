package mapper

import (
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"
)

//
// @Author yfy2001
// @Date 2025/7/22 10 20
//

type DeviceMapper struct {
	bm *base.Mapper[model.Device]
}

func NewDeviceMapper() *DeviceMapper {
	return &DeviceMapper{
		bm: base.NewMapper[model.Device](),
	}
}
