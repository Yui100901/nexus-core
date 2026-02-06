package api

import (
	"nexus-core/api/dto"
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
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /product/create [post]
func (c *ProductController) CreateProduct(ctx *gin.Context) {
	var cmd dto.CreateProductCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	p, err := dto.ToEntityProduct(cmd)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.CreateProduct(ctx, p); err != nil {
		InternalError(ctx, err.Error())
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
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /product/createVersion [post]
func (c *ProductController) CreateProductVersion(ctx *gin.Context) {
	var cmd dto.CreateProductVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	v, err := dto.ToEntityVersion(cmd)
	if err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.CreateNewVersion(ctx, cmd.ProductID, v); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, v)
}

// BatchCreate 批量创建产品
// @Summary Batch create products
// @Tags products
// @Accept json
// @Produce json
// @Param body body []dto.CreateProductCommand true "Create Products"
// @Success 200 {array} entity.Product
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /product/batchCreate [post]
func (c *ProductController) BatchCreate(ctx *gin.Context) {
	var cmds []dto.CreateProductCommand
	if err := ctx.ShouldBindJSON(&cmds); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	var products []*entity.Product
	for _, cmd := range cmds {
		p, err := dto.ToEntityProduct(cmd)
		if err != nil {
			BadRequest(ctx, err.Error())
			return
		}
		products = append(products, p)
	}
	if err := c.ps.BatchCreateProduct(ctx, products); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	Success(ctx, products)
}

// GetByID 根据 ID 获取产品
// @Summary Get product by ID
// @Tags products
// @Accept json
// @Produce json
// @Param id query uint true "Product ID"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.APIResponse
// @Failure 404 {object} api.APIResponse
// @Router /product/getByID [get]
func (c *ProductController) GetByID(ctx *gin.Context) {
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

	// 调用服务层
	p, err := c.ps.GetByID(ctx, id)
	if err != nil {
		NotFound(ctx, err.Error())
		return
	}
	Success(ctx, p)
}

// GetByName 根据名称查询产品
// @Summary Get product by name
// @Tags products
// @Accept json
// @Produce json
// @Param name query string true "Product Name"
// @Success 200 {object} entity.Product
// @Failure 400 {object} api.APIResponse
// @Failure 404 {object} api.APIResponse
// @Router /product/getByName [get]
func (c *ProductController) GetByName(ctx *gin.Context) {
	var q dto.GetProductByNameQuery
	if err := ctx.ShouldBindQuery(&q); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	p, err := c.ps.GetByName(ctx, q.Name)
	if err != nil {
		NotFound(ctx, err.Error())
		return
	}
	Success(ctx, p)
}

// SetMinVersion 设置最小支持版本
// @Summary Set min supported version
// @Tags products
// @Accept json
// @Produce json
// @Param body body dto.UpdateMinVersionCommand true "Set Min Version"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /product/setMinVersion [post]
func (c *ProductController) SetMinVersion(ctx *gin.Context) {
	var cmd dto.UpdateMinVersionCommand
	if err := ctx.ShouldBindJSON(&cmd); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.SetMinSupportedVersion(ctx, cmd.ProductID, cmd.VersionID); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	SuccessMsg(ctx, "min supported version updated")
}

// DeleteProduct 删除产品
// @Summary Delete product
// @Tags products
// @Accept json
// @Produce json
// @Param body body object true "{\"id\": <product id>}"
// @Success 200 {object} api.APIResponse
// @Failure 400 {object} api.APIResponse
// @Failure 500 {object} api.APIResponse
// @Router /product/delete [post]
func (c *ProductController) DeleteProduct(ctx *gin.Context) {
	var q struct {
		ID uint `json:"id" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&q); err != nil {
		BadRequest(ctx, err.Error())
		return
	}
	if err := c.ps.DeleteProduct(ctx, q.ID); err != nil {
		InternalError(ctx, err.Error())
		return
	}
	SuccessMsg(ctx, "product deleted")
}

// ReleaseNewVersion 发布新版本
