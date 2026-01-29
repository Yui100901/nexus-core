package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 定义统一的API响应格式
// 用于标准化接口返回的数据结构
type APIResponse struct {
	Code    int         `json:"code"`           // 业务状态码
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据，可选
}

const (
	CodeOK         = 200 // 成功状态码
	CodeBadRequest = 400 // 请求错误状态码
	CodeNotFound   = 404 // 未找到状态码
	CodeInternal   = 500 // 内部错误状态码
)

// JSON 发送自定义响应
// 根据指定的HTTP状态码、业务码、消息和数据构造响应
func JSON(ctx *gin.Context, httpStatus int, code int, message string, data interface{}) {
	ctx.JSON(httpStatus, APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Success 返回成功响应并包含数据
func Success(ctx *gin.Context, data interface{}) {
	JSON(ctx, http.StatusOK, CodeOK, "ok", data)
}

// SuccessMsg 返回成功响应但仅包含消息
func SuccessMsg(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusOK, CodeOK, message, nil)
}

// BadRequest 返回400错误响应
func BadRequest(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusBadRequest, CodeBadRequest, message, nil)
}

// NotFound 返回404错误响应
func NotFound(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusNotFound, CodeNotFound, message, nil)
}

// InternalError 返回500错误响应
func InternalError(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusInternalServerError, CodeInternal, message, nil)
}
