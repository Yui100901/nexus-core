package service

import (
	"context"
	"errors"
	"nexus-core/global"
	"nexus-core/persistence/model"
	"strings"
	"time"

	"nexus-core/domain/entity"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/19 14 49
//

// LicenseService 提供许可证相关的业务逻辑服务
// 包括许可证的创建、更新、查询、激活和验证等功能
type LicenseService struct {
}

// NewLicenseService 创建新的许可证服务实例
func NewLicenseService() *LicenseService {
	return &LicenseService{}
}

// CreateLicense 创建单个许可证
func (s *LicenseService) CreateLicense(ctx context.Context, cmd CreateLicenseCommand) (*LicenseData, error) {
	if err := validateCreateLicenseCommand(cmd); err != nil {
		return nil, err
	}
	var product model.Product
	if err := global.DB.WithContext(ctx).Model(&model.Product{}).Where("id = ?", cmd.ProductID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound("product not found")
		}
		return nil, WrapInternal("get product failed", err)
	}
	license := &model.License{
		ProductID:     product.ID,
		LicenseKey:    strings.ReplaceAll(uuid.New().String(), "-", ""),
		ValidityHours: cmd.ValidityHours,
		ActivatedAt:   nil,
		ExpiredAt:     nil,
		Status:        int(entity.StatusInactive), //默认未激活
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		FeatureMask:   "",
		Remark:        cmd.Remark,
	}
	err := licenseRepo.Create(ctx, global.DB.WithContext(ctx), license)
	if err != nil {
		return nil, WrapInternal("create license failed", err)
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "license", license.ID, "create", map[string]interface{}{
		"product_id": license.ProductID,
	})
	return toLicenseData(license), nil
}

func (s *LicenseService) BatchCreateLicenses(ctx context.Context, cmd BatchCreateLicenseCommand) ([]LicenseData, error) {
	if err := validateBatchCreateLicenseCommand(cmd); err != nil {
		return nil, err
	}

	var product model.Product
	if err := global.DB.WithContext(ctx).Model(&model.Product{}).Where("id = ?", cmd.ProductID).First(&product).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound("product not found")
		}
		return nil, WrapInternal("get product failed", err)
	}

	licenses := make([]model.License, 0, cmd.Count)
	for i := 0; i < cmd.Count; i++ {
		licenses = append(licenses, model.License{
			ProductID:     product.ID,
			LicenseKey:    strings.ReplaceAll(uuid.New().String(), "-", ""),
			ValidityHours: cmd.ValidityHours,
			Status:        int(entity.StatusInactive),
			MaxNodes:      cmd.MaxNodes,
			MaxConcurrent: cmd.MaxConcurrent,
			FeatureMask:   "",
			Remark:        cmd.Remark,
		})
	}

	if err := global.DB.WithContext(ctx).Create(&licenses).Error; err != nil {
		return nil, WrapInternal("batch create licenses failed", err)
	}

	data := make([]LicenseData, 0, len(licenses))
	for i := range licenses {
		recordAuditLog(ctx, global.DB.WithContext(ctx), "license", licenses[i].ID, "create", map[string]interface{}{
			"product_id":   licenses[i].ProductID,
			"batch_create": true,
		})
		data = append(data, *toLicenseData(&licenses[i]))
	}
	return data, nil
}

// RevokeLicense 吊销许可证
// todo 后续可能需要强制下线？
func (s *LicenseService) RevokeLicense(ctx context.Context, licenseID uint) error {
	result := global.DB.WithContext(ctx).Model(&model.License{}).Where("id = ?", licenseID).Update("status", entity.StatusRevoked)
	if result.Error != nil {
		return WrapInternal("revoke license failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound("license not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "license", licenseID, "revoke", nil)
	return nil
}

func (s *LicenseService) RestoreLicense(ctx context.Context, cmd RestoreLicenseCommand) (*LicenseData, error) {
	if cmd.ID == 0 {
		return nil, ErrBadRequest("id is required")
	}

	var license model.License
	err := global.DB.WithContext(ctx).Where("id = ?", cmd.ID).First(&license).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound("license not found")
	}
	if err != nil {
		return nil, WrapInternal("get license failed", err)
	}
	if license.Status != int(entity.StatusRevoked) {
		return toLicenseData(&license), nil
	}

	status := int(entity.StatusInactive)
	if license.ActivatedAt != nil {
		status = int(entity.StatusActive)
		if license.ExpiredAt != nil && time.Now().After(*license.ExpiredAt) {
			status = int(entity.StatusExpired)
		}
	}

	if err := global.DB.WithContext(ctx).Model(&license).Update("status", status).Error; err != nil {
		return nil, WrapInternal("restore license failed", err)
	}
	license.Status = status
	recordAuditLog(ctx, global.DB.WithContext(ctx), "license", license.ID, "restore", map[string]interface{}{
		"status": status,
	})
	return toLicenseData(&license), nil
}

