package service

import (
	"fmt"
	"nexus-core/sc"
	"time"

	"nexus-core/domain/entity"
	"nexus-core/persistence/repository"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/19 14 49
//

// LicenseService 提供许可证相关的业务逻辑服务
// 包括许可证的创建、更新、查询、激活和验证等功能
type LicenseService struct {
	lr  *repository.LicenseRepository // 许可证仓库，用于数据持久化操作
	pr  *repository.ProductRepository
	nlr *repository.NodeLicenseBindingRepository
}

// NewLicenseService 创建新的许可证服务实例
func NewLicenseService() *LicenseService {
	return &LicenseService{
		lr:  repository.NewLicenseRepository(),
		pr:  repository.NewProductRepository(),
		nlr: repository.NewNodeLicenseBindingRepository(),
	}
}

// CreateLicense 创建单个许可证
func (s *LicenseService) CreateLicense(ctx *sc.ServiceContext, license *entity.License) error {
	productIDs := license.GetScopeProductIdList()
	if len(productIDs) == 0 {
		return fmt.Errorf("license scope cannot be empty")
	}

	// 检查产品是否都存在
	db := ctx.MustDefaultDB()
	exist, err := s.pr.ExistIds(ctx, db, productIDs)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("some products in scope do not exist")
	}

	// 插入 License
	return s.lr.CreateLicense(ctx, db, license)
}

// BatchCreateLicense 批量创建许可证
func (s *LicenseService) BatchCreateLicense(ctx *sc.ServiceContext, licenses []*entity.License) error {
	if len(licenses) == 0 {
		return fmt.Errorf("licenses list cannot be empty")
	}

	// 收集所有需要的产品 ID
	allIDs := make(map[uint]struct{})
	for _, license := range licenses {
		for _, id := range license.GetScopeProductIdList() {
			allIDs[id] = struct{}{}
		}
	}

	var allIDList []uint
	for k := range allIDs {
		allIDList = append(allIDList, k)
	}

	db := ctx.MustDefaultDB()
	exists, err := s.pr.ExistIds(ctx, db, allIDList)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("some products in scope do not exist")
	}

	// 批量插入
	plain := ctx.MustPlainDB()
	if plain == nil {
		return fmt.Errorf("database not initialized in service context")
	}
	return ctx.WithTransactionUsingDB(plain, func(txCtx *sc.ServiceContext) error {
		err := s.lr.BatchCreateLicense(txCtx, txCtx.MustDefaultDB(), licenses)
		if err != nil {
			return err
		}
		return s.lr.BatchCreateLicenseScope(txCtx, txCtx.MustDefaultDB(), licenses)
	})
}

// ActivateLicenseIfNeeded 激活许可证
func (s *LicenseService) ActivateLicenseIfNeeded(ctx *sc.ServiceContext, license *entity.License) error {
	// Use transaction wrapper for consistency
	db := ctx.MustPlainDB()
	if db == nil {
		return fmt.Errorf("database not initialized in service context")
	}
	return ctx.WithTransactionUsingDB(db, func(txCtx *sc.ServiceContext) error {
		return s.ActivateLicenseIfNeededWithTx(txCtx, txCtx.MustDefaultDB(), license)
	})
}

// ActivateLicenseIfNeededWithTx 在已有事务中激活许可证（供其他 service 在同一事务中调用）
func (s *LicenseService) ActivateLicenseIfNeededWithTx(ctx *sc.ServiceContext, tx *gorm.DB, license *entity.License) error {
	if license.IsActive() {
		return nil
	}

	if err := license.Activate(time.Now()); err != nil {
		return err
	}

	return s.lr.UpdateLicenseStatus(ctx, tx, license.ID, entity.StatusActive)
}

// GetLicenseBindList 获取许可证绑定列表
func (s *LicenseService) GetLicenseBindList(ctx *sc.ServiceContext, licenseID uint) ([]entity.NodeLicenseBinding, error) {
	db := ctx.MustDefaultDB()
	return s.nlr.GetBindingsByLicenseID(ctx, db, licenseID)
}

// UpdateLicenseStatus 更新许可证状态
func (s *LicenseService) UpdateLicenseStatus(ctx *sc.ServiceContext, licenseID uint, status int) error {
	db := ctx.MustDefaultDB()
	return s.lr.UpdateLicenseStatus(ctx, db, licenseID, status)
}

// UpdateLicense 更新许可证信息
func (s *LicenseService) UpdateLicense(ctx *sc.ServiceContext, license *entity.License) error {
	db := ctx.MustDefaultDB()
	return s.lr.UpdateLicense(ctx, db, license)
}

// GetLicenseByID 根据ID获取许可证
func (s *LicenseService) GetLicenseByID(ctx *sc.ServiceContext, id uint) (*entity.License, error) {
	db := ctx.MustDefaultDB()
	return s.lr.GetByID(ctx, db, id)
}

// GetLicenseByKey 根据许可证密钥获取许可证
func (s *LicenseService) GetLicenseByKey(ctx *sc.ServiceContext, key string) (*entity.License, error) {
	db := ctx.MustDefaultDB()
	return s.lr.GetByKey(ctx, db, key)
}

// DeleteExpiredLicenses 删除所有过期的许可证
func (s *LicenseService) DeleteExpiredLicenses(ctx *sc.ServiceContext) error {
	db := ctx.MustDefaultDB()
	ids, err := s.lr.GetIdListByStatus(ctx, db, entity.StatusExpired)
	if err != nil {
		return err
	}

	plain := ctx.MustPlainDB()
	if plain == nil {
		return fmt.Errorf("database not initialized in service context")
	}

	return ctx.WithTransactionUsingDB(plain, func(txCtx *sc.ServiceContext) error {
		err := s.lr.BatchDeleteScopeByLicenseIdList(txCtx, txCtx.MustDefaultDB(), ids)
		if err != nil {
			return err
		}
		return s.lr.BatchDeleteByIdList(txCtx, txCtx.MustDefaultDB(), ids)
	})
}
