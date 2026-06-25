package repository

import (
	"context"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/16 15 58
//

type LicenseRepository struct {
}

func NewLicenseRepository() *LicenseRepository {
	return &LicenseRepository{}
}

// Create 创建 License
func (r *LicenseRepository) Create(ctx context.Context, db *gorm.DB, license *model.License) error {
	if err := gorm.G[model.License](db).Create(ctx, license); err != nil {
		return err
	}
	return nil
}

// BatchCreateLicense 批量创建 License
func (r *LicenseRepository) BatchCreateLicense(ctx context.Context, db *gorm.DB, licenses []model.License) error {
	return gorm.G[model.License](db).CreateInBatches(ctx, &licenses, 100)
}

// UpdateLicenseStatus 更新 License 状态
func (r *LicenseRepository) UpdateLicenseStatus(ctx context.Context, db *gorm.DB, id uint, status int) error {
	_, err := gorm.G[model.License](db).
		Where("id = ?", id).
		Update(ctx, "status", status)
	return err
}

// UpdateLicense 更新 License
func (r *LicenseRepository) UpdateLicense(ctx context.Context, db *gorm.DB, license *model.License) error {
	_, err := gorm.G[model.License](db).
		Where("id = ?", license.ID).
		Where("license_key = ?", license.LicenseKey).
		Updates(ctx, *license)
	if err != nil {
		return err
	}

	return nil
}

// GetByID 根据 id 获取领域对象 License
func (r *LicenseRepository) GetByID(ctx context.Context, db *gorm.DB, id uint) (*model.License, error) {
	m, err := GetOneByUniqueColumn[model.License](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}

// GetByKey 根据 LicenseKey 获取领域对象 License
func (r *LicenseRepository) GetByKey(ctx context.Context, db *gorm.DB, key string) (*model.License, error) {
	m, err := GetOneByUniqueColumn[model.License](ctx, db, "id", key)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}

func (r *LicenseRepository) GetIdListByStatus(ctx context.Context, db *gorm.DB, status int) ([]uint, error) {
	licenses, err := gorm.G[model.License](db).
		Select("id").
		Where("status = ?", status).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	var ids []uint
	for _, l := range licenses {
		ids = append(ids, l.ID)
	}
	return ids, nil
}

// BatchDeleteByIdList 批量删除 License
func (r *LicenseRepository) BatchDeleteByIdList(ctx context.Context, db *gorm.DB, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	//  删除 License
	if _, err := gorm.G[model.License](db).
		Where("id IN ?", ids).
		Delete(ctx); err != nil {
		return err
	}

	return nil
}
