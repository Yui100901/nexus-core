package model

import "time"

//
// @Author yfy2001
// @Date 2026/3/26 15 55
//

const (
	VersionStatusUnreleased = 0 // 0 未发布
	VersionStatusAvailable  = 1 // 1 可用
	VersionStatusDeprecated = 2 // 2 已经弃用
)

// ProductVersion 产品版本信息
type ProductVersion struct {
	BaseModel
	ProductID   uint       `gorm:"index;not null"`            // 所属产品 Product.ID
	VersionCode string     `gorm:"type:varchar(50);not null"` // 版本号,产品内唯一
	ReleaseDate *time.Time `gorm:"type:datetime"`             // 发布时间
	Description *string    `gorm:"type:text"`                 // 版本说明
	Status      int        `gorm:"type:int;index;not null"`   // 是否启用
}

func (ProductVersion) TableName() string {
	return "product_version"
}
