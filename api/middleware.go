package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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

//func ServiceContextMiddleware() gin.HandlerFunc {
//	return func(c *gin.Context) {
//
//		sc := ctx.InitContext(c)
//		c.Set("ServiceContext", sc)
//
//		c.Next()
//	}
//}
