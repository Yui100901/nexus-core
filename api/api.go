package api

import (
	"net/http"
	"nexus-core/sc"

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

// 兼容历史 swagger 注释中使用的 APIResponse 名称，避免文档与代码不一致
// 采用类型别名方式保持向后兼容，同时不破坏现有逻辑
type APIResponse = CommonResponse

const (
	CodeOK         = 200 // 成功状态码
	CodeBadRequest = 400 // 请求错误状态码
	CodeNotFound   = 404 // 未找到状态码
	CodeInternal   = 500 // 内部错误状态码
)

// ServiceContextKey 是注入到 gin.Context 中的键名，统一使用常量避免字符串散落
const ServiceContextKey = "ServiceContext"

type Api struct {
}

// JSON 发送自定义响应
// 根据指定的HTTP状态码、业务码、消息和数据构造响应
func (a *Api) JSON(ctx *sc.ServiceContext, httpStatus int, code int, message string, data interface{}) {
	ctx.GinContext.JSON(httpStatus, CommonResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Success 返回成功响应并包含数据
func (a *Api) Success(ctx *sc.ServiceContext, data interface{}) {
	a.JSON(ctx, http.StatusOK, CodeOK, "ok", data)
}

// SuccessMsg 返回成功响应但仅包含消息
func (a *Api) SuccessMsg(ctx *sc.ServiceContext, message string) {
	a.JSON(ctx, http.StatusOK, CodeOK, message, nil)
}

// BadRequest 返回400错误响应
func (a *Api) BadRequest(ctx *sc.ServiceContext, message string) {
	a.JSON(ctx, http.StatusBadRequest, CodeBadRequest, message, nil)
}

// NotFound 返回404错误响应
func (a *Api) NotFound(ctx *sc.ServiceContext, message string) {
	a.JSON(ctx, http.StatusNotFound, CodeNotFound, message, nil)
}

// InternalError 返回500错误响应
func (a *Api) InternalError(ctx *sc.ServiceContext, message string) {
	a.JSON(ctx, http.StatusInternalServerError, CodeInternal, message, nil)
}

// getServiceContextFromGin 从 gin.Context 中安全获取 *sc.ServiceContext
// 返回 (nil, false) 当未注入或类型不匹配
func getServiceContextFromGin(gCtx *gin.Context) (*sc.ServiceContext, bool) {
	if v, ok := gCtx.Get(ServiceContextKey); ok {
		if sctx, ok2 := v.(*sc.ServiceContext); ok2 {
			return sctx, true
		}
	}
	return nil, false
}
