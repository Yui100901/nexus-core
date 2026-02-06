package service

import (
	"context"
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/persistence/repository"
	"time"
)

// ProductService 提供产品相关的业务逻辑服务
// 管理产品的创建、查询、版本控制等操作
type ProductService struct {
	pr *repository.ProductRepository // 产品仓库，用于数据持久化操作
}

// NewProductService 创建新的产品服务实例
func NewProductService() *ProductService {
	return &ProductService{
		pr: repository.NewProductRepository(),
	}
}

// CreateProduct 创建新产品
// 包括产品基本信息和版本列表的持久化存储
func (s *ProductService) CreateProduct(ctx context.Context, p *entity.Product) error {
	return s.pr.CreateProduct(ctx, p)
}

// BatchCreateProduct 批量创建产品
// 支持一次性创建多个产品及其版本信息
func (s *ProductService) BatchCreateProduct(ctx context.Context, products []*entity.Product) error {
	// 1. 校验重复名称
	seen := make(map[string]bool)
	for _, p := range products {
		if seen[p.Name] {
			return fmt.Errorf("duplicate product name: %s", p.Name)
		}
		seen[p.Name] = true
	}

	// 2. 调用仓储层批量创建
	return s.pr.BatchCreateProduct(ctx, products)
}

// GetByID 根据ID获取产品信息
// 返回指定ID的完整产品信息，包括所有版本
func (s *ProductService) GetByID(ctx context.Context, id uint) (*entity.Product, error) {
	return s.pr.GetByID(ctx, id)
}

// GetByName 根据名称获取产品信息
// 返回指定名称的完整产品信息，包括所有版本
func (s *ProductService) GetByName(ctx context.Context, name string) (*entity.Product, error) {
	return s.pr.GetByName(ctx, name)
}

// SetMinSupportedVersion 设置产品的最低支持版本
// 用于控制产品版本的兼容性要求
func (s *ProductService) SetMinSupportedVersion(ctx context.Context, productID, versionID uint) error {
	product, err := s.pr.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	err = product.SetMinSupportedVersion(versionID)
	if err != nil {
		return err
	}
	return s.pr.UpdateMinSupportedVersion(ctx, product.ID, *product.MinSupportedVersionID)
}

// DeleteProduct 删除产品
// 同时删除产品相关的所有版本信息
func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	return s.pr.DeleteProduct(ctx, id)
}

// CreateNewVersion 创建新产品版本
func (s *ProductService) CreateNewVersion(ctx context.Context, productID uint, v *entity.Version) error {
	product, err := s.pr.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if err := product.CreateNewVersion(*v); err != nil {
		return err
	}
	if err := s.pr.CreateNewVersion(ctx, product.ID, v); err != nil {
		return err
	}

	// 如果设置了发布时间，则注册定时任务
	if v.ReleaseDate != nil {
		s.ScheduleReleaseTask(ctx, product.ID, v.ID, *v.ReleaseDate)
	}

	return nil
}

func (s *ProductService) ReleaseVersion(ctx context.Context, productID, versionID uint, releaseDate time.Time) error {
	product, err := s.pr.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	err = product.ReleaseVersion(versionID, releaseDate)
	if err != nil {
		return err
	}
	return s.pr.ReleaseVersion(ctx, versionID, releaseDate)
}

// ScheduleReleaseTask 简易定时发布
func (s *ProductService) ScheduleReleaseTask(ctx context.Context, productID, versionID uint, releaseDate time.Time) {
	delay := time.Until(releaseDate)
	if delay <= 0 {
		// 已经过了发布时间，直接发布
		_ = s.ReleaseVersion(ctx, productID, versionID, releaseDate)
		return
	}
	go func() {
		<-time.After(delay)
		_ = s.ReleaseVersion(ctx, productID, versionID, releaseDate)
	}()
}
