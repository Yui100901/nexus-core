package repository

import (
	"context"
	"nexus-core/domain/entity"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/16 15 58
//

type LicenseRepository struct {
	db *gorm.DB
}

func NewLicenseRepository() *LicenseRepository {
	return &LicenseRepository{
		db: base.Connect(),
	}
}

// CreateLicense 创建 License 及其 Scope 列表（回填 ID、时间戳、Scope ID）
func (r *LicenseRepository) CreateLicense(ctx context.Context, license *entity.License) error {
	pLicense := &model.License{
		LicenseKey:    license.LicenseKey,
		ValidityHours: license.ValidityHours,
		Status:        0,
	}
	if err := gorm.G[model.License](r.db).Create(ctx, pLicense); err != nil {
		return err
	}

	// 回填 License 信息
	license.ID = pLicense.ID
	license.ActivatedAt = pLicense.ActivatedAt
	license.ExpiredAt = pLicense.ExpiredAt
	license.Status = pLicense.Status

	// 保存 Scope 列表
	var pScopeList []model.LicenseScope
	for _, scope := range license.ScopeList {
		pScopeList = append(pScopeList, model.LicenseScope{
			LicenseID:   license.ID,
			ProductID:   scope.ProductID,
			FeatureMask: scope.FeatureMask,
		})
	}
	if len(pScopeList) > 0 {
		if err := gorm.G[model.LicenseScope](r.db).CreateInBatches(ctx, &pScopeList, 0); err != nil {
			return err
		}
		// 回填 Scope ID
		for i := range license.ScopeList {
			license.ScopeList[i].ID = pScopeList[i].ID
		}
	}

	return nil
}

// BatchCreateLicense 批量创建 License 及其 Scope 列表（回填 ID、Scope ID）
func (r *LicenseRepository) BatchCreateLicense(ctx context.Context, licenses []*entity.License) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pLicenses []model.License
		for _, license := range licenses {
			pLicenses = append(pLicenses, model.License{
				LicenseKey:    license.LicenseKey,
				ValidityHours: license.ValidityHours,
				Status:        0,
			})
		}

		if err := gorm.G[model.License](tx).CreateInBatches(ctx, &pLicenses, 100); err != nil {
			return err
		}

		// 回填 License 信息
		for i := range licenses {
			licenses[i].ID = pLicenses[i].ID
			licenses[i].ActivatedAt = pLicenses[i].ActivatedAt
			licenses[i].ExpiredAt = pLicenses[i].ExpiredAt
			licenses[i].Status = pLicenses[i].Status
		}

		var pScopeList []model.LicenseScope
		for _, license := range licenses {
			for _, scope := range license.ScopeList {
				pScopeList = append(pScopeList, model.LicenseScope{
					LicenseID:   license.ID,
					ProductID:   scope.ProductID,
					FeatureMask: scope.FeatureMask,
				})
			}
		}

		if len(pScopeList) > 0 {
			if err := gorm.G[model.LicenseScope](tx).CreateInBatches(ctx, &pScopeList, 100); err != nil {
				return err
			}
			// 回填 Scope ID
			idx := 0
			for i := range licenses {
				for j := range licenses[i].ScopeList {
					licenses[i].ScopeList[j].ID = pScopeList[idx].ID
					idx++
				}
			}
		}

		return nil
	})
}

// UpdateLicenseStatus 更新 License 状态
func (r *LicenseRepository) UpdateLicenseStatus(ctx context.Context, id uint, status int) error {
	_, err := gorm.G[model.License](r.db).
		Where("id = ?", id).
		Update(ctx, "status", status)
	return err
}

// UpdateLicense 更新 License
func (r *LicenseRepository) UpdateLicense(ctx context.Context, license *entity.License) error {
	pLicense := model.License{
		ValidityHours: license.ValidityHours,
		ActivatedAt:   license.ActivatedAt,
		ExpiredAt:     license.ExpiredAt,
		Status:        license.Status,
		Remark:        license.Remark,
	}

	_, err := gorm.G[model.License](r.db).
		Where("id = ?", license.ID).
		Where("license_key = ?", license.LicenseKey).
		Updates(ctx, pLicense)
	if err != nil {
		return err
	}

	return nil
}

// GetByID 根据 id 获取领域对象 License
func (r *LicenseRepository) GetByID(ctx context.Context, id uint) (*entity.License, error) {
	m, err := gorm.G[*model.License](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}

	scopes, err := r.GetScopeListByLicenseId(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityLicense(m, scopes), nil
}

// GetByKey 根据 LicenseKey 获取领域对象 License
func (r *LicenseRepository) GetByKey(ctx context.Context, key string) (*entity.License, error) {
	m, err := gorm.G[*model.License](r.db).
		Where("license_key = ?", key).
		First(ctx)
	if err != nil {
		return nil, err
	}

	scopes, err := r.GetScopeListByLicenseId(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityLicense(m, scopes), nil
}

func (r *LicenseRepository) GetIdListByStatus(ctx context.Context, status int) ([]uint, error) {
	licenses, err := gorm.G[model.License](r.db).
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

// BatchDeleteByIdList 批量删除 License 及其 Scope
func (r *LicenseRepository) BatchDeleteByIdList(ctx context.Context, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: 删除 Scope（依赖 LicenseID）
		if _, err := gorm.G[model.LicenseScope](tx).
			Where("license_id IN ?", ids).
			Delete(ctx); err != nil {
			return err
		}

		// Step 2: 删除 License
		if _, err := gorm.G[model.License](tx).
			Where("id IN ?", ids).
			Delete(ctx); err != nil {
			return err
		}

		return nil
	})
}

// GetScopeListByLicenseId 获取许可范围列表
func (r *LicenseRepository) GetScopeListByLicenseId(ctx context.Context, id uint) ([]model.LicenseScope, error) {
	return gorm.G[model.LicenseScope](r.db).Where("license_id = ?", id).Find(ctx)
}

// AddScope 添加 Scope
func (r *LicenseRepository) AddScope(ctx context.Context, licenseID uint, scope entity.Scope) error {
	pScope := &model.LicenseScope{
		LicenseID:   licenseID,
		ProductID:   scope.ProductID,
		FeatureMask: scope.FeatureMask,
	}
	return gorm.G[model.LicenseScope](r.db).Create(ctx, pScope)
}

func toEntityLicense(m *model.License, scopes []model.LicenseScope) *entity.License {
	var scopeList []entity.Scope
	for _, s := range scopes {
		scopeList = append(scopeList, entity.Scope{
			ID:          s.ID,
			ProductID:   s.ProductID,
			FeatureMask: s.FeatureMask,
		})
	}

	return &entity.License{
		ID:            m.ID,
		LicenseKey:    m.LicenseKey,
		ValidityHours: m.ValidityHours,
		ActivatedAt:   m.ActivatedAt,
		ExpiredAt:     m.ExpiredAt,
		Status:        m.Status,
		Remark:        m.Remark,
		ScopeList:     scopeList,
	}
}
