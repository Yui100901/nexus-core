package repository

import (
	"context"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/3/26 15 59
//

type ProductVersionRepository struct {
}

func NewProductVersionRepository() *ProductVersionRepository {
	return &ProductVersionRepository{}
}

func (r *ProductVersionRepository) Create(ctx context.Context, db *gorm.DB, productVersion *model.ProductVersion) error {
	if err := gorm.G[model.ProductVersion](db).Create(ctx, productVersion); err != nil {
		return err
	}
	return nil
}

func (r *ProductVersionRepository) GetByID(ctx context.Context,
	db *gorm.DB, id string) (*model.ProductVersion, error) {
	m, err := GetOneByUniqueColumn[model.ProductVersion](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}

func (r *ProductVersionRepository) ListByProductID(ctx context.Context,
	db *gorm.DB, productID string) ([]model.ProductVersion, error) {
	return FindByColumn[model.ProductVersion](ctx, db, "product_id", productID)
}
