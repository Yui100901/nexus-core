package repository

import (
	"errors"
	"nexus-core/domain/entity"
	"nexus-core/persistence/model"
	"nexus-core/sc"

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

// CreateNode 创建节点（回填 ID）
func (r *NodeRepository) CreateNode(ctx *sc.ServiceContext, db *gorm.DB, node *entity.Node) error {
	pNode := &model.Node{
		DeviceCode: node.DeviceCode,
		MetaInfo:   node.Metadata,
	}
	if err := gorm.G[model.Node](db).Create(ctx, pNode); err != nil {
		return err
	}
	// 回填信息
	node.ID = pNode.ID
	return nil
}

// BatchCreateNode 批量创建节点（回填 ID）
func (r *NodeRepository) BatchCreateNode(ctx *sc.ServiceContext, db *gorm.DB, nodes []*entity.Node) error {
	// Step 1: 转换成持久化模型
	var pNodes []model.Node
	for _, node := range nodes {
		pNodes = append(pNodes, model.Node{
			DeviceCode: node.DeviceCode,
			MetaInfo:   node.Metadata,
		})
	}

	// Step 2: 批量插入 Node
	if err := gorm.G[model.Node](db).CreateInBatches(ctx, &pNodes, 100); err != nil {
		return err
	}

	// Step 3: 回填 ID
	for i := range nodes {
		nodes[i].ID = pNodes[i].ID
	}

	return nil
}

// GetByID 根据 ID 获取节点信息
func (r *NodeRepository) GetByID(ctx *sc.ServiceContext, db *gorm.DB, id uint) (*entity.Node, error) {
	m, err := GetOneByUniqueColumn[model.Node](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}

	return toEntityNode(m), nil
}

// GetByDeviceCode 根据设备码获取节点信息
func (r *NodeRepository) GetByDeviceCode(ctx *sc.ServiceContext, db *gorm.DB, deviceCode string) (*entity.Node, error) {
	m, err := gorm.G[*model.Node](db).
		Where("device_code = ?", deviceCode).
		First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return toEntityNode(m), nil
}

// DeleteNode 删除节点
func (r *NodeRepository) DeleteNode(ctx *sc.ServiceContext, db *gorm.DB, id uint) error {
	if _, err := gorm.G[model.Node](db).
		Where("id = ?", id).
		Delete(ctx); err != nil {
		return err
	}
	return nil
}

// ForceUnbind 强制解绑节点绑定
// 将指定绑定的解绑时间设置为当前时间并将状态更新为解绑状态
func (r *NodeRepository) ForceUnbind(ctx *sc.ServiceContext, db *gorm.DB, bindingID uint) error {
	_, err := gorm.G[model.NodeLicenseBinding](db).
		Where("id = ?", bindingID).
		Updates(ctx, model.NodeLicenseBinding{
			IsBound: 0,
		})
	return err
}

// GetBindingByID 根据ID获取绑定信息
func (r *NodeRepository) GetBindingByID(ctx *sc.ServiceContext, db *gorm.DB, id uint) (*entity.NodeLicenseBinding, error) {
	m, err := GetOneByUniqueColumn[model.NodeLicenseBinding](ctx, db, "id", id)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &entity.NodeLicenseBinding{
		ID:        m.ID,
		LicenseID: m.LicenseID,
		IsBound:   m.IsBound,
	}, nil
}

// GetBindingsByLicenseAndProduct 根据许可证获取绑定列表
func (r *NodeRepository) GetBindingsByLicenseAndProduct(ctx *sc.ServiceContext, db *gorm.DB, licenseID, productID uint) ([]entity.NodeLicenseBinding, error) {
	modelBindings, err := gorm.G[model.NodeLicenseBinding](db).
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
		Banned:     m.Banned,
		Metadata:   m.MetaInfo,
	}
}
