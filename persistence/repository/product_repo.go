package repository

import (
	"errors"
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
	"nexus-core/sc"
	"time"

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

// CreateProduct 创建产品（回填 ID）
func (r *ProductRepository) CreateProduct(ctx *sc.ServiceContext, db *gorm.DB, product *entity.Product) error {
	pProduct := &model.Product{
		Name:                  product.Name,
		Description:           product.Description,
		MinSupportedVersionID: product.MinSupportedVersionID,
	}
	if err := gorm.G[model.Product](db).Create(ctx, pProduct); err != nil {
		return err
	}

	// 回填 ID
	product.ID = pProduct.ID

	return nil
}

// BatchCreateProduct 批量创建产品及其版本
func (r *ProductRepository) BatchCreateProduct(ctx *sc.ServiceContext, db *gorm.DB, products []*entity.Product) error {
	var pProducts []model.Product
	for _, product := range products {
		pProducts = append(pProducts, model.Product{
			Name:                  product.Name,
			Description:           product.Description,
			MinSupportedVersionID: product.MinSupportedVersionID,
		})
	}

	if err := gorm.G[model.Product](db).CreateInBatches(ctx, &pProducts, 100); err != nil {
		return err
	}

	// 回填 ID
	for i := range products {
		products[i].ID = pProducts[i].ID
	}

	return nil
}

// GetByID 根据 ID 获取产品及其版本
func (r *ProductRepository) GetByID(ctx *sc.ServiceContext, db *gorm.DB, id uint) (*entity.Product, error) {
	m, err := gorm.G[*model.Product](db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 没找到，返回 nil
			return nil, nil
		} else {
			// 数据库错误
			return nil, err
		}
	}

	versions, err := r.GetVersionsByProductID(ctx, db, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityProduct(m, versions), nil
}

// GetByName 根据名称获取产品
func (r *ProductRepository) GetByName(ctx *sc.ServiceContext, db *gorm.DB, name string) (*entity.Product, error) {
	m, err := gorm.G[*model.Product](db).
		Where("name = ?", name).
		First(ctx)
	if err != nil {
		return nil, err
	}

	versions, err := r.GetVersionsByProductID(ctx, db, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityProduct(m, versions), nil
}

func (r *ProductRepository) ExistIds(ctx *sc.ServiceContext, db *gorm.DB, ids []uint) (bool, error) {
	if len(ids) == 0 {
		return false, fmt.Errorf("id list cannot be empty")
	}

	var count int64
	err := db.WithContext(ctx).
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
func (r *ProductRepository) CreateNewVersion(ctx *sc.ServiceContext, db *gorm.DB, productID uint, v *entity.Version) error {
	m := &model.ProductVersion{
		ProductID:   productID,
		VersionCode: v.VersionCode,
		ReleaseDate: v.ReleaseDate,
		Description: v.Description,
		IsEnabled:   v.IsEnabled,
	}
	err := gorm.G[model.ProductVersion](db).
		Create(ctx, m)
	if err != nil {
		return err
	}
	v.ID = m.ID
	return nil
}

// ReleaseVersion 发布新版本
func (r *ProductRepository) ReleaseVersion(ctx *sc.ServiceContext, db *gorm.DB, versionID uint, releaseDate time.Time) error {
	_, err := gorm.G[model.ProductVersion](db).
		Where("id = ?", versionID).
		Updates(ctx, model.ProductVersion{
			IsEnabled:   1,
			ReleaseDate: &releaseDate,
		})
	return err
}

// DeprecateVersion 废弃版本
func (r *ProductRepository) DeprecateVersion(ctx *sc.ServiceContext, db *gorm.DB, productID, versionID uint) error {
	_, err := gorm.G[model.ProductVersion](db).
		Where("product_id = ? AND id = ?", productID, versionID).
		Update(ctx, "is_enabled", 0)
	return err
}

// GetVersionsByProductID 获取产品的版本列表
func (r *ProductRepository) GetVersionsByProductID(ctx *sc.ServiceContext, db *gorm.DB, productID uint) ([]model.ProductVersion, error) {
	return gorm.G[model.ProductVersion](db).
		Where("product_id = ?", productID).
		Find(ctx)
}

// UpdateMinSupportedVersion 设置产品的最低支持版本
func (r *ProductRepository) UpdateMinSupportedVersion(ctx *sc.ServiceContext, db *gorm.DB, productID, versionID uint) error {
	_, err := gorm.G[model.Product](db).
		Where("id = ?", productID).
		Update(ctx, "min_supported_version_id", versionID)
	return err
}

// DeleteProduct 删除产品
func (r *ProductRepository) DeleteProduct(ctx *sc.ServiceContext, db *gorm.DB, id uint) error {

	if _, err := gorm.G[model.Product](db).
		Where("id = ?", id).
		Delete(ctx); err != nil {
		return err
	}
	return nil
}

// DeleteVersion 删除版本
func (r *ProductRepository) DeleteVersion(ctx *sc.ServiceContext, db *gorm.DB, id uint) error {
	if _, err := gorm.G[model.ProductVersion](db).
		Where("id = ?", id).
		Delete(ctx); err != nil {
		return err
	}
	return nil
}

// GetVersionByProductAndCode 查找指定产品的版本信息
func (r *ProductRepository) GetVersionByProductAndCode(ctx *sc.ServiceContext, db *gorm.DB, productID uint, versionCode string) (*model.ProductVersion, error) {
	m, err := gorm.G[*model.ProductVersion](db).
		Where("product_id = ? AND version_code = ?", productID, versionCode).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// GetVersionByID 根据版本 ID 获取版本信息
func (r *ProductRepository) GetVersionByID(ctx *sc.ServiceContext, db *gorm.DB, id uint) (*model.ProductVersion, error) {
	m, err := gorm.G[*model.ProductVersion](db).
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
