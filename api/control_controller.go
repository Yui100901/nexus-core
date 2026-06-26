package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

type ControlController struct {
	cs *service.ControlService
}

func NewControlController() *ControlController {
	return &ControlController{cs: service.NewControlService()}
}

func (c *ControlController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/control-services")
	{
		g.POST("", c.CreateControlService)
		g.GET("", c.ListControlServices)
		g.GET("/:id", c.GetControlServiceByID)
		g.PATCH("/:id", c.UpdateControlService)
		g.DELETE("/:id", c.DeleteControlService)
		g.POST("/:id/status", c.UpdateControlServiceStatus)
	}

	capabilities := r.Group("/node-capabilities")
	{
		capabilities.POST("", c.ReportNodeCapability)
		capabilities.GET("", c.ListNodeCapabilities)
	}

	commands := r.Group("/control-commands")
	{
		commands.POST("", c.CreateControlCommand)
		commands.GET("", c.ListControlCommands)
		commands.GET("/:id", c.GetControlCommandByID)
		commands.POST("/:id/complete", c.CompleteControlCommand)
	}

	nodeControl := r.Group("/node-control")
	{
		nodeControl.GET("/ws", c.ConnectNodeControlWebSocket)
	}
}

// ListControlCommands 查询控制指令列表
// @Summary List control commands
// @Tags control-commands
// @Accept json
// @Produce json
// @Param node_id query uint false "Node ID"
// @Param service_identifier query string false "Service identifier fuzzy filter"
// @Param status query int false "Command status"
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-commands [get]
func (c *ControlController) ListControlCommands(ctx *gin.Context) {
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	nodeID, err := UintQuery(ctx, "node_id")
	if err != nil {
		BadRequest(ctx, "invalid node_id")
		return
	}
	status, err := IntQueryPtr(ctx, "status")
	if err != nil {
		BadRequest(ctx, "invalid status")
		return
	}

	data, err := c.cs.ListControlCommandsPage(ctx.Request.Context(), service.ListControlCommandsCommand{
		NodeID:            nodeID,
		ServiceIdentifier: StringQuery(ctx, "service_identifier"),
		Status:            status,
		Limit:             page.Limit,
		Offset:            page.Offset,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// CreateControlService 创建控制服务定义
// @Summary Create a control service
// @Tags control-services
// @Accept json
// @Produce json
// @Param body body dto.CreateControlServiceCommand true "Create Control Service"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 409 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services [post]
func (c *ControlController) CreateControlService(ctx *gin.Context) {
	var cmd dto.CreateControlServiceCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.CreateControlService(ctx.Request.Context(), service.CreateControlServiceCommand{
		ProductID:    cmd.ProductID,
		Identifier:   cmd.Identifier,
		Name:         cmd.Name,
		Description:  cmd.Description,
		ServiceType:  cmd.ServiceType,
		InputSchema:  cmd.InputSchema,
		OutputSchema: cmd.OutputSchema,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// GetControlServiceByID 查询控制服务定义
// @Summary Get a control service
// @Tags control-services
// @Accept json
// @Produce json
// @Param id path uint true "Control Service ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services/{id} [get]
func (c *ControlController) GetControlServiceByID(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	data, err := c.cs.GetControlServiceByID(ctx.Request.Context(), id)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ListControlServices 查询控制服务列表
// @Summary List control services
// @Tags control-services
// @Accept json
// @Produce json
// @Param product_id query uint false "Product ID"
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit, compatible with page_size"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services [get]
func (c *ControlController) ListControlServices(ctx *gin.Context) {
	productID, err := UintQuery(ctx, "product_id")
	if err != nil {
		BadRequest(ctx, "invalid product_id")
		return
	}
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.ListControlServicesPage(ctx.Request.Context(), service.ListControlServicesCommand{
		ProductID: productID,
		Limit:     page.Limit,
		Offset:    page.Offset,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ReportNodeCapability 上报节点控制能力
// @Summary Report node capability
// @Tags node-capabilities
// @Accept json
// @Produce json
// @Param body body dto.ReportNodeCapabilityCommand true "Report Node Capability"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-capabilities [post]
func (c *ControlController) ReportNodeCapability(ctx *gin.Context) {
	var cmd dto.ReportNodeCapabilityCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.ReportNodeCapability(ctx.Request.Context(), service.ReportNodeCapabilityCommand{
		NodeID:            cmd.NodeID,
		ServiceIdentifier: cmd.ServiceIdentifier,
		Schema:            cmd.Schema,
		Protocol:          cmd.Protocol,
		Endpoint:          cmd.Endpoint,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ListNodeCapabilities 查询节点控制能力
// @Summary List node capabilities
// @Tags node-capabilities
// @Accept json
// @Produce json
// @Param node_id query uint false "Node ID"
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit, compatible with page_size"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-capabilities [get]
func (c *ControlController) ListNodeCapabilities(ctx *gin.Context) {
	nodeID, err := UintQuery(ctx, "node_id")
	if err != nil {
		BadRequest(ctx, "invalid node_id")
		return
	}
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	var nodeIDValue uint
	if nodeID != nil {
		nodeIDValue = *nodeID
	}
	data, err := c.cs.ListNodeCapabilitiesPage(ctx.Request.Context(), service.ListNodeCapabilitiesCommand{
		NodeID: nodeIDValue,
		Limit:  page.Limit,
		Offset: page.Offset,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// CreateControlCommand 创建并下发控制指令
// @Summary Create a control command
// @Tags control-commands
// @Accept json
// @Produce json
// @Param body body dto.CreateControlCommand true "Create Control Command"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 403 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-commands [post]
func (c *ControlController) CreateControlCommand(ctx *gin.Context) {
	var cmd dto.CreateControlCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.CreateControlCommand(ctx.Request.Context(), service.CreateControlCommand{
		NodeID:            cmd.NodeID,
		ServiceIdentifier: cmd.ServiceIdentifier,
		Payload:           cmd.Payload,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// GetControlCommandByID 查询控制指令结果
// @Summary Get a control command
// @Tags control-commands
// @Accept json
// @Produce json
// @Param id path uint true "Control Command ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-commands/{id} [get]
func (c *ControlController) GetControlCommandByID(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.GetControlCommandByID(ctx.Request.Context(), id)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ConnectNodeControlWebSocket accepts a node-owned websocket connection for server-to-node commands.
// @Summary Connect node control websocket
// @Description Nodes connect to this endpoint, then the server can dispatch websocket control commands by node_id.
// @Tags node-control
// @Accept json
// @Produce json
// @Param node_id query uint true "Node ID"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /node-control/ws [get]
func (c *ControlController) ConnectNodeControlWebSocket(ctx *gin.Context) {
	nodeID, err := UintParamOrQuery(ctx, "node_id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := service.DefaultControlWebSocketHub.ServeHTTP(ctx.Writer, ctx.Request, nodeID); err != nil {
		HandleError(ctx, err)
		return
	}
}

// UpdateControlService updates a control service definition.
// @Summary Update a control service
// @Tags control-services
// @Accept json
// @Produce json
// @Param id path uint true "Control Service ID"
// @Param body body dto.UpdateControlServiceCommand true "Update Control Service"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services/{id} [patch]
func (c *ControlController) UpdateControlService(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	var cmd dto.UpdateControlServiceCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.UpdateControlService(ctx.Request.Context(), service.UpdateControlServiceCommand{
		ID:           id,
		ProductID:    cmd.ProductID,
		Name:         cmd.Name,
		Description:  cmd.Description,
		ServiceType:  cmd.ServiceType,
		InputSchema:  cmd.InputSchema,
		OutputSchema: cmd.OutputSchema,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// UpdateControlServiceStatus enables or disables a control service.
// @Summary Update control service status
// @Tags control-services
// @Accept json
// @Produce json
// @Param id path uint true "Control Service ID"
// @Param body body dto.UpdateControlServiceStatusCommand true "Update Control Service Status"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services/{id}/status [post]
func (c *ControlController) UpdateControlServiceStatus(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	var cmd dto.UpdateControlServiceStatusCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.cs.UpdateControlServiceStatus(ctx.Request.Context(), service.UpdateControlServiceStatusCommand{
		ID:     id,
		Status: cmd.Status,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// DeleteControlService deletes an unused control service definition.
// @Summary Delete a control service
// @Tags control-services
// @Accept json
// @Produce json
// @Param id path uint true "Control Service ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 409 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-services/{id} [delete]
func (c *ControlController) DeleteControlService(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.cs.DeleteControlService(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "control service deleted")
}

// CompleteControlCommand records an async node control command response.
// @Summary Complete a control command
// @Tags control-commands
// @Accept json
// @Produce json
// @Param id path uint true "Control Command ID"
// @Param body body dto.CompleteControlCommand true "Complete Control Command"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /control-commands/{id}/complete [post]
func (c *ControlController) CompleteControlCommand(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	var cmd dto.CompleteControlCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	cmd.CommandID = id

	data, err := c.cs.CompleteControlCommand(ctx.Request.Context(), service.CompleteControlCommandCommand{
		CommandID:    cmd.CommandID,
		Status:       cmd.Status,
		Result:       cmd.Result,
		ErrorMessage: cmd.ErrorMessage,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}
