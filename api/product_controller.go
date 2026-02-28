package api

import (
	"nexus-core/api/dto"
	"nexus-core/ctx"
	"nexus-core/domain/entity"
	"nexus-core/domain/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProductController 处理产品相关的API请求
// 管理产品的创建、查询、版本控制等操作
// @Author yfy2001
// @Date 2026/1/20 10 54
type ProductController struct {
	Api
	ps *service.ProductService // 产品服务，处理产品相关的业务逻辑
}

// NewProductController 创建新的产品控制器实例
func NewProductController() *ProductController {
	return &ProductController{ps: service.NewProductService()}
}

// RegisterRoutes 注册产品相关的路由
// 包括产品创建、查询、版本控制等操作的路由
func (c *ProductController) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/product")
	{
		g.POST("/create", c.CreateProduct)               // 创建产品
		g.POST("/createVersion", c.CreateProductVersion) // 创建产品版本
		g.POST("/batchCreate", c.BatchCreate)            // 批量创建产品
		g.GET("/getByID", c.GetByID)                     // 根据ID获取产品
		g.GET("/getByName", c.GetByName)                 // 根据名称获取产品
		g.POST("/setMinVersion", c.SetMinVersion)        // 设置最小支持版本
		g.POST("/delete", c.DeleteProduct)               // 删除产品
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
// @Router /product/create [post]
func (c *ProductController) CreateProduct(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmd dto.CreateProductCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	p, err := dto.ToEntityProduct(cmd)
	if err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ps.CreateProduct(gCtx, p); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, p)
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
// @Router /product/createVersion [post]
func (c *ProductController) CreateProductVersion(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmd dto.CreateProductVersionCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	v, err := dto.ToEntityVersion(cmd)
	if err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ps.CreateNewVersion(gCtx, cmd.ProductID, v); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, v)
}

// ReleaseNewVersion 发布新版本
// @Summary Release a new version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.ReleaseNewVersionCommand true "Release New Version"
// @Success 200 {object} entity.Version
func (c *ProductController) ReleaseNewVersion(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmd dto.ReleaseNewVersionCommand

	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}

	err := c.ps.ReleaseVersion(gCtx, cmd.ProductID, cmd.VersionID, cmd.ReleaseDate)
	if err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}

	c.Success(sCtx, cmd.VersionID)

}

// BatchCreate 批量创建产品
// @Summary Batch create products
// @Tags products
// @Accept json
// @Produce json
// @Param body body []dto.CreateProductCommand true "Create Products"
// @Success 200 {array} entity.Product
// @Failure 400 {object} api.CommonResponse
// @Failure 500 {object} api.CommonResponse
// @Router /product/batchCreate [post]
func (c *ProductController) BatchCreate(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmds []dto.CreateProductCommand
	if err := gCtx.ShouldBindJSON(&cmds); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	var products []*entity.Product
	for _, cmd := range cmds {
		p, err := dto.ToEntityProduct(cmd)
		if err != nil {
			c.BadRequest(sCtx, err.Error())
			return
		}
		products = append(products, p)
	}
	if err := c.ps.BatchCreateProduct(gCtx, products); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.Success(sCtx, products)
}

// GetByID 根据 ID 获取产品
// @Summary Get product by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id query uint true "Product ID"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /product/getByID [get]
func (c *ProductController) GetByID(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	// 获取 query 参数
	idStr := gCtx.Query("id")
	if idStr == "" {
		c.NotFound(sCtx, "id is required")
		return
	}

	// 转换为 uint
	idUint64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.NotFound(sCtx, "invalid id")
		return
	}
	id := uint(idUint64)

	// 调用服务层
	p, err := c.ps.GetByID(gCtx, id)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, p)
}

// GetByName 根据名称查询产品
// @Summary Get product by name
// @Tags products
// @Accept json
// @Produce json
// @Param name query string true "Product Name"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.CommonResponse
// @Failure 404 {object} api.CommonResponse
// @Router /product/getByName [get]
func (c *ProductController) GetByName(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var q dto.GetProductByNameQuery
	if err := gCtx.ShouldBindQuery(&q); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	p, err := c.ps.GetByName(gCtx, q.Name)
	if err != nil {
		c.NotFound(sCtx, err.Error())
		return
	}
	c.Success(sCtx, p)
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
// @Router /product/setMinVersion [post]
func (c *ProductController) SetMinVersion(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmd dto.UpdateMinVersionCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ps.SetMinSupportedVersion(gCtx, cmd.ProductID, cmd.VersionID); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "min supported version updated")
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
// @Router /product/deprecateVersion [post]
func (c *ProductController) DeprecateVersion(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var cmd dto.DeprecateVersionCommand
	if err := gCtx.ShouldBindJSON(&cmd); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ps.DeprecateVersion(gCtx, cmd.ProductID, cmd.VersionID); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "version deprecated")
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
// @Router /product/delete [post]
func (c *ProductController) DeleteProduct(gCtx *gin.Context) {
	sCtx := ctx.InitContext(gCtx)
	var q struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := gCtx.ShouldBindJSON(&q); err != nil {
		c.BadRequest(sCtx, err.Error())
		return
	}
	if err := c.ps.DeleteProduct(gCtx, q.ID); err != nil {
		c.InternalError(sCtx, err.Error())
		return
	}
	c.SuccessMsg(sCtx, "product deleted")
}
