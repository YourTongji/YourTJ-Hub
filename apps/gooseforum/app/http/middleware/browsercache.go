package middleware

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/httputil"
	"github.com/gin-gonic/gin"
)

// BrowserCache applies long public cache headers in production.
func BrowserCache(c *gin.Context) {
	if !setting.IsProduction() {
		c.Next()
		return
	}
	httputil.SetLongPublic(c)
	c.Next()
}

// AssetsCache applies immutable long cache headers to the /assets mount in
// production. Vite build output carries a content hash in the filename, so a
// byte change always changes the URL — a year-long immutable cache is safe
// and removes revalidation requests on repeat visits. Local/dev responses
// stay header-free so editors never fight a stale cached bundle.
func AssetsCache(c *gin.Context) {
	if !setting.IsProduction() {
		c.Next()
		return
	}
	httputil.SetImmutable(c)
	c.Next()
}
