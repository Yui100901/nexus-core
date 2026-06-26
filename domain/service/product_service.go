package service

import (
	"context"
	"errors"
	"fmt"
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
		ID:                    pProduct.ID,
		Name:                  pProduct.Name,
		Description:           pProduct.Description,
		Status:                pProduct.Status,
		MinSupportedVersionID: pProduct.MinSupportedVersionID,
		Versions:              []ProductVersionData{},
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
	return s.productDataFromModel(ctx, global.DB.WithContext(ctx), pProduct)
}

func (s *ProductService) ListProducts(ctx context.Context, cmd ListProductsCommand) ([]ProductData, error) {
	query := global.DB.WithContext(ctx).Model(&model.Product{}).Order("id DESC")
	if cmd.Name != nil && strings.TrimSpace(*cmd.Name) != "" {
		query = query.Where("name LIKE ?", "%"+strings.TrimSpace(*cmd.Name)+"%")
	}
	if cmd.Status != nil {
		query = query.Where("status = ?", *cmd.Status)
	}
	if cmd.Limit > 0 {
		query = query.Limit(cmd.Limit)
	}
	if cmd.Offset > 0 {
		query = query.Offset(cmd.Offset)
	}

	var products []model.Product
	if err := query.Find(&products).Error; err != nil {
		return nil, WrapInternal("list products failed", err)
	}

	productIDs := make([]uint, 0, len(products))
	for i := range products {
		productIDs = append(productIDs, products[i].ID)
	}

	versionsByProduct := map[uint][]ProductVersionData{}
	if len(productIDs) > 0 {
		var versions []model.ProductVersion
		if err := global.DB.WithContext(ctx).
			Where("product_id IN ?", productIDs).
			Order("product_id ASC, release_date ASC, id ASC").
			Find(&versions).Error; err != nil {
			return nil, WrapInternal("list product versions failed", err)
		}
		for i := range versions {
			item := toProductVersionData(&versions[i])
			versionsByProduct[versions[i].ProductID] = append(versionsByProduct[versions[i].ProductID], *item)
		}
	}

	data := make([]ProductData, 0, len(products))
	for i := range products {
		data = append(data, ProductData{
			ID:                    products[i].ID,
			Name:                  products[i].Name,
			Description:           products[i].Description,
			Status:                products[i].Status,
			MinSupportedVersionID: products[i].MinSupportedVersionID,
			Versions:              versionsByProduct[products[i].ID],
		})
	}
	return data, nil
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
	if version.ProductID != productID {
		return ErrBadRequest("version does not belong to product")
	}
	if version.Status != int(entity.VersionStatusAvailable) || version.ReleaseDate == nil {
		return ErrConflict("min supported version must be available")
	}
	if err := global.DB.WithContext(ctx).Model(&model.Product{}).
		Where("id = ?", productID).
		Update("min_supported_version_id", versionID).Error; err != nil {
		return WrapInternal("update min supported version failed", err)
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "product", productID, "set_min_version", map[string]interface{}{
		"version_id": versionID,
	})
	return nil
}

// DeleteProduct 删除产品
// 若产品仍有关联 License 或控制服务定义，则阻断删除，避免破坏授权和控制链路。
func (s *ProductService) DeleteProduct(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var product model.Product
		err := tx.Where("id = ?", id).First(&product).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound("product not found")
		}
		if err != nil {
			return WrapInternal("get product failed", err)
		}

		var licenseCount int64
		if err := tx.Model(&model.License{}).Where("product_id = ?", id).Count(&licenseCount).Error; err != nil {
			return WrapInternal("count product licenses failed", err)
		}
		if licenseCount > 0 {
			return ErrConflict("product has licenses")
		}

		var controlServiceCount int64
		if err := tx.Model(&model.ControlService{}).Where("product_id = ?", id).Count(&controlServiceCount).Error; err != nil {
			return WrapInternal("count product control services failed", err)
		}
		if controlServiceCount > 0 {
			return ErrConflict("product has control services")
		}

		if err := tx.Where("product_id = ?", id).Delete(&model.ProductVersion{}).Error; err != nil {
			return WrapInternal("delete product versions failed", err)
		}
		if err := tx.Delete(&product).Error; err != nil {
			return WrapInternal("delete product failed", err)
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
		if cmd.Method == ReleaseScheduled && cmd.ReleaseDate == nil {
			return ErrBadRequest("release_date is required for scheduled release")
		}
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
		if !newVersion.ReleaseDate.After(time.Now()) {
			if err := s.ReleaseDueProductVersions(ctx); err != nil {
				return nil, err
			}
		}
	}

	if cmd.Method == ReleaseImmediate {
		newVersion.Status = int(entity.VersionStatusAvailable)
		now := time.Now()
		newVersion.ReleaseDate = &now
	}

	return toProductVersionData(newVersion), nil
}

