package dto

import (
	"errors"
)

// -------------------- Command --------------------

// CreateLicenseCommand 创建许可证的命令对象
// @Description Command to create a license
// @Tags License
type CreateLicenseCommand struct {
	ProductID     uint    `json:"product_id" binding:"required"`     // 授权范围列表
	ValidityHours int     `json:"validity_hours" binding:"required"` // 有效时长（小时）
	MaxNodes      int     `json:"max_nodes" binding:"required"`      // 最大节点数
	MaxConcurrent int     `json:"max_concurrent" binding:"required"` // 并发限制
	Remark        *string `json:"remark"`                            // 备注
}

// Validate 对 CreateLicenseCommand 做轻量校验，供 controller / service 使用
func (c *CreateLicenseCommand) Validate() error {
	if c == nil {
		return errors.New("command is nil")
	}
	if c.ValidityHours <= 0 {
		return errors.New("validity_hours must be > 0")
	}
	if c.MaxNodes < 0 {
		return errors.New("max_nodes must be >= 0")
	}
	if c.MaxConcurrent < 0 {
		return errors.New("max_concurrent must be >= 0")
	}
	if c.ProductID <= 0 {
		return errors.New("product_id must be > 0")
	}

	return nil
}

type LicenseData struct {
	ID            uint    `json:"id"`             // 许可证ID
	ProductID     uint    `json:"product_id"`     // 产品ID
	LicenseKey    string  `json:"license_key"`    // 许可证密钥
	ValidityHours int     `json:"validity_hours"` // 有效时长（小时）
	Status        int     `json:"status"`         // 状态
	Remark        *string `json:"remark"`         // 备注
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
