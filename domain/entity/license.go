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

// LicenseStatus 定义许可证状态枚举类型
type LicenseStatus int

const (
	StatusInactive LicenseStatus = iota // 0 未激活
	StatusActive                        // 1 已激活
	StatusExpired                       // 2 已过期
	StatusRevoked                       // 3 已吊销
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
	Status        LicenseStatus
	Remark        *string // 备注信息
	MaxNodes      int     // 最大节点数 (0 = 不限制)
	MaxConcurrent int     // 并发限制 (0 = 不限制)
	FeatureMask   string  // 功能模块掩码
}

// NewLicense 工厂方法
// 创建一个新的许可证对象，默认状态为未激活
func NewLicense(productID uint, validityHours, maxNodes, concurrentLimit int, remark *string) *License {
	if validityHours <= 0 {
		validityHours = 0
	}

	return &License{
		ProductID:     productID,
		LicenseKey:    strings.ReplaceAll(uuid.New().String(), "-", ""),
		ValidityHours: validityHours,
		IssuedAt:      time.Now(),
		Status:        StatusInactive,
		MaxNodes:      maxNodes,
		MaxConcurrent: concurrentLimit,
		Remark:        remark,
	}
}

// CalculateStatus 根据当前时间返回状态
func (l *License) CalculateStatus(now time.Time) LicenseStatus {
	switch l.Status {
	case StatusInactive:
		return StatusInactive
	case StatusActive, StatusExpired:
		if l.ExpiredAt != nil && now.After(*l.ExpiredAt) {
			return StatusExpired
		} else {
			return StatusActive
		}
	case StatusRevoked:
		return StatusRevoked
	}
	return StatusInactive
}

// IsActive 检查许可证是否处于激活状态
func (l *License) IsActive() bool {
	return l.CalculateStatus(time.Now()) == StatusActive
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

	if l.ActivatedAt == nil {
		l.ActivatedAt = &now
	}

	expired := now.Add(time.Duration(l.ValidityHours) * time.Hour)
	l.ExpiredAt = &expired
	l.Status = StatusActive
	return nil
}

// Renew 续期或缩短许可证
// 根据 extraHours 参数增加或减少许可证的有效期
func (l *License) Renew(now time.Time, extraHours int) {
	l.ValidityHours += extraHours
	if l.ValidityHours < 0 {
		l.ValidityHours = 0
	}

	if l.Status == StatusInactive {
		return
	}

	if l.ExpiredAt == nil || now.After(*l.ExpiredAt) {
		expired := now.Add(time.Duration(l.ValidityHours) * time.Hour)
		l.ExpiredAt = &expired
	} else {
		expired := l.ExpiredAt.Add(time.Duration(extraHours) * time.Hour)
		l.ExpiredAt = &expired
	}

	l.Status = l.CalculateStatus(now)
}

// Revoke 吊销许可证
// 将许可证状态设置为已吊销，使其立即失效
func (l *License) Revoke() {
	l.Status = StatusRevoked
}

// UnRevoke UnRevoke恢复
func (l *License) UnRevoke() {
	//先设为激活状态，然后重新计算
	l.Status = StatusActive
	l.Status = l.CalculateStatus(time.Now())
}

// IsValid 是否有效
func (l *License) IsValid() bool {
	return l.CalculateStatus(time.Now()) == StatusActive
}

// ValidateMaxNodes 验证最大节点数限制
func (l *License) ValidateMaxNodes(currentBindings int) bool {
	return validateLimit(l.MaxNodes, currentBindings)
}

// ValidateMaxConcurrent 验证最大并发数限制
func (l *License) ValidateMaxConcurrent(currentConcurrent int) bool {
	return validateLimit(l.MaxConcurrent, currentConcurrent)
}

// 通用限制验证函数
func validateLimit(limit, current int) bool {
	return limit == 0 || current < limit
}
