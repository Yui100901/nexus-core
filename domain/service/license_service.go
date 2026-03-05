package service

import (
	"fmt"
	"nexus-core/persistence/base"
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
	db  *gorm.DB
	lr  *repository.LicenseRepository // 许可证仓库，用于数据持久化操作
	pr  *repository.ProductRepository
	nlr *repository.NodeLicenseBindingRepository
}

// NewLicenseService 创建新的许可证服务实例
func NewLicenseService() *LicenseService {
	return &LicenseService{
		db:  base.Connect(),
		lr:  repository.NewLicenseRepository(),
		pr:  repository.NewProductRepository(),
		nlr: repository.NewNodeLicenseBindingRepository(),
	}
}

// CreateLicense 创建单个许可证
// 包括许可证及其授权范围的持久化存储
func (s *LicenseService) CreateLicense(ctx *sc.ServiceContext, license *entity.License) error {
	productIDs := license.GetScopeProductIdList()
	if len(productIDs) == 0 {
		return fmt.Errorf("license scope cannot be empty")
	}

	// 检查产品是否都存在
	exist, err := s.pr.ExistIds(ctx, s.db, productIDs)
	// err := s.pr.ExistIds(ctx, s.db, productIDs)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("some products in scope do not exist")
	}

	// 插入 License
	return s.lr.CreateLicense(ctx, s.db, license)
}

// BatchCreateLicense 批量创建许可证
// 支持一次性创建多个许可证及其授权范围
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

	// 一次性查询数据库
	var allIDList []uint
	for k := range allIDs {
		allIDList = append(allIDList, k)
	}
	exists, err := s.pr.ExistIds(ctx, s.db, allIDList) // 假设 repo
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("some products in scope do not exist")
	}

	// 批量插入
	return s.db.Transaction(func(tx *gorm.DB) error {
		err := s.lr.BatchCreateLicense(ctx, tx, licenses)
		if err != nil {
			return err
		}
		return s.lr.BatchCreateLicenseScope(ctx, tx, licenses)
	})
}

// ActivateLicenseIfNeeded 激活许可证
func (s *LicenseService) ActivateLicenseIfNeeded(ctx *sc.ServiceContext, license *entity.License) error {
	// Use transaction wrapper for consistency
	return repository.WithTransaction(s.db, func(tx *gorm.DB) error {
		return s.ActivateLicenseIfNeededWithTx(ctx, tx, license)
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
// 返回指定许可证的所有绑定信息
func (s *LicenseService) GetLicenseBindList(ctx *sc.ServiceContext, licenseID uint) ([]entity.NodeLicenseBinding, error) {
	return s.nlr.GetBindingsByLicenseID(ctx, s.db, licenseID)
}

// UpdateLicenseStatus 更新许可证状态
// 如激活、过期、吊销等状态变更
func (s *LicenseService) UpdateLicenseStatus(ctx *sc.ServiceContext, licenseID uint, status int) error {
	// 可以加业务逻辑，比如只有过期的才能更新为失效
	return s.lr.UpdateLicenseStatus(ctx, s.db, licenseID, status)
}

// UpdateLicense 更新许可证信息
// 包括有效期、备注等信息的更新
func (s *LicenseService) UpdateLicense(ctx *sc.ServiceContext, license *entity.License) error {
	// 可以加业务逻辑，比如校验有效期、状态转换是否合法
	return s.lr.UpdateLicense(ctx, s.db, license)
}

// GetLicenseByID 根据ID获取许可证
// 返回指定ID的完整许可证信息，包括授权范围
func (s *LicenseService) GetLicenseByID(ctx *sc.ServiceContext, id uint) (*entity.License, error) {
	return s.lr.GetByID(ctx, s.db, id)
}

// GetLicenseByKey 根据许可证密钥获取许可证
// 主要用于客户端验证时根据输入的许可证密钥查找许可证信息
func (s *LicenseService) GetLicenseByKey(ctx *sc.ServiceContext, key string) (*entity.License, error) {
	return s.lr.GetByKey(ctx, s.db, key)
}

// DeleteExpiredLicenses 删除所有过期的许可证
// 清理数据库中已过期的许可证记录
func (s *LicenseService) DeleteExpiredLicenses(ctx *sc.ServiceContext) error {
	ids, err := s.lr.GetIdListByStatus(ctx, s.db, entity.StatusExpired)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		err := s.lr.BatchDeleteScopeByLicenseIdList(ctx, tx, ids)
		if err != nil {
			return err
		}
		return s.lr.BatchDeleteByIdList(ctx, tx, ids)
	})
}
