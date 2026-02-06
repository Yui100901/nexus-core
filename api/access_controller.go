package api

import (
	"context"
	"fmt"
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"nexus-core/monitor"
	"nexus-core/persistence/repository"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessController 处理客户端心跳请求
// 负责验证许可证、管理节点绑定和控制并发访问
type AccessController struct {
	ls  *service.LicenseService       // 许可证服务，处理许可证验证和激活
	ns  *service.NodeService          // 节点服务，管理节点创建和绑定
	ps  *service.ProductService       // 产品服务，处理产品版本验证
	lr  *repository.LicenseRepository // 许可证仓库，直接访问许可证数据
	nr  *repository.NodeRepository    // 节点仓库，直接访问节点数据
	nlr *repository.NodeLicenseBindingRepository
}

// NewAccessController 创建新的设备控制器实例
func NewAccessController() *AccessController {
	return &AccessController{
		ls:  service.NewLicenseService(),
		ns:  service.NewNodeService(),
		ps:  service.NewProductService(),
		lr:  repository.NewLicenseRepository(),
		nr:  repository.NewNodeRepository(),
		nlr: repository.NewNodeLicenseBindingRepository(),
	}
}

// RegisterRoutes 注册设备相关的路由
func (c *AccessController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/access")
	{
		g.POST("/heartbeat", c.Heartbeat)
	}
}

// Heartbeat 心跳接口处理
// 客户端定期发送心跳以验证许可证有效性并更新节点状态
// @Summary Client heartbeat
// @Tags access
// @Accept json
// @Produce json
// @Param body body dto.HeartbeatCommand true "Heartbeat"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /access/heartbeat [post]
func (c *AccessController) Heartbeat(ctx *gin.Context) {
	var cmd dto.HeartbeatCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	// 先找产品
	product, err := c.ps.GetByID(context.Background(), cmd.ProductID)
	if err != nil || product == nil {
		BadRequest(ctx, "product not found")
		return
	}
	// 验证产品版本
	if !product.CheckVersionSupportedByCode(cmd.VersionCode) {
		BadRequest(ctx, "product version not supported")
		return
	}
	// 找到 license
	license, err := c.ls.GetLicenseByKey(context.Background(), cmd.LicenseKey)
	if err != nil {
		BadRequest(ctx, "invalid license")
		return
	}
	// 验证许可证是否对产品有效
	scope := license.GetScope(cmd.ProductID)
	if scope == nil {
		BadRequest(ctx, "product not supported")
		return
	}
	// 检查许可证是否过期
	currentStatus := license.CheckStatus(time.Now())
	// 验证许可证状态
	switch currentStatus {
	case entity.StatusInactive:
		// 激活 license
		if err := c.ls.ActivateLicenseIfNeeded(context.Background(), license); err != nil {
			InternalError(ctx, err.Error())
			return
		}
	case entity.StatusActive:
	case entity.StatusExpired:
		BadRequest(ctx, "license expired")
		return
	case entity.StatusRevoked:
		BadRequest(ctx, "invalid license")
		return
	}
	// 查找或创建 node
	node, err := c.ns.GetByDeviceCode(context.Background(), cmd.DeviceCode)
	if err != nil {
		// create new node
		n, err := entity.NewNode(cmd.DeviceCode, nil)
		if err != nil {
			InternalError(ctx, "create node failed")
			return
		}
		if err := c.ns.CreateNode(context.Background(), n); err != nil {
			InternalError(ctx, "create node failed")
			return
		}
		node = n
	}

	// 检查绑定
	binding, err := c.nlr.GetBindingByNodeAndLicense(context.Background(), node.ID, license.ID)
	if err != nil {
		InternalError(ctx, "check binding failed")
		return
	}
	if binding == nil {
		//不存在绑定
		//检查许可证的 MaxNodes 限制
		bindingsCount, err := c.nlr.CountActiveBindingsByLicenseForProduct(context.Background(), license.ID, product.ID)
		if err != nil {
			InternalError(ctx, "check binding failed")
			return
		}
		if ok := license.ValidateMaxNodesForProduct(product.ID, int(bindingsCount)); !ok {
			BadRequest(ctx, "maximum nodes exceeded")
			return
		}
		//添加绑定
		binding, _ = entity.NewNodeLicenseBinding(node.ID, license.ID, product.ID)
		binding.IsBound = 1
		if err := c.nlr.AddBinding(context.Background(), binding); err != nil {
			InternalError(ctx, "add binding failed")
			return
		}
	} else {
		//存在绑定，更新绑定状态为已绑定
		if binding.IsBound == 0 {
			binding.IsBound = 1
			if err := c.nlr.UpdateBindingStatus(context.Background(), binding.ID, 1); err != nil {
				InternalError(ctx, "update binding status failed")
				return
			}
		}
	}

	// 检查并发限制
	totalConcurrent := monitor.GlobalStat.GetConcurrentByLicenseForProduct(license.LicenseKey, product.ID)
	if !license.ValidateMaxConcurrentForProduct(product.ID, totalConcurrent) {
		BadRequest(ctx, "maximum concurrent exceeded")
		return
	}

	monitor.GlobalMonitor.HeartBeat(fmt.Sprintf("%d|%s|%s",
		product.ID, node.DeviceCode, license.LicenseKey), time.Second*60)

	Success(ctx, map[string]interface{}{})
}
