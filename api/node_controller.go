package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"nexus-core/sc"

	"github.com/gin-gonic/gin"
)

// NodeController 处理节点相关的API请求
// 管理节点的创建、查询、绑定等操作
type NodeController struct {
	Api
	ns *service.NodeService // 节点服务，处理节点相关的业务逻辑
}

// NewNodeController 创建新的节点控制器实例
func NewNodeController() *NodeController {
	return &NodeController{ns: service.NewNodeService()}
}

// RegisterRoutes 注册节点相关的路由
// 包括节点创建、查询、绑定等操作的路由
func (c *NodeController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/node")
	{
		g.POST("/create", c.CreateNode)                       // 创建节点
		g.POST("/batchCreate", c.BatchCreate)                 // 批量创建节点
		g.GET("/getByID", c.GetByID)                          // 根据ID获取节点
		g.GET("/getByDevice", c.GetByDeviceCode)              // 根据设备码获取节点
		g.POST("/addBinding", c.AddBinding)                   // 添加节点绑定
		g.POST("/updateBindingStatus", c.UpdateBindingStatus) // 更新绑定状态
		g.POST("/unbind", c.ForceUnbind)                      // 强制解绑节点
		g.POST("/delete", c.DeleteNode)                       // 删除节点
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
// @Router /node/create [post]
func (c *NodeController) CreateNode(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.CreateNodeCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	n := dto.ToEntityNode(cmd)
	if err := c.ns.CreateNode(sCtx, n); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, n)
}

// BatchCreate 批量创建节点
// @Summary Batch create nodes
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body []dto.CreateNodeCommand true "Create Nodes"
// @Success 200 {array} entity.Node
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node/batchCreate [post]
func (c *NodeController) BatchCreate(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmds []dto.CreateNodeCommand
	if err := gCtx.ShouldBindJSON(&cmds); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	var nodes []*entity.Node
	for _, cmd := range cmds {
		nodes = append(nodes, dto.ToEntityNode(cmd))
	}
	if err := c.ns.BatchCreateNode(sCtx, nodes); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, nodes)
}

// GetByID 根据 ID 获取节点信息
// @Summary Get node by ID
// @Tags nodes
// @Accept json
// @Produce json
// @Param id query uint true "Node ID"
// @Success 200 {object} entity.Node
// @Failure 400 {object} api.CommonResponse
// @Failure 504 {object} api.CommonResponse
// @Router /node/getByID [get]
func (c *NodeController) GetByID(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var q dto.GetNodeByIDQuery
	if err := gCtx.ShouldBindQuery(&q); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	n, err := c.ns.GetByID(sCtx, q.ID)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, n)
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
// @Router /node/getByDevice [get]
func (c *NodeController) GetByDeviceCode(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var q dto.GetNodeByDeviceCodeQuery
	if err := gCtx.ShouldBindQuery(&q); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	n, err := c.ns.GetByDeviceCode(sCtx, q.DeviceCode)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, n)
}

// AddBinding 添加节点绑定
// @Summary Add node binding
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.AddBindingCommand true "Add Binding"
// @Success 200 {object} entity.NodeLicenseBinding
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node/addBinding [post]
func (c *NodeController) AddBinding(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.AddBindingCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ns.AddBinding(sCtx, cmd.NodeID, cmd.LicenseID, cmd.ProductID); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, "")
}

// UpdateBindingStatus 更新绑定状态
// @Summary Update binding status
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.UpdateBindingStatusCommand true "Update binding status"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node/updateBindingStatus [post]
func (c *NodeController) UpdateBindingStatus(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.UpdateBindingStatusCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ns.UpdateBindingStatus(sCtx, cmd.ID, cmd.Status); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "binding status updated")
}

// ForceUnbind 强制解绑节点
// @Summary Force unbind a node binding using node and license IDs
// @Tags nodes
// @Accept json
// @Produce json
// @Param body body dto.ForceUnbindCommand true "Force unbind command"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node/unbind [post]
func (c *NodeController) ForceUnbind(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var cmd dto.ForceUnbindCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	// todo: implement force unbind logic
	//if err := c.ns.ForceUnbindByNodeAndLicense(gCtx, cmd.NodeID, cmd.LicenseID); err != nil {
	//	InternalError(gCtx, err.Error())
	//	return
	//}
	c.SuccessMsg(sCtx, "node binding forced to unbind")
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
// @Router /node/delete [post]
func (c *NodeController) DeleteNode(gCtx *gin.Context) {
	sCtx, ok := getServiceContextFromGin(gCtx)
	if !ok {
		tmp := &sc.ServiceContext{GinContext: gCtx}
		c.InternalError(tmp, "service context missing")
		return
	}
	var q struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := gCtx.ShouldBindJSON(&q); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ns.DeleteNode(sCtx, q.ID); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "node deleted")
}
