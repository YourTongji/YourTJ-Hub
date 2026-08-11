package routes

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/service/mcpservice"
)

// mcpRoute mounts the MCP streamable HTTP endpoint at /mcp.
//
// It is intentionally a separate registration from apiRoute: several route
// tests call apiRoute(router) directly and must not instantiate the MCP
// server. The endpoint honors the mcp.enabled preference (default off) so an
// operator opts in explicitly by setting [mcp] enabled = true; the write tool
// opt-in is mcp.writes (default off), read by the MCP server per session.
func mcpRoute(ginApp *gin.Engine) {
	if !preferences.GetBool("mcp.enabled", false) {
		slog.Info("mcp endpoint disabled by preference mcp.enabled")
		return
	}
	svc := mcpservice.NewService()
	handler := svc.HTTPHandler()
	// Streamable HTTP uses POST for JSON-RPC and GET for the SSE stream; the
	// MCP endpoint is mounted outside any gzip group so streaming is not
	// buffered away. The gin layer resolves the client IP through the same
	// trusted-proxies rules as the REST stack and hands it to the MCP auth
	// verifier, so a raw X-Forwarded-For header on a direct connection cannot
	// spoof the IP dimension of the write rate limits.
	ginApp.Any("/mcp", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), mcpservice.ClientIPContextKey{}, c.ClientIP())
		handler.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
	})
	slog.Info("mcp endpoint mounted", "path", "/mcp")
}
