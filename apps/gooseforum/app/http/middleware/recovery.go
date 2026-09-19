package middleware

import (
	"net/http"

	paniclog "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
	"github.com/gin-gonic/gin"
)

// Recovery logs request context for panics and returns HTTP 500.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				attrs := []any{
					"method", c.Request.Method,
					"path", c.Request.URL.Path,
					"query", logQuery(c.Request.URL),
					"route", c.FullPath(),
					"ip", c.ClientIP(),
					"user_agent", c.Request.UserAgent(),
				}
				if referer := c.Request.Referer(); referer != "" {
					attrs = append(attrs, "referer", referer)
				}
				paniclog.LogPanic("http_request", err, attrs...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
