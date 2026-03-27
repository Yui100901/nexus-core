package service

import (
	"context"
	"fmt"
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/model"
	"time"

	"gorm.io/gorm"
)

// ProductService 提供产品相关的业务逻辑服务
// 管理产品的创建、查询、版本控制等操作
type ProductService struct {
}

// NewProductService 创建新的产品服务实例
func NewProductService() *ProductService {
	return &ProductService{}
}

// CreateProduct 创建新产品
// 包括产品基本信息和版本列表的持久化存储
func (s *ProductService) CreateProduct(cmd dto.CreateProductCommand) (*dto.ProductData, error) {
	pProduct := &model.Product{
		Name:        cmd.Name,
		Description: cmd.Description,
	}
	err := productRepo.Create(context.Background(), global.DB, pProduct)
	if err != nil {
		return nil, err
	}
	return &dto.ProductData{
		ID:          pProduct.ID,
		Name:        pProduct.Name,
		Description: pProduct.Description,
	}, nil
}

//// BatchCreateProduct 批量创建产品
//// 支持一次性创建多个产品及其版本信息
//func (s *ProductService) BatchCreateProduct(products []dto.CreateProductCommand) error {
//	// 1. 校验重复名称
//	seen := make(map[string]bool)
//	for _, p := range products {
//		if seen[p.Name] {
//			return fmt.Errorf("duplicate product name: %s", p.Name)
//		}
//		seen[p.Name] = true
//	}
//
//	// 2. 调用仓储层批量创建 在事务中
//	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
//		// txCtx.MustDefaultDB() returns tx
//		return s.pr.BatchCreateProduct(txCtx, txCtx.MustDefaultDB(), products)
//	})
//}

// GetProductDataByID 根据ID获取产品信息
func (s *ProductService) GetProductDataByID(id uint) (*dto.ProductData, error) {
	pProduct, err := productRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	if pProduct == nil {
		return nil, fmt.Errorf("product not found")
	}
	return &dto.ProductData{
		ID:          pProduct.ID,
		Name:        pProduct.Name,
		Description: pProduct.Description,
	}, nil
}

func (s *ProductService) GetProductEntityByID(id uint) (*entity.Product, error) {
	pProduct, err := productRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	if pProduct == nil {
		return nil, fmt.Errorf("pProduct not found")
	}
	pVersionList, err := productVersionRepo.ListByProductID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	var versionList []entity.Version
	for _, v := range pVersionList {
		version := entity.Version{
			ID:          v.ID,
			VersionCode: v.VersionCode,
			ReleaseDate: v.ReleaseDate,
			Description: v.Description,
			Status:      v.Status,
		}
		versionList = append(versionList, version)
	}
	return &entity.Product{
		ID:                    pProduct.ID,
		Name:                  pProduct.Name,
		Description:           pProduct.Description,
		MinSupportedVersionID: pProduct.MinSupportedVersionID,
		VersionList:           versionList,
	}, nil
}

// SetMinSupportedVersion 设置产品的最低支持版本
// 用于控制产品版本的兼容性要求
func (s *ProductService) SetMinSupportedVersion(productID, versionID uint) error {
	version, err := productVersionRepo.GetByID(context.Background(), global.DB, versionID)
	if err != nil {
		return err
	}
	if version == nil {
		return fmt.Errorf("unkonwn version id %d", versionID)
	}
	if version.ProductID == productID {
		return global.DB.Model(&model.Product{}).
			Where("id = ?", productID).
			Update("min_supported_version_id", versionID).Error
	}
	return fmt.Errorf("version id %d not supported for %d", versionID, productID)
}

// DeleteProduct 删除产品
// 同时删除产品相关的所有版本信息
func (s *ProductService) DeleteProduct(id uint) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&model.Product{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&model.ProductVersion{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// CreateProductVersion 创建新产品版本
// 创建新产品版本，若指定了发布时间，则注册定时发布任务
func (s *ProductService) CreateProductVersion(cmd dto.CreateProductVersionCommand) (*dto.ProductVersionData, error) {
	product, err := s.GetProductEntityByID(cmd.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, fmt.Errorf("product not found")
	}
	if product.ExistsVersionCode(cmd.VersionCode) {
		return nil, fmt.Errorf("version code already exists")
	}
	newVersion := &model.ProductVersion{
		ProductID:   cmd.ProductID,
		VersionCode: cmd.VersionCode,
		ReleaseDate: cmd.ReleaseDate,
		Description: cmd.Description,
		Status:      model.VersionStatusUnreleased, //默认未发布
	}
	if err := productVersionRepo.Create(context.Background(), global.DB, newVersion); err != nil {
		return nil, err
	}

	switch cmd.Method {
	case dto.ReleaseImmediate:
		err := s.doReleaseVersion(newVersion.ID, time.Now())
		if err != nil {
			return nil, fmt.Errorf("failed to release version: %w", err)
		}
	case dto.ReleaseScheduled:
		var releaseDate time.Time
		if newVersion.ReleaseDate != nil {
			releaseDate = *newVersion.ReleaseDate
		} else {
			releaseDate = time.Now()
		}
		s.ScheduleReleaseTask(newVersion.ID, releaseDate)
	case dto.ReleaseHold:

	}

	return &dto.ProductVersionData{
		ID:          newVersion.ID,
		ProductID:   newVersion.ProductID,
		VersionCode: newVersion.VersionCode,
	}, nil
}

// ReleaseVersion 发布指定产品的指定版本
// 若指定了发布时间则定时发布，否则立即发布
func (s *ProductService) ReleaseVersion(versionID uint, releaseDate *time.Time) error {
	if releaseDate == nil {
		return s.doReleaseVersion(versionID, time.Now())
	} else {
		//创建定时任务
		s.ScheduleReleaseTask(versionID, *releaseDate)
		return nil
	}
}

// ScheduleReleaseTask 简易定时发布
// todo 后续考虑如何管理定时任务
func (s *ProductService) ScheduleReleaseTask(versionID uint, releaseDate time.Time) {
	delay := time.Until(releaseDate)
	if delay <= 0 {
		// 已经过了发布时间，直接发布
		_ = s.doReleaseVersion(versionID, releaseDate)
		return
	}
	go func() {
		<-time.After(delay)
		_ = s.doReleaseVersion(versionID, releaseDate)
	}()
}

// 内部方法执行版本发布
func (s *ProductService) doReleaseVersion(versionID uint, releaseDate time.Time) error {

	return global.DB.Model(&model.ProductVersion{}).Where("id = ?", versionID).Updates(map[string]interface{}{
		"status":       model.VersionStatusAvailable,
		"release_date": releaseDate,
	}).Error
}

func (s *ProductService) DeprecateVersion(versionID uint) error {
	return global.DB.Model(&model.ProductVersion{}).Where("id = ?", versionID).Updates(map[string]interface{}{
		"status": model.VersionStatusDeprecated,
	}).Error
}
