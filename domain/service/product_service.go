package service

import (
	"context"
	"errors"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/model"
	"strings"
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
func (s *ProductService) CreateProduct(ctx context.Context, cmd CreateProductCommand) (*ProductData, error) {
	pProduct := &model.Product{
		Name:        cmd.Name,
		Description: cmd.Description,
	}
	err := productRepo.Create(ctx, global.DB.WithContext(ctx), pProduct)
	if err != nil {
		return nil, err
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "product", pProduct.ID, "create", map[string]interface{}{
		"name": pProduct.Name,
	})
	return &ProductData{
		ID:          pProduct.ID,
		Name:        pProduct.Name,
		Description: pProduct.Description,
	}, nil
}

// GetProductDataByID 根据ID获取产品信息
func (s *ProductService) GetProductDataByID(ctx context.Context, id uint) (*ProductData, error) {
	pProduct, err := productRepo.GetByID(ctx, global.DB.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if pProduct == nil {
		return nil, ErrNotFound("product not found")
	}
	return &ProductData{
		ID:          pProduct.ID,
		Name:        pProduct.Name,
		Description: pProduct.Description,
	}, nil
}

func (s *ProductService) UpdateProduct(ctx context.Context, cmd UpdateProductCommand) (*ProductData, error) {
	if cmd.ID == 0 {
		return nil, ErrBadRequest("id is required")
	}

	updates := map[string]interface{}{}
	if cmd.Name != nil {
		name := strings.TrimSpace(*cmd.Name)
		if name == "" {
			return nil, ErrBadRequest("name is required")
		}
		updates["name"] = name
	}
	if cmd.Description != nil {
		updates["description"] = cmd.Description
	}
	if len(updates) == 0 {
		return nil, ErrBadRequest("no product fields to update")
	}

	result := global.DB.WithContext(ctx).Model(&model.Product{}).Where("id = ?", cmd.ID).Updates(updates)
	if result.Error != nil {
		return nil, WrapInternal("update product failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound("product not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "product", cmd.ID, "update", updates)
	return s.GetProductDataByID(ctx, cmd.ID)
}

// SetMinSupportedVersion 设置产品的最低支持版本
// 用于控制产品版本的兼容性要求
func (s *ProductService) SetMinSupportedVersion(ctx context.Context, cmd UpdateMinVersionCommand) error {
	versionID, productID := cmd.VersionID, cmd.ProductID
	version, err := productVersionRepo.GetByID(ctx, global.DB.WithContext(ctx), versionID)
	if err != nil {
		return err
	}
	if version == nil {
		return ErrNotFound("version not found")
	}
	if version.ProductID == productID {
		return global.DB.WithContext(ctx).Model(&model.Product{}).
			Where("id = ?", productID).
			Update("min_supported_version_id", versionID).Error
	}
	return ErrBadRequest("version does not belong to product")
}

// DeleteProduct 删除产品
// 同时删除产品相关的所有版本信息
func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&model.Product{}).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", id).Delete(&model.ProductVersion{}).Error; err != nil {
			return err
		}
		recordAuditLog(ctx, tx, "product", id, "delete", nil)
		return nil
	})
}

// CreateProductVersion 创建新产品版本
// 创建新产品版本，若指定了发布时间，则注册定时发布任务
func (s *ProductService) CreateProductVersion(ctx context.Context, cmd CreateProductVersionCommand) (*ProductVersionData, error) {
	var newVersion *model.ProductVersion
	if err := global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		product, err := GetProductEntityByID(ctx, tx, cmd.ProductID)
		if err != nil {
			return err
		}
		if product == nil {
			return ErrNotFound("product not found")
		}
		if product.ExistsVersionCode(cmd.VersionCode) {
			return ErrConflict("version code already exists")
		}
		newVersion = &model.ProductVersion{
			ProductID:   cmd.ProductID,
			VersionCode: cmd.VersionCode,
			ReleaseDate: cmd.ReleaseDate,
			Description: cmd.Description,
			Status:      int(entity.VersionStatusUnreleased), //默认未发布
			IsEnabled:   true,
		}
		if err := productVersionRepo.Create(ctx, tx, newVersion); err != nil {
			return err
		}

		if cmd.Method == ReleaseImmediate {
			if err := s.doReleaseVersion(ctx, tx, newVersion.ID, time.Now()); err != nil {
				return WrapInternal("failed to release version", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if cmd.Method == ReleaseScheduled {
		var releaseDate time.Time
		if newVersion.ReleaseDate != nil {
			releaseDate = *newVersion.ReleaseDate
		} else {
			releaseDate = time.Now()
		}
		s.scheduleReleaseTask(newVersion.ID, releaseDate)
	}

	return &ProductVersionData{
		ID:          newVersion.ID,
		ProductID:   newVersion.ProductID,
		VersionCode: newVersion.VersionCode,
	}, nil
}

// ReleaseVersion 发布指定产品的指定版本
// 若指定了发布时间则定时发布，否则立即发布
func (s *ProductService) ReleaseVersion(ctx context.Context, cmd ReleaseNewVersionCommand) error {
	versionID, releaseDate := cmd.VersionID, cmd.ReleaseDate
	if releaseDate == nil {
		return s.doReleaseVersion(ctx, global.DB.WithContext(ctx), versionID, time.Now())
	} else {
		//创建定时任务
		s.scheduleReleaseTask(versionID, *releaseDate)
		return nil
	}
}

// ScheduleReleaseTask 简易定时发布
// todo 后续考虑如何管理定时任务
func (s *ProductService) scheduleReleaseTask(versionID uint, releaseDate time.Time) {
	delay := time.Until(releaseDate)
	if delay <= 0 {
		// 已经过了发布时间，直接发布
		bg := context.Background()
		_ = s.doReleaseVersion(bg, global.DB.WithContext(bg), versionID, releaseDate)
		return
	}
	go func() {
		<-time.After(delay)
		bg := context.Background()
		_ = s.doReleaseVersion(bg, global.DB.WithContext(bg), versionID, releaseDate)
	}()
}

// 内部方法执行版本发布
func (s *ProductService) doReleaseVersion(ctx context.Context, db *gorm.DB, versionID uint, releaseDate time.Time) error {
	var version model.ProductVersion
	if err := db.WithContext(ctx).Where("id = ?", versionID).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound("version not found")
		}
		return WrapInternal("get version failed", err)
	}

	if err := db.WithContext(ctx).Model(&model.ProductVersion{}).Where("id = ?", versionID).Updates(map[string]interface{}{
		"status":       entity.VersionStatusAvailable,
		"release_date": releaseDate,
	}).Error; err != nil {
		return WrapInternal("release version failed", err)
	}

	if err := db.WithContext(ctx).Model(&model.Product{}).
		Where("id = ? AND min_supported_version_id IS NULL", version.ProductID).
		Update("min_supported_version_id", versionID).Error; err != nil {
		return WrapInternal("update min supported version failed", err)
	}

	return nil
}

func (s *ProductService) DeprecateVersion(ctx context.Context, versionID uint) error {
	return global.DB.WithContext(ctx).Model(&model.ProductVersion{}).Where("id = ?", versionID).Updates(map[string]interface{}{
		"status": entity.VersionStatusDeprecated,
	}).Error
}
