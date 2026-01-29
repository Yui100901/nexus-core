package dto

import "nexus-core/domain/entity"

// CreateProductCommand 创建产品的命令对象
// @Description Command to create a product
// @Tags Product
type CreateProductCommand struct {
	Name        string  `json:"name" binding:"required"` // 产品名称
	Description *string `json:"description"`             // 产品描述
}

// ProductVDTO 产品版本的DTO对象
// @Description Product version DTO
type ProductVDTO struct {
	VersionCode string `json:"version_code"` // 版本号
	Description string `json:"description"`  // 版本描述
	// ReleaseDate omitted for simplicity in DTO
	Status int `json:"status"` // 版本状态
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
