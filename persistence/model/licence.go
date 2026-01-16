package model

import "time"

//
// @Author yfy2001
// @Date 2026/1/16 10 16
//

// 授权状态枚举
const (
	StatusInactive = iota // 0 未激活
	StatusActive          // 1 已激活
	StatusExpired         // 2 已过期
	StatusRevoked         // 3 已吊销
)

// License 授权许可
type License struct {
	ID            uint       `gorm:"primaryKey;autoIncrement"`
	LicenseKey    string     `gorm:"uniqueIndex;type:varchar(255);not null"` // 注册码
	ValidityHours int        `gorm:"type:int;not null"`                      // 有效时长（小时）
	ValidFrom     *time.Time `gorm:"type:datetime"`                          // 有效期开始
	ValidUntil    *time.Time `gorm:"type:datetime"`                          // 有效期结束
	Status        int        `gorm:"type:int;index;not null"`                // 状态枚举
	ActivatedAt   *time.Time `gorm:"type:datetime"`
	ExpiredAt     *time.Time `gorm:"type:datetime"`
	CreatedAt     time.Time  `gorm:"autoCreateTime"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime"`
}

func (License) TableName() string {
	return "license"
}

// LicenseScope 授权范围：支持的产品及限制
type LicenseScope struct {
	ID              uint      `gorm:"primaryKey;autoIncrement"`
	LicenseID       uint      `gorm:"index;not null"`              // 对应 License.ID
	ProductID       uint      `gorm:"index;not null"`              // 对应 Product.ID
	FeatureMask     string    `gorm:"type:varchar(255)"`           // 功能模块掩码
	MaxNodes        int       `gorm:"type:int;not null;default:0"` // 最大节点数 (0 = 不限制)
	ConcurrentLimit int       `gorm:"type:int;not null;default:0"` // 并发限制 (0 = 不限制)
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

func (LicenseScope) TableName() string {
	return "license_scope"
}
