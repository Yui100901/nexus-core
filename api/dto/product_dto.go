package dto

import (
	"nexus-core/domain/entity"
	"time"
)

// CreateProductCommand 创建产品的命令对象
// @Description Command to create a product
// @Tags Product
type CreateProductCommand struct {
	Name        string  `json:"name" binding:"required"` // 产品名称
	Description *string `json:"description"`             // 产品描述
}

// CreateProductVersionCommand 产品版本的DTO对象
// @Description Product version DTO
type CreateProductVersionCommand struct {
	ProductID   uint       `json:"product_id" binding:"required"`   // 所属产品ID
	VersionCode string     `json:"version_code" binding:"required"` // 版本号
	ReleaseDate *time.Time `json:"release_date"`                    // 发布时间
	Description *string    `json:"description"`                     // 版本描述
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

// ToEntityVersion 将创建产品版本命令转换为实体对象
func ToEntityVersion(cmd CreateProductVersionCommand) (*entity.Version, error) {
	return entity.NewVersion(cmd.VersionCode, cmd.ReleaseDate, cmd.Description)
}

// ToEntityProduct 将创建产品命令转换为实体对象
func ToEntityProduct(cmd CreateProductCommand) (*entity.Product, error) {
	return entity.NewProduct(cmd.Name, cmd.Description, nil)
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