// UpdateLicense 更新许可证信息
func (s *LicenseService) UpdateLicense(ctx context.Context, cmd UpdateLicenseCommand) error {
	id := cmd.ID
	if id == 0 {
		return ErrBadRequest("id is required")
	}
	if cmd.MaxNodes < 0 {
		return ErrBadRequest("max_nodes must be greater than or equal to 0")
	}
	if cmd.MaxConcurrent < 0 {
		return ErrBadRequest("max_concurrent must be greater than or equal to 0")
	}
	updates := map[string]interface{}{
		"max_nodes":      cmd.MaxNodes,
		"max_concurrent": cmd.MaxConcurrent,
		"feature_mask":   cmd.FeatureMask,
		"remark":         cmd.Remark,
	}
	result := global.DB.WithContext(ctx).Model(&model.License{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return WrapInternal("update license failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound("license not found")
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "license", id, "update", map[string]interface{}{
		"max_nodes":      cmd.MaxNodes,
		"max_concurrent": cmd.MaxConcurrent,
		"feature_mask":   cmd.FeatureMask,
	})
	return nil
}

// RenewLicense 增加或减少许可证时间
func (s *LicenseService) RenewLicense(ctx context.Context, cmd RenewLicenseCommand) error {
	licenseID, extraHours := cmd.ID, cmd.ExtraHours
	if licenseID == 0 {
		return ErrBadRequest("id is required")
	}
	if extraHours == 0 {
		return ErrBadRequest("extra_hours must not be 0")
	}
	license, err := GetLicenseEntityByID(ctx, global.DB.WithContext(ctx), licenseID)
	if err != nil {
		return err
	}
	if license == nil {
		return ErrNotFound("license not found")
	}
	if license.Status == entity.StatusRevoked {
		return ErrForbidden("revoked license must be restored before renew")
	}
	license.Renew(time.Now(), extraHours)
	if err := global.DB.WithContext(ctx).Model(&model.License{}).Where("id = ?", licenseID).
		Updates(map[string]interface{}{
			"validity_hours": license.ValidityHours,
			"expired_at":     license.ExpiredAt,
			"status":         int(license.Status),
		}).Error; err != nil {
		return WrapInternal("renew license failed", err)
	}
	recordAuditLog(ctx, global.DB.WithContext(ctx), "license", licenseID, "renew", map[string]interface{}{
		"extra_hours":    extraHours,
		"validity_hours": license.ValidityHours,
	})
	return nil
}

// RemoveBindings 移除许可证的所有绑定关系
func (s *LicenseService) RemoveBindings(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var license model.License
		err := tx.Where("id = ?", id).First(&license).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound("license not found")
		}
		if err != nil {
			return WrapInternal("get license failed", err)
		}
		if err := tx.Where("license_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return WrapInternal("remove license bindings failed", err)
		}
		if err := resetLicenseNodeCount(ctx, tx, id); err != nil {
			return WrapInternal("reset license node count failed", err)
		}
		recordAuditLog(ctx, tx, "license", id, "remove_bindings", nil)
		return nil
	})
}

// DeleteLicense 删除许可证
func (s *LicenseService) DeleteLicense(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("license_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return WrapInternal("delete license bindings failed", err)
		}
		if err := tx.Where("license_id = ?", id).Delete(&model.LicenseProductScope{}).Error; err != nil {
			return WrapInternal("delete license product scopes failed", err)
		}
		if err := tx.Where("license_id = ?", id).Delete(&model.LicenseServiceScope{}).Error; err != nil {
			return WrapInternal("delete license service scopes failed", err)
		}
		result := tx.Where("id = ?", id).Delete(&model.License{})
		if result.Error != nil {
			return WrapInternal("delete license failed", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrNotFound("license not found")
		}
		recordAuditLog(ctx, tx, "license", id, "delete", nil)
		return nil
	})
}

// GetLicenseDataByID 根据ID获取许可证
func (s *LicenseService) GetLicenseDataByID(ctx context.Context, id uint) (*LicenseData, error) {
	license, err := GetLicenseEntityByID(ctx, global.DB.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if license == nil {
		return nil, ErrNotFound("license not found")
	}
	return &LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        int(license.Status),
		Remark:        license.Remark,
		MaxNodes:      license.MaxNodes,
		MaxConcurrent: license.MaxConcurrent,
		FeatureMask:   license.FeatureMask,
	}, nil
}

// GetLicenseDataByKey 根据许可证密钥获取许可证
func (s *LicenseService) GetLicenseDataByKey(ctx context.Context, key string) (*LicenseData, error) {
	license, err := GetLicenseEntityByKey(ctx, global.DB.WithContext(ctx), key)
	if err != nil {
		return nil, err
	}
	if license == nil {
		return nil, ErrNotFound("license not found")
	}
	return &LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        int(license.Status),
		Remark:        license.Remark,
		MaxNodes:      license.MaxNodes,
		MaxConcurrent: license.MaxConcurrent,
		FeatureMask:   license.FeatureMask,
	}, nil
}

