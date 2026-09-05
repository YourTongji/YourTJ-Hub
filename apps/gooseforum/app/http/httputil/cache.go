// Package httputil contains small HTTP response helpers.
package httputil

import (
	"github.com/gin-gonic/gin"
)

// Cache-Control values shared by the static asset mounts and the file
// download controller.
const (
	LongPublicCacheControl = "public, max-age=18144000"
	ImmutableCacheControl  = "public, max-age=31536000, immutable"
	NoStoreCacheControl    = "no-store"
)

// SetLongPublic marks the response as publicly cacheable for static assets.
// Used by the file download controller, which only reaches it on the success
// path.
func SetLongPublic(c *gin.Context) {
	c.Header("Cache-Control", LongPublicCacheControl)
}

// statusAwareCacheWriter defers the Cache-Control decision until the response
// status is known. The static mounts must cache long, but gin's StaticFS
// answers a missing file by switching the same request onto the NoRoute chain
// — the status is only final when gin flushes the response. Writing the
// header up front would pin 404s (e.g. a missing content-hashed chunk during
// a deploy rollback) into caches under the long max-age.
type statusAwareCacheWriter struct {
	gin.ResponseWriter
	successHeader string
	decided       bool
}

// decide applies the Cache-Control header exactly once: 2xx carries the
// success header, every other status is explicitly no-store so failure
// responses can never be cached by browsers or shared caches.
func (w *statusAwareCacheWriter) decide(code int) {
	if w.decided {
		return
	}
	w.decided = true
	if code >= 200 && code < 300 {
		w.Header().Set("Cache-Control", w.successHeader)
	} else {
		w.Header().Set("Cache-Control", NoStoreCacheControl)
	}
}

func (w *statusAwareCacheWriter) WriteHeader(code int) {
	if code > 0 {
		w.decide(code)
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusAwareCacheWriter) WriteHeaderNow() {
	w.decide(w.Status())
	w.ResponseWriter.WriteHeaderNow()
}

func (w *statusAwareCacheWriter) Write(data []byte) (int, error) {
	w.decide(w.Status())
	return w.ResponseWriter.Write(data)
}

func (w *statusAwareCacheWriter) WriteString(s string) (int, error) {
	w.decide(w.Status())
	return w.ResponseWriter.WriteString(s)
}

// DeferCacheHeader wraps c.Writer so Cache-Control is chosen from the final
// response status: 2xx → successHeader, anything else → no-store. Install it
// in mount middleware before the handler, so the wrapper also covers the
// NoRoute fallback gin uses for missing static files.
func DeferCacheHeader(c *gin.Context, successHeader string) {
	c.Writer = &statusAwareCacheWriter{ResponseWriter: c.Writer, successHeader: successHeader}
}
