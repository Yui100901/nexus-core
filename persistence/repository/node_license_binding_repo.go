package repository

import (
	"context"
	"errors"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/30 13 58
//

type NodeLicenseBindingRepository struct {
}

func NewNodeLicenseBindingRepository() *NodeLicenseBindingRepository {
	return &NodeLicenseBindingRepository{}
}

// AddBinding 添加节点绑定关系（回填 ID）
func (r *NodeLicenseBindingRepository) AddBinding(ctx context.Context, db *gorm.DB, binding *model.NodeLicenseBinding) error {
	if err := gorm.G[model.NodeLicenseBinding](db).Create(ctx, binding); err != nil {
		return err
	}
	return nil
}

// UpdateBindingStatus 更新绑定状态
func (r *NodeLicenseBindingRepository) UpdateBindingStatus(ctx context.Context, db *gorm.DB, id uint, status int) error {
	_, err := gorm.G[model.NodeLicenseBinding](db).
		Where("id = ?", id).
		Update(ctx, "status", status)
	return err
}

// GetBindingsByNodeID 获取节点的绑定关系列表
func (r *NodeLicenseBindingRepository) GetBindingsByNodeID(ctx context.Context, db *gorm.DB, nodeID uint) ([]model.NodeLicenseBinding, error) {
	bindings, err := gorm.G[model.NodeLicenseBinding](db).
		Where("node_id = ?", nodeID).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

// GetBindingsByLicenseID 获取许可证的绑定关系列表
func (r *NodeLicenseBindingRepository) GetBindingsByLicenseID(ctx context.Context, db *gorm.DB, licenseID uint) ([]model.NodeLicenseBinding, error) {
	bindings, err := gorm.G[model.NodeLicenseBinding](db).
		Where("license_id = ?", licenseID).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

// GetBindingByNodeAndLicense 查询指定节点和许可证的绑定关系
func (r *NodeLicenseBindingRepository) GetBindingByNodeAndLicense(
	ctx context.Context,
	db *gorm.DB,
	nodeID,
	licenseID uint,
) (*model.NodeLicenseBinding, error) {
	m, err := gorm.G[*model.NodeLicenseBinding](db).
		Where("node_id = ? AND license_id = ?", nodeID, licenseID).
		First(ctx)
	if err != nil {
		// 如果是未找到记录的错误，返回 false
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		// 其他错误直接返回
		return nil, err
	}
	// 找到记录，返回 true
	return m, nil
}

// CountActiveBindingsByLicense 统计某许可已绑定的节点数量（Status = 1）
func (r *NodeLicenseBindingRepository) CountActiveBindingsByLicense(ctx context.Context, db *gorm.DB, licenseID uint) (int64, error) {
	var cnt int64
	if err := db.WithContext(ctx).
		Model(&model.NodeLicenseBinding{}).
		Where("license_id = ? AND status = ?", licenseID, 1).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// CountActiveBindingsByLicenseForProduct 统计某许可下某个产品已绑定的节点数量（Status = 1）
func (r *NodeLicenseBindingRepository) CountActiveBindingsByLicenseForProduct(
	ctx context.Context,
	db *gorm.DB,
	licenseID, productID uint) (int64, error) {
	var cnt int64
	if err := db.WithContext(ctx).
		Model(&model.NodeLicenseBinding{}).
		Where("license_id = ? AND product_id = ? AND status = ?", licenseID, productID, 1).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// DeleteBindingByNodeID 删除指定节点的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByNodeID(ctx context.Context, db *gorm.DB, nodeID uint) error {
	return db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteBindingByLicenseID 删除指定许可证的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByLicenseID(ctx context.Context, db *gorm.DB, licenseID uint) error {
	return db.WithContext(ctx).
		Where("license_id = ?", licenseID).
		Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteBindingByProductID 删除指定产品的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByProductID(ctx context.Context, db *gorm.DB, productID uint) error {
	return db.WithContext(ctx).
		Where("product_id = ?", productID).
		Delete(&model.NodeLicenseBinding{}).Error
}
