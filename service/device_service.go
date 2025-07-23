package service

import (
	"nexus-core/domain/entity"
	"nexus-core/persistence/repository"
)

//
// @Author yfy2001
// @Date 2025/7/22 10 33
//

type DeviceService struct {
	dr *repository.DeviceRepository
}

func NewDeviceService() *DeviceService {
	return &DeviceService{
		dr: repository.NewDeviceRepository(),
	}
}

func (ds *DeviceService) CreateDevice(e *entity.Device) {
	ds.dr.SaveOrUpdate(e)
}
