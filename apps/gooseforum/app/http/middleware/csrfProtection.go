package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/gin-gonic/gin"
)

// accessTokenCookieName mirrors the session cookie written by bundles/jwtopt
// (TokenSetting/TokenClean). Browsers authenticate the forum API with this
// HttpOnly cookie; non-browser clients authenticate with the Authorization
// header and never depend on the cookie (see docs/product/identity-and-access.md
// "认证与 CSRF 边界", issue #406).
const accessTokenCookieName = "access_token"

// CSRFProtection rejects cross-site state-changing requests that would
// otherwise be authenticated by the access_token cookie.
//
// Auth contract (issue #406):
//   - Bearer clients (mobile app, Agent agt_, MCP, curl/API clients) send an
//     Authorization header and are exempt even when a cookie is also present.
//   - Browser pages (site + admin SPA, GoHTML SSR) are same-origin with the
//     API in the single-binary deployment, so every fetch/POST carries the
//     page Origin (and usually a Referer). Dev mode proxies only /assets/* to
//     Vite; the page itself is still served by the Go backend, so the Origin
//     always equals the request Host (localhost, LAN IP, or public domain).
//   - GET/HEAD/OPTIONS are exempt (no state change; preflight support).
//   - Requests without an access_token cookie are exempt: there is no
//     cookie-authenticated state change to protect (anonymous writes, the
//     GitHub wiki webhook's HMAC auth, OIDC token endpoints).
//
// Enforcement for the remaining cookie-authenticated state changes: the
// Origin must match the site origin (derived from the request Host plus the
// X-Forwarded-Proto/TLS scheme so TLS-terminating proxies pass) or one of the
// explicitly configured origins in `csrf.allowedOrigins`. When the Origin
// header is missing (legacy browsers that omit Origin on same-origin POSTs),
// the Referer origin is checked instead. Any mismatch is rejected with 403
// (envelope messageCode `auth.csrf.rejected`).
//
// SameSite=Lax + HttpOnly + Secure cookies remain as defense in depth; the
// origin check additionally covers the cases SameSite=Lax does not: same-site
// cross-origin (subdomain) attacks and browsers without SameSite support.
//
// Mounted per write-route group (route4api.go) after the authentication
// middleware, never globally, so anonymous 401 semantics and public POSTs are
// untouched. Do not move this into bridge.go's global chain.
func CSRFProtection(c *gin.Context) {
	method := c.Request.Method
	if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
		c.Next()
		return
	}
	// Bearer-authenticated API clients are exempt: their credential cannot be
	// attached cross-site by a browser, so Origin checking would only add
	// friction (mobile, Agent, MCP, curl with an explicit token).
	if strings.TrimSpace(c.GetHeader("Authorization")) != "" {
		c.Next()
		return
	}
	// No session cookie -> nothing cookie-authenticated rides along. Public or
	// server-to-server writes (e.g. wiki webhook HMAC, OIDC token endpoint)
	// legitimately arrive without Origin and must keep working.
	cookie, err := c.Cookie(accessTokenCookieName)
	if err != nil || strings.TrimSpace(cookie) == "" {
		c.Next()
		return
	}
	if sameSiteRequest(c) {
		c.Next()
		return
	}
	c.JSON(http.StatusForbidden, component.FailDataCode(component.MessageAuthCsrfRejected, nil))
	c.Abort()
}

// sameSiteRequest reports whether the request comes from a site-controlled
// origin: the Origin header when present, otherwise the Referer origin.
func sameSiteRequest(c *gin.Context) bool {
	if origin := strings.TrimSpace(c.GetHeader("Origin")); origin != "" {
		return originMatches(c, origin)
	}
	// Referer fallback: only meaningful for browser requests. The header is
	// absent on curl/scripting clients, which then get rejected fail-closed.
	if referer := strings.TrimSpace(c.GetHeader("Referer")); referer != "" {
		if ref, err := url.Parse(referer); err == nil && ref.Host != "" {
			return originMatches(c, ref.Scheme+"://"+ref.Host)
		}
	}
	return false
}

// originMatches compares origin against the site-controlled origin set: the
// request's own Host-derived origin and the configured csrf.allowedOrigins.
func originMatches(c *gin.Context, origin string) bool {
	candidate, err := url.Parse(origin)
	if err != nil || candidate.Host == "" {
		return false
	}
	for _, allowed := range siteAllowedOrigins(c) {
		parsed, err := url.Parse(allowed)
		if err != nil || parsed.Host == "" {
			continue
		}
		if sameOrigin(candidate, parsed) {
			return true
		}
	}
	return false
}

// siteAllowedOrigins returns the origins this deployment accepts for
// cookie-authenticated writes. The request's own Host always wins: the browser
// Origin equals the URL the user visited, which is also the domain the
// host-only access_token cookie is scoped to. X-Forwarded-Proto is honored
// (first hop) so TLS-terminating reverse proxies (the supported deployment)
// produce https origins; widening the allowed set by scheme is safe because an
// attacker still cannot make its own Origin equal the victim host. Extra
// origins (e.g. an alternate front-end domain that must share the cookie
// session) are configured via `csrf.allowedOrigins` (comma separated).
func siteAllowedOrigins(c *gin.Context) []string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwardedProto != "" {
		if first, _, ok := strings.Cut(forwardedProto, ","); ok {
			forwardedProto = first
		}
		if proto := strings.ToLower(strings.TrimSpace(forwardedProto)); proto == "http" || proto == "https" {
			scheme = proto
		}
	}
	allowed := []string{scheme + "://" + c.Request.Host}
	for _, raw := range strings.Split(preferences.GetString("csrf.allowedOrigins", ""), ",") {
		if origin := strings.TrimSpace(raw); origin != "" {
			allowed = append(allowed, origin)
		}
	}
	return allowed
}

// sameOrigin compares scheme, hostname, and effective port (default ports are
// normalized: browsers omit :80/:443 from Origin while Host may carry them).
func sameOrigin(a, b *url.URL) bool {
	if !strings.EqualFold(a.Scheme, b.Scheme) {
		return false
	}
	if !strings.EqualFold(a.Hostname(), b.Hostname()) {
		return false
	}
	return effectivePort(a) == effectivePort(b)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	if u.Scheme == "https" {
		return "443"
	}
	return "80"
}
