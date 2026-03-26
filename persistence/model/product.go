package model

import "gorm.io/datatypes"

//
// @Author yfy2001
// @Date 2026/1/16 10 15
//

const (
	VersionStatusUnreleased = 0 // 0 未发布
	VersionStatusAvailable  = 1 // 1 可用
	VersionStatusDeprecated = 2 // 2 已经弃用
)

// Product 产品信息
type Product struct {
	BaseModel
	Name                  string         `gorm:"uniqueIndex;type:varchar(100);not null"` // 产品名称
	Description           *string        `gorm:"type:text"`                              // 产品描述
	MinSupportedVersionID *uint          `gorm:"index"`                                  // 最低支持版本
	FeatureList           datatypes.JSON `gorm:"type:json"`                              // json保存功能列表
}

func (Product) TableName() string {
	return "product"
}
