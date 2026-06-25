package service

import (
	"context"
	"errors"
	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
	"time"

	"gorm.io/gorm"
)

func bindNodeToLicense(ctx context.Context, tx *gorm.DB, nodeID uint, license *entity.License, productID uint) (bool, error) {
	var binding model.NodeLicenseBinding
	err := tx.WithContext(ctx).
		Where("node_id = ? AND license_id = ?", nodeID, license.ID).
		First(&binding).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, WrapInternal("get binding failed", err)
	}
	if err == nil && binding.Status == int(entity.BindingStatusBound) {
		return false, nil
	}

	if err := incrementLicenseNodeCount(ctx, tx, license.ID); err != nil {
		return false, err
	}

	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		binding = model.NodeLicenseBinding{
			NodeID:    nodeID,
			LicenseID: license.ID,
			ProductID: productID,
			Status:    int(entity.BindingStatusBound),
			BoundAt:   &now,
		}
		if err := tx.WithContext(ctx).Create(&binding).Error; err != nil {
			return false, WrapInternal("create binding failed", err)
		}
	} else {
		if err := tx.WithContext(ctx).Model(&binding).Updates(map[string]interface{}{
			"product_id": productID,
			"status":     entity.BindingStatusBound,
			"bound_at":   &now,
			"unbound_at": nil,
			"updated_at": now,
		}).Error; err != nil {
			return false, WrapInternal("update binding failed", err)
		}
	}

	license.CurrentNodeCount++
	return true, nil
}

func incrementLicenseNodeCount(ctx context.Context, tx *gorm.DB, licenseID uint) error {
	result := tx.WithContext(ctx).Model(&model.License{}).
		Where("id = ? AND (max_nodes = 0 OR current_node_count < max_nodes)", licenseID).
		Update("current_node_count", gorm.Expr("current_node_count + ?", 1))
	if result.Error != nil {
		return WrapInternal("update license node count failed", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrConflict("license has reached max nodes")
	}
	return nil
}

func decrementLicenseNodeCount(ctx context.Context, tx *gorm.DB, licenseID uint) error {
	return tx.WithContext(ctx).Model(&model.License{}).
		Where("id = ?", licenseID).
		Update("current_node_count", gorm.Expr("CASE WHEN current_node_count > 0 THEN current_node_count - 1 ELSE 0 END")).Error
}

func resetLicenseNodeCount(ctx context.Context, tx *gorm.DB, licenseID uint) error {
	return tx.WithContext(ctx).Model(&model.License{}).
		Where("id = ?", licenseID).
		Update("current_node_count", 0).Error
}
