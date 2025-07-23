package repository

import (
	"nexus-core/domain/entity"
	"nexus-core/persistence/converter"
	"nexus-core/persistence/mapper"
)

//
// @Author yfy2001
// @Date 2025/7/22 10 16
//

type DeviceRepository struct {
	dm *mapper.DeviceMapper
	am *mapper.AuthMapper
	dc *converter.DeviceConverter
}

func NewDeviceRepository() *DeviceRepository {
	return &DeviceRepository{
		dm: mapper.NewDeviceMapper(),
		am: mapper.NewAuthMapper(),
	}
}

func (dr *DeviceRepository) SaveOrUpdate(e *entity.Device) {
	d, a := dr.dc.ToPersistence(e)
	dr.dm.SaveOrUpdate(d)
	dr.am.SaveOrUpdate(a)
}
