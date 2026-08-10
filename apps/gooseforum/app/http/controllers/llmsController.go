package controllers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/service/llmsservice"
	"github.com/spf13/cast"
)

const llmsCacheControl = "public, max-age=10"

func RenderLLMSIndex(c *gin.Context) {
	content, err := llmsservice.BuildIndex(component.GetHost(c))
	renderLLMS(c, "text/plain; charset=utf-8", content, err)
}

func RenderLLMSFull(c *gin.Context) {
	content, err := llmsservice.BuildFull(component.GetHost(c))
	renderLLMS(c, "text/plain; charset=utf-8", content, err)
}

func RenderLLMSTopic(c *gin.Context) {
	document := c.Param("document")
	if !strings.HasSuffix(document, ".md") {
		c.Status(http.StatusNotFound)
		return
	}
	topicID := cast.ToUint64(strings.TrimSuffix(document, ".md"))
	content, err := llmsservice.BuildTopic(component.GetHost(c), topicID)
	renderLLMS(c, "text/markdown; charset=utf-8", content, err)
}

func renderLLMS(c *gin.Context, contentType string, content string, err error) {
	if errors.Is(err, llmsservice.ErrUnavailable) || errors.Is(err, llmsservice.ErrTopicMissing) {
		c.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "LLMS export build error")
		return
	}
	c.Header("Cache-Control", llmsCacheControl)
	// 输出内嵌 Host 相关绝对链接：Vary 保证 CDN/反向代理按 Host 分开缓存，防止恶意 Host 投毒。
	c.Header("Vary", "Host")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, contentType, []byte(content))
}