// ReleaseVersion 发布指定产品的指定版本
// 若指定了发布时间则定时发布，否则立即发布
func (s *ProductService) ReleaseVersion(ctx context.Context, cmd ReleaseNewVersionCommand) error {
	if cmd.ProductID == 0 {
		return ErrBadRequest("product_id is required")
	}
	versionID, releaseDate := cmd.VersionID, cmd.ReleaseDate
	if releaseDate == nil {
		return s.doReleaseVersion(ctx, global.DB.WithContext(ctx), versionID, time.Now())
	}
	if !releaseDate.After(time.Now()) {
		return s.doReleaseVersion(ctx, global.DB.WithContext(ctx), versionID, *releaseDate)
	}
	return s.scheduleVersionRelease(ctx, cmd.ProductID, versionID, *releaseDate)
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

func (s *ProductService) scheduleVersionRelease(ctx context.Context, productID uint, versionID uint, releaseDate time.Time) error {
	result := global.DB.WithContext(ctx).Model(&model.ProductVersion{}).
		Where("id = ? AND product_id = ?", versionID, productID).
		Updates(map[string]interface{}{
			"status":       int(entity.VersionStatusUnreleased),
			"release_date": releaseDate,
		})
	if result.Error != nil {
		return WrapInternal("schedule version release failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound("version not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "product", productID, "schedule_version_release", map[string]interface{}{
		"version_id":    versionID,
		"release_date":  releaseDate,
		"release_after": releaseDate.Format(time.RFC3339),
	})
	return nil
}

func (s *ProductService) ReleaseDueProductVersions(ctx context.Context) error {
	now := time.Now()
	var versions []model.ProductVersion
	if err := global.DB.WithContext(ctx).
		Where("status = ? AND release_date IS NOT NULL AND release_date <= ?", int(entity.VersionStatusUnreleased), now).
		Order("release_date ASC, id ASC").
		Find(&versions).Error; err != nil {
		return WrapInternal("list due product versions failed", err)
	}

	for _, version := range versions {
		if err := s.doReleaseVersion(ctx, global.DB.WithContext(ctx), version.ID, *version.ReleaseDate); err != nil {
			return err
		}
		recordAuditLog(ctx, global.DB.WithContext(ctx), "product", version.ProductID, "release_due_version", map[string]interface{}{
			"version_id": version.ID,
		})
	}
	return nil
}

func (s *ProductService) StartScheduledReleaseWorker(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		if err := s.ReleaseDueProductVersions(ctx); err != nil {
			fmt.Printf("release due product versions failed: %v\n", err)
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := s.ReleaseDueProductVersions(ctx); err != nil {
					fmt.Printf("release due product versions failed: %v\n", err)
				}
			}
		}
	}()
}

func (s *ProductService) DeprecateVersion(ctx context.Context, cmd DeprecateVersionCommand) error {
	if cmd.ProductID == 0 || cmd.VersionID == 0 {
		return ErrBadRequest("product_id and version_id are required")
	}
	var product model.Product
	err := global.DB.WithContext(ctx).Where("id = ?", cmd.ProductID).First(&product).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound("product not found")
	}
	if err != nil {
		return WrapInternal("get product failed", err)
	}
	if product.MinSupportedVersionID != nil && *product.MinSupportedVersionID == cmd.VersionID {
		return ErrConflict("min supported version cannot be deprecated")
	}

	result := global.DB.WithContext(ctx).Model(&model.ProductVersion{}).
		Where("id = ? AND product_id = ?", cmd.VersionID, cmd.ProductID).
		Update("status", entity.VersionStatusDeprecated)
	if result.Error != nil {
		return WrapInternal("deprecate version failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound("version not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "product", cmd.ProductID, "deprecate_version", map[string]interface{}{
		"version_id": cmd.VersionID,
	})
	return nil
}

func (s *ProductService) productDataFromModel(ctx context.Context, db *gorm.DB, product *model.Product) (*ProductData, error) {
	var versions []model.ProductVersion
	if err := db.WithContext(ctx).
		Where("product_id = ?", product.ID).
		Order("release_date ASC, id ASC").
		Find(&versions).Error; err != nil {
		return nil, WrapInternal("list product versions failed", err)
	}

	data := &ProductData{
		ID:                    product.ID,
		Name:                  product.Name,
		Description:           product.Description,
		Status:                product.Status,
		MinSupportedVersionID: product.MinSupportedVersionID,
		Versions:              make([]ProductVersionData, 0, len(versions)),
	}
	for i := range versions {
		data.Versions = append(data.Versions, *toProductVersionData(&versions[i]))
	}
	return data, nil
}

func toProductVersionData(version *model.ProductVersion) *ProductVersionData {
	return &ProductVersionData{
		ID:          version.ID,
		ProductID:   version.ProductID,
		VersionCode: version.VersionCode,
		ReleaseDate: version.ReleaseDate,
		Description: version.Description,
		Status:      version.Status,
	}
}
