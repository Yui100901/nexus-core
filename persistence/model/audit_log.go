package model

import "gorm.io/datatypes"

// AuditLog 记录产品、License、节点、绑定和控制指令的关键操作。
type AuditLog struct {
	BaseModel
	ResourceType string         `gorm:"type:varchar(50);index;not null"`
	ResourceID   uint           `gorm:"index;not null;default:0"`
	Action       string         `gorm:"type:varchar(100);index;not null"`
	Operator     string         `gorm:"type:varchar(100);index;not null;default:system"`
	Data         datatypes.JSON `gorm:"type:json"`
}

func (AuditLog) TableName() string {
	return "audit_log"
}
