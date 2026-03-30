package service

import (
	"context"
	"fmt"
	"nexus-core/api/dto"
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
func (s *LicenseService) CreateLicense(cmd dto.CreateLicenseCommand) (*dto.LicenseData, error) {
	var product model.Product
	if err := global.DB.Model(&model.Product{}).Where("id = ?", &cmd.ProductID).First(product).Error; err != nil {
		return nil, err
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
	err := licenseRepo.Create(context.Background(), global.DB, license)
	if err != nil {
		return nil, err
	}
	return &dto.LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
}

//// BatchCreateLicense 批量创建许可证
//func (s *LicenseService) BatchCreateLicense(ctx *sc.ServiceContext, licenses []*entity.License) error {
//	if len(licenses) == 0 {
//		return fmt.Errorf("licenses list cannot be empty")
//	}
//
//	// 收集所有需要的产品 ID
//	allIDs := make(map[uint]struct{})
//	for _, license := range licenses {
//		id := license.ProductID
//		allIDs[id] = struct{}{}
//	}
//
//	var allIDList []uint
//	for k := range allIDs {
//		allIDList = append(allIDList, k)
//	}
//
//	db := ctx.MustDefaultDB()
//	exists, err := s.pr.ExistIds(ctx, db, allIDList)
//	if err != nil {
//		return err
//	}
//
//	if !exists {
//		return fmt.Errorf("some products in scope do not exist")
//	}
//
//	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
//		return s.lr.BatchCreateLicense(txCtx, txCtx.MustDefaultDB(), licenses)
//	})
//}

//// ActivateLicenseIfNeeded 激活许可证
//func (s *LicenseService) ActivateLicenseIfNeeded(ctx *sc.ServiceContext, license *entity.License) error {
//	// Use transaction wrapper for consistency
//	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
//		return s.ActivateLicenseIfNeededWithCtx(txCtx, license)
//	})
//}

//// ActivateLicenseIfNeededWithCtx 在已有事务中激活许可证（供其他 service 在同一事务中调用）
//// 使用 ServiceContext 来获取当前活跃的 DB（可能是 tx）
//func (s *LicenseService) ActivateLicenseIfNeededWithCtx(ctx *sc.ServiceContext, license *entity.License) error {
//	if license.IsActive() {
//		return nil
//	}
//
//	if err := license.Activate(time.Now()); err != nil {
//		return err
//	}
//
//	// use ctx.MustDefaultDB() so callers don't need to pass tx explicitly
//	return s.lr.UpdateLicenseStatus(ctx, ctx.MustDefaultDB(), license.ID, int(entity.StatusActive))
//}
//
//// GetLicenseBindList 获取许可证绑定列表
//func (s *LicenseService) GetLicenseBindList(ctx *sc.ServiceContext, licenseID uint) ([]entity.NodeLicenseBinding, error) {
//	db := ctx.MustDefaultDB()
//	return s.nlr.GetBindingsByLicenseID(ctx, db, licenseID)
//}

// RevokeLicense 吊销许可证
// todo 后续可能需要强制下线？
func (s *LicenseService) RevokeLicense(licenseID uint) error {
	return global.DB.Model(&model.License{}).Where("id = ?", licenseID).Update("status", entity.StatusRevoked).Error
}

// UpdateLicense 更新许可证信息
func (s *LicenseService) UpdateLicense(cmd dto.UpdateLicenseCommand) error {
	id := cmd.ID
	updates := model.License{
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		FeatureMask:   cmd.FeatureMask,
		Remark:        cmd.Remark,
	}
	return global.DB.Model(&model.License{}).Where("id = ?", id).Updates(updates).Error
}

// RenewLicense 增加或减少许可证时间
func (s *LicenseService) RenewLicense(cmd dto.RenewLicenseCommand) error {
	licenseID, extraHours := cmd.ID, cmd.ExtraHours
	license, err := s.GetLicenseEntityById(licenseID)
	if err != nil {
		return err
	}
	license.Renew(time.Now(), extraHours)
	return global.DB.Model(model.License{}).Where("id = ?", licenseID).
		Updates(model.License{
			ValidityHours: license.ValidityHours,
			ExpiredAt:     license.ExpiredAt,
			Status:        int(license.Status),
		}).Error
}

// RemoveBindings 移除许可证的所有绑定关系
func (s *LicenseService) RemoveBindings(id uint) error {
	return global.DB.Where("license_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteLicense 删除许可证
func (s *LicenseService) DeleteLicense(id uint) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
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
func (s *LicenseService) GetLicenseDataByID(id uint) (*dto.LicenseData, error) {
	license, err := licenseRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	if license == nil {
		return nil, fmt.Errorf("product not found")
	}
	return &dto.LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
}

// GetLicenseDataByKey 根据许可证密钥获取许可证
func (s *LicenseService) GetLicenseDataByKey(key string) (*dto.LicenseData, error) {
	license, err := licenseRepo.GetByKey(context.Background(), global.DB, key)
	if err != nil {
		return nil, err
	}
	if license == nil {
		return nil, fmt.Errorf("product not found")
	}
	return &dto.LicenseData{
		ID:            license.ID,
		ProductID:     license.ProductID,
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        license.Status,
		Remark:        license.Remark,
	}, nil
}

func (s *LicenseService) GetLicenseEntityById(id uint) (*entity.License, error) {
	pLicense, err := licenseRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	return &entity.License{
		ID:            pLicense.ID,
		ProductID:     pLicense.ProductID,
		LicenseKey:    pLicense.LicenseKey,
		ValidityHours: pLicense.ValidityHours,
		IssuedAt:      pLicense.CreatedAt,
		ActivatedAt:   pLicense.ActivatedAt,
		ExpiredAt:     pLicense.ExpiredAt,
		Status:        entity.LicenseStatus(pLicense.Status),
		Remark:        pLicense.Remark,
		MaxNodes:      pLicense.MaxNodes,
		MaxConcurrent: pLicense.MaxConcurrent,
		FeatureMask:   pLicense.FeatureMask,
	}, nil
}

// DeleteInvalidLicenses 删除所有过期的许可证，同时删除节点许可证绑定
func (s *LicenseService) DeleteInvalidLicenses() error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		var expiredLicenses []model.License

		// 查询所有已过期或被吊销的许可证
		if err := tx.Where("(expired_at IS NOT NULL AND expired_at < ?) OR status IN (?, ?)", time.Now(), model.StatusExpired, model.StatusRevoked).
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
