package api

import (
	"net/http"
	"nexus-core/ctx"

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

type Api struct {
	ctx ctx.ServiceContext
}

// JSON 发送自定义响应
// 根据指定的HTTP状态码、业务码、消息和数据构造响应
func (a *Api) JSON(ctx *gin.Context, httpStatus int, code int, message string, data interface{}) {
	ctx.JSON(httpStatus, CommonResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Success 返回成功响应并包含数据
func (a *Api) Success(ctx *gin.Context, data interface{}) {
	JSON(ctx, http.StatusOK, CodeOK, "ok", data)
}

// SuccessMsg 返回成功响应但仅包含消息
func (a *Api) SuccessMsg(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusOK, CodeOK, message, nil)
}

// BadRequest 返回400错误响应
func (a *Api) BadRequest(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusBadRequest, CodeBadRequest, message, nil)
}

// NotFound 返回404错误响应
func (a *Api) NotFound(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusNotFound, CodeNotFound, message, nil)
}

// InternalError 返回500错误响应
func (a *Api) InternalError(ctx *gin.Context, message string) {
	JSON(ctx, http.StatusInternalServerError, CodeInternal, message, nil)
}
