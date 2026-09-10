package oidcservice

import (
	"net/url"
	"strings"
	"time"

	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

// client implements op.Client for a configured first-party OIDC client.
// It only ever supports the authorization code flow with PKCE.
type client struct {
	cfg ClientConfig
	ttl time.Duration
}

// GetID returns the client identifier.
func (c *client) GetID() string { return c.cfg.ID }

// RedirectURIs returns the exact configured redirect URIs. The library
// performs exact matching first; clients that configure RedirectURIGlobs get
// doublestar fallback matching (see HasRedirectGlobs below). Both paths only
// ever match against registered values, so open redirects stay impossible.
func (c *client) RedirectURIs() []string { return c.cfg.RedirectURIs }

// RedirectURIGlobs returns the configured doublestar patterns, enabling
// zitadel/oidc's op.HasRedirectGlobs fallback: after exact matching fails,
// each pattern is matched with doublestar.Match. Patterns are validated at
// config load time (fail-closed), so a bad pattern never reaches runtime.
func (c *client) RedirectURIGlobs() []string { return c.cfg.RedirectURIGlobs }

// PostLogoutRedirectURIGlobs is not supported (no RP-initiated logout).
func (c *client) PostLogoutRedirectURIGlobs() []string { return nil }

// PostLogoutRedirectURIs is not supported (no RP-initiated logout).
func (c *client) PostLogoutRedirectURIs() []string { return nil }

// ApplicationType reports web for https clients and native for custom-scheme
// clients (e.g. yourtj://callback on mobile). Native clients are required for
// custom-scheme redirect URIs by the OIDC spec.
func (c *client) ApplicationType() op.ApplicationType {
	for _, uri := range c.cfg.RedirectURIs {
		if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
			return op.ApplicationTypeNative
		}
	}
	return op.ApplicationTypeWeb
}

// AuthMethod is `none` for public clients (PKCE only) and
// client_secret_basic for confidential clients.
func (c *client) AuthMethod() oidc.AuthMethod {
	if c.cfg.Secret == "" {
		return oidc.AuthMethodNone
	}
	return oidc.AuthMethodBasic
}

// ResponseTypes only allows the authorization code flow.
func (c *client) ResponseTypes() []oidc.ResponseType {
	return []oidc.ResponseType{oidc.ResponseTypeCode}
}

// GrantTypes only allows authorization_code.
func (c *client) GrantTypes() []oidc.GrantType {
	return []oidc.GrantType{oidc.GrantTypeCode}
}

// LoginURL redirects the end user to the forum login page, preserving the
// server-side authorization request ID in the redirect target so the browser
// returns to /api/oauth/authorize/callback?id=<requestID> after login. The
// callback is a same-site relative path: the login page only accepts
// redirect values starting with "/", which also rules out open redirects.
func (c *client) LoginURL(authReqID string) string {
	callback := "/api/oauth/authorize/callback?id=" + url.QueryEscape(authReqID)
	return "/login?redirect=" + url.QueryEscape(callback)
}

// AccessTokenType issues opaque bearer tokens (never the forum JWT).
func (c *client) AccessTokenType() op.AccessTokenType {
	return op.AccessTokenTypeBearer
}

// IDTokenLifetime returns the configured ID token lifetime.
func (c *client) IDTokenLifetime() time.Duration { return c.ttl }

// DevMode relaxes http redirect rules for local development clients.
func (c *client) DevMode() bool { return c.cfg.DevMode }

// RestrictAdditionalIdTokenScopes keeps scopes unchanged (we only allow
// openid/profile/email which are checked in IsScopeAllowed).
func (c *client) RestrictAdditionalIdTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string { return scopes }
}

// RestrictAdditionalAccessTokenScopes keeps scopes unchanged.
func (c *client) RestrictAdditionalAccessTokenScopes() func(scopes []string) []string {
	return func(scopes []string) []string { return scopes }
}

// IsScopeAllowed accepts only the three standard OIDC scopes we implement.
func (c *client) IsScopeAllowed(scope string) bool {
	switch scope {
	case oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail:
		return true
	}
	return false
}

// IDTokenUserinfoClaimsAssertion keeps profile/email claims out of the ID
// token; clients must call userinfo for those.
func (c *client) IDTokenUserinfoClaimsAssertion() bool { return false }

// ClockSkew is zero: tokens are validated with the server clock only.
func (c *client) ClockSkew() time.Duration { return 0 }
