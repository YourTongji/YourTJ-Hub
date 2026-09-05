package middleware

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/httputil"
	"github.com/gin-gonic/gin"
)

// BrowserCache applies long public cache headers to successful /static
// responses in production; error responses get no-store. The header decision
// is deferred until the status is known (see httputil.DeferCacheHeader):
// gin's StaticFS answers missing files through the NoRoute chain inside the
// same request, so a 404 must never carry the long max-age.
func BrowserCache(c *gin.Context) {
	if !setting.IsProduction() {
		c.Next()
		return
	}
	httputil.DeferCacheHeader(c, httputil.LongPublicCacheControl)
	c.Next()
}

// AssetsCache applies immutable long cache headers to successful /assets
// responses in production; error responses get no-store. Vite build output
// carries a content hash in the filename, so a byte change always changes
// the URL — a year-long immutable cache is safe for 200s and removes
// revalidation requests on repeat visits, while a missing hashed chunk
// (deploy rollback window) stays uncachable instead of being pinned for a
// year. Local/dev responses stay header-free so editors never fight a stale
// cached bundle.
func AssetsCache(c *gin.Context) {
	if !setting.IsProduction() {
		c.Next()
		return
	}
	httputil.DeferCacheHeader(c, httputil.ImmutableCacheControl)
	c.Next()
}
