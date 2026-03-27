package dto

import (
	"time"
)

// CreateProductCommand 创建产品的命令对象
// @Description Command to create a product
// @Tags Product
type CreateProductCommand struct {
	Name        string  `json:"name" binding:"required"` // 产品名称
	Description *string `json:"description"`             // 产品描述
}

type ProductData struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// ReleaseMethod 表示版本发布方式
type ReleaseMethod int

const (
	ReleaseImmediate ReleaseMethod = iota // 0 立即发布
	ReleaseScheduled                      // 1 定时发布
	ReleaseHold                           // 2 暂不发布
)

// CreateProductVersionCommand 产品版本的DTO对象
// @Description Product version DTO
type CreateProductVersionCommand struct {
	ProductID   uint          `json:"product_id" binding:"required"`     // 所属产品ID
	VersionCode string        `json:"version_code" binding:"required"`   // 版本号
	ReleaseDate *time.Time    `json:"release_date"`                      // 发布时间
	Description *string       `json:"description"`                       // 版本描述
	Method      ReleaseMethod `json:"release_method" binding:"required"` // 0立即发布，1定时发布，2暂不发布
}

type ProductVersionData struct {
	ID          uint       `json:"id"`
	ProductID   uint       `json:"product_id"`   // 所属产品ID
	VersionCode string     `json:"version_code"` // 版本号
	ReleaseDate *time.Time `json:"release_date"` // 发布时间
}

type ReleaseNewVersionCommand struct {
	ProductID   uint       `json:"product_id" binding:"required"` // 所属产品ID
	VersionID   uint       `json:"version_id" binding:"required"` // 版本ID
	ReleaseDate *time.Time `json:"release_date"`                  // 发布时间
}

type DeprecateVersionCommand struct {
	ProductID uint `json:"product_id" binding:"required"`
	VersionID uint `json:"version_id" binding:"required"`
}

// Query DTOs

// GetProductByIDQuery 根据ID查询产品的查询对象
type GetProductByIDQuery struct {
	ID uint `form:"id" binding:"required"` // 产品ID
}

// GetProductByNameQuery 根据名称查询产品的查询对象
type GetProductByNameQuery struct {
	Name string `form:"name" binding:"required"` // 产品名称
}

// UpdateMinVersionCommand 更新最低支持版本的命令对象
// @Description Command to update min supported version
type UpdateMinVersionCommand struct {
	ProductID uint `json:"product_id" binding:"required"` // 产品ID
	VersionID uint `json:"version_id" binding:"required"` // 版本ID
}
