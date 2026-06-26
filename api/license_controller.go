package api

import (
	"nexus-core/api/dto"
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
	licenses := r.Group("/licenses")
	{
		licenses.POST("", c.CreateLicense)
		licenses.POST("/batch", c.BatchCreateLicenses)
		licenses.GET("", c.ListLicenses)
		licenses.GET("/:id", c.GetByID)
		licenses.PATCH("/:id", c.UpdateLicense)
		licenses.DELETE("/:id", c.DeleteLicense)
		licenses.POST("/:id/revoke", c.RevokeLicense)
		licenses.POST("/:id/restore", c.RestoreLicense)
		licenses.POST("/:id/renew", c.RenewLicense)
		licenses.DELETE("/:id/bindings", c.CleanLicenseBindings)
	}
	r.GET("/license-keys/:key", c.GetByKey)
	r.DELETE("/license-cleanups/invalid", c.CleanInvalidLicense)

	licenseGroup := r.Group("/license")
	{
		licenseGroup.POST("/createLicense", c.CreateLicense)               // 创建单个许可证
		licenseGroup.GET("/getByID", c.GetByID)                            // 根据ID查询许可证
		licenseGroup.GET("/getByKey", c.GetByKey)                          // 根据许可证密钥查询
		licenseGroup.POST("/revokeLicense", c.RevokeLicense)               // 吊销许可证
		licenseGroup.POST("/renewLicense", c.RenewLicense)                 // 续期
		licenseGroup.POST("/deleteLicense", c.DeleteLicense)               // 删除许可证
		licenseGroup.POST("/cleanLicenseBindings", c.CleanLicenseBindings) // 清理许可证绑定
		licenseGroup.POST("/update", c.UpdateLicense)                      // 更新许可证信息
		licenseGroup.POST("/cleanInvalidLicense", c.CleanInvalidLicense)   // 清理过期的许可证
	}
}

