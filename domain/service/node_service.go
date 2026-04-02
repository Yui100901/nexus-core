package service

import (
	"context"
	"errors"
	"fmt"
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/global"
	"nexus-core/persistence/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
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
func (s *NodeService) CreateNode(cmd dto.CreateNodeCommand) (*dto.NodeData, error) {
	n := &model.Node{
		DeviceCode: cmd.DeviceCode,
		Metadata:   datatypes.JSON(*cmd.Metadata),
		Status:     0, //默认正常
	}
	err := nodeRepo.Create(context.Background(), global.DB, n)
	if err != nil {
		return nil, err
	}
	metadata := string(n.Metadata)
	return &dto.NodeData{
		ID:         n.ID,
		DeviceCode: n.DeviceCode,
		Status:     n.Status,
		Metadata:   &metadata,
	}, nil
}

// BatchCreateNode 批量创建节点
// 支持一次性创建多个节点
//func (s *NodeService) BatchCreateNode(nodes []*entity.Node) error {
//	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
//		return s.nr.BatchCreateNode(txCtx, txCtx.MustDefaultDB(), nodes)
//	})
//}

// GetNodeDataByID 根据ID获取节点信息
func (s *NodeService) GetNodeDataByID(id uint) (*dto.NodeData, error) {
	pNode, err := nodeRepo.GetByID(context.Background(), global.DB, id)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, fmt.Errorf("product not found")
	}
	metadata := string(pNode.Metadata)
	return &dto.NodeData{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// GetByDeviceCode 根据设备码获取节点信息
// 主要用于心跳验证时根据设备码查找节点
func (s *NodeService) GetByDeviceCode(code string) (*dto.NodeData, error) {
	pNode, err := nodeRepo.GetByDeviceCode(context.Background(), global.DB, code)
	if err != nil {
		return nil, err
	}
	if pNode == nil {
		return nil, fmt.Errorf("product not found")
	}
	metadata := string(pNode.Metadata)
	return &dto.NodeData{
		ID:         pNode.ID,
		DeviceCode: pNode.DeviceCode,
		Status:     pNode.Status,
		Metadata:   &metadata,
	}, nil
}

// DeleteNode 删除节点，并移除所有绑定
func (s *NodeService) DeleteNode(id uint) error {
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", id).Delete(&model.Node{}).Error; err != nil {
			return err
		}
		if err := tx.Where("node_id = ?", id).Delete(&model.NodeLicenseBinding{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (s *NodeService) AddBinding(cmd dto.AddBindingCommand) error {
	nodeID, licenseID := cmd.NodeID, cmd.LicenseID

	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 检查 License 是否存在
		license, err := GetLicenseEntityByID(licenseID)
		if err != nil || license == nil {
			return fmt.Errorf("invalid license")
		}

		// 检查 Node 是否存在
		n, err := GetNodeEntityByID(nodeID)
		if err != nil || n == nil {
			return fmt.Errorf("invalid node")
		}

		// 检查当前绑定数量是否超过 MaxNodes
		if !license.ValidateNodeLimit() {
			return fmt.Errorf("license has reached max nodes")
		}

		// 查找是否已有绑定关系
		var binding model.NodeLicenseBinding
		err = tx.Where("node_id = ? AND license_id = ?", nodeID, licenseID).
			First(&binding).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 已存在绑定记录
		if err == nil {
			if binding.Status == int(entity.BindingStatusBound) {
				return nil // 已绑定，无需重复绑定
			}
			// 更新绑定状态
			return tx.Model(&binding).Update("status", entity.BindingStatusBound).Error
		}

		// 插入新绑定
		newBinding := model.NodeLicenseBinding{
			NodeID:    nodeID,
			LicenseID: licenseID,
			Status:    int(entity.BindingStatusBound),
		}
		return tx.Create(&newBinding).Error
	})
}

func (s *NodeService) AutoBind(cmd dto.AutoBindCommand) error {
	deviceCode, licenseID := cmd.DeviceCode, cmd.LicenseID
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 检查 License 是否存在
		license, err := GetLicenseEntityByID(licenseID)
		if err != nil || license == nil {
			return fmt.Errorf("invalid license")
		}

		// 检查 Node 是否存在
		node, err := GetNodeEntityByCode(deviceCode)
		if err != nil {
			return fmt.Errorf("failed to get node")
		}
		if node == nil {
			//创建节点
			newNode := &model.Node{
				DeviceCode: cmd.DeviceCode,
				Status:     0, //默认正常
			}
			err = nodeRepo.Create(context.Background(), tx, newNode)
			if err != nil {
				return err
			}
			metadata := string(newNode.Metadata)
			node = &entity.Node{
				ID:         newNode.ID,
				DeviceCode: newNode.DeviceCode,
				Status:     0,
				Metadata:   &metadata,
			}
			// 插入新绑定
			newBinding := model.NodeLicenseBinding{
				NodeID:    node.ID,
				LicenseID: license.ID,
				Status:    int(entity.BindingStatusBound),
			}
			if err := tx.Create(&newBinding).Error; err != nil {
				return fmt.Errorf("create binding failed")
			}
		} else {
			// 查找是否已有绑定关系
			var binding model.NodeLicenseBinding
			err = tx.Where("node_id = ? AND license_id = ?", node.ID, license.ID).
				First(&binding).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			// 已存在绑定记录
			if err == nil {
				if binding.Status == 1 {
					return nil // 已绑定，无需重复绑定
				}
				if err := tx.Model(&binding).Update("is_bound", entity.BindingStatusBound).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

}

func (s *NodeService) UnbindByID(cmd dto.UnbindCommand) error {
	nodeID, licenseID := cmd.NodeID, cmd.LicenseID
	return global.DB.Transaction(func(tx *gorm.DB) error {
		// 查找是否已有绑定关系
		var binding model.NodeLicenseBinding
		err := tx.Where("node_id = ? AND license_id = ?", nodeID, licenseID).
			First(&binding).Error
		if err != nil {
			return err
		}
		if binding.Status == 0 {
			return nil
		}
		return tx.Model(&binding).Update("is_bound", entity.BindingStatusUnbound).Error
	})
}

func (s *NodeService) CleanUnboundNode() error {
	// 1. 查出所有已绑定的节点 ID（status=1）
	var boundNodeIDs []uint
	if err := global.DB.Model(&model.NodeLicenseBinding{}).
		Where("status = ?", entity.BindingStatusBound).
		Pluck("node_id", &boundNodeIDs).Error; err != nil {
		return err
	}

	// 2. 如果没有任何绑定，则删除所有节点
	if len(boundNodeIDs) == 0 {
		if err := global.DB.Delete(&model.Node{}).Error; err != nil {
			return err
		}
		return nil
	}

	// 3. 删除未绑定的节点
	if err := global.DB.Where("id NOT IN ?", boundNodeIDs).Delete(&model.Node{}).Error; err != nil {
		return err
	}

	return nil
}
