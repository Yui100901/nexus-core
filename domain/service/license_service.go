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
	return &LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
}

// RevokeLicense 吊销许可证
// todo 后续可能需要强制下线？
func (s *LicenseService) RevokeLicense(ctx context.Context, licenseID uint) error {
	return global.DB.WithContext(ctx).Model(&model.License{}).Where("id = ?", licenseID).Update("status", entity.StatusRevoked).Error
}

// UpdateLicense 更新许可证信息
func (s *LicenseService) UpdateLicense(ctx context.Context, cmd UpdateLicenseCommand) error {
	id := cmd.ID
	updates := model.License{
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		FeatureMask:   cmd.FeatureMask,
		Remark:        cmd.Remark,
	}
	return global.DB.WithContext(ctx).Model(&model.License{}).Where("id = ?", id).Updates(updates).Error
}

// RenewLicense 增加或减少许可证时间
func (s *LicenseService) RenewLicense(ctx context.Context, cmd RenewLicenseCommand) error {
	licenseID, extraHours := cmd.ID, cmd.ExtraHours
	license, err := GetLicenseEntityByID(ctx, global.DB.WithContext(ctx), licenseID)
	if err != nil {
		return err
	}
	if license == nil {
		return ErrNotFound("license not found")
	}
	license.Renew(time.Now(), extraHours)
	return global.DB.WithContext(ctx).Model(model.License{}).Where("id = ?", licenseID).
		Updates(model.License{
			ValidityHours: license.ValidityHours,
			ExpiredAt:     license.ExpiredAt,
			Status:        int(license.Status),
		}).Error
}

// RemoveBindings 移除许可证的所有绑定关系
func (s *LicenseService) RemoveBindings(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Where("license_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteLicense 删除许可证
func (s *LicenseService) DeleteLicense(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("license_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&model.License{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// GetLicenseDataByID 根据ID获取许可证
func (s *LicenseService) GetLicenseDataByID(ctx context.Context, id uint) (*LicenseData, error) {
	license, err := licenseRepo.GetByID(ctx, global.DB.WithContext(ctx), id)
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
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
}

// GetLicenseDataByKey 根据许可证密钥获取许可证
func (s *LicenseService) GetLicenseDataByKey(ctx context.Context, key string) (*LicenseData, error) {
	license, err := licenseRepo.GetByKey(ctx, global.DB.WithContext(ctx), key)
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
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
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

		// 删除许可证
		if err := tx.Where("id IN ?", ids).Delete(&model.License{}).Error; err != nil {
			return err
		}

		return nil
	})
}
