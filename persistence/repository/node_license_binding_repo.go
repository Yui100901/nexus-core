package repository

import (
	"context"
	"errors"
	"nexus-core/domain/entity"
	"nexus-core/persistence/base"
	"nexus-core/persistence/model"

	"gorm.io/gorm"
)

//
// @Author yfy2001
// @Date 2026/1/30 13 58
//

type NodeLicenseBindingRepository struct {
	db *gorm.DB
}

func NewNodeLicenseBindingRepository() *NodeLicenseBindingRepository {
	return &NodeLicenseBindingRepository{
		db: base.Connect(),
	}
}

// AddBinding 添加节点绑定关系（回填 ID）
func (r *NodeLicenseBindingRepository) AddBinding(ctx context.Context, binding *entity.NodeLicenseBinding) error {
	pBinding := &model.NodeLicenseBinding{
		NodeID:    binding.NodeID,
		LicenseID: binding.LicenseID,
		IsBound:   binding.IsBound,
	}
	if err := gorm.G[model.NodeLicenseBinding](r.db).Create(ctx, pBinding); err != nil {
		return err
	}
	binding.ID = pBinding.ID
	return nil
}

// UpdateBindingStatus 更新绑定状态
func (r *NodeLicenseBindingRepository) UpdateBindingStatus(ctx context.Context, id uint, status int) error {
	_, err := gorm.G[model.NodeLicenseBinding](r.db).
		Where("id = ?", id).
		Update(ctx, "bound_status", status)
	return err
}

// GetBindingsByNodeID 获取节点的绑定关系列表
func (r *NodeLicenseBindingRepository) GetBindingsByNodeID(ctx context.Context, nodeID uint) ([]entity.NodeLicenseBinding, error) {
	var res []entity.NodeLicenseBinding
	ms, err := gorm.G[model.NodeLicenseBinding](r.db).
		Where("node_id = ?", nodeID).
		Find(ctx)
	if err != nil {
		return res, err
	}
	for _, b := range ms {
		res = append(res, *toEntityNodeLicenseBinding(&b))
	}
	return res, nil
}

// GetBindingsByLicenseID 获取许可证的绑定关系列表
func (r *NodeLicenseBindingRepository) GetBindingsByLicenseID(ctx context.Context, licenseID uint) ([]entity.NodeLicenseBinding, error) {
	var res []entity.NodeLicenseBinding
	ms, err := gorm.G[model.NodeLicenseBinding](r.db).
		Where("license_id = ?", licenseID).
		Find(ctx)
	if err != nil {
		return res, err
	}
	for _, b := range ms {
		res = append(res, *toEntityNodeLicenseBinding(&b))
	}
	return res, nil
}

// GetBindingByNodeAndLicense 查询指定节点和许可证的绑定关系
func (r *NodeLicenseBindingRepository) GetBindingByNodeAndLicense(
	ctx context.Context,
	nodeID,
	licenseID uint,
) (*entity.NodeLicenseBinding, bool, error) {
	m, err := gorm.G[*model.NodeLicenseBinding](r.db).
		Where("node_id = ? AND license_id = ?", nodeID, licenseID).
		First(ctx)
	if err != nil {
		// 如果是未找到记录的错误，返回 false
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		// 其他错误直接返回
		return nil, false, err
	}
	// 找到记录，返回 true
	return toEntityNodeLicenseBinding(m), true, nil
}

// CountActiveBindingsByLicense 统计某许可已绑定的节点数量（IsBound = active (0)）
func (r *NodeLicenseBindingRepository) CountActiveBindingsByLicense(ctx context.Context, licenseID uint) (int64, error) {
	var cnt int64
	if err := r.db.WithContext(ctx).
		Model(&model.NodeLicenseBinding{}).
		Where("license_id = ? AND bound_status = ?", licenseID, model.BoundStatusActive).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

func toEntityNodeLicenseBinding(b *model.NodeLicenseBinding) *entity.NodeLicenseBinding {
	return &entity.NodeLicenseBinding{
		ID:        b.ID,
		NodeID:    b.NodeID,
		LicenseID: b.LicenseID,
		IsBound:   b.IsBound,
	}
}
