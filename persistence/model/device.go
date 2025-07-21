package model

//
// @Author yfy2001
// @Date 2025/7/21 14 53
//

// 设备表

type Device struct {
	ID          string `gorm:"primaryKey;type:varchar(100)"` //id
	Name        string //名称
	DeviceType  string //设备类型
	Model       string //设备型号
	Description string //备注
	Protocol    string //网络协议
	IP          string //设备ip
}

func (Device) TableName() string {
	return "device"
}
