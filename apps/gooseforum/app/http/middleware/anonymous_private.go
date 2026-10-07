package middleware

import "github.com/gin-gonic/gin"

func AnonymousPrivateResponse(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Next()
}
