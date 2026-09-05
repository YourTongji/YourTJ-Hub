// Package httputil contains small HTTP response helpers.
package httputil

import (
	"github.com/gin-gonic/gin"
)

// SetLongPublic marks the response as publicly cacheable for static assets.
func SetLongPublic(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=18144000")
}

// SetImmutable marks the response as immutable-cached for one year. Reserved
// for content-addressed assets (Vite build output carries a content hash in
// the filename), where any byte change is guaranteed to change the URL.
func SetImmutable(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
}
