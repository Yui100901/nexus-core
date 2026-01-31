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
// @Date 2026/1/20 10 29
//

type NodeRepository struct {
	db *gorm.DB
}

func NewNodeRepository() *NodeRepository {
	return &NodeRepository{
		db: base.Connect(),
	}
}

// CreateNode 创建节点（回填 ID）
func (r *NodeRepository) CreateNode(ctx context.Context, node *entity.Node) error {
	pNode := &model.Node{
		DeviceCode: node.DeviceCode,
		MetaInfo:   node.MetaInfo,
	}
	if err := gorm.G[model.Node](r.db).Create(ctx, pNode); err != nil {
		return err
	}
	// 回填信息
	node.ID = pNode.ID
	return nil
}

// BatchCreateNode 批量创建节点（回填 ID）
func (r *NodeRepository) BatchCreateNode(ctx context.Context, nodes []*entity.Node) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Step 1: 转换成持久化模型
		var pNodes []model.Node
		for _, node := range nodes {
			pNodes = append(pNodes, model.Node{
				DeviceCode: node.DeviceCode,
				MetaInfo:   node.MetaInfo,
			})
		}

		// Step 2: 批量插入 Node
		if err := gorm.G[model.Node](tx).CreateInBatches(ctx, &pNodes, 100); err != nil {
			return err
		}

		// Step 3: 回填 ID
		for i := range nodes {
			nodes[i].ID = pNodes[i].ID
		}

		return nil
	})
}

// GetByID 根据 ID 获取节点信息
func (r *NodeRepository) GetByID(ctx context.Context, id uint) (*entity.Node, error) {
	m, err := gorm.G[*model.Node](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return toEntityNode(m), nil
}

// GetByDeviceCode 根据设备码获取节点信息
func (r *NodeRepository) GetByDeviceCode(ctx context.Context, deviceCode string) (*entity.Node, error) {
	m, err := gorm.G[*model.Node](r.db).
		Where("device_code = ?", deviceCode).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return toEntityNode(m), nil
}

// DeleteNode 删除节点及其绑定关系
func (r *NodeRepository) DeleteNode(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := gorm.G[model.NodeLicenseBinding](tx).
			Where("node_id = ?", id).
			Delete(ctx); err != nil {
			return err
		}
		if _, err := gorm.G[model.Node](tx).
			Where("id = ?", id).
			Delete(ctx); err != nil {
			return err
		}
		return nil
	})
}

// ForceUnbind 强制解绑节点绑定
// 将指定绑定的解绑时间设置为当前时间并将状态更新为解绑状态
func (r *NodeRepository) ForceUnbind(ctx context.Context, bindingID uint) error {
	_, err := gorm.G[model.NodeLicenseBinding](r.db).
		Where("id = ?", bindingID).
		Updates(ctx, model.NodeLicenseBinding{
			IsBound: 0,
		})
	return err
}

// GetBindingByID 根据ID获取绑定信息
func (r *NodeRepository) GetBindingByID(ctx context.Context, id uint) (*entity.NodeLicenseBinding, error) {
	m, err := gorm.G[*model.NodeLicenseBinding](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &entity.NodeLicenseBinding{
		ID:        m.ID,
		LicenseID: m.LicenseID,
		IsBound:   m.IsBound,
	}, nil
}

// GetBindingsByLicenseAndProduct 根据许可证获取绑定列表
func (r *NodeRepository) GetBindingsByLicenseAndProduct(ctx context.Context, licenseID, productID uint) ([]entity.NodeLicenseBinding, error) {
	modelBindings, err := gorm.G[model.NodeLicenseBinding](r.db).
		Where("license_id = ?", licenseID).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	var bindings []entity.NodeLicenseBinding
	for _, mb := range modelBindings {
		bindings = append(bindings, entity.NodeLicenseBinding{
			ID:        mb.ID,
			LicenseID: mb.LicenseID,
			IsBound:   mb.IsBound,
		})
	}
	return bindings, nil
}

// 转换为领域对象
func toEntityNode(m *model.Node) *entity.Node {
	return &entity.Node{
		ID:         m.ID,
		DeviceCode: m.DeviceCode,
		MetaInfo:   m.MetaInfo,
	}
}
