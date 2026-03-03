package api

import (
	"fmt"
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"nexus-core/monitor"
	"nexus-core/persistence/repository"
	"nexus-core/sc"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessController 处理客户端心跳请求
// 负责验证许可证、管理节点绑定和控制并发访问
type AccessController struct {
	Api
	ls  *service.LicenseService       // 许可证服务，处理许可证验证和激活
	ns  *service.NodeService          // 节点服务，管理节点创建和绑定
	ps  *service.ProductService       // 产品服务，处理产品版本验证
	lr  *repository.LicenseRepository // 许可证仓库，直接访问许可证数据
	nr  *repository.NodeRepository    // 节点仓库，直接访问节点数据
	nlr *repository.NodeLicenseBindingRepository
}

// NewAccessController 创建新的访问控制器实例
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

// RegisterRoutes 注册访问相关的路由
func (c *AccessController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/access")
	{
		g.POST("/auto-bind", c.AutoBind)
		g.POST("/heartbeat", c.Heartbeat)
	}
}

// AutoBind 自动绑定接口处理
// 客户端启动时，会自动绑定节点和许可证
// @Summary Client auto bind
// @Tags access
// @Accept json
// @Produce json
// @Param body body dto.AutoBindCommand true "Auto Bind"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /access/auto-bind [post]
func (c *AccessController) AutoBind(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmd dto.AutoBindCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	//验证产品和版本是否支持
	ok, err := c.ps.CheckProductVersionSupported(gCtx, cmd.ProductID, nil, &cmd.VersionCode)
	if err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	if !ok {
		c.InternalError(sCtx, "product version not supported")
		return
	}
	// 找到 license
	license, err := c.ls.GetLicenseByKey(gCtx, cmd.LicenseKey)
	if err != nil {
		c.BadRequest(sCtx, "invalid license")
		return
	}
	// 验证许可证是否对产品有效
	scope := license.GetScope(cmd.ProductID)
	if scope == nil {
		c.BadRequest(sCtx, "product not supported")
		return
	}
	// 检查许可证是否过期
	currentStatus := license.CheckStatus(time.Now())
	toActivate := false
	// 验证许可证状态
	switch currentStatus {
	case entity.StatusInactive:
		toActivate = true
	case entity.StatusActive:
	case entity.StatusExpired:
		c.BadRequest(sCtx, "license expired")
		return
	case entity.StatusRevoked:
		c.BadRequest(sCtx, "invalid license")
		return
	}

	node, err := c.ns.AutoCreateNode(gCtx, cmd.DeviceCode, nil)
	if err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}

	if !node.IsValid() {
		c.BadRequest(sCtx, "invalid node")
		return
	}

	// 检查绑定
	binding, err := c.nlr.GetBindingByNodeAndLicense(gCtx, node.ID, license.ID)
	if err != nil {
		c.InternalError(sCtx, "check binding failed")
		return
	}
	if binding == nil {
		err := c.ns.AutoCreateBind(gCtx, node.ID, cmd.ProductID, license)
		if err != nil {
			c.InternalError(sCtx, "auto bind failed")
			return
		}
	} else {
		//存在绑定，更新绑定状态为已绑定
		if binding.IsBound == 0 {
			binding.IsBound = 1
			if err := c.nlr.UpdateBindingStatus(sCtx, binding.ID, 1); err != nil {
				c.InternalError(sCtx, "update binding status failed")
				return
			}
		}
	}
	if toActivate {
		// 激活 license
		if err := c.ls.ActivateLicenseIfNeeded(gCtx, license); err != nil {
			c.InternalError(sCtx, err.Error())
			return
		}
	}

	c.Success(sCtx, map[string]interface{}{})
}

// Heartbeat 心跳接口处理
// 客户端定期发送心跳以验证许可证有效性并更新节点状态
// 若期间在线的节点解绑，或过期等操作会导致强制下线
// @Summary Client heartbeat
// @Tags access
// @Accept json
// @Produce json
// @Param body body dto.HeartbeatCommand true "Heartbeat"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /access/heartbeat [post]
func (c *AccessController) Heartbeat(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmd dto.HeartbeatCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	//验证产品和版本是否支持
	ok, err := c.ps.CheckProductVersionSupported(gCtx, cmd.ProductID, nil, &cmd.VersionCode)
	if err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	if !ok {
		c.InternalError(sCtx, "product version not supported")
		return
	}
	// 找到 license
	license, err := c.ls.GetLicenseByKey(gCtx, cmd.LicenseKey)
	if err != nil {
		c.BadRequest(sCtx, "invalid license")
		return
	}
	// 验证许可证是否对产品有效
	scope := license.GetScope(cmd.ProductID)
	if scope == nil {
		c.BadRequest(sCtx, "product not supported")
		return
	}
	// 检查许可证是否过期
	currentStatus := license.CheckStatus(time.Now())
	// 验证许可证状态
	switch currentStatus {
	case entity.StatusInactive:
		c.BadRequest(sCtx, "license not active")
		return
	case entity.StatusActive:
	case entity.StatusExpired:
		c.BadRequest(sCtx, "license expired")
		return
	case entity.StatusRevoked:
		c.BadRequest(sCtx, "invalid license")
		return
	}

	node, err := c.ns.GetByDeviceCode(gCtx, cmd.DeviceCode)
	if err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	if node == nil {
		c.InternalError(sCtx, "node not found")
		return
	} else {
		if !node.IsValid() {
			c.BadRequest(sCtx, "invalid node")
			return
		}
	}

	// 检查绑定
	binding, err := c.nlr.GetBindingByNodeAndLicense(gCtx, node.ID, license.ID)
	if err != nil {
		c.InternalError(sCtx, "check binding failed")
		return
	}
	if binding == nil {
		c.InternalError(sCtx, "binding not found")
		return
	} else {
		if binding.IsBound == 0 {
			c.InternalError(sCtx, "binding not bound")
			return
		}
	}

	// 检查并发限制
	totalConcurrent := monitor.GlobalStat.GetConcurrentByLicenseForProduct(license.LicenseKey, cmd.ProductID)
	if !license.ValidateMaxConcurrentForProduct(cmd.ProductID, totalConcurrent) {
		c.BadRequest(sCtx, "maximum concurrent exceeded")
		return
	}

	monitor.GlobalMonitor.HeartBeat(fmt.Sprintf("%d|%s|%s",
		cmd.ProductID, node.DeviceCode, license.LicenseKey), time.Second*60)

	c.Success(sCtx, map[string]interface{}{})
}
