package api

import (
	"net/http"

	"nexus-core/persistence/base"
	"nexus-core/sc"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var appDB *gorm.DB

// SetDB sets the application-wide DB instance used by the middleware
func SetDB(db *gorm.DB) {
	appDB = db
}

// GetDB returns the configured app DB (may be nil)
func GetDB() *gorm.DB {
	return appDB
}

//
// @Author yfy2001
// @Date 2026/2/27 15 23
//

// CorsMiddleware 跨域配置中间件
// 配置CORS头部信息，允许跨域请求
func CorsMiddleware() gin.HandlerFunc {
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
		context.Next()
	}
}

// ServiceContextMiddleware 将 ServiceContext 注入到 gin.Context 中
func ServiceContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sctx := sc.InitContext(c)
		// inject app DB into service context so handlers/services can use sctx.GetDB()
		if appDB != nil {
			sctx.SetDB(appDB)
		} else {
			// fallback to base.Connect if SetDB was not called; this keeps backward compatibility
			sctx.SetDB(base.Connect())
		}
		c.Set(sc.ServiceContextKey, sctx)
		c.Next()
	}
}
