package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nexus-core/domain/entity"
	"nexus-core/persistence/repository"

	"github.com/google/uuid"
)

//
// @Author yfy2001
// @Date 2026/1/19 14 49
//

// LicenseService 提供许可证相关的业务逻辑服务
// 包括许可证的创建、更新、查询、激活和验证等功能
type LicenseService struct {
	lr *repository.LicenseRepository // 许可证仓库，用于数据持久化操作
	pr *repository.ProductRepository
}

// NewLicenseService 创建新的许可证服务实例
func NewLicenseService() *LicenseService {
	return &LicenseService{
		lr: repository.NewLicenseRepository(),
	}
}

// CreateLicense 创建单个许可证
// 包括许可证及其授权范围的持久化存储
func (s *LicenseService) CreateLicense(ctx context.Context, license *entity.License) error {
	productIDs := license.GetScopeProductIdList()
	if len(productIDs) == 0 {
		return fmt.Errorf("license scope cannot be empty")
	}

	// 检查产品是否都存在
	exist, err := s.pr.ExistIds(ctx, productIDs)
	if err != nil {
		return err
	}
	if !exist {
		return fmt.Errorf("some products in scope do not exist")
	}

	// 插入 License
	return s.lr.CreateLicense(ctx, license)
}

// BatchCreateLicense 批量创建许可证
// 支持一次性创建多个许可证及其授权范围
func (s *LicenseService) BatchCreateLicense(ctx context.Context, licenses []*entity.License) error {
	// 业务规则校验，比如批量大小限制、LicenseKey 唯一性等
	return s.lr.BatchCreateLicense(ctx, licenses)
}

// ActivateLicense 激活许可证
func (s *LicenseService) ActivateLicense(ctx context.Context, licenseID uint) error {
	license, err := s.lr.GetByID(ctx, licenseID)
	if err != nil {
		return err
	}
	err = license.Activate(time.Now())
	if err != nil {
		return err
	}

	return s.lr.UpdateLicenseStatus(ctx, license.ID, entity.StatusActive)
}

// UpdateLicenseStatus 更新许可证状态
// 如激活、过期、吊销等状态变更
func (s *LicenseService) UpdateLicenseStatus(ctx context.Context, licenseID uint, status int) error {
	// 可以加业务逻辑，比如只有过期的才能更新为失效
	return s.lr.UpdateLicenseStatus(ctx, licenseID, status)
}

// UpdateLicense 更新许可证信息
// 包括有效期、备注等信息的更新
func (s *LicenseService) UpdateLicense(ctx context.Context, license *entity.License) error {
	// 可以加业务逻辑，比如校验有效期、状态转换是否合法
	return s.lr.UpdateLicense(ctx, license)
}

// GetLicenseByID 根据ID获取许可证
// 返回指定ID的完整许可证信息，包括授权范围
func (s *LicenseService) GetLicenseByID(ctx context.Context, id uint) (*entity.License, error) {
	return s.lr.GetByID(ctx, id)
}

// GetLicenseByKey 根据许可证密钥获取许可证
// 主要用于客户端验证时根据输入的许可证密钥查找许可证信息
func (s *LicenseService) GetLicenseByKey(ctx context.Context, key string) (*entity.License, error) {
	return s.lr.GetByKey(ctx, key)
}

// DeleteExpiredLicenses 删除所有过期的许可证
// 清理数据库中已过期的许可证记录
func (s *LicenseService) DeleteExpiredLicenses(ctx context.Context) error {
	ids, err := s.lr.GetIdListByStatus(ctx, entity.StatusExpired)
	if err != nil {
		return err
	}
	return s.lr.BatchDeleteByIdList(ctx, ids)
}

// GenerateLicenseKey 生成唯一的许可证密钥
// 使用UUID生成器创建全局唯一的许可证密钥
func (s *LicenseService) GenerateLicenseKey() string {
	return uuid.New().String()
}

// ActivateLicenseIfNeeded 按需激活许可证
// 如果许可证处于非激活状态，则激活它并更新数据库
func (s *LicenseService) ActivateLicenseIfNeeded(ctx context.Context, license *entity.License) error {
	if license.Status == entity.StatusActive {
		return nil
	}
	if license.Status == entity.StatusRevoked {
		return errors.New("license revoked")
	}
	if license.ValidityHours <= 0 {
		return errors.New("invalid validity hours")
	}
	if err := license.Activate(time.Now()); err != nil {
		return err
	}
	return s.lr.UpdateLicense(ctx, license)
}

// ValidateLicenseForUsage 验证许可证对特定产品的使用权限
// 检查许可证是否存在、是否有效、是否包含指定产品的授权以及是否超过限制
func (s *LicenseService) ValidateLicenseForUsage(ctx context.Context, license *entity.License, productID uint, currentNodes int, currentConcurrent int) (bool, error) {
	// Refresh expiration status
	license.CheckAndUpdateExpiration(time.Now())
	if license.Status != entity.StatusActive {
		return false, errors.New("license not active")
	}
	// Check scope for product
	for _, scope := range license.ScopeList {
		if scope.ProductID == productID {
			// check node limit and concurrent
			if (scope.MaxNodes == 0 || currentNodes <= scope.MaxNodes) && (scope.ConcurrentLimit == 0 || currentConcurrent <= scope.ConcurrentLimit) {
				return true, nil
			}
			return false, errors.New("license scope limits exceeded")
		}
	}
	return false, errors.New("license does not cover product")
}
