package api

import (
	"nexus-core/api/dto"
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

// ProductController 处理产品相关的API请求
// 管理产品的创建、查询、版本控制等操作
// @Author yfy2001
// @Date 2026/1/20 10 54
type ProductController struct {
	ps *service.ProductService // 产品服务，处理产品相关的业务逻辑
}

// NewProductController 创建新的产品控制器实例
func NewProductController() *ProductController {
	return &ProductController{ps: service.NewProductService()}
}

// RegisterRoutes 注册产品相关的路由
// 包括产品创建、查询、版本控制等操作的路由
func (c *ProductController) RegisterRoutes(r *gin.Engine) {
	products := r.Group("/products")
	{
		products.POST("", c.CreateProduct)
		products.GET("/:id", c.GetByID)
		products.PATCH("/:id", c.UpdateProduct)
		products.DELETE("/:id", c.DeleteProduct)
		products.POST("/versions", c.CreateProductVersion)
		products.POST("/versions/release", c.ReleaseNewVersion)
		products.POST("/versions/deprecate", c.DeprecateVersion)
		products.POST("/min-supported-version", c.SetMinVersion)
	}

	g := r.Group("/product")
	{
		g.POST("/createProduct", c.CreateProduct)               // 创建产品
		g.POST("/createProductVersion", c.CreateProductVersion) // 创建产品版本
		g.POST("/releaseNewVersion", c.ReleaseNewVersion)       //手动发布版本
		g.GET("/getByID", c.GetByID)                            // 根据ID获取产品
		g.POST("/setMinVersion", c.SetMinVersion)               // 设置最小支持版本
		g.POST("deprecateVersion", c.DeprecateVersion)          // 废弃版本
		g.POST("/deleteProduct", c.DeleteProduct)               // 删除产品
	}
}

// CreateProduct 创建产品
// @Summary Create a product
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.CreateProductCommand true "Create Product"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products [post]
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var cmd dto.CreateProductCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	p, err := c.ps.CreateProduct(ctx.Request.Context(), service.CreateProductCommand{
		Name:        cmd.Name,
		Description: cmd.Description,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, p)
}

// CreateProductVersion 创建产品版本
// @Summary Create a product version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.CreateProductVersionCommand true "Create Product Version"
// @Success 200 {object} entity.Version
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products/versions [post]
func (c *ProductController) CreateProductVersion(ctx *gin.Context) {
	var cmd dto.CreateProductVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	v, err := c.ps.CreateProductVersion(ctx.Request.Context(), service.CreateProductVersionCommand{
		ProductID:   cmd.ProductID,
		VersionCode: cmd.VersionCode,
		ReleaseDate: cmd.ReleaseDate,
		Description: cmd.Description,
		Method:      service.ReleaseMethod(cmd.Method),
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, v)
}

// ReleaseNewVersion 发布新版本
// @Summary Release a new version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.ReleaseNewVersionCommand true "Release New Version"
// @Success 200 {object} entity.Version
// @Router /products/versions/release [post]
func (c *ProductController) ReleaseNewVersion(ctx *gin.Context) {
	var cmd dto.ReleaseNewVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	err := c.ps.ReleaseVersion(ctx.Request.Context(), service.ReleaseNewVersionCommand{
		ProductID:   cmd.ProductID,
		VersionID:   cmd.VersionID,
		ReleaseDate: cmd.ReleaseDate,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, cmd.VersionID)
}

// GetByID 根据 ID 获取产品
// @Summary Get product by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path uint true "Product ID"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /products/{id} [get]
func (c *ProductController) GetByID(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}

	data, err := c.ps.GetProductDataByID(ctx.Request.Context(), id)
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}

// SetMinVersion 设置最小支持版本
// @Summary Set min supported version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.UpdateMinVersionCommand true "Set Min Version"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products/min-supported-version [post]
func (c *ProductController) SetMinVersion(ctx *gin.Context) {
	var cmd dto.UpdateMinVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.SetMinSupportedVersion(ctx.Request.Context(), service.UpdateMinVersionCommand{
		ProductID: cmd.ProductID,
		VersionID: cmd.VersionID,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "min supported version updated")
}

// DeprecateVersion 废弃版本
// @Summary Deprecate a version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.DeprecateVersionCommand true "Deprecate Version"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products/versions/deprecate [post]
func (c *ProductController) DeprecateVersion(ctx *gin.Context) {
	var cmd dto.DeprecateVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.DeprecateVersion(ctx.Request.Context(), service.DeprecateVersionCommand{
		ProductID: cmd.ProductID,
		VersionID: cmd.VersionID,
	}); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "version deprecated")
}

// DeleteProduct 删除产品
// @Summary Delete product
// @Tags products
// @Accept json
// @Produce json
// @Param body body object true "{\"id\": <product id>}"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products/{id} [delete]
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	id, err := UintParamOrQuery(ctx, "id")
	if err != nil {
		var q struct {
			ID uint `json:"id" binding:"required"`
		}
		if bindErr := ctx.ShouldBindJSON(&q); bindErr != nil {
			BadRequest(ctx, err.Error())
			return
		}
		id = q.ID
	}
	if err := c.ps.DeleteProduct(ctx.Request.Context(), id); err != nil {
		HandleError(ctx, err)
		return
	}
	SuccessMsg(ctx, "product deleted")
}

// UpdateProduct 更新产品基础信息
// @Summary Update product
// @Tags products
// @Accept json
// @Produce json
// @Param id path uint true "Product ID"
// @Param body body dto.UpdateProductCommand true "Update Product"
// @Success 200 {object} api.CommonResponse
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /products/{id} [patch]
func (c *ProductController) UpdateProduct(ctx *gin.Context) {
	var cmd dto.UpdateProductCommand
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

	data, err := c.ps.UpdateProduct(ctx.Request.Context(), service.UpdateProductCommand{
		ID:          cmd.ID,
		Name:        cmd.Name,
		Description: cmd.Description,
	})
	if err != nil {
		HandleError(ctx, err)
		return
	}
	Success(ctx, data)
}
