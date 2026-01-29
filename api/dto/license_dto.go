package dto

import (
	"errors"
	"fmt"

	"nexus-core/domain/entity"
)

// -------------------- Command --------------------

// CreateLicenseCommand 创建许可证的命令对象
// @Description Command to create a license
// @Tags License
type CreateLicenseCommand struct {
	ValidityHours   int        `json:"validity_hours" binding:"required"`   // 有效时长（小时）
	MaxNodes        int        `json:"max_nodes" binding:"required"`        // 最大节点数
	ConcurrentLimit int        `json:"concurrent_limit" binding:"required"` // 并发限制
	ScopeList       []ScopeDTO `json:"scope_list" binding:"required"`       // 授权范围列表
	Remark          *string    `json:"remark"`                              // 备注
}

// Validate 对 CreateLicenseCommand 做轻量校验，供 controller / service 使用
func (c *CreateLicenseCommand) Validate() error {
	if c == nil {
		return errors.New("command is nil")
	}
	if c.ValidityHours <= 0 {
		return errors.New("validity_hours must be > 0")
	}
	if len(c.ScopeList) == 0 {
		return errors.New("scope_list is required and must contain at least one scope")
	}
	for i := range c.ScopeList {
		if err := c.ScopeList[i].Validate(); err != nil {
			return fmt.Errorf("scope_list[%d]: %w", i, err)
		}
	}
	return nil
}

// UpdateLicenseCommand 更新许可证的命令对象
// @Description Command to update a license
type UpdateLicenseCommand struct {
	ID            uint    `json:"id" binding:"required"`          // 许可证ID
	LicenseKey    string  `json:"license_key" binding:"required"` // 许可证密钥
	ValidityHours int     `json:"validity_hours"`                 // 有效时长（小时）
	Status        int     `json:"status"`                         // 状态
	Remark        *string `json:"remark"`                         // 备注
}

// UpdateLicenseStatusCommand 更新许可证状态的命令对象
type UpdateLicenseStatusCommand struct {
	ID     uint `json:"id" binding:"required"`     // 许可证ID
	Status int  `json:"status" binding:"required"` // 新状态
}

// -------------------- Query --------------------

// GetLicenseByIDQuery 按ID查询许可证的查询对象
// @Description Query by license ID
type GetLicenseByIDQuery struct {
	ID uint `form:"id" binding:"required"` // 许可证ID
}

// GetLicenseByKeyQuery 按密钥查询许可证的查询对象
// @Description Query by license key
type GetLicenseByKeyQuery struct {
	Key string `form:"key" binding:"required"` // 许可证密钥
}

// -------------------- DTO 辅助 --------------------

// ScopeDTO 用于传输授权范围的DTO对象
// @Description Scope transfer object
type ScopeDTO struct {
	ProductID   uint   `json:"product_id" binding:"required"` // 产品ID
	FeatureMask string `json:"feature_mask"`                  // 功能模块掩码
}

// Validate 对 ScopeDTO 做轻量校验
func (s *ScopeDTO) Validate() error {
	if s == nil {
		return errors.New("scope is nil")
	}
	if s.ProductID == 0 {
		return errors.New("product_id is required and must be > 0")
	}
	return nil
}

// ToEntityScopes 将DTO对象列表转换为领域对象列表
// 注意：为了最小改动，保留当前映射函数（controller/service 仍然依赖 entity.Scope）。
func ToEntityScopes(scopeDTOs []ScopeDTO) []entity.Scope {
	var scopes []entity.Scope
	for _, s := range scopeDTOs {
		scopes = append(scopes, entity.Scope{
			ProductID:   s.ProductID,
			FeatureMask: s.FeatureMask,
		})
	}
	return scopes
}
