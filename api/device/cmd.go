package device

import (
	"nexus-core/domain/entity"
	"time"
)

//
// @Author yfy2001
// @Date 2025/7/23 13 28
//

type CreateDeviceCommand struct {
	ID            string `json:"id"`            //id
	Name          string `json:"name"`          //名称
	DeviceType    string `json:"deviceType"`    //设备类型
	Model         string `json:"model"`         //设备型号
	Description   string `json:"description"`   //备注
	ValidDuration int    `json:"validDuration"` //有效时长（小时）
}

func (c *CreateDeviceCommand) ToDomain() *entity.Device {
	return &entity.Device{
		ID:          c.ID,
		Name:        c.Name,
		DeviceType:  c.DeviceType,
		Model:       c.Model,
		Description: c.Description,
		Protocol:    "",
		IP:          "",
		Auth: &entity.Auth{
			CreatedAt:     time.Now(),
			ActivatedAt:   nil,
			ValidDuration: c.ValidDuration,
			ExpiredAt:     nil,
			Status:        0,
		},
	}
}
