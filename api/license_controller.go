package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/service"
	"strconv"

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
		licenseGroup.POST("/createLicense", c.CreateLicense)               // 创建单个许可证
		licenseGroup.POST("/batchCreateLicense", c.BatchCreateLicense)     // 批量创建许可证
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
// @Router /license/create [post]
func (c *LicenseController) CreateLicense(ctx *gin.Context) {
	var cmd dto.CreateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	license, err := c.ls.CreateLicense(cmd)
	if err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, license)
}

// BatchCreateLicense 批量创建 License
// @Summary Batch create licenses
// @Description Create multiple licenses in batch
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body []dto.CreateLicenseCommand true "Create Licenses"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/batchCreate [post]
func (c *LicenseController) BatchCreateLicense(ctx *gin.Context) {
	//var cmds []dto.CreateLicenseCommand
	//if err := ctx.ShouldBindJSON(&cmds); err != nil {
	//	BadRequest(ctx, err.Error())
	//	return
	//}
	//
	//var licenses []*entity.License
	//for _, cmd := range cmds {
	//	license := entity.NewLicense(cmd.ProductID, cmd.ValidityHours, cmd.MaxNodes, cmd.MaxConcurrent, cmd.Remark)
	//	licenses = append(licenses, license)
	//}
	//
	//if err := c.ls.BatchCreateLicense(ctx, licenses); err != nil {
	//	InternalError(ctx, err.Error())
	//	return
	//}
	//Success(ctx, licenses)
}

// GetByID 根据 ID 获取 License（Query 参数传递）
// @Summary Get license by ID
// @Tags licenses
// @Accept json
// @Produce json
// @Param id query uint true "License ID"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /license/getByID [get]
func (c *LicenseController) GetByID(ctx *gin.Context) {
	// 获取 query 参数
	idStr := ctx.Query("id")
	if idStr == "" {
		NotFound(ctx, "id is required")
		return
	}

	// 转换为 uint
	idUint64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		NotFound(ctx, "invalid id")
		return
	}
	id := uint(idUint64)

	license, err := c.ls.GetLicenseDataByID(id)
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
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /license/getByKey [get]
func (c *LicenseController) GetByKey(ctx *gin.Context) {
	// 获取 query 参数
	key := ctx.Query("deviceCode")

	license, err := c.ls.GetLicenseDataByKey(key)
	if err != nil {
		NotFound(ctx, err.Error())
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
// @Router /license/updateStatus [post]
func (c *LicenseController) RevokeLicense(ctx *gin.Context) {
	var cmd dto.UpdateLicenseStatusCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ls.RevokeLicense(cmd.ID); err != nil {
		InternalError(ctx, err.Error())
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
// @Router /license/update [post]
func (c *LicenseController) UpdateLicense(ctx *gin.Context) {
	var cmd dto.UpdateLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	if err := c.ls.UpdateLicense(cmd); err != nil {
		InternalError(ctx, err.Error())
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
// @Router /license/update [post]
func (c *LicenseController) RenewLicense(ctx *gin.Context) {
	var cmd dto.RenewLicenseCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	if err := c.ls.RenewLicense(cmd); err != nil {
		InternalError(ctx, err.Error())
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
// @Router /license/update [post]
func (c *LicenseController) CleanLicenseBindings(ctx *gin.Context) {
	var cmd struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	if err := c.ls.RemoveBindings(cmd.ID); err != nil {
		InternalError(ctx, err.Error())
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
// @Router /license/deleteExpired [post]
func (c *LicenseController) CleanInvalidLicense(ctx *gin.Context) {
	if err := c.ls.CleanInvalidLicense(); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	SuccessMsg(ctx, "expired licenses deleted")
}

// DeleteLicense 删除单个license
// @Summary Delete expired licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/deleteExpired [post]
func (c *LicenseController) DeleteLicense(ctx *gin.Context) {
	var cmd struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	if err := c.ls.DeleteLicense(cmd.ID); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, "renew success")
}
