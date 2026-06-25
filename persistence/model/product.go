package model

import "gorm.io/datatypes"

//
// @Author yfy2001
// @Date 2026/1/16 10 15
//

// Product 产品信息
type Product struct {
	BaseModel
	Name                  string         `gorm:"uniqueIndex;type:varchar(100);not null"` // 产品名称
	Description           *string        `gorm:"type:text"`                              // 产品描述
	Status                int            `gorm:"type:int;index;not null;default:1"`      // 状态：1启用，2禁用，3废弃
	MinSupportedVersionID *uint          `gorm:"index"`                                  // 最低支持版本
	FeatureList           datatypes.JSON `gorm:"type:json"`                              // 兼容旧字段，后续迁移至服务/功能关联表
}

func (Product) TableName() string {
	return "product"
}
