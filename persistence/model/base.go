package model

import (
	"time"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/16 17 13
//

type BaseModel struct {
	ID        uint           `gorm:"primary_key;auto_increment"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;index;not null"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;index;not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	CreatedBy string         `gorm:"size:64"` // 记录创建人
	UpdatedBy string         `gorm:"size:64"` // 记录修改人
}

func (m *BaseModel) BeforeCreate(tx *gorm.DB) (err error) {
	// 假设从上下文获取当前用户
	m.CreatedBy = "system"
	m.UpdatedBy = "system"
	return
}

func (m *BaseModel) BeforeUpdate(tx *gorm.DB) (err error) {
	m.UpdatedBy = "system"
	return
}
