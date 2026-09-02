package middleware

import (
	_ "embed"
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

//go:embed startup.html
var startupHTML []byte

// StartupGate keeps every request on a temporary loading page until the
// startup work (database migration and dependent service boot) completes.
// Load balancers and health probes observe 503 + Retry-After, so a
// half-migrated instance is never treated as available.
type StartupGate struct {
	ready atomic.Int32
}

func NewStartupGate() *StartupGate {
	return &StartupGate{}
}

// Complete marks the instance as ready to serve business traffic.
func (g *StartupGate) Complete() {
	g.ready.Store(1)
}

// Ready reports whether the startup gate has been opened.
func (g *StartupGate) Ready() bool {
	return g.ready.Load() == 1
}

// Handler returns the loading page until Complete is called, then passes
// through. It must be registered before any empty routes so nothing
// DB-dependent runs while the schema/data migrations are still in progress.
func (g *StartupGate) Handler(c *gin.Context) {
	if g.ready.Load() == 1 {
		c.Next()
		return
	}
	c.Header("Retry-After", "5")
	c.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	c.Data(http.StatusServiceUnavailable, "text/html; charset=utf-8", startupHTML)
	c.Abort()
}
