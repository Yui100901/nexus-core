package repository

import (
	"context"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/20 10 54
//

type ProductRepository struct {
}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) Create(ctx context.Context, db *gorm.DB, product *model.Product) error {
	if err := gorm.G[model.Product](db).Create(ctx, product); err != nil {
		return err
	}
	return nil
}

// GetByID 根据 ID 获取产品及其版本
func (r *ProductRepository) GetByID(ctx context.Context, db *gorm.DB, id uint) (*model.Product, error) {
	m, err := GetOneByUniqueColumn[model.Product](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}
