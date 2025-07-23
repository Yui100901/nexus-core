package converter

import (
	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
)

//
// @Author yfy2001
// @Date 2025/7/23 08 48
//

type DeviceConverter struct {
}

func NewDeviceConverter() *DeviceConverter {
	return &DeviceConverter{}
}

func (dc *DeviceConverter) ToPersistence(e *entity.Device) (*model.Device, *model.Auth) {
	if e == nil {
		return nil, nil
	}
	d := &model.Device{
		ID:          e.ID,
		Name:        e.Name,
		DeviceType:  e.DeviceType,
		Model:       e.Model,
		Description: e.Description,
		Protocol:    e.Protocol,
		IP:          e.IP,
	}
	if e.Auth == nil {
		return d, nil
	}
	a := &model.Auth{
		DeviceId:      e.ID,
		CreatedAt:     e.Auth.CreatedAt,
		ActivatedAt:   e.Auth.ActivatedAt,
		ValidDuration: e.Auth.ValidDuration,
		ExpiredAt:     e.Auth.ExpiredAt,
		Status:        e.Auth.Status.Int(),
	}
	return d, a
}

func (dc *DeviceConverter) ToDomain(d *model.Device, a *model.Auth) *entity.Device {
	if d == nil {
		return nil
	}
	e := &entity.Device{
		ID:          d.ID,
		Name:        d.Name,
		DeviceType:  d.DeviceType,
		Model:       d.Model,
		Description: d.Description,
		Protocol:    d.Protocol,
		IP:          d.IP,
		Auth:        nil,
	}
	if a == nil {
		return e
	}
	e.Auth = &entity.Auth{
		CreatedAt:     a.CreatedAt,
		ActivatedAt:   a.ActivatedAt,
		ValidDuration: a.ValidDuration,
		ExpiredAt:     a.ExpiredAt,
		Status:        entity.AuthStatus(a.Status),
	}
	return e
}
