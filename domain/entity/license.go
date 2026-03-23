package entity

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

//
// @Author yfy2001
// @Date 2026/1/16 15 42
//

// LicenseStatus 定义许可证状态枚举
const (
	StatusInactive = iota // 0 未激活
	StatusActive          // 1 已激活
	StatusExpired         // 2 已过期
	StatusRevoked         // 3 已吊销
)

// License 表示许可证领域的核心实体
// 包含许可证的基本信息、激活状态、有效期和授权范围
type License struct {
	ID            uint
	ProductID     uint       // 产品id
	LicenseKey    string     // 许可证密钥，用于客户端验证
	ValidityHours int        // 有效时长（小时），从激活时刻开始计算
	IssuedAt      time.Time  // 颁发时间，许可证创建时设置
	ActivatedAt   *time.Time // 激活时间，首次激活时设置
	ExpiredAt     *time.Time // 过期时间，基于激活时间和有效时长计算
	Status        int        // 当前状态，使用LicenseStatus枚举值
	Remark        *string    // 备注信息
	MaxNodes      int        // 最大节点数 (0 = 不限制)
	MaxConcurrent int        // 并发限制 (0 = 不限制)
	FeatureMask   string     // 功能模块掩码
}

// NewLicense 工厂方法
// 创建一个新的许可证对象，默认状态为未激活
func NewLicense(productID uint, validityHours int, maxNodes int, concurrentLimit int, remark *string) (*License, error) {
	if validityHours <= 0 {
		return nil, fmt.Errorf("validity hours must be positive")
	}

	license := &License{
		ProductID:     productID,
		LicenseKey:    strings.ReplaceAll(uuid.New().String(), "-", ""),
		ValidityHours: validityHours,
		IssuedAt:      time.Now(),
		Status:        StatusInactive, // 初始状态必须是未激活
		MaxNodes:      maxNodes,
		MaxConcurrent: concurrentLimit,
		Remark:        remark,
	}

	return license, nil
}

func (l *License) IsActive() bool {
	return l.CheckStatus(time.Now()) == StatusActive
}

// Activate 激活许可证
// 将许可证从未激活状态转为激活状态，并设置激活时间和过期时间
func (l *License) Activate(now time.Time) error {
	if l.Status != StatusInactive {
		return fmt.Errorf("license cannot be activated from status %d", l.Status)
	}
	if l.ValidityHours <= 0 {
		return fmt.Errorf("validity hours must be positive")
	}

	// 只在第一次激活时设置 ActivatedAt
	if l.ActivatedAt == nil {
		l.ActivatedAt = &now
	}

	expired := now.Add(time.Duration(l.ValidityHours) * time.Hour)
	l.ExpiredAt = &expired
	l.Status = StatusActive
	return nil
}

// Renew 续期或缩短许可证
// 根据extraHours参数增加或减少许可证的有效期
func (l *License) Renew(now time.Time, extraHours int) error {
	if l.Status == StatusRevoked {
		return fmt.Errorf("license %s has been revoked and cannot be renewed", l.LicenseKey)
	}

	// 如果已过期，恢复为 Active，但不修改 ActivatedAt
	if l.Status == StatusExpired {
		l.Status = StatusActive
	}

	// 调整过期时间
	if l.ExpiredAt == nil || now.After(*l.ExpiredAt) {
		expired := now.Add(time.Duration(extraHours) * time.Hour)
		l.ExpiredAt = &expired
	} else {
		expired := l.ExpiredAt.Add(time.Duration(extraHours) * time.Hour)
		l.ExpiredAt = &expired
	}

	// 更新总有效时长
	l.ValidityHours += extraHours
	if l.ValidityHours < 0 {
		l.ValidityHours = 0
	}

	// 如果缩短后已经过期，更新状态
	if l.ExpiredAt != nil && now.After(*l.ExpiredAt) {
		l.Status = StatusExpired
	}

	return nil
}

// Revoke 吊销许可证
// 将许可证状态设置为已吊销，使其立即失效
func (l *License) Revoke(now time.Time) bool {
	if l.Status == StatusRevoked {
		return false
	}
	l.ExpiredAt = &now
	l.Status = StatusRevoked
	return true
}

// IsExpired 检查许可证是否已过期
// 根据当前时间和过期时间判断
func (l *License) IsExpired(now time.Time) bool {
	if l.ExpiredAt == nil {
		return false
	}
	return now.After(*l.ExpiredAt)
}

// CheckStatus 自动检查并更新许可证状态
// 如果许可证处于活动状态且已过期，则将其状态更新为过期
func (l *License) CheckStatus(now time.Time) int {
	if l.Status == StatusActive && l.ExpiredAt != nil && now.After(*l.ExpiredAt) {
		l.Status = StatusExpired
	}
	return l.Status
}

// ValidateMaxNodesForProduct 验证许可证特定产品授权中的最大节点数
func (l *License) ValidateMaxNodesForProduct(currentBindings int) bool {
	if l.MaxNodes > 0 && currentBindings >= l.MaxNodes {
		return false
	}
	return true
}

// ValidateMaxConcurrentForProduct 验证许可证特定产品授权中的最大并发数
func (l *License) ValidateMaxConcurrentForProduct(currentConcurrent int) bool {
	if l.MaxConcurrent > 0 && currentConcurrent >= l.MaxConcurrent {
		return false
	}
	return true
}
