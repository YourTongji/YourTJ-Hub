package pkcontroller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response PK 统一信封 {code, msg, data}（Issue #187 / PRD §5.4.4）。
// 成功 code=0；错误 code≠0 且 msg 为可读中文，HTTP 状态码与 code 对齐。
type Response struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

const (
	CodeOK         = 0
	CodeBadRequest = http.StatusBadRequest
	CodeInternal   = http.StatusInternalServerError
)

// Ok 成功响应（默认文案"查询成功"，对齐上游 jsonOk）。
func Ok(data any) Response {
	return Response{Code: CodeOK, Msg: "查询成功", Data: data}
}

// BadRequest 参数错误响应。
func BadRequest(msg string) Response {
	return Response{Code: CodeBadRequest, Msg: msg, Data: map[string]any{}}
}

// Internal 服务器内部错误响应。
func Internal(msg string) Response {
	return Response{Code: CodeInternal, Msg: msg, Data: map[string]any{}}
}

// HTTPStatus 信封对应的 HTTP 状态码。
func (r Response) HTTPStatus() int {
	if r.Code == CodeOK {
		return http.StatusOK
	}
	return r.Code
}

// Request PK handler 请求包装（对齐 component.BetterRequest 的角色）。
type Request[T any] struct {
	Params     T
	GinContext *gin.Context
}
