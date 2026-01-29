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

	bindings, err := r.GetBindingsByNodeID(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityNode(m, bindings), nil
}

// GetByDeviceCode 根据设备码获取节点信息
func (r *NodeRepository) GetByDeviceCode(ctx context.Context, deviceCode string) (*entity.Node, error) {
	m, err := gorm.G[*model.Node](r.db).
		Where("device_code = ?", deviceCode).
		First(ctx)
	if err != nil {
		return nil, err
	}

	bindings, err := r.GetBindingsByNodeID(ctx, m.ID)
	if err != nil {
		return nil, err
	}

	return toEntityNode(m, bindings), nil
}

// AddBinding 添加节点绑定关系（回填 ID）
func (r *NodeRepository) AddBinding(ctx context.Context, nodeId uint, binding *entity.NodeBinding) error {
	pBinding := &model.NodeBinding{
		NodeID:    nodeId,
		LicenseID: binding.LicenseID,
		IsBound:   binding.IsBound,
	}
	if err := gorm.G[model.NodeBinding](r.db).Create(ctx, pBinding); err != nil {
		return err
	}
	binding.ID = pBinding.ID
	return nil
}

// UpdateBindingStatus 更新绑定状态
func (r *NodeRepository) UpdateBindingStatus(ctx context.Context, id uint, status int) error {
	_, err := gorm.G[model.NodeBinding](r.db).
		Where("id = ?", id).
		Update(ctx, "bound_status", status)
	return err
}

// GetBindingsByNodeID 获取节点的绑定关系列表
func (r *NodeRepository) GetBindingsByNodeID(ctx context.Context, nodeID uint) ([]model.NodeBinding, error) {
	return gorm.G[model.NodeBinding](r.db).
		Where("node_id = ?", nodeID).
		Find(ctx)
}

// DeleteNode 删除节点及其绑定关系
func (r *NodeRepository) DeleteNode(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := gorm.G[model.NodeBinding](tx).
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

// GetBindingByNodeAndLicense 查询指定节点和许可证的绑定关系
func (r *NodeRepository) GetBindingByNodeAndLicense(ctx context.Context, nodeID, licenseID uint) (*entity.NodeBinding, error) {
	m, err := gorm.G[*model.NodeBinding](r.db).
		Where("node_id = ? AND license_id = ?", nodeID, licenseID).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &entity.NodeBinding{
		ID:        m.ID,
		LicenseID: m.LicenseID,
		IsBound:   m.IsBound,
	}, nil
}

// GetBindingByNodeAndLicenseProduct 查询指定节点上指定许可和产品的绑定关系
func (r *NodeRepository) GetBindingByNodeAndLicenseProduct(ctx context.Context, nodeID, licenseID, productID uint) (*entity.NodeBinding, error) {
	m, err := gorm.G[*model.NodeBinding](r.db).
		Where("node_id = ? AND license_id = ?", nodeID, licenseID).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &entity.NodeBinding{
		ID:        m.ID,
		LicenseID: m.LicenseID,
		IsBound:   m.IsBound,
	}, nil
}

// CountActiveBindingsByLicenseAndProduct 统计某许可下某产品已绑定的节点数量（IsBound = active (0)）
func (r *NodeRepository) CountActiveBindingsByLicenseAndProduct(ctx context.Context, licenseID, productID uint) (int64, error) {
	var cnt int64
	if err := r.db.WithContext(ctx).
		Model(&model.NodeBinding{}).
		Where("license_id = ? AND bound_status = ?", licenseID, model.BoundStatusActive).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// ForceUnbind 强制解绑节点绑定
// 将指定绑定的解绑时间设置为当前时间并将状态更新为解绑状态
func (r *NodeRepository) ForceUnbind(ctx context.Context, bindingID uint) error {
	_, err := gorm.G[model.NodeBinding](r.db).
		Where("id = ?", bindingID).
		Updates(ctx, model.NodeBinding{
			IsBound: model.BoundStatusUnbound,
		})
	return err
}

// GetBindingByID 根据ID获取绑定信息
func (r *NodeRepository) GetBindingByID(ctx context.Context, id uint) (*entity.NodeBinding, error) {
	m, err := gorm.G[*model.NodeBinding](r.db).
		Where("id = ?", id).
		First(ctx)
	if err != nil {
		return nil, err
	}
	return &entity.NodeBinding{
		ID:        m.ID,
		LicenseID: m.LicenseID,
		IsBound:   m.IsBound,
	}, nil
}

// GetBindingsByLicenseAndProduct 根据许可证获取绑定列表
func (r *NodeRepository) GetBindingsByLicenseAndProduct(ctx context.Context, licenseID, productID uint) ([]entity.NodeBinding, error) {
	modelBindings, err := gorm.G[model.NodeBinding](r.db).
		Where("license_id = ?", licenseID).
		Find(ctx)
	if err != nil {
		return nil, err
	}

	var bindings []entity.NodeBinding
	for _, mb := range modelBindings {
		bindings = append(bindings, entity.NodeBinding{
			ID:        mb.ID,
			LicenseID: mb.LicenseID,
			IsBound:   mb.IsBound,
		})
	}
	return bindings, nil
}

// 转换为领域对象
func toEntityNode(m *model.Node, bindings []model.NodeBinding) *entity.Node {
	var bindingList []entity.NodeBinding
	for _, b := range bindings {
		bindingList = append(bindingList, entity.NodeBinding{
			ID:        b.ID,
			LicenseID: b.LicenseID,
			IsBound:   b.IsBound,
		})
	}
	return &entity.Node{
		ID:         m.ID,
		DeviceCode: m.DeviceCode,
		MetaInfo:   m.MetaInfo,
		Bindings:   bindingList,
	}
}
