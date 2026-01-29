package api

import (
	"context"
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"nexus-core/persistence/repository"
	"nexus-core/runtimecache"

	"github.com/gin-gonic/gin"
)

// AccessController 处理客户端心跳请求
// 负责验证许可证、管理节点绑定和控制并发访问
type AccessController struct {
	ls *service.LicenseService       // 许可证服务，处理许可证验证和激活
	ns *service.NodeService          // 节点服务，管理节点创建和绑定
	ps *service.ProductService       // 产品服务，处理产品版本验证
	lr *repository.LicenseRepository // 许可证仓库，直接访问许可证数据
	nr *repository.NodeRepository    // 节点仓库，直接访问节点数据
}

// NewAccessController 创建新的设备控制器实例
func NewAccessController() *AccessController {
	return &AccessController{
		ls: service.NewLicenseService(),
		ns: service.NewNodeService(),
		ps: service.NewProductService(),
		lr: repository.NewLicenseRepository(),
		nr: repository.NewNodeRepository(),
	}
}

// RegisterRoutes 注册设备相关的路由
func (c *AccessController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/device")
	{
		g.POST("/heartbeat", c.Heartbeat)
	}
}

// Heartbeat 心跳接口处理
// 客户端定期发送心跳以验证许可证有效性并更新节点状态
// @Summary Client heartbeat
// @Tags device
// @Accept json
// @Produce json
// @Param body body dto.HeartbeatCommand true "Heartbeat"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /device/heartbeat [post]
func (c *AccessController) Heartbeat(ctx *gin.Context) {
	var cmd dto.HeartbeatCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	// 1. 找到 license
	lic, err := c.ls.GetLicenseByKey(context.Background(), cmd.LicenseKey)
	if err != nil {
		BadRequest(ctx, "license not found")
		return
	}
	// 1.5 检查版本是否被产品支持
	ok, err := c.ps.HasSupportedVersion(context.Background(), cmd.ProductID, cmd.VersionCode)
	if err != nil {
		InternalError(ctx, err.Error())
		return
	}
	if !ok {
		BadRequest(ctx, "product version not supported")
		return
	}
	// 2. 激活 license 如果需要
	if err := c.ls.ActivateLicenseIfNeeded(context.Background(), lic); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	// 3. 查找或创建 node
	node, err := c.ns.GetByDeviceCode(context.Background(), cmd.DeviceCode)
	if err != nil {
		// create new node
		n := &entity.Node{DeviceCode: cmd.DeviceCode, MetaInfo: nil}
		if err := c.ns.CreateNode(context.Background(), n); err != nil {
			InternalError(ctx, err.Error())
			return
		}
		node = n
	}
	// 4. 更新运行时并发统计（记录当前节点的 concurrent）
	totalNodes, totalConcurrent := runtimecache.SetNodeConcurrent(lic.ID, cmd.ProductID, node.ID, cmd.Concurrent)

	// 5. 检查是否已有绑定
	binding, err := c.nr.GetBindingByNodeAndLicenseProduct(context.Background(), node.ID, lic.ID, cmd.ProductID)
	if err == nil && binding != nil {
		// already bound, check license validity for usage using runtime totals
		valid, err := c.ls.ValidateLicenseForUsage(ctx, lic, cmd.ProductID, totalNodes, totalConcurrent)
		if !valid || err != nil {
			BadRequest(ctx, "license validation failed: "+err.Error())
			return
		}
		Success(ctx, map[string]interface{}{"message": "heartbeat ok", "bound": true, "nodes": totalNodes, "concurrent": totalConcurrent})
		return
	}
	// 6. enforce MaxNodes limit before creating binding
	if lic != nil {
		maxNodes := 0
		for _, s := range lic.ScopeList {
			if s.ProductID == cmd.ProductID {
				maxNodes = s.MaxNodes
				break
			}
		}
		if maxNodes > 0 && totalNodes >= maxNodes {
			BadRequest(ctx, "max nodes limit reached")
			return
		}
	}
	// 7. 添加绑定
	nb := &entity.NodeBinding{LicenseID: lic.ID, NodeID: node.ID, IsBound: 0}
	if err := c.ns.AddBinding(context.Background(), node.ID, nb); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, map[string]interface{}{"message": "bound", "binding_id": nb.ID, "nodes": totalNodes, "concurrent": totalConcurrent})
}
