package service

import (
	"context"
	"errors"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/model"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NodeService 提供节点相关的业务逻辑服务
// 管理节点的创建、查询、绑定等操作
type NodeService struct {
}

// NewNodeService 创建新的节点服务实例
func NewNodeService() *NodeService {
	return &NodeService{}
}

// CreateNode 创建新节点
// 将节点信息持久化到数据库
func (s *NodeService) CreateNode(ctx context.Context, cmd CreateNodeCommand) (*NodeData, error) {
	var metadata datatypes.JSON
	if cmd.Metadata != nil {
		metadata = datatypes.JSON([]byte(*cmd.Metadata))
	}
	n := &model.Node{
		DeviceCode: cmd.DeviceCode,
		Metadata:   metadata,
		Status:     entity.NodeStatusNormal,
	}
	err := nodeRepo.Create(ctx, global.DB.WithContext(ctx), n)
	if err != nil {
		return nil, err
	}
	metadataString := string(n.Metadata)
	return &NodeData{
		ID:         n.ID,
		DeviceCode: n.DeviceCode,
		Status:     n.Status,
		Metadata:   &metadataString,
	}, nil
}

// GetNodeDataByID 根据ID获取节点信息
func (s *NodeService) GetNodeDataByID(ctx context.Context, id uint) (*NodeData, error) {
	pNode, err := nodeRepo.GetByID(ctx, global.DB.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, ErrNotFound("node not found")
	}
	metadata := string(pNode.Metadata)
	return &NodeData{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// GetByDeviceCode 根据设备码获取节点信息
// 主要用于心跳验证时根据设备码查找节点
func (s *NodeService) GetByDeviceCode(ctx context.Context, code string) (*NodeData, error) {
	pNode, err := nodeRepo.GetByDeviceCode(ctx, global.DB.WithContext(ctx), code)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, ErrNotFound("node not found")
	}
	metadata := string(pNode.Metadata)
	return &NodeData{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// DeleteNode 删除节点，并移除所有绑定
func (s *NodeService) DeleteNode(ctx context.Context, id uint) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&model.Node{}).Error; err != nil {
			return err
		}
		if err := tx.Where("node_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *NodeService) AddBinding(ctx context.Context, cmd AddBindingCommand) error {
	nodeID, licenseID := cmd.NodeID, cmd.LicenseID

	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查 License 是否存在
		var pLicense model.License
		if err := tx.Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
			Where("id = ?", licenseID).
			First(&pLicense).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound("license not found")
			}
			return WrapInternal("get license failed", err)
		}
		license := ToEntityLicense(&pLicense)

		// 检查 Node 是否存在
		n, err := GetNodeEntityByID(ctx, tx, nodeID)
		if err != nil {
			return WrapInternal("get node failed", err)
		}
		if n == nil {
			return ErrNotFound("node not found")
		}

		// 检查当前绑定数量是否超过 MaxNodes
		if !license.ValidateNodeLimit() {
			return ErrConflict("license has reached max nodes")
		}

		// 查找是否已有绑定关系
		var binding model.NodeLicenseBinding
		err = tx.Where("node_id = ? AND license_id = ?", nodeID, licenseID).
			First(&binding).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return WrapInternal("get binding failed", err)
		}
		toUpdate := false
		// 已存在绑定记录
		if err == nil {
			if binding.Status == int(entity.BindingStatusBound) {
				return nil // 已绑定，无需重复绑定
			}
			toUpdate = true
		}

		result := tx.Model(&model.License{}).
			Where("id = ? AND (max_nodes = 0 OR current_node_count < max_nodes)", licenseID).
			Update("current_node_count", gorm.Expr("current_node_count + ?", 1))
		if result.Error != nil {
			return WrapInternal("update license node count failed", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrConflict("license has reached max nodes")
		}

		if toUpdate {
			// 更新绑定状态
			now := time.Now()
			return tx.Model(&binding).Updates(map[string]interface{}{
				"status":     entity.BindingStatusBound,
				"bound_at":   &now,
				"unbound_at": nil,
			}).Error
		} else {
			// 插入新绑定
			now := time.Now()
			newBinding := model.NodeLicenseBinding{
				NodeID:    nodeID,
				LicenseID: licenseID,
				ProductID: license.ProductID,
				Status:    int(entity.BindingStatusBound),
				BoundAt:   &now,
			}
			return tx.Create(&newBinding).Error
		}

	})
}

