package service

import (
	"context"
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/repository"
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
func GetLicenseEntityByID(id uint) (*entity.License, error) {
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

// GetLicenseEntityByKey 获取license实体
func GetLicenseEntityByKey(key string) (*entity.License, error) {
	pLicense, err := licenseRepo.GetByKey(context.Background(), global.DB, key)
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

// GetNodeEntityByID 获取node实体
func GetNodeEntityByID(id uint) (*entity.Node, error) {
	pNode, err := nodeRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	metadata := string(pNode.Metadata)
	return &entity.Node{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// GetNodeEntityByCode 获取node实体
func GetNodeEntityByCode(code string) (*entity.Node, error) {
	pNode, err := nodeRepo.GetByDeviceCode(context.Background(), global.DB, code)
	if err != nil {
		return nil, err
	}
	metadata := string(pNode.Metadata)
	return &entity.Node{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// GetProductEntityByID 获取产品实体
func GetProductEntityByID(id uint) (*entity.Product, error) {
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
