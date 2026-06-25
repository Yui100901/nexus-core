package repository

import (
	"context"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/20 10 29
//

type NodeRepository struct {
}

func NewNodeRepository() *NodeRepository {
	return &NodeRepository{}
}

// Create 创建节点
func (r *NodeRepository) Create(ctx context.Context, db *gorm.DB, node *model.Node) error {
	if err := gorm.G[model.Node](db).Create(ctx, node); err != nil {
		return err
	}
	return nil
}

// BatchCreateNode 批量创建节点（回填 ID）
func (r *NodeRepository) BatchCreateNode(ctx context.Context, db *gorm.DB, nodes []model.Node) error {
	return gorm.G[model.Node](db).CreateInBatches(ctx, &nodes, 100)
}

// GetByID 根据 ID 获取节点信息
func (r *NodeRepository) GetByID(ctx context.Context, db *gorm.DB, id uint) (*model.Node, error) {
	m, err := GetOneByUniqueColumn[model.Node](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}

// GetByDeviceCode 根据设备码获取节点信息
func (r *NodeRepository) GetByDeviceCode(ctx context.Context, db *gorm.DB, deviceCode string) (*model.Node, error) {
	m, err := GetOneByUniqueColumn[model.Node](ctx, db, "device_code", deviceCode)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return m, nil
}

// DeleteNode 删除节点
func (r *NodeRepository) DeleteNode(ctx context.Context, db *gorm.DB, id uint) error {
	if _, err := gorm.G[model.Node](db).
		Where("id = ?", id).
		Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ForceUnbind 强制解绑节点绑定
// 将指定绑定的解绑时间设置为当前时间并将状态更新为解绑状态
func (r *NodeRepository) ForceUnbind(ctx context.Context, db *gorm.DB, bindingID uint) error {
	_, err := gorm.G[model.NodeLicenseBinding](db).
		Where("id = ?", bindingID).
		Updates(ctx, model.NodeLicenseBinding{
			Status: 0,
		})
	return err
}

// GetBindingsByLicenseAndProduct 根据许可证获取绑定列表
func (r *NodeRepository) GetBindingsByLicenseAndProduct(ctx context.Context, db *gorm.DB, licenseID, productID uint) ([]model.NodeLicenseBinding, error) {
	bindings, err := gorm.G[model.NodeLicenseBinding](db).
		Where("license_id = ? AND product_id = ?", licenseID, productID).
		Find(ctx)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}
