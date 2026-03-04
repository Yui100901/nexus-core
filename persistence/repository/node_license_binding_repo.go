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
// @Date 2026/1/30 13 58
//

type NodeLicenseBindingRepository struct {
}

func NewNodeLicenseBindingRepository() *NodeLicenseBindingRepository {
	return &NodeLicenseBindingRepository{}
}

// AddBinding 添加节点绑定关系（回填 ID）
func (r *NodeLicenseBindingRepository) AddBinding(ctx *sc.ServiceContext, db *gorm.DB, binding *entity.NodeLicenseBinding) error {
	pBinding := &model.NodeLicenseBinding{
		NodeID:    binding.NodeID,
		LicenseID: binding.LicenseID,
		IsBound:   binding.IsBound,
	}
	if err := gorm.G[model.NodeLicenseBinding](db).Create(ctx, pBinding); err != nil {
		return err
	}
	binding.ID = pBinding.ID
	return nil
}

// UpdateBindingStatus 更新绑定状态
func (r *NodeLicenseBindingRepository) UpdateBindingStatus(ctx *sc.ServiceContext, db *gorm.DB, id uint, status int) error {
	_, err := gorm.G[model.NodeLicenseBinding](db).
		Where("id = ?", id).
		Update(ctx, "bound_status", status)
	return err
}

// GetBindingsByNodeID 获取节点的绑定关系列表
func (r *NodeLicenseBindingRepository) GetBindingsByNodeID(ctx *sc.ServiceContext, db *gorm.DB, nodeID uint) ([]entity.NodeLicenseBinding, error) {
	var res []entity.NodeLicenseBinding
	ms, err := gorm.G[model.NodeLicenseBinding](db).
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
func (r *NodeLicenseBindingRepository) GetBindingsByLicenseID(ctx *sc.ServiceContext, db *gorm.DB, licenseID uint) ([]entity.NodeLicenseBinding, error) {
	var res []entity.NodeLicenseBinding
	ms, err := gorm.G[model.NodeLicenseBinding](db).
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
	ctx *sc.ServiceContext,
	db *gorm.DB,
	nodeID,
	licenseID uint,
) (*entity.NodeLicenseBinding, error) {
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
	return toEntityNodeLicenseBinding(m), nil
}

// CountActiveBindingsByLicense 统计某许可已绑定的节点数量（IsBound = 1）
func (r *NodeLicenseBindingRepository) CountActiveBindingsByLicense(ctx *sc.ServiceContext, db *gorm.DB, licenseID uint) (int64, error) {
	var cnt int64
	if err := db.WithContext(ctx).
		Model(&model.NodeLicenseBinding{}).
		Where("license_id = ? AND is_bound = ?", licenseID, 1).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// CountActiveBindingsByLicenseForProduct 统计某许可下某个产品已绑定的节点数量（IsBound = 1）
func (r *NodeLicenseBindingRepository) CountActiveBindingsByLicenseForProduct(
	ctx *sc.ServiceContext,
	db *gorm.DB,
	licenseID, productID uint) (int64, error) {
	var cnt int64
	if err := db.WithContext(ctx).
		Model(&model.NodeLicenseBinding{}).
		Where("license_id = ? AND product_id = ? AND is_bound = ?", licenseID, productID, 1).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	return cnt, nil
}

// DeleteBindingByNodeID 删除指定节点的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByNodeID(ctx *sc.ServiceContext, db *gorm.DB, nodeID uint) error {
	return db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteBindingByLicenseID 删除指定许可证的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByLicenseID(ctx *sc.ServiceContext, db *gorm.DB, licenseID uint) error {
	return db.WithContext(ctx).
		Where("license_id = ?", licenseID).
		Delete(&model.NodeLicenseBinding{}).Error
}

// DeleteBindingByProductID 删除指定产品的绑定关系
func (r *NodeLicenseBindingRepository) DeleteBindingByProductID(ctx *sc.ServiceContext, db *gorm.DB, productID uint) error {
	return db.WithContext(ctx).
		Where("product_id = ?", productID).
		Delete(&model.NodeLicenseBinding{}).Error
}

func toEntityNodeLicenseBinding(b *model.NodeLicenseBinding) *entity.NodeLicenseBinding {
	return &entity.NodeLicenseBinding{
		ID:        b.ID,
		NodeID:    b.NodeID,
		LicenseID: b.LicenseID,
		IsBound:   b.IsBound,
	}
}
