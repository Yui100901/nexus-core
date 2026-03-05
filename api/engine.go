package api

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"nexus-core/config"
	"nexus-core/sc"

	_ "nexus-core/docs"
)

//
// @Author yfy2001
// @Date 2025/7/22 09 26
//

var WebEngine *gin.Engine

// NewServer 创建并配置Gin服务器引擎
// 包括跨域配置、请求ID中间件和简单日志中间件
func NewServer() *gin.Engine {
	r := gin.Default()
	r.Use(CorsMiddleware())
	r.Use(ServiceContextMiddleware())
	// simple logger
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		fmt.Printf("%s %s %d %s\n", c.Request.Method, c.Request.URL.Path, status, latency)
	})
	return r
}

// RegisterDefaultRoutes 注册默认的API路由
// 包括健康检查端点、控制器路由和Swagger文档
func RegisterDefaultRoutes() {
	// health
	WebEngine.GET("/health", func(c *gin.Context) {
		// try to get service context injected by middleware
		var sCtx *sc.ServiceContext
		if v, ok := c.Get("ServiceContext"); ok {
			if vCtx, ok2 := v.(*sc.ServiceContext); ok2 {
				sCtx = vCtx
			}
		}
		if sCtx == nil {
			// fallback to init local context
			sCtx = sc.InitContext(c)
		}
		(&Api{}).Success(sCtx, map[string]string{"status": "ok"})
	})

	// controllers
	NewLicenseController().RegisterRoutes(WebEngine)
	NewProductController().RegisterRoutes(WebEngine)
	NewNodeController().RegisterRoutes(WebEngine)
	NewAccessController().RegisterRoutes(WebEngine)

	// serve swagger UI under /swagger when enabled in config
	cfg := config.Get()
	if cfg.SwaggerEnabled {
		WebEngine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/swagger.json")))
	}
}

func init() {
	WebEngine = NewServer()
}
