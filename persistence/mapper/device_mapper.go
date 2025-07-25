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
	*base.Mapper[model.Device]
}

func NewDeviceMapper() *DeviceMapper {
	return &DeviceMapper{
		Mapper: base.NewMapper[model.Device](),
	}
}