func (s *LicenseService) ListLicenses(ctx context.Context, cmd ListLicensesCommand) ([]LicenseData, error) {
	query := global.DB.WithContext(ctx).Model(&model.License{}).Order("id DESC")
	if cmd.ProductID != nil {
		query = query.Where("product_id = ?", *cmd.ProductID)
	}
	if cmd.Status != nil {
		query = query.Where("status = ?", *cmd.Status)
	}
	if cmd.LicenseKey != nil && strings.TrimSpace(*cmd.LicenseKey) != "" {
		query = query.Where("license_key LIKE ?", "%"+strings.TrimSpace(*cmd.LicenseKey)+"%")
	}
	if cmd.Limit > 0 {
		query = query.Limit(cmd.Limit)
	}
	if cmd.Offset > 0 {
		query = query.Offset(cmd.Offset)
	}

	var licenses []model.License
	if err := query.Find(&licenses).Error; err != nil {
		return nil, WrapInternal("list licenses failed", err)
	}

	data := make([]LicenseData, 0, len(licenses))
	for i := range licenses {
		entityLicense := ToEntityLicense(&licenses[i])
		status := int(entityLicense.CalculateStatus(time.Now()))
		data = append(data, LicenseData{
			ID:            licenses[i].ID,
			ProductID:     licenses[i].ProductID,
			LicenseKey:    licenses[i].LicenseKey,
			ValidityHours: licenses[i].ValidityHours,
			Status:        status,
			Remark:        licenses[i].Remark,
			MaxNodes:      licenses[i].MaxNodes,
			MaxConcurrent: licenses[i].MaxConcurrent,
			FeatureMask:   licenses[i].FeatureMask,
		})
	}
	return data, nil
}

// CleanInvalidLicense 删除所有过期的许可证，同时删除节点许可证绑定
func (s *LicenseService) CleanInvalidLicense(ctx context.Context) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var expiredLicenses []model.License

		// 查询所有已过期或被吊销的许可证
		if err := tx.Where("(expired_at IS NOT NULL AND expired_at < ?) OR status IN (?, ?)",
			time.Now(), entity.StatusExpired, entity.StatusRevoked).
			Find(&expiredLicenses).Error; err != nil {
			return err
		}

		if len(expiredLicenses) == 0 {
			return nil // 没有无效的许可证
		}

		// 提取所有无效许可证的 ID
		var ids []uint
		for _, lic := range expiredLicenses {
			ids = append(ids, lic.ID)
		}

		// 删除节点绑定关系
		if err := tx.Where("license_id IN ?", ids).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("license_id IN ?", ids).Delete(&model.LicenseProductScope{}).Error; err != nil {
			return err
		}
		if err := tx.Where("license_id IN ?", ids).Delete(&model.LicenseServiceScope{}).Error; err != nil {
			return err
		}

		// 删除许可证
		if err := tx.Where("id IN ?", ids).Delete(&model.License{}).Error; err != nil {
			return err
		}

		for _, id := range ids {
			recordAuditLog(ctx, tx, "license", id, "clean_invalid", nil)
		}
		return nil
	})
}

func validateCreateLicenseCommand(cmd CreateLicenseCommand) error {
	if cmd.ProductID == 0 {
		return ErrBadRequest("product_id is required")
	}
	if cmd.ValidityHours <= 0 {
		return ErrBadRequest("validity_hours must be greater than 0")
	}
	if cmd.MaxNodes < 0 {
		return ErrBadRequest("max_nodes must be greater than or equal to 0")
	}
	if cmd.MaxConcurrent < 0 {
		return ErrBadRequest("max_concurrent must be greater than or equal to 0")
	}
	return nil
}

func validateBatchCreateLicenseCommand(cmd BatchCreateLicenseCommand) error {
	if err := validateCreateLicenseCommand(CreateLicenseCommand{
		ProductID:     cmd.ProductID,
		ValidityHours: cmd.ValidityHours,
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		Remark:        cmd.Remark,
	}); err != nil {
		return err
	}
	if cmd.Count <= 0 {
		return ErrBadRequest("count must be greater than 0")
	}
	if cmd.Count > 1000 {
		return ErrBadRequest("count must be less than or equal to 1000")
	}
	return nil
}

func toLicenseData(license *model.License) *LicenseData {
	return &LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        license.Status,
		Remark:        license.Remark,
		MaxNodes:      license.MaxNodes,
		MaxConcurrent: license.MaxConcurrent,
		FeatureMask:   license.FeatureMask,
	}
}
