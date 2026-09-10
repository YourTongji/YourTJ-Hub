package controllers

import (
	"net/http"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"

	"github.com/gin-gonic/gin"
)

const (
	contentTypeHTML      = "text/html"
	errorCodeNotFound    = 404
	errorMessageNotFound = "路由未定义，请确认 url 和请求方法是否正确。"
)

func NotFound(c *gin.Context) {
	if strings.Contains(c.GetHeader("Accept"), contentTypeHTML) {
		forum.RenderNotFoundPage(c, component.MessageRouteNotFound)
		return
	}
	c.JSON(http.StatusNotFound, component.DataMap{
		"code":        errorCodeNotFound,
		"msg":         errorMessageNotFound,
		"messageCode": component.MessageRouteNotFound,
	})
}
