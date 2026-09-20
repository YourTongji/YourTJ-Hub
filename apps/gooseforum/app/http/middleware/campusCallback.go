package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const guardFailureRedirect = "guardFailureRedirect"

// CampusCallbackPrivacy runs before session/account/rate guards. Their failures
// must also discard the school's code/state URL without exchanging credentials.
func CampusCallbackPrivacy(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	if c.FullPath() == "/api/campus/tongji/callback" {
		destination := "/campus?authorization=failed"
		if strings.HasPrefix(c.Query("state"), "login.") {
			destination = "/login?tongjiNotice=failed"
		}
		c.Set(guardFailureRedirect, destination)
	}
	c.Next()
}

func abortGuardFailure(c *gin.Context, status int, body any) {
	if destination := c.GetString(guardFailureRedirect); destination != "" {
		c.Redirect(http.StatusSeeOther, destination)
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(status, body)
}
