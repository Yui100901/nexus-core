package api

import (
	"fmt"
	"log"
	"net/http"
	"nexus-core/ctx"
	"nexus-core/persistence/base"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

func ServiceContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		method := c.Request.Method
		path := c.Request.URL.Path
		prefix := fmt.Sprintf("[TraceID:%s] [RequestID:%s] [%s %s] ", traceID, requestID, method, path)
		logger := log.New(os.Stdout, prefix, log.LstdFlags)

		// 根据请求选择数据库
		db := base.Connect()

		sc := ctx.NewServiceContext(c, traceID, requestID, db, logger)
		c.Set("ServiceContext", sc)

		c.Next()
	}
}
