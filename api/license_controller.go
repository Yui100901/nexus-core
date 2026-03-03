package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"nexus-core/sc"

	"github.com/gin-gonic/gin"
)

// LicenseController 处理许可证相关的API端点
// 提供许可证的创建、查询、更新和删除等功能
// @Author yfy2001
// @Date 2026/1/20 09 11
type LicenseController struct {
	Api
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
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/create [post]
func (c *LicenseController) CreateLicense(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmd dto.CreateLicenseCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	license, err := entity.NewLicense(cmd.ValidityHours, cmd.MaxNodes, cmd.MaxConcurrent, cmd.Remark, dto.ToEntityScopes(cmd.ScopeList))

	if err != nil {
		c.BadRequest(sCtx, err.Error())
	}

	if err := c.ls.CreateLicense(sCtx, license); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, license)
}

// BatchCreate 批量创建 License
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
func (c *LicenseController) BatchCreate(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmds []dto.CreateLicenseCommand
	if err := gCtx.ShouldBindJSON(&cmds); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	var licenses []*entity.License
	for _, cmd := range cmds {
		license, err := entity.NewLicense(cmd.ValidityHours, cmd.MaxNodes, cmd.MaxConcurrent, cmd.Remark, dto.ToEntityScopes(cmd.ScopeList))
		if err != nil {
			c.BadRequest(sCtx, err.Error())
			return
		}
		licenses = append(licenses, license)
	}

	if err := c.ls.BatchCreateLicense(sCtx, licenses); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, licenses)
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
func (c *LicenseController) GetByID(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var query dto.GetLicenseByIDQuery
	if err := gCtx.ShouldBindQuery(&query); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	license, err := c.ls.GetLicenseByID(sCtx, query.ID)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, license)
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
func (c *LicenseController) GetByKey(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var query dto.GetLicenseByKeyQuery
	if err := gCtx.ShouldBindQuery(&query); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	license, err := c.ls.GetLicenseByKey(sCtx, query.Key)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, license)
}

// UpdateStatus 更新 License 状态（POST Body）
// @Summary Update license status
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseStatusCommand true "Update status"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/updateStatus [post]
func (c *LicenseController) UpdateStatus(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmd dto.UpdateLicenseStatusCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	if err := c.ls.UpdateLicenseStatus(sCtx, cmd.ID, cmd.Status); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "status updated")
}

// UpdateLicense 更新 License（POST Body）
// @Summary Update license
// @Tags licenses
// @Accept json
// @Produce json
// @Param body body dto.UpdateLicenseCommand true "Update License"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/update [post]
func (c *LicenseController) UpdateLicense(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	var cmd dto.UpdateLicenseCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	license := &entity.License{
		ID:            cmd.ID,
		LicenseKey:    cmd.LicenseKey,
		ValidityHours: cmd.ValidityHours,
		Status:        cmd.Status,
		Remark:        cmd.Remark,
	}

	if err := c.ls.UpdateLicense(sCtx, license); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, license)
}

// DeleteExpired 删除过期 License（POST）
// @Summary Delete expired licenses
// @Tags licenses
// @Accept json
// @Produce json
// @Success 200 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /license/deleteExpired [post]
func (c *LicenseController) DeleteExpired(gCtx *gin.Context) {
	sCtx := sc.InitContext(gCtx)
	if err := c.ls.DeleteExpiredLicenses(sCtx); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "expired licenses deleted")
}
