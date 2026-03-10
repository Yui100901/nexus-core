package service

import (
	"fmt"
	"nexus-core/domain/entity"
	"nexus-core/persistence/base"
	"nexus-core/persistence/repository"
	"nexus-core/sc"
)

// NodeService 提供节点相关的业务逻辑服务
// 管理节点的创建、查询、绑定等操作
type NodeService struct {
	// db removed; use DB from sc.ServiceContext at runtime
	nr  *repository.NodeRepository // 节点仓库，用于数据持久化操作
	nlr *repository.NodeLicenseBindingRepository
}

// NewNodeService 创建新的节点服务实例
func NewNodeService() *NodeService {
	return &NodeService{
		nr:  repository.NewNodeRepository(),
		nlr: repository.NewNodeLicenseBindingRepository(),
	}
}

// CreateNode 创建新节点
// 将节点信息持久化到数据库
func (s *NodeService) CreateNode(ctx *sc.ServiceContext, n *entity.Node) error {
	db := ctx.MustDefaultDB()
	return s.nr.CreateNode(ctx, db, n)
}

// AutoCreateNode 自动创建节点
// 根据设备码自动创建节点，适用于心跳验证时自动注册新节点
func (s *NodeService) AutoCreateNode(ctx *sc.ServiceContext, deviceCode string, metadata *string) (*entity.Node, error) {
	db := ctx.MustDefaultDB()
	// 查找或创建 node
	node, err := s.nr.GetByDeviceCode(ctx, db, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get node failed")
	}
	//node不存在
	if node == nil {
		n, err := entity.NewNode(deviceCode, metadata)
		if err != nil {
			return nil, fmt.Errorf("create node failed")
		}
		if err := s.nr.CreateNode(ctx, db, n); err != nil {
			return nil, fmt.Errorf("create node failed")
		}
		node = n
	}
	return node, nil
}

// AutoCreateNodeWithContext variant uses DB/Tx from sCtx (if present) and does NOT start a transaction itself
func (s *NodeService) AutoCreateNodeWithContext(sCtx *sc.ServiceContext, deviceCode string, metadata *string) (*entity.Node, error) {
	db := sCtx.MustDefaultDB()

	// 查找或创建 node using provided db (which may be a tx)
	node, err := s.nr.GetByDeviceCode(sCtx, db, deviceCode)
	if err != nil {
		return nil, fmt.Errorf("get node failed")
	}
	if node == nil {
		n, err := entity.NewNode(deviceCode, metadata)
		if err != nil {
			return nil, fmt.Errorf("create node failed")
		}
		if err := s.nr.CreateNode(sCtx, db, n); err != nil {
			return nil, fmt.Errorf("create node failed")
		}
		node = n
	}
	return node, nil
}

// BatchCreateNode 批量创建节点
// 支持一次性创建多个节点
func (s *NodeService) BatchCreateNode(ctx *sc.ServiceContext, nodes []*entity.Node) error {
	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
		return s.nr.BatchCreateNode(txCtx, txCtx.MustDefaultDB(), nodes)
	})
}

// GetByID 根据ID获取节点信息
// 返回指定ID的完整节点信息，包括所有绑定关系
func (s *NodeService) GetByID(ctx *sc.ServiceContext, id uint) (*entity.Node, error) {
	db := ctx.MustDefaultDB()
	return s.nr.GetByID(ctx, db, id)
}

// GetByDeviceCode 根据设备码获取节点信息
// 主要用于心跳验证时根据设备码查找节点
func (s *NodeService) GetByDeviceCode(ctx *sc.ServiceContext, code string) (*entity.Node, error) {
	db := ctx.MustDefaultDB()
	return s.nr.GetByDeviceCode(ctx, db, code)
}

// AddBinding 为节点添加许可证绑定关系
// 将指定的许可证与节点进行绑定
func (s *NodeService) AddBinding(ctx *sc.ServiceContext, nodeID, licenseID, productID uint) error {
	binding, err := entity.NewNodeLicenseBinding(nodeID, licenseID, productID)
	if err != nil {
		return err
	}
	binding.IsBound = 1
	db := ctx.MustDefaultDB()
	return s.nlr.AddBinding(ctx, db, binding)
}

// AutoCreateBind 节点自动绑定
func (s *NodeService) AutoCreateBind(ctx *sc.ServiceContext, nodeID, productID uint, license *entity.License) error {
	// Use WithTransaction helper
	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
		count, err := s.nlr.CountActiveBindingsByLicenseForProduct(txCtx, txCtx.MustDefaultDB(), license.ID, productID)
		if err != nil {
			return fmt.Errorf("check binding failed")
		}
		if ok := license.ValidateMaxNodesForProduct(productID, int(count)); !ok {
			return fmt.Errorf("maximum nodes exceeded")
		}

		binding, _ := entity.NewNodeLicenseBinding(nodeID, license.ID, productID)
		binding.IsBound = 1
		if err := s.nlr.AddBinding(txCtx, txCtx.MustDefaultDB(), binding); err != nil {
			return fmt.Errorf("add binding failed")
		}
		return nil
	})
}

// AutoCreateBindWithContext does binding using DB/Tx from sCtx and does NOT start transaction itself
func (s *NodeService) AutoCreateBindWithContext(sCtx *sc.ServiceContext, nodeID, productID uint, license *entity.License) error {
	db := sCtx.MustDefaultDB()

	count, err := s.nlr.CountActiveBindingsByLicenseForProduct(sCtx, db, license.ID, productID)
	if err != nil {
		return fmt.Errorf("check binding failed")
	}
	if ok := license.ValidateMaxNodesForProduct(productID, int(count)); !ok {
		return fmt.Errorf("maximum nodes exceeded")
	}

	binding, _ := entity.NewNodeLicenseBinding(nodeID, license.ID, productID)
	binding.IsBound = 1
	if err := s.nlr.AddBinding(sCtx, db, binding); err != nil {
		return fmt.Errorf("add binding failed")
	}
	return nil
}

func (s *NodeService) UpdateBindingStatus(ctx *sc.ServiceContext, id uint, status int) error {
	db := ctx.MustDefaultDB()
	return s.nlr.UpdateBindingStatus(ctx, db, id, status)
}

func (s *NodeService) ForceUnbind(ctx *sc.ServiceContext, bindingID uint) error {
	db := ctx.MustDefaultDB()

	// 执行强制解绑操作
	err := s.nr.ForceUnbind(ctx, db, bindingID)
	if err != nil {
		return err
	}

	return nil
}

func (s *NodeService) DeleteNode(ctx *sc.ServiceContext, id uint) error {
	return ctx.RunInTransaction(base.DefaultDBName, func(txCtx *sc.ServiceContext) error {
		err := s.nr.DeleteNode(txCtx, txCtx.MustDefaultDB(), id)
		if err != nil {
			return err
		}
		return s.nlr.DeleteBindingByNodeID(txCtx, txCtx.MustDefaultDB(), id)
	})
}

func (s *NodeService) GetBindingByNodeAndLicense(ctx *sc.ServiceContext, nodeID, licenseID uint) (*entity.NodeLicenseBinding, error) {
	db := ctx.MustDefaultDB()
	return s.nlr.GetBindingByNodeAndLicense(ctx, db, nodeID, licenseID)
}
