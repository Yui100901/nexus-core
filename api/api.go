package api

import (
	"net/http"
	"nexus-core/domain/service"

	"github.com/gin-gonic/gin"
)

//
// @Author yfy2001
// @Date 2026/2/28 09 58
//

// CommonResponse 定义统一的API响应格式
// 用于标准化接口返回的数据结构
type CommonResponse struct {
	Code    int         `json:"code"`           // 业务状态码
	Message string      `json:"message"`        // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据，可选
}

// APIResponse 兼容历史 swagger 注释中使用的 APIResponse 名称，避免文档与代码不一致
// 采用类型别名方式保持向后兼容，同时不破坏现有逻辑
type APIResponse = CommonResponse

const (
	CodeOK         = 200 // 成功状态码
	CodeBadRequest = 400 // 请求错误状态码
	CodeForbidden  = 403 // 权限不足状态码
	CodeNotFound   = 404 // 未找到状态码
	CodeConflict   = 409 // 状态冲突状态码
	CodeInternal   = 500 // 内部错误状态码
)

// JSON 发送自定义响应
// 根据指定的HTTP状态码、业务码、消息和数据构造响应
func JSON(ctx *gin.Context, httpStatus int, code int, message string, data interface{}) {
	ctx.JSON(httpStatus, CommonResponse{
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

// Forbidden 返回403错误响应
func Forbidden(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusForbidden, CodeForbidden, message, nil)
}

// NotFound 返回404错误响应
func NotFound(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusNotFound, CodeNotFound, message, nil)
}

// Conflict 返回409错误响应
func Conflict(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusConflict, CodeConflict, message, nil)
}

// InternalError 返回500错误响应
func InternalError(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusInternalServerError, CodeInternal, message, nil)
}

// HandleError 将 service 层业务错误统一映射为 HTTP 响应。
func HandleError(ctx *gin.Context, err error) {
	if err == nil {
		return
	}
	switch service.ErrorKindOf(err) {
	case service.ErrorKindBadRequest:
		BadRequest(ctx, err.Error())
	case service.ErrorKindNotFound:
		NotFound(ctx, err.Error())
	case service.ErrorKindForbidden:
		Forbidden(ctx, err.Error())
	case service.ErrorKindConflict:
		Conflict(ctx, err.Error())
	default:
		InternalError(ctx, err.Error())
	}
}
