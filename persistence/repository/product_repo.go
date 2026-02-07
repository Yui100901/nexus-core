package repository

import (
	"context"
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"
	"time"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/20 10 54
//

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{
		db: base.Connect(),
	}
}

// CreateProduct 创建产品（回填 ID）
func (r *ProductRepository) CreateProduct(ctx context.Context, product *entity.Product) error {
	pProduct := &model.Product{
		Name:                  product.Name,
		Description:           product.Description,
		MinSupportedVersionID: product.MinSupportedVersionID,
	}
	if err := gorm.G[model.Product](r.db).Create(ctx, pProduct); err != nil {
		return err
	}

	// 回填 ID
	product.ID = pProduct.ID

	return nil
}

// BatchCreateProduct 批量创建产品及其版本
func (r *ProductRepository) BatchCreateProduct(ctx context.Context, products []*entity.Product) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pProducts []model.Product
		for _, product := range products {
			pProducts = append(pProducts, model.Product{
				Name:                  product.Name,
				Description:           product.Description,
				MinSupportedVersionID: product.MinSupportedVersionID,
			})
		}

		if err := gorm.G[model.Product](tx).CreateInBatches(ctx, &pProducts, 100); err != nil {
			return err
		}

		// 回填 ID
		for i := range products {
			products[i].ID = pProducts[i].ID
		}

		return nil
	})
}

// GetByID 根据 ID 获取产品及其版本
func (r *ProductRepository) GetByID(ctx context.Context, id uint) (*entity.Product, error) {
	m, err := gorm.G[*model.Product](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}

	versions, err := r.GetVersionsByProductID(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityProduct(m, versions), nil
}

// GetByName 根据名称获取产品
func (r *ProductRepository) GetByName(ctx context.Context, name string) (*entity.Product, error) {
	m, err := gorm.G[*model.Product](r.db).
		Where("name = ?", name).
		First(ctx)
	if err != nil {
		return nil, err
	}

	versions, err := r.GetVersionsByProductID(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityProduct(m, versions), nil
}

func (r *ProductRepository) ExistIds(ctx context.Context, ids []uint) (bool, error) {
	if len(ids) == 0 {
		return false, fmt.Errorf("id list cannot be empty")
	}

	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Product{}).
		Where("id IN ?", ids).
		Count(&count).Error
	if err != nil {
		return false, err
	}

	// 如果数据库里找到的数量和传入的数量一致，说明都存在
	return count == int64(len(ids)), nil
}

// CreateNewVersion 创建新产品版本
func (r *ProductRepository) CreateNewVersion(ctx context.Context, productID uint, v *entity.Version) error {
	m := &model.ProductVersion{
		ProductID:   productID,
		VersionCode: v.VersionCode,
		ReleaseDate: v.ReleaseDate,
		Description: v.Description,
		IsEnabled:   v.IsEnabled,
	}
	err := gorm.G[model.ProductVersion](r.db).
		Create(ctx, m)
	if err != nil {
		return err
	}
	v.ID = m.ID
	return nil
}

// ReleaseVersion 发布新版本
func (r *ProductRepository) ReleaseVersion(ctx context.Context, versionID uint, releaseDate time.Time) error {
	_, err := gorm.G[model.ProductVersion](r.db).
		Where("id = ?", versionID).
		Updates(ctx, model.ProductVersion{
			IsEnabled:   1,
			ReleaseDate: &releaseDate,
		})
	return err
}

// DeprecateVersion 废弃版本
func (r *ProductRepository) DeprecateVersion(ctx context.Context, productID, versionID uint) error {
	_, err := gorm.G[model.ProductVersion](r.db).
		Where("product_id = ? AND id = ?", productID, versionID).
		Update(ctx, "is_enabled", 0)
	return err
}

// GetVersionsByProductID 获取产品的版本列表
func (r *ProductRepository) GetVersionsByProductID(ctx context.Context, productID uint) ([]model.ProductVersion, error) {
	return gorm.G[model.ProductVersion](r.db).
		Where("product_id = ?", productID).
		Find(ctx)
}

// UpdateMinSupportedVersion 设置产品的最低支持版本
func (r *ProductRepository) UpdateMinSupportedVersion(ctx context.Context, productID, versionID uint) error {
	_, err := gorm.G[model.Product](r.db).
		Where("id = ?", productID).
		Update(ctx, "min_supported_version_id", versionID)
	return err
}

// DeleteProduct 删除产品及其版本
func (r *ProductRepository) DeleteProduct(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := gorm.G[model.ProductVersion](tx).
			Where("product_id = ?", id).
			Delete(ctx); err != nil {
			return err
		}
		if _, err := gorm.G[model.Product](tx).
			Where("id = ?", id).
			Delete(ctx); err != nil {
			return err
		}
		return nil
	})
}

// GetVersionByProductAndCode 查找指定产品的版本信息
func (r *ProductRepository) GetVersionByProductAndCode(ctx context.Context, productID uint, versionCode string) (*model.ProductVersion, error) {
	m, err := gorm.G[*model.ProductVersion](r.db).
		Where("product_id = ? AND version_code = ?", productID, versionCode).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetVersionByID 根据版本 ID 获取版本信息
func (r *ProductRepository) GetVersionByID(ctx context.Context, id uint) (*model.ProductVersion, error) {
	m, err := gorm.G[*model.ProductVersion](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// 转换为领域对象
func toEntityProduct(m *model.Product, versions []model.ProductVersion) *entity.Product {
	var versionList []entity.Version
	for _, v := range versions {
		versionList = append(versionList, entity.Version{
			ID:          v.ID,
			VersionCode: v.VersionCode,
			ReleaseDate: v.ReleaseDate,
			Description: v.Description,
			IsEnabled:   v.IsEnabled,
		})
	}
	return &entity.Product{
		ID:                    m.ID,
		Name:                  m.Name,
		Description:           m.Description,
		MinSupportedVersionID: m.MinSupportedVersionID,
		VersionList:           versionList,
	}
}
