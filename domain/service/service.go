package service

import (
	"context"
	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
	"nexus-core/persistence/repository"
	"time"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/3/26 14 52
//

var (
	productRepo        = repository.NewProductRepository()
	productVersionRepo = repository.NewProductVersionRepository()
	nodeRepo           = repository.NewNodeRepository()
	licenseRepo        = repository.NewLicenseRepository()
)

// GetLicenseEntityByID 获取license实体
func GetLicenseEntityByID(ctx context.Context, db *gorm.DB, id uint) (*entity.License, error) {
	pLicense, err := licenseRepo.GetByID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if pLicense == nil {
		return nil, nil
	}
	return hydrateLicenseEntity(ctx, db, pLicense)
}

// GetLicenseEntityByKey 获取license实体
func GetLicenseEntityByKey(ctx context.Context, db *gorm.DB, key string) (*entity.License, error) {
	pLicense, err := licenseRepo.GetByKey(ctx, db, key)
	if err != nil {
		return nil, err
	}
	if pLicense == nil {
		return nil, nil
	}
	return hydrateLicenseEntity(ctx, db, pLicense)
}

func ToEntityLicense(pLicense *model.License) *entity.License {
	return &entity.License{
		ID:               pLicense.ID,
		ProductID:        pLicense.ProductID,
		LicenseKey:       pLicense.LicenseKey,
		ValidityHours:    pLicense.ValidityHours,
		IssuedAt:         pLicense.CreatedAt,
		ActivatedAt:      pLicense.ActivatedAt,
		ExpiredAt:        pLicense.ExpiredAt,
		Status:           entity.LicenseStatus(pLicense.Status),
		Remark:           pLicense.Remark,
		MaxNodes:         pLicense.MaxNodes,
		CurrentNodeCount: pLicense.CurrentNodeCount,
		MaxConcurrent:    pLicense.MaxConcurrent,
		FeatureMask:      pLicense.FeatureMask,
	}
}

func hydrateLicenseEntity(ctx context.Context, db *gorm.DB, pLicense *model.License) (*entity.License, error) {
	license := ToEntityLicense(pLicense)
	currentStatus := license.CalculateStatus(time.Now())
	if currentStatus != license.Status {
		license.Status = currentStatus
		if err := db.WithContext(ctx).Model(&model.License{}).
			Where("id = ?", license.ID).
			Update("status", int(currentStatus)).Error; err != nil {
			return nil, err
		}
	}
	return license, nil
}

// GetNodeEntityByID 获取node实体
func GetNodeEntityByID(ctx context.Context, db *gorm.DB, id uint) (*entity.Node, error) {
	pNode, err := nodeRepo.GetByID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, nil
	}
	return ToEntityNode(pNode), nil
}

// GetNodeEntityByCode 获取node实体
func GetNodeEntityByCode(ctx context.Context, db *gorm.DB, code string) (*entity.Node, error) {
	pNode, err := nodeRepo.GetByDeviceCode(ctx, db, code)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, nil
	}
	return ToEntityNode(pNode), nil
}

func ToEntityNode(pNode *model.Node) *entity.Node {
	metadata := string(pNode.Metadata)
	return &entity.Node{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}
}

// GetProductEntityByID 获取产品实体
func GetProductEntityByID(ctx context.Context, db *gorm.DB, id uint) (*entity.Product, error) {
	pProduct, err := productRepo.GetByID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	if pProduct == nil {
		return nil, nil
	}
	pVersionList, err := productVersionRepo.ListByProductID(ctx, db, id)
	if err != nil {
		return nil, err
	}
	return ToEntityProduct(pProduct, pVersionList), nil
}

func ToEntityProduct(pProduct *model.Product, pVersionList []model.ProductVersion) *entity.Product {
	var versionList []entity.Version
	for _, v := range pVersionList {
		version := *ToEntityVersion(&v)
		versionList = append(versionList, version)
	}
	return &entity.Product{
		ID:                    pProduct.ID,
		Name:                  pProduct.Name,
		Description:           pProduct.Description,
		MinSupportedVersionID: pProduct.MinSupportedVersionID,
		VersionList:           versionList,
	}
}

func ToEntityVersion(pVersion *model.ProductVersion) *entity.Version {
	return &entity.Version{
		ID:          pVersion.ID,
		VersionCode: pVersion.VersionCode,
		ReleaseDate: pVersion.ReleaseDate,
		Description: pVersion.Description,
		Status:      pVersion.Status,
	}
}
