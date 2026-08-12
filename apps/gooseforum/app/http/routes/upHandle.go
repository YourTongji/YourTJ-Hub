package routes

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/validate"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"

	"github.com/gin-gonic/gin"
)

// ginUpNP wraps handlers that do not need request parameters.
func ginUpNP(action func() component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		response := action()
		c.JSON(response.Code, response.Data)
	}
}

func UpButterReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindAndExecute(c, c.ShouldBind, action, false)
	}
}

// UpJsonReq binds JSON request bodies.
func UpJsonReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindAndExecute(c, c.ShouldBindJSON, action, true)
	}
}

// UpQueryReq binds query parameters.
func UpQueryReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindAndExecute(c, c.ShouldBindQuery, action, true)
	}
}

// UpUriReq binds URI path parameters.
func UpUriReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindAndExecute(c, c.ShouldBindUri, action, true)
	}
}

// UpFormReq binds form or multipart form data.
func UpFormReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindAndExecute(c, c.ShouldBind, action, true)
	}
}

// UpUriQueryReq binds URI path parameters then query parameters, both strictly.
func UpUriQueryReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindUriThenExecute(c, c.ShouldBindQuery, action)
	}
}

// UpUriJsonReq binds URI path parameters then the JSON body, both strictly.
func UpUriJsonReq[T any](action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	return func(c *gin.Context) {
		bindUriThenExecute(c, c.ShouldBindJSON, action)
	}
}

// bindUriThenExecute binds path parameters first, then the remaining payload
// source. Any binding failure is a strict HTTP 400 parse error.
// 400 不返回原始解析错误串（与 500 不泄漏内部信息一致），只给稳定 messageCode。
func bindUriThenExecute[T any](c *gin.Context, binder func(any) error, action func(component.BetterRequest[T]) component.Response) {
	userId := c.GetUint64("userId")
	var params T
	if err := c.ShouldBindUri(&params); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(
			component.MessageRequestParseFailed, nil))
		return
	}
	if err := binder(&params); err != nil {
		c.JSON(http.StatusBadRequest, component.FailDataCode(
			component.MessageRequestParseFailed, nil))
		return
	}
	executeValidated(c, params, userId, action)
}

// bindAndExecute binds params, validates them, and executes the controller action.
// strict 模式下绑定失败返回 400 稳定 messageCode，不泄漏原始错误串。
func bindAndExecute[T any](c *gin.Context, binder func(any) error, action func(component.BetterRequest[T]) component.Response, strict bool) {
	userId := c.GetUint64("userId")
	var params T
	if err := binder(&params); err != nil {
		if strict {
			c.JSON(http.StatusBadRequest, component.FailDataCode(
				component.MessageRequestParseFailed, nil))
			return
		}
	}
	executeValidated(c, params, userId, action)
}

// executeValidated validates params and executes the controller action.
func executeValidated[T any](c *gin.Context, params T, userId uint64, action func(component.BetterRequest[T]) component.Response) {
	if err := validate.Valid(params); err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageRequestInvalidParams, nil))
		return
	}

	response := action(component.BetterRequest[T]{
		Params:     params,
		UserId:     userId,
		GinContext: c,
	})
	c.JSON(response.Code, response.Data)
}
