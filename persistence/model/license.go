package model

import "time"

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

// 状态枚举
const (
	StatusInactive = iota // 0 未激活
	StatusActive          // 1 已激活
	StatusExpired         // 2 已过期
	StatusRevoked         // 3 已吊销
)

// License 许可
type License struct {
	BaseModel
	LicenseKey    string     `gorm:"uniqueIndex;type:varchar(255);not null"` // 注册码
	ValidityHours int        `gorm:"type:int;not null"`                      // 有效时长（小时）
	ActivatedAt   *time.Time `gorm:"type:datetime"`                          // 激活时间
	ExpiredAt     *time.Time `gorm:"type:datetime"`                          // 过期时间
	Status        int        `gorm:"type:int;index;not null"`                // 状态枚举
	Remark        *string    `gorm:"type:text"`                              // 备注
}

func (License) TableName() string {
	return "license"
}

// LicenseScope 许可范围：支持的产品及限制
type LicenseScope struct {
	BaseModel
	LicenseID     uint   `gorm:"index;not null"`              // 对应 License.ID
	ProductID     uint   `gorm:"index;not null"`              // 对应 Product.ID
	MaxNodes      int    `gorm:"type:int;not null;default:0"` // 最大节点数 (0 = 不限制)
	MaxConcurrent int    `gorm:"type:int;not null;default:0"` // 并发限制 (0 = 不限制)
	FeatureMask   string `gorm:"type:varchar(255)"`           // 功能模块掩码
}

func (LicenseScope) TableName() string {
	return "license_scope"
}
