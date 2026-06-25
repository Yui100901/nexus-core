package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

// NodeController 处理节点相关的API请求
// 管理节点的创建、查询、绑定等操作
type NodeController struct {
	ns *service.NodeService // 节点服务，处理节点相关的业务逻辑
}

// NewNodeController 创建新的节点控制器实例
func NewNodeController() *NodeController {
	return &NodeController{ns: service.NewNodeService()}
}

// RegisterRoutes 注册节点相关的路由
// 包括节点创建、查询、绑定等操作的路由
func (c *NodeController) RegisterRoutes(r *gin.Engine) {
	nodes := r.Group("/nodes")
	{
		nodes.POST("", c.CreateNode)
		nodes.GET("/:id", c.GetByID)
		nodes.DELETE("/:id", c.DeleteNode)
		nodes.POST("/:id/ban", c.BanNode)
		nodes.POST("/:id/unban", c.UnbanNode)
	}
	r.GET("/node-devices/:device_code", c.GetByDeviceCode)
	r.POST("/node-bindings", c.AddBinding)
	r.DELETE("/node-bindings", c.Unbind)
	r.DELETE("/node-cleanups/unbound", c.CleanUnboundNode)

	g := r.Group("/node")
	{
		g.POST("/createNode", c.CreateNode)      // 创建节点
		g.GET("/getByID", c.GetByID)             // 根据ID获取节点
		g.GET("/getByDevice", c.GetByDeviceCode) // 根据设备码获取节点
		g.POST("/addBinding", c.AddBinding)      // 添加节点绑定
		g.POST("/unbind", c.Unbind)              // 解绑
		g.POST("/delete", c.DeleteNode)          // 删除节点
		g.POST("/ban", c.BanNode)                // 封禁节点
		g.POST("/unban", c.UnbanNode)            // 解封节点
		g.POST("/cleanUnboundNode", c.CleanUnboundNode)
	}
}

// CreateNode 创建节点
// @Summary Create a node
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.CreateNodeCommand true "Create Node"
// @Success 200 {object} entity.Node
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /nodes [post]
func (c *NodeController) CreateNode(ctx *gin.Context) {
	var cmd dto.CreateNodeCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	n, err := c.ns.CreateNode(ctx.Request.Context(), service.CreateNodeCommand{
		DeviceCode: cmd.DeviceCode,
		Metadata:   cmd.Metadata,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, n)
}

// GetByID 根据 ID 获取节点信息
// @Summary Get node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path uint true "Node ID"
// @Success 200 {object} entity.Node
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /nodes/{id} [get]
func (c *NodeController) GetByID(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.ns.GetNodeDataByID(ctx.Request.Context(), id)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// GetByDeviceCode 根据设备码查询节点
// @Summary Get node by device code
// @Tags nodes
// @Accept json
// @Produce json
// @Param device_code query string true "Device Code"
// @Success 200 {object} entity.Node
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /node-devices/{device_code} [get]
func (c *NodeController) GetByDeviceCode(ctx *gin.Context) {
	// 获取 query 参数
	code := ctx.Param("device_code")
	if code == "" {
		code = ctx.Query("device_code")
	}
	if code == "" {
		code = ctx.Query("deviceCode")
	}
	if code == "" {
		BadRequest(ctx, "device_code is required")
		return
	}
	n, err := c.ns.GetByDeviceCode(ctx.Request.Context(), code)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, n)
}

// AddBinding 手动添加节点绑定
// @Summary Add node binding
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.AddBindingCommand true "Add Binding"
// @Success 200 {object} entity.NodeLicenseBinding
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-bindings [post]
func (c *NodeController) AddBinding(ctx *gin.Context) {
	var cmd dto.AddBindingCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ns.AddBinding(ctx.Request.Context(), service.AddBindingCommand{
		NodeID:    cmd.NodeID,
		LicenseID: cmd.LicenseID,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, "")
}

// Unbind 解除绑定状态
// @Summary Update binding status
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.UnbindCommand true "Update binding status"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-bindings [delete]
func (c *NodeController) Unbind(ctx *gin.Context) {
	var cmd dto.UnbindCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ns.UnbindByID(ctx.Request.Context(), service.UnbindCommand{
		NodeID:    cmd.NodeID,
		LicenseID: cmd.LicenseID,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "binding status updated")
}

// DeleteNode 删除节点
// @Summary Delete a node
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body object true "{\"id\": <node id>}"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /nodes/{id} [delete]
func (c *NodeController) DeleteNode(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		var cmd struct {
			ID uint `json:"id" binding:"required"`
		}
		if bindErr := ctx.ShouldBindJSON(&cmd); bindErr != nil {
			BadRequest(ctx, err.Error())
			return
		}
		id = cmd.ID
	}
	if err := c.ns.DeleteNode(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "node deleted")
}

// BanNode 封禁节点
// @Summary Ban a node
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path uint true "Node ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /nodes/{id}/ban [post]
func (c *NodeController) BanNode(ctx *gin.Context) {
	id, ok := c.nodeIDFromParamOrBody(ctx)
	if !ok {
		return
	}
	if err := c.ns.BanNode(ctx.Request.Context(), service.UpdateNodeStatusCommand{NodeID: id}); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "node banned")
}

// UnbanNode 解封节点
// @Summary Unban a node
// @Tags nodes
// @Accept json
// @Produce json
// @Param id path uint true "Node ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /nodes/{id}/unban [post]
func (c *NodeController) UnbanNode(ctx *gin.Context) {
	id, ok := c.nodeIDFromParamOrBody(ctx)
	if !ok {
		return
	}
	if err := c.ns.UnbanNode(ctx.Request.Context(), service.UpdateNodeStatusCommand{NodeID: id}); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "node unbanned")
}

func (c *NodeController) nodeIDFromParamOrBody(ctx *gin.Context) (uint, bool) {
	id, err := UintParamOrQuery(ctx, "id")
	if err == nil {
		return id, true
	}
	var cmd dto.UpdateNodeStatusCommand
	if bindErr := ctx.ShouldBindJSON(&cmd); bindErr != nil {
		BadRequest(ctx, err.Error())
		return 0, false
	}
	return cmd.NodeID, true
}

// CleanUnboundNode 清理无任何绑定的节点
// @Summary Delete a node
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body object true "{\"id\": <node id>}"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-cleanups/unbound [delete]
func (c *NodeController) CleanUnboundNode(ctx *gin.Context) {
	if err := c.ns.CleanUnboundNode(ctx.Request.Context()); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "node deleted")
}
