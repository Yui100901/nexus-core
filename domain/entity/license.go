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
	LicenseKey    string     // 许可证密钥，用于客户端验证
	ValidityHours int        // 有效时长（小时），从激活时刻开始计算
	ActivatedAt   *time.Time // 激活时间，首次激活时设置
	ExpiredAt     *time.Time // 过期时间，基于激活时间和有效时长计算
	Status        int        // 当前状态，使用LicenseStatus枚举值
	Remark        *string    // 备注信息
	ScopeList     []Scope    // 授权范围列表，定义了许可证对产品的使用权限
}

// NewLicense 工厂方法
// 创建一个新的许可证对象，默认状态为未激活
func NewLicense(validityHours int, maxNodes int, concurrentLimit int, remark *string, scopes []Scope) (*License, error) {
	if validityHours <= 0 {
		return nil, fmt.Errorf("validity hours must be positive")
	}

	license := &License{
		LicenseKey:    strings.ReplaceAll(uuid.New().String(), "-", ""),
		ValidityHours: validityHours,
		Status:        StatusInactive, // 初始状态必须是未激活
		Remark:        remark,
		ScopeList:     scopes,
	}

	return license, nil
}

// Scope 定义许可证对特定产品的授权范围
// 包括功能模块掩码、节点数限制和并发限制
type Scope struct {
	ID            uint
	ProductID     uint   // 关联的产品ID，指向Product实体
	MaxNodes      int    `gorm:"type:int;not null;default:0"` // 最大节点数 (0 = 不限制)
	MaxConcurrent int    `gorm:"type:int;not null;default:0"` // 并发限制 (0 = 不限制)
	FeatureMask   string // 功能模块掩码，用于控制功能模块访问权限
}

func NewScope(productID uint, maxNodes, concurrentLimit int, featureMask string) *Scope {
	return &Scope{
		ProductID:     productID,
		MaxNodes:      maxNodes,
		MaxConcurrent: concurrentLimit,
		FeatureMask:   featureMask,
	}
}

// GetScopeProductIdList 获取授权范围中的产品ID列表
func (l *License) GetScopeProductIdList() []uint {
	ids := make([]uint, 0, len(l.ScopeList))
	for _, p := range l.ScopeList {
		ids = append(ids, p.ID)
	}
	return ids
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

// AddScope 添加授权范围
// 为许可证添加对特定产品的授权
func (l *License) AddScope(scope Scope) bool {
	for _, s := range l.ScopeList {
		if s.ProductID == scope.ProductID {
			return false
		}
	}
	l.ScopeList = append(l.ScopeList, scope)
	return true
}

// UpdateScope 更新特定产品的授权范围
// 根据产品ID找到对应的授权范围并替换为新的授权范围
func (l *License) UpdateScope(productID uint, newScope Scope) bool {
	for i, s := range l.ScopeList {
		if s.ProductID == productID {
			l.ScopeList[i] = newScope
			return true
		}
	}
	return false
}

// RemoveScope 删除特定产品的授权范围
// 根据产品ID移除对应的授权范围
func (l *License) RemoveScope(productID uint) bool {
	for i, s := range l.ScopeList {
		if s.ProductID == productID {
			l.ScopeList = append(l.ScopeList[:i], l.ScopeList[i+1:]...)
			return true
		}
	}
	return false
}

// ValidateScope 验证对特定产品的访问权限
// 检查许可证的授权范围中是否包含指定的产品ID
func (l *License) ValidateScope(productID uint) bool {
	for _, s := range l.ScopeList {
		if s.ProductID == productID {
			return true
		}
	}
	return false
}
