package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

// LicenseController 处理许可证相关的API端点
// 提供许可证的创建、查询、更新和删除等功能
// @Author yfy2001
// @Date 2026/1/20 09 11
type LicenseController struct {
	ls *service.LicenseService // 许可证服务，处理业务逻辑
}

// NewLicenseController 创建新的许可证控制器实例
func NewLicenseController() *LicenseController {
	return &LicenseController{
		ls: service.NewLicenseService(),
	}
}

// RegisterRoutes 注册许可证相关的路由
// 包括创建、查询、更新和删除等操作的路由
func (c *LicenseController) RegisterRoutes(r *gin.Engine) {
	licenseGroup := r.Group("/license")
	{
		licenseGroup.POST("/create", c.CreateLicense)        // 创建单个许可证
		licenseGroup.POST("/batchCreate", c.BatchCreate)     // 批量创建许可证
		licenseGroup.GET("/getByID", c.GetByID)              // 根据ID查询许可证
		licenseGroup.GET("/getByKey", c.GetByKey)            // 根据许可证密钥查询
		licenseGroup.POST("/updateStatus", c.UpdateStatus)   // 更新许可证状态
		licenseGroup.POST("/update", c.UpdateLicense)        // 更新许可证信息
		licenseGroup.POST("/deleteExpired", c.DeleteExpired) // 删除过期的许可证
	}
}

// CreateLicense 创建 License
// @Summary Create a license
// @Description Create a new license with scopes
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.CreateLicenseCommand true "Create License"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /license/create [post]
func (c *LicenseController) CreateLicense(ctx *gin.Context) {
	var cmd dto.CreateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	license, err := entity.NewLicense(cmd.ValidityHours, cmd.MaxNodes, cmd.ConcurrentLimit, cmd.Remark, dto.ToEntityScopes(cmd.ScopeList))

	if err != nil {
		BadRequest(ctx, err.Error())
	}

	if err := c.ls.CreateLicense(ctx, license); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, license)
}

// BatchCreate 批量创建 License
// @Summary Batch create licenses
// @Description Create multiple licenses in batch
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body []dto.CreateLicenseCommand true "Create Licenses"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /license/batchCreate [post]
func (c *LicenseController) BatchCreate(ctx *gin.Context) {
	var cmds []dto.CreateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmds); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	var licenses []*entity.License
	for _, cmd := range cmds {
		license, err := entity.NewLicense(cmd.ValidityHours, cmd.MaxNodes, cmd.ConcurrentLimit, cmd.Remark, dto.ToEntityScopes(cmd.ScopeList))
		if err != nil {
			BadRequest(ctx, err.Error())
			return
		}
		licenses = append(licenses, license)
	}

	if err := c.ls.BatchCreateLicense(ctx, licenses); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, licenses)
}

// GetByID 根据 ID 获取 License（Query 参数传递）
// @Summary Get license by ID
// @Tags licenses
// @Accept json
// @Produce json
// @Param id query uint true "License ID"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 404 {object} api.APIResponse
// @Router /license/getByID [get]
func (c *LicenseController) GetByID(ctx *gin.Context) {
	var query dto.GetLicenseByIDQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	license, err := c.ls.GetLicenseByID(ctx, query.ID)
	if err != nil {
		NotFound(ctx, err.Error())
		return
	}
	Success(ctx, license)
}

// GetByKey 根据 Key 获取 License（Query 参数传递）
// @Summary Get license by key
// @Tags licenses
// @Accept json
// @Produce json
// @Param key query string true "License Key"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 404 {object} api.APIResponse
// @Router /license/getByKey [get]
func (c *LicenseController) GetByKey(ctx *gin.Context) {
	var query dto.GetLicenseByKeyQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	license, err := c.ls.GetLicenseByKey(ctx, query.Key)
	if err != nil {
		NotFound(ctx, err.Error())
		return
	}
	Success(ctx, license)
}

// UpdateStatus 更新 License 状态（POST Body）
// @Summary Update license status
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseStatusCommand true "Update status"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /license/updateStatus [post]
func (c *LicenseController) UpdateStatus(ctx *gin.Context) {
	var cmd dto.UpdateLicenseStatusCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	if err := c.ls.UpdateLicenseStatus(ctx, cmd.ID, cmd.Status); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	SuccessMsg(ctx, "status updated")
}

// UpdateLicense 更新 License（POST Body）
// @Summary Update license
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseCommand true "Update License"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /license/update [post]
func (c *LicenseController) UpdateLicense(ctx *gin.Context) {
	var cmd dto.UpdateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	license := &entity.License{
		ID:            cmd.ID,
		LicenseKey:    cmd.LicenseKey,
		ValidityHours: cmd.ValidityHours,
		Status:        cmd.Status,
		Remark:        cmd.Remark,
	}

	if err := c.ls.UpdateLicense(ctx, license); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, license)
}

// DeleteExpired 删除过期 License（POST）
// @Summary Delete expired licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Success 200 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /license/deleteExpired [post]
func (c *LicenseController) DeleteExpired(ctx *gin.Context) {
	if err := c.ls.DeleteExpiredLicenses(ctx); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	SuccessMsg(ctx, "expired licenses deleted")
}
