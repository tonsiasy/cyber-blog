package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一 API 响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// Success 成功返回 (HTTP 200)
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "success",
		Data:    data,
	})
}

// Created 成功创建资源 (HTTP 201)
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    http.StatusCreated,
		Message: "created",
		Data:    data,
	})
}

// Fail 业务或客户端错误返回
func Fail(c *gin.Context, httpStatus int, message string, errDetail ...string) {
	detail := ""
	if len(errDetail) > 0 {
		detail = errDetail[0]
	}
	c.JSON(httpStatus, Response{
		Code:    httpStatus,
		Message: message,
		Error:   detail,
	})
}

// BadRequest 参数校验失败 (HTTP 400)
func BadRequest(c *gin.Context, message string, errDetail ...string) {
	Fail(c, http.StatusBadRequest, message, errDetail...)
}

// Unauthorized 未授权或 Token 无效 (HTTP 401)
func Unauthorized(c *gin.Context, message string) {
	Fail(c, http.StatusUnauthorized, message)
}

// Forbidden 权限不足 (HTTP 403)
func Forbidden(c *gin.Context, message string) {
	Fail(c, http.StatusForbidden, message)
}

// NotFound 资源不存在 (HTTP 404)
func NotFound(c *gin.Context, message string) {
	Fail(c, http.StatusNotFound, message)
}

// InternalServerError 服务器内部错误 (HTTP 500)
func InternalServerError(c *gin.Context, message string, errDetail ...string) {
	Fail(c, http.StatusInternalServerError, message, errDetail...)
}
