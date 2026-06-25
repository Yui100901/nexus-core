package model

import "time"

//
// @Author yfy2001
// @Date 2026/3/26 15 55
//

// ProductVersion 产品版本信息
type ProductVersion struct {
	BaseModel
	ProductID   uint       `gorm:"uniqueIndex:idx_product_version_code;index;not null"` // 所属产品 Product.ID
	VersionCode string     `gorm:"uniqueIndex:idx_product_version_code;type:varchar(50);not null"`
	ReleaseDate *time.Time `gorm:"type:datetime"`                                    // 发布时间
	Description *string    `gorm:"type:text"`                                        // 版本说明
	Status      int        `gorm:"type:int;index;not null;default:0"`                // 状态：0未发布，1可用，2废弃
	IsEnabled   bool       `json:"-" gorm:"column:is_enabled;not null;default:true"` // 兼容旧 SQLite 表，业务逻辑统一使用 Status
}

func (ProductVersion) TableName() string {
	return "product_version"
}
