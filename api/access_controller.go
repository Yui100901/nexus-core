package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/service"
	"nexus-core/sc"

	"github.com/gin-gonic/gin"
)

// AccessController 处理客户端心跳请求
// 负责验证许可证、管理节点绑定和控制并发访问
type AccessController struct {
	Api
	ls *service.LicenseService // 许可证服务，处理许可证验证和激活
	ns *service.NodeService    // 节点服务，管理节点创建和绑定
	ps *service.ProductService // 产品服务，处理产品版本验证
	as *service.AccessService  // 新增：业务服务层
}

// NewAccessController 创建新的访问控制器实例
func NewAccessController() *AccessController {
	ls := service.NewLicenseService()
	ns := service.NewNodeService()
	ps := service.NewProductService()
	as := service.NewAccessService(ls, ns, ps)
	return &AccessController{
		ls: ls,
		ns: ns,
		ps: ps,
		as: as,
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

// AutoBind 自动绑定接口处理（现在非常薄）
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
	// get service context
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.AutoBindCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	res, err := c.as.AutoBind(sCtx, cmd.DeviceCode, cmd.ProductID, cmd.VersionCode, cmd.LicenseKey)
	if err != nil {
		if se, ok := err.(*service.ServiceError); ok {
			// map service-defined HTTP status
			switch se.HTTPStatus {
			case 400:
				c.BadRequest(sCtx, se.Error())
			case 500:
				c.InternalError(sCtx, se.Error())
			default:
				c.InternalError(sCtx, se.Error())
			}
			return
		}
		c.InternalError(sCtx, err.Error())
		return
	}

	c.Success(sCtx, res)
}

// Heartbeat 现在也很薄
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
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.HeartbeatCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	res, err := c.as.Heartbeat(sCtx, cmd.DeviceCode, cmd.ProductID, cmd.VersionCode, cmd.LicenseKey)
	if err != nil {
		if se, ok := err.(*service.ServiceError); ok {
			switch se.HTTPStatus {
			case 400:
				c.BadRequest(sCtx, se.Error())
			case 500:
				c.InternalError(sCtx, se.Error())
			default:
				c.InternalError(sCtx, se.Error())
			}
			return
		}
		c.InternalError(sCtx, err.Error())
		return
	}

	c.Success(sCtx, res)
}
