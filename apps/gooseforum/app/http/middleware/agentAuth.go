package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
)

// AgentAuth authenticates an Agent through its opaque bearer token only.
// Cookies, human JWTs, session credentials, OAuth flows, and fallback
// credentials are never accepted here. Every failure (missing, malformed,
// unknown, wrong-hash, disabled, frozen, deleted, or non-bot credential)
// resolves to the same byte-identical 401 envelope with
// messageCode auth.required, so no failure reason, token, prefix, or hash
// is ever exposed.
func AgentAuth(c *gin.Context) {
	token := bearerToken(c)
	if token == "" {
		unauthorizedAgent(c)
		return
	}
	agent, _, err := agentservice.ResolveByToken(token)
	if err != nil || agent == nil {
		unauthorizedAgent(c)
		return
	}
	c.Set("userId", agent.UserId)
	c.Next()
}

// bearerToken extracts the opaque agent token from the Authorization header.
// The auth scheme is compared case-insensitively per RFC 9110 ("Bearer",
// "bearer", "BEARER" all match) and the credential is trimmed of surrounding
// whitespace; cookie and other header sources are ignored so a human session
// can never authenticate an Agent.
func bearerToken(c *gin.Context) string {
	header := c.GetHeader("Authorization")
	scheme, rest, found := strings.Cut(header, " ")
	if !found {
		return ""
	}
	if !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(rest)
}

func unauthorizedAgent(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, component.FailDataCode(component.MessageAuthRequired, nil))
}
