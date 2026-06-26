package model

import (
	"time"

	"gorm.io/datatypes"
)

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

// Node 节点信息
type Node struct {
	BaseModel
	DeviceCode          string         `gorm:"type:varchar(100);uniqueIndex;not null"` // 设备唯一识别码
	Status              int            `gorm:"type:int;index;not null;default:0"`      // 状态：0正常，1离线，2封禁，3强制下线
	Metadata            datatypes.JSON `gorm:"type:json"`                              // 其他元信息
	LastSeenAt          *time.Time     `gorm:"type:datetime;index"`                    // 最近心跳时间
	OnlineAt            *time.Time     `gorm:"type:datetime"`                          // 最近上线时间
	OfflineAt           *time.Time     `gorm:"type:datetime"`                          // 最近离线时间
	BannedAt            *time.Time     `gorm:"type:datetime"`                          // 封禁时间
	BanReason           *string        `gorm:"type:text"`                              // 封禁原因
	ForcedOfflineAt     *time.Time     `gorm:"type:datetime"`                          // 强制下线时间
	ForcedOfflineReason *string        `gorm:"type:text"`                              // 强制下线原因
}

func (Node) TableName() string {
	return "node"
}