// ListLicenses 查询 License 列表
// @Summary List licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Param product_id query uint false "Product ID"
// @Param status query int false "License status"
// @Param license_key query string false "License key fuzzy filter"
// @Param page query int false "Page"
// @Param page_size query int false "Page Size"
// @Param limit query int false "Limit"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses [get]
func (c *LicenseController) ListLicenses(ctx *gin.Context) {
	page, err := PaginationQuery(ctx)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	productID, err := UintQuery(ctx, "product_id")
	if err != nil {
		BadRequest(ctx, "invalid product_id")
		return
	}
	status, err := IntQueryPtr(ctx, "status")
	if err != nil {
		BadRequest(ctx, "invalid status")
		return
	}
	data, err := c.ls.ListLicenses(ctx.Request.Context(), service.ListLicensesCommand{
		ProductID:  productID,
		Status:     status,
		LicenseKey: StringQuery(ctx, "license_key"),
		Limit:      page.Limit,
		Offset:     page.Offset,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// CreateLicense 创建 License
// @Summary Create a license
// @Description Create a new license with scopes
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.CreateLicenseCommand true "Create License"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses [post]
func (c *LicenseController) CreateLicense(ctx *gin.Context) {
	var cmd dto.CreateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	license, err := c.ls.CreateLicense(ctx.Request.Context(), service.CreateLicenseCommand{
		ProductID:     cmd.ProductID,
		ValidityHours: cmd.ValidityHours,
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		Remark:        cmd.Remark,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, license)
}

// GetByID 根据 ID 获取 License（Query 参数传递）
// @Summary Get license by ID
// @Tags licenses
// @Accept json
// @Produce json
// @Param id path uint true "License ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /licenses/{id} [get]
func (c *LicenseController) GetByID(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	license, err := c.ls.GetLicenseDataByID(ctx.Request.Context(), id)
	if err != nil {
		HandleError(ctx, err)
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
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /license-keys/{key} [get]
func (c *LicenseController) GetByKey(ctx *gin.Context) {
	// 获取 query 参数
	key := ctx.Param("key")
	if key == "" {
		key = ctx.Query("key")
	}
	if key == "" {
		key = ctx.Query("deviceCode")
	}
	if key == "" {
		BadRequest(ctx, "key is required")
		return
	}

	license, err := c.ls.GetLicenseDataByKey(ctx.Request.Context(), key)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, license)
}

// RevokeLicense 更新 License 状态（POST Body）
// @Summary Update license status
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseStatusCommand true "Update status"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id}/revoke [post]
func (c *LicenseController) RevokeLicense(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		var cmd dto.UpdateLicenseStatusCommand
		if bindErr := ctx.ShouldBindJSON(&cmd); bindErr != nil {
			BadRequest(ctx, err.Error())
			return
		}
		id = cmd.ID
	}
	if err := c.ls.RevokeLicense(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "status updated")
}

// UpdateLicense 通用的更新 License（POST Body）
// @Summary Update license
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseCommand true "Update License"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id} [patch]
func (c *LicenseController) UpdateLicense(ctx *gin.Context) {
	var cmd dto.UpdateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if id, err := UintParamOrQuery(ctx, "id"); err == nil {
		cmd.ID = id
	}
	if cmd.ID == 0 {
		BadRequest(ctx, "id is required")
		return
	}

	if err := c.ls.UpdateLicense(ctx.Request.Context(), service.UpdateLicenseCommand{
		ID:            cmd.ID,
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		FeatureMask:   cmd.FeatureMask,
		Remark:        cmd.Remark,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, "update success")
}

// RenewLicense 增加或减少许可证时间
// @Summary renew license
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseCommand true "renew License"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id}/renew [post]
func (c *LicenseController) RenewLicense(ctx *gin.Context) {
	var cmd dto.RenewLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if id, err := UintParamOrQuery(ctx, "id"); err == nil {
		cmd.ID = id
	}
	if cmd.ID == 0 {
		BadRequest(ctx, "id is required")
		return
	}

	if err := c.ls.RenewLicense(ctx.Request.Context(), service.RenewLicenseCommand{
		ID:         cmd.ID,
		ExtraHours: cmd.ExtraHours,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, "renew success")
}

// CleanLicenseBindings 清理该许可证相关的绑定
// @Summary Remove all node bindings of a license
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseCommand true "CleanLicenseBindings"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id}/bindings [delete]
func (c *LicenseController) CleanLicenseBindings(ctx *gin.Context) {
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

	if err := c.ls.RemoveBindings(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, "renew success")
}

// CleanInvalidLicense 清理无效的 License 包括过期，和已经被吊销
// @Summary Delete expired licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license-cleanups/invalid [delete]
func (c *LicenseController) CleanInvalidLicense(ctx *gin.Context) {
	if err := c.ls.CleanInvalidLicense(ctx.Request.Context()); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "invalid licenses deleted")
}

// DeleteLicense 删除单个license
// @Summary Delete expired licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id} [delete]
func (c *LicenseController) DeleteLicense(ctx *gin.Context) {
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

	if err := c.ls.DeleteLicense(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, "renew success")
}

// BatchCreateLicenses 批量创建 License
// @Summary Batch create licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.BatchCreateLicenseCommand true "Batch Create Licenses"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/batch [post]
func (c *LicenseController) BatchCreateLicenses(ctx *gin.Context) {
	var cmd dto.BatchCreateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.ls.BatchCreateLicenses(ctx.Request.Context(), service.BatchCreateLicenseCommand{
		ProductID:     cmd.ProductID,
		ValidityHours: cmd.ValidityHours,
		MaxNodes:      cmd.MaxNodes,
		MaxConcurrent: cmd.MaxConcurrent,
		Remark:        cmd.Remark,
		Count:         cmd.Count,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// RestoreLicense 恢复已吊销 License
// @Summary Restore a revoked license
// @Tags licenses
// @Accept json
// @Produce json
// @Param id path uint true "License ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /licenses/{id}/restore [post]
func (c *LicenseController) RestoreLicense(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		var cmd dto.RestoreLicenseCommand
		if bindErr := ctx.ShouldBindJSON(&cmd); bindErr != nil {
			BadRequest(ctx, err.Error())
			return
		}
		id = cmd.ID
	}

	data, err := c.ls.RestoreLicense(ctx.Request.Context(), service.RestoreLicenseCommand{ID: id})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}
