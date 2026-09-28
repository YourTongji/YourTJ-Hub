package routes

import (
	"errors"
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

// maxContentWriteBodyBytes 是正文/回复写接口的请求体硬上限。控制器内还有与
// maxPostLength 联动的源文本护栏作为语义上限，这里只兜住解析超大请求体的内存
// 与 CPU：2 MiB 覆盖默认配置下 UTF-8 最坏情况（maxPostLength×4 码点 × 4 字节
// ≈ 800 KB）并留有余量。
const maxContentWriteBodyBytes = 2 << 20

// UpLimitedJsonReq applies a hard body limit before the standard strict JSON
// binding path. It is intended for public endpoints whose work is more
// expensive than decoding the request itself.
func UpLimitedJsonReq[T any](maxBytes int64, action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	handler := UpJsonReq(action)
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		handler(c)
	}
}

// UpLimitedButterReq applies a hard body limit while preserving the lenient JSON
// binding of content write endpoints. A body over maxBytes is rejected with the
// standard parse-failure response before the controller runs; malformed bodies
// within the limit keep the legacy lenient behavior (bound to zero values and
// failed as business errors).
func UpLimitedButterReq[T any](maxBytes int64, action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	handler := UpButterReq(action)
	return func(c *gin.Context) {
		// Content-Length 决定了大多数请求的拒绝路径；谎报/分块请求由
		// MaxBytesReader + bindAndExecute 的 MaxBytesError 分支兜底。
		if c.Request.ContentLength > maxBytes {
			c.JSON(http.StatusBadRequest, component.FailDataCode(component.MessageRequestParseFailed, nil))
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		handler(c)
	}
}

// UpUriLimitedJsonReq binds URI path parameters then a size-limited strict JSON
// body. Bodies over maxBytes fail as parse errors, like UpLimitedJsonReq.
func UpUriLimitedJsonReq[T any](maxBytes int64, action func(ctx component.BetterRequest[T]) component.Response) func(c *gin.Context) {
	handler := UpUriJsonReq(action)
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		handler(c)
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
		// 请求体越过 MaxBytesReader 上限时一律按解析失败拒绝，即使路由使用
		// 宽松绑定：被截断的请求体不能降级成零值业务错误。其余解析错误保持
		// 宽松/严格各自的既有语义。
		var maxBytesErr *http.MaxBytesError
		if strict || errors.As(err, &maxBytesErr) {
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
