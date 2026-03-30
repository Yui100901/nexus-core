package service

import (
	"context"
	"errors"
	"fmt"
	"nexus-core/api/dto"
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
		// 检查 Node 是否存在
		if _, err := GetNodeEntityByID(nodeID); err != nil {
			return err
		}

		// 检查 License 是否存在
		license, err := GetLicenseEntityByID(licenseID)
		if err != nil {
			return err
		}

		// 查找是否已有绑定关系
		var binding model.NodeLicenseBinding
		err = tx.Where("node_id = ? AND license_id = ?", nodeID, licenseID).
			First(&binding).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 检查当前绑定数量是否超过 MaxNodes
		var nodeCount int64
		if err := tx.Model(&model.NodeLicenseBinding{}).
			Where("license_id = ? AND is_bound = ?", licenseID, 1).
			Count(&nodeCount).Error; err != nil {
			return err
		}
		if int64(license.MaxNodes) <= nodeCount {
			return fmt.Errorf("license %d has reached max nodes (%d)", licenseID, license.MaxNodes)
		}

		// 已存在绑定记录
		if err == nil {
			if binding.Status == 1 {
				return nil // 已绑定，无需重复绑定
			}
			return tx.Model(&binding).Update("is_bound", 1).Error
		}

		// 插入新绑定
		newBinding := model.NodeLicenseBinding{
			NodeID:    nodeID,
			LicenseID: licenseID,
			Status:    1,
		}
		return tx.Create(&newBinding).Error
	})
}

func (s *NodeService) Unbind(cmd dto.UnbindCommand) error {
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
		return tx.Model(&binding).Update("is_bound", 0).Error
	})
}
