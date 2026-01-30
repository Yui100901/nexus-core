package service

import (
	"context"
	"nexus-core/domain/entity"
	"nexus-core/persistence/repository"
	"nexus-core/runtimecache"
)

// NodeService 提供节点相关的业务逻辑服务
// 管理节点的创建、查询、绑定等操作
type NodeService struct {
	nr  *repository.NodeRepository // 节点仓库，用于数据持久化操作
	nlr *repository.NodeLicenseBindingRepository
}

// NewNodeService 创建新的节点服务实例
func NewNodeService() *NodeService {
	return &NodeService{
		nr: repository.NewNodeRepository(),
	}
}

// CreateNode 创建新节点
// 将节点信息持久化到数据库
func (s *NodeService) CreateNode(ctx context.Context, n *entity.Node) error {
	return s.nr.CreateNode(ctx, n)
}

// BatchCreateNode 批量创建节点
// 支持一次性创建多个节点
func (s *NodeService) BatchCreateNode(ctx context.Context, nodes []*entity.Node) error {
	return s.nr.BatchCreateNode(ctx, nodes)
}

// GetByID 根据ID获取节点信息
// 返回指定ID的完整节点信息，包括所有绑定关系
func (s *NodeService) GetByID(ctx context.Context, id uint) (*entity.Node, error) {
	return s.nr.GetByID(ctx, id)
}

// GetByDeviceCode 根据设备码获取节点信息
// 主要用于心跳验证时根据设备码查找节点
func (s *NodeService) GetByDeviceCode(ctx context.Context, code string) (*entity.Node, error) {
	return s.nr.GetByDeviceCode(ctx, code)
}

// AddBinding 为节点添加许可证绑定关系
// 将指定的许可证与节点进行绑定
func (s *NodeService) AddBinding(ctx context.Context, nodeId uint, binding *entity.NodeLicenseBinding) error {
	node, err := s.nr.GetByID(ctx, nodeId)
	if err != nil {
		return err
	}
	node.Bind(binding)
	return s.nlr.AddBinding(ctx, nodeId, binding)
}

// UpdateBindingStatus 更新绑定状态
// 修改节点与许可证之间的绑定状态
func (s *NodeService) UpdateBindingStatus(ctx context.Context, id uint, status int) error {
	return s.nlr.UpdateBindingStatus(ctx, id, status)
}

// ForceUnbind 强制解绑节点绑定（根据绑定ID）
// 将指定的绑定状态更新为解绑状态，并记录解绑时间
// 同时更新运行时缓存，减少对应许可证和产品的节点计数
func (s *NodeService) ForceUnbind(ctx context.Context, bindingID uint) error {
	// 先获取完整的绑定信息以获取节点ID
	binding, err := s.nr.GetBindingByID(ctx, bindingID)
	if err != nil {
		return err
	}

	// 执行强制解绑操作
	err = s.nr.ForceUnbind(ctx, bindingID)
	if err != nil {
		return err
	}

	// 从运行时缓存中移除该节点的并发计数
	// 由于没有ProductID，我们暂时使用默认值0
	runtimecache.RemoveNodeConcurrent(binding.LicenseID, 0, binding.NodeID)

	return nil
}

// ForceUnbindByNodeAndLicense 根据节点ID和许可证ID强制解绑节点
// 查找对应的绑定关系并将其状态更新为解绑状态
func (s *NodeService) ForceUnbindByNodeAndLicense(ctx context.Context, nodeID, licenseID uint) error {
	// 查找节点和许可证之间的绑定关系
	binding, err := s.nlr.GetBindingByNodeAndLicense(ctx, nodeID, licenseID)
	if err != nil {
		return err
	}

	// 执行强制解绑操作
	err = s.nr.ForceUnbind(ctx, binding.ID)
	if err != nil {
		return err
	}

	// 从运行时缓存中移除该节点的并发计数
	// 由于没有ProductID，我们暂时使用默认值0
	runtimecache.RemoveNodeConcurrent(binding.LicenseID, 0, binding.NodeID)

	return nil
}

// DeleteNode 删除节点
// 同时删除节点的所有绑定关系
func (s *NodeService) DeleteNode(ctx context.Context, id uint) error {
	return s.nr.DeleteNode(ctx, id)
}
