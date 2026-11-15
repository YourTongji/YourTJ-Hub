package forum

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/gin-gonic/gin"
)

func renderHomeMarkdown(c *gin.Context) {
	// The homepage route refreshes a near-expiry session before this handler,
	// so any response here — including the error branch — may carry New-Token
	// and the access_token cookie. Keep every representation uncacheable.
	c.Header("Vary", "Accept, X-Goose-Page, Host")
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	content, err := llmsservice.BuildHomeMarkdown(component.GetHost(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "home markdown build error")
		return
	}
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
}
