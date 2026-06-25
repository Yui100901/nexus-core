package api

import (
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

type MonitorController struct {
	ms *service.MonitorService
	as *service.AuditService
}

func NewMonitorController() *MonitorController {
	return &MonitorController{
		ms: service.NewMonitorService(),
		as: service.NewAuditService(),
	}
}

func (c *MonitorController) RegisterRoutes(r *gin.Engine) {
	monitorGroup := r.Group("/monitor")
	{
		monitorGroup.GET("/online", c.GetOnlineSummary)
		monitorGroup.GET("/nodes/heartbeats", c.ListNodeHeartbeats)
	}
	r.GET("/audit-logs", c.ListAuditLogs)
}

// GetOnlineSummary 查询在线节点统计
// @Summary Get online node summary
// @Tags monitor
// @Accept json
// @Produce json
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /monitor/online [get]
func (c *MonitorController) GetOnlineSummary(ctx *gin.Context) {
	data, err := c.ms.GetOnlineSummary(ctx.Request.Context())
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ListNodeHeartbeats 查询节点最近心跳
// @Summary List node heartbeat stats
// @Tags monitor
// @Accept json
// @Produce json
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit"
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /monitor/nodes/heartbeats [get]
func (c *MonitorController) ListNodeHeartbeats(ctx *gin.Context) {
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	data, err := c.ms.ListNodeHeartbeatsPage(ctx.Request.Context(), page.Limit, page.Offset)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// ListAuditLogs 查询审计日志
// @Summary List audit logs
// @Tags audit
// @Accept json
// @Produce json
// @Param resource_type query string false "Resource Type"
// @Param resource_id query uint false "Resource ID"
// @Param action query string false "Action"
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /audit-logs [get]
func (c *MonitorController) ListAuditLogs(ctx *gin.Context) {
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	resourceID, err := UintQuery(ctx, "resource_id")
	if err != nil {
		BadRequest(ctx, "invalid resource_id")
		return
	}

	data, err := c.as.ListAuditLogs(ctx.Request.Context(), service.ListAuditLogsCommand{
		ResourceType: StringQuery(ctx, "resource_type"),
		ResourceID:   resourceID,
		Action:       StringQuery(ctx, "action"),
		Limit:        page.Limit,
		Offset:       page.Offset,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}
