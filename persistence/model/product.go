package model

import "time"

//
// @Author yfy2001
// @Date 2026/1/16 10 15
//

// Product 产品信息
type Product struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	Name        string    `gorm:"uniqueIndex;type:varchar(100);not null"` // 产品名称
	Description string    `gorm:"type:text"`                              // 产品描述
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (Product) TableName() string {
	return "product"
}

// ProductVersion 产品版本信息
type ProductVersion struct {
	ID          uint      `gorm:"primaryKey;autoIncrement"`
	ProductID   uint      `gorm:"index;not null"`            // 所属产品 Product.ID
	VersionCode string    `gorm:"type:varchar(50);not null"` // 版本号
	ReleaseDate time.Time `gorm:"type:datetime;not null"`    // 发布时间
	Description string    `gorm:"type:text"`                 // 版本说明
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (ProductVersion) TableName() string {
	return "product_version"
}
