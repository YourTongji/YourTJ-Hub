package forum

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/llmsservice"
	"github.com/gin-gonic/gin"
)

func renderHomeMarkdown(c *gin.Context) {
	content, err := llmsservice.BuildHomeMarkdown(component.GetHost(c))
	if err != nil {
		c.String(http.StatusInternalServerError, "home markdown build error")
		return
	}
	c.Header("Vary", "Accept, X-Goose-Page, Host")
	// The homepage route refreshes a near-expiry session before this handler.
	// A public cache could store that New-Token and access_token cookie.
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", []byte(content))
}
