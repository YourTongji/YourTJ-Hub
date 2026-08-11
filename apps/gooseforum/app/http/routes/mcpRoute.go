package routes

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/http/middleware"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/mcpservice"
)

// mcpRoute mounts the MCP streamable HTTP endpoint at /mcp.
//
// It is intentionally a separate registration from apiRoute: several route
// tests call apiRoute(router) directly and must not instantiate the MCP
// server. The endpoint is always registered (gin's route tree is immutable at
// runtime) and the enabled flag is checked per request from the admin-panel
// MCP settings (hotdataserve 5s cache), so toggling mcp.enabled in the panel
// takes effect without restarting the process. When disabled the endpoint
// answers 404 and exposes no MCP surface; the write-tool opt-in (mcp.writes,
// default off) is read by the MCP server per session.
func mcpRoute(ginApp *gin.Engine) {
	svc := mcpservice.NewService()
	handler := svc.HTTPHandler()
	// Streamable HTTP uses POST for JSON-RPC and GET for the SSE stream; the
	// MCP endpoint is mounted outside any gzip group so streaming is not
	// buffered away. The gin layer resolves the client IP through the same
	// trusted-proxies rules as the REST stack and hands it to the MCP auth
	// verifier, so a raw X-Forwarded-For header on a direct connection cannot
	// spoof the IP dimension of the write rate limits.
	ginApp.Any("/mcp", middleware.RateLimit(middleware.RateLimitMCPAuth), func(c *gin.Context) {
		if !hotdataserve.GetMCPSettingsConfigCache().Enabled {
			c.Status(http.StatusNotFound)
			return
		}
		ctx := context.WithValue(c.Request.Context(), mcpservice.ClientIPContextKey{}, c.ClientIP())
		handler.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
	})
	slog.Info("mcp endpoint registered", "path", "/mcp")
}