func (s *NodeService) AutoBind(ctx context.Context, cmd AutoBindCommand) error {
	deviceCode, licenseID := cmd.DeviceCode, cmd.LicenseID
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 检查 License 是否存在
		license, err := GetLicenseEntityByID(ctx, tx, licenseID)
		if err != nil {
			return WrapInternal("get license failed", err)
		}
		if license == nil {
			return ErrNotFound("license not found")
		}

		// 检查 Node 是否存在
		node, err := GetNodeEntityByCode(ctx, tx, deviceCode)
		if err != nil {
			return WrapInternal("get node failed", err)
		}
		if node == nil {
			//创建节点
			newNode := &model.Node{
				DeviceCode: cmd.DeviceCode,
				Status:     0, //默认正常
			}
			err = nodeRepo.Create(ctx, tx, newNode)
			if err != nil {
				return WrapInternal("create node failed", err)
			}
			metadata := string(newNode.Metadata)
			node = &entity.Node{
				ID:         newNode.ID,
				DeviceCode: newNode.DeviceCode,
				Status:     0,
				Metadata:   &metadata,
			}
			// 插入新绑定
			now := time.Now()
			newBinding := model.NodeLicenseBinding{
				NodeID:    node.ID,
				LicenseID: license.ID,
				ProductID: license.ProductID,
				Status:    int(entity.BindingStatusBound),
				BoundAt:   &now,
			}
			if err := tx.Create(&newBinding).Error; err != nil {
				return WrapInternal("create binding failed", err)
			}
		} else {
			// 查找是否已有绑定关系
			var binding model.NodeLicenseBinding
			err = tx.Where("node_id = ? AND license_id = ?", node.ID, license.ID).
				First(&binding).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return WrapInternal("get binding failed", err)
			}
			// 已存在绑定记录
			if err == nil {
				if binding.Status == 1 {
					return nil // 已绑定，无需重复绑定
				}
				now := time.Now()
				if err := tx.Model(&binding).Updates(map[string]interface{}{
					"status":     entity.BindingStatusBound,
					"bound_at":   &now,
					"unbound_at": nil,
				}).Error; err != nil {
					return WrapInternal("update binding failed", err)
				}
			}
		}
		return nil
	})

}

func (s *NodeService) UnbindByID(ctx context.Context, cmd UnbindCommand) error {
	nodeID, licenseID := cmd.NodeID, cmd.LicenseID
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查找是否已有绑定关系
		var binding model.NodeLicenseBinding
		err := tx.Where("node_id = ? AND license_id = ?", nodeID, licenseID).
			First(&binding).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound("binding not found")
			}
			return WrapInternal("get binding failed", err)
		}
		if binding.Status == 0 {
			return nil
		}
		now := time.Now()
		return tx.Model(&binding).Updates(map[string]interface{}{
			"status":     entity.BindingStatusUnbound,
			"unbound_at": &now,
		}).Error
	})
}

func (s *NodeService) CleanUnboundNode(ctx context.Context) error {
	return global.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查出所有已绑定的节点 ID（status=1）
		var boundNodeIDs []uint
		if err := tx.Model(&model.NodeLicenseBinding{}).
			Where("status = ?", entity.BindingStatusBound).
			Pluck("node_id", &boundNodeIDs).Error; err != nil {
			return err
		}

		// 2. 如果没有任何绑定，则删除所有节点
		if len(boundNodeIDs) == 0 {
			if err := tx.Delete(&model.Node{}).Error; err != nil {
				return err
			}
			return nil
		}

		// 3. 删除未绑定的节点
		if err := tx.Where("id NOT IN ?", boundNodeIDs).Delete(&model.Node{}).Error; err != nil {
			return err
		}

		return nil
	})
}
