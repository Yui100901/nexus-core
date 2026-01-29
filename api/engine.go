package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "nexus-core/docs"
)

//
// @Author yfy2001
// @Date 2025/7/22 09 26
//

var WebEngine *gin.Engine

// Cors 跨域配置中间件
// 配置CORS头部信息，允许跨域请求
func Cors() gin.HandlerFunc {
	return func(context *gin.Context) {
		method := context.Request.Method
		context.Header("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token, x-token")
		context.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, DELETE, PATCH, PUT")
		context.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
		context.Header("Access-Control-Allow-Credentials", "true")
		if method == "OPTIONS" {
			context.AbortWithStatus(http.StatusNoContent)
		}
	}
}

// NewServer 创建并配置Gin服务器引擎
// 包括跨域配置、请求ID中间件和简单日志中间件
func NewServer() *gin.Engine {
	r := gin.Default()
	r.Use(Cors())
	// request id middleware
	r.Use(func(c *gin.Context) {
		id := uuid.New().String()
		c.Writer.Header().Set("X-Request-ID", id)
		c.Set("request_id", id)
		c.Next()
	})
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
		// use unified success response
		Success(c, map[string]string{"status": "ok"})
	})

	// controllers
	NewLicenseController().RegisterRoutes(WebEngine)
	NewProductController().RegisterRoutes(WebEngine)
	NewNodeController().RegisterRoutes(WebEngine)
	NewDeviceController().RegisterRoutes(WebEngine)

	// serve swagger UI under /docs to avoid conflicts with potential /swagger/doc.json
	WebEngine.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/docs/swagger.json")))
}

func init() {
	WebEngine = NewServer()
}
