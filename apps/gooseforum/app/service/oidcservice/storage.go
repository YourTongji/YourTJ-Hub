package oidcservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/oidcprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAccessTokens"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/oidcAuthRequests"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	jose "github.com/go-jose/go-jose/v4"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

var (
	errClientNotFound = errors.New("oidc: client not found")
)

// storage implements op.Storage backed by GORM rows. It deliberately does not
// implement ClientCredentialsStorage, DeviceAuthorizationStorage or
// TokenExchangeStorage so those grant types stay unsupported.
type storage struct {
	cfg     Config
	km      *oidcprovider.KeyManager
	clients map[string]*client
}

func newStorage(cfg Config, km *oidcprovider.KeyManager) *storage {
	clients := make(map[string]*client, len(cfg.Clients))
	for i := range cfg.Clients {
		c := &client{cfg: cfg.Clients[i], ttl: cfg.IDTokenTTL}
		clients[c.GetID()] = c
	}
	return &storage{cfg: cfg, km: km, clients: clients}
}

// browserBindingTTL returns the lifetime of a pending authorization request
// for this storage instance. It bounds the browser-binding cookie set on
// /authorize so the cookie never outlives the request it binds.
func (s *storage) browserBindingTTL() time.Duration {
	if s.cfg.AuthRequestTTL <= 0 {
		return defaultAuthRequestTTL
	}
	return s.cfg.AuthRequestTTL
}

// --- op.OPStorage ---

// GetClientByClientID loads a configured client. Unknown clients are rejected;
// the caller (authorize/token) maps the error to the standard OIDC error.
func (s *storage) GetClientByClientID(_ context.Context, clientID string) (op.Client, error) {
	if c, ok := s.clients[clientID]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("%w: %s", errClientNotFound, clientID)
}

// AuthorizeClientIDSecret validates a confidential client's secret with a
// constant-time comparison. The secret is never logged.
func (s *storage) AuthorizeClientIDSecret(_ context.Context, clientID, clientSecret string) error {
	c, ok := s.clients[clientID]
	if !ok {
		return errClientNotFound
	}
	if c.cfg.Secret == "" {
		return errors.New("oidc: public client must not present a secret")
	}
	if subtle.ConstantTimeCompare([]byte(c.cfg.Secret), []byte(clientSecret)) != 1 {
		return errors.New("oidc: invalid client secret")
	}
	return nil
}

// SetUserinfoFromScopes is deprecated by the library and intentionally empty;
// ID token userinfo is filled by SetUserinfoFromRequest below.
func (s *storage) SetUserinfoFromScopes(_ context.Context, _ *oidc.UserInfo, _, _ string, _ []string) error {
	return nil
}

// SetUserinfoFromRequest fills userinfo from the user record based on the
// requested scopes (used when minting ID tokens). The library calls this
// optional interface after SetUserinfoFromScopes, which is deprecated.
func (s *storage) SetUserinfoFromRequest(_ context.Context, userinfo *oidc.UserInfo, request op.IDTokenRequest, scopes []string) error {
	sub, err := strconv.ParseUint(request.GetSubject(), 10, 64)
	if err != nil || sub == 0 {
		return oidc.ErrInvalidRequest().WithDescription("invalid subject")
	}
	user, err := users.Get(sub)
	if err != nil || user.Id == 0 {
		return oidc.ErrInvalidRequest().WithDescription("user not found")
	}
	fillUserinfo(userinfo, request.GetSubject(), user, scopes)
	return nil
}

// SetUserinfoFromToken fills userinfo for the userinfo endpoint after
// validating the opaque access token. TokenVersion changes (password change /
// revoke-all), frozen accounts and expiry are enforced here.
func (s *storage) SetUserinfoFromToken(_ context.Context, userinfo *oidc.UserInfo, tokenID, subject, _ string) error {
	row := s.accessTokenByID(tokenID)
	if row == nil || row.Revoked || row.ExpiresAt.Before(now()) {
		return oidc.ErrInvalidGrant().WithDescription("access token invalid or expired")
	}
	user, err := users.Get(row.Subject)
	if err != nil || user.Id == 0 {
		return oidc.ErrInvalidGrant().WithDescription("user not found")
	}
	if user.IsFrozen == users.StatusFrozen {
		return oidc.ErrInvalidGrant().WithDescription("user frozen")
	}
	if user.TokenVersion != row.TokenVersion {
		return oidc.ErrInvalidGrant().WithDescription("token version mismatch")
	}
	fillUserinfo(userinfo, strconv.FormatUint(row.Subject, 10), user, strings.Fields(row.Scopes))
	return nil
}

// accessTokenByID loads an access token row by its token ID.
func (s *storage) accessTokenByID(tokenID string) *oidcAccessTokens.Entity {
	return oidcAccessTokens.GetByTokenId(tokenID)
}

// SetIntrospectionFromToken is never used: the introspection endpoint is not
// mounted and not advertised.
func (s *storage) SetIntrospectionFromToken(_ context.Context, _ *oidc.IntrospectionResponse, _, _, _ string) error {
	return nil
}

// GetPrivateClaimsFromScopes returns no private claims.
func (s *storage) GetPrivateClaimsFromScopes(_ context.Context, _, _ string, _ []string) (map[string]any, error) {
	return nil, nil
}

// GetKeyByIDAndClientID is only used by the (unsupported) JWT profile grant.
func (s *storage) GetKeyByIDAndClientID(_ context.Context, _, _ string) (*jose.JSONWebKey, error) {
	return nil, oidc.ErrInvalidRequest().WithDescription("private_key_jwt not supported")
}

// ValidateJWTProfileScopes is only used by the (unsupported) JWT profile grant.
func (s *storage) ValidateJWTProfileScopes(_ context.Context, _ string, scopes []string) ([]string, error) {
	return scopes, nil
}

// --- op.AuthStorage ---

// CreateAuthRequest persists a validated authorization request. It enforces
// state/nonce/PKCE S256 and rejects unsupported prompts and scopes. The
// SHA-256 hash of the browser-binding cookie (from the request context) is
// persisted so the login bridge can verify the callback comes from the same
// browser that started the request.
func (s *storage) CreateAuthRequest(ctx context.Context, authReq *oidc.AuthRequest, _ string) (op.AuthRequest, error) {
	if authReq.State == "" {
		return nil, oidc.ErrInvalidRequest().WithDescription("state is required")
	}
	if authReq.Nonce == "" {
		return nil, oidc.ErrInvalidRequest().WithDescription("nonce is required")
	}
	if authReq.CodeChallenge == "" || authReq.CodeChallengeMethod != oidc.CodeChallengeMethodS256 {
		return nil, oidc.ErrInvalidRequest().WithDescription("PKCE S256 is required")
	}
	if slices.Contains(authReq.Scopes, oidc.ScopeOfflineAccess) {
		return nil, oidc.ErrInvalidScope().WithDescription("offline_access not supported")
	}
	if len(authReq.Prompt) > 0 {
		if slices.Contains(authReq.Prompt, oidc.PromptNone) {
			return nil, oidc.ErrLoginRequired().WithDescription("prompt=none not supported")
		}
		return nil, oidc.ErrInvalidRequest().WithDescription("prompt not supported")
	}
	if authReq.ResponseType != oidc.ResponseTypeCode {
		return nil, oidc.ErrInvalidRequest().WithDescription("only response_type=code is supported")
	}

	requestID, err := randomHex(16)
	if err != nil {
		return nil, oidc.ErrServerError().WithParent(err)
	}
	entity := &oidcAuthRequests.Entity{
		RequestId:      requestID,
		ClientId:       authReq.ClientID,
		Scopes:         strings.Join(authReq.Scopes, " "),
		RedirectUri:    authReq.RedirectURI,
		State:          authReq.State,
		Nonce:          authReq.Nonce,
		ResponseType:   string(authReq.ResponseType),
		CodeChallenge:  authReq.CodeChallenge,
		BrowserBinding: browserBindingHashFromContext(ctx),
		ExpiresAt:      now().Add(s.cfg.AuthRequestTTL),
	}
	// Opportunistic housekeeping: expired rows are otherwise only removed at
	// startup/daily cron; cleaning here is cheap (indexed) and owner-level.
	oidcAuthRequests.DeleteExpired(now())
	if err := oidcAuthRequests.Create(entity); err != nil {
		return nil, oidc.ErrServerError().WithParent(err)
	}
	return &authRequest{entity: entity}, nil
}

// AuthRequestByID loads an auth request by its request ID (login bridge key).
// Expired requests are rejected.
func (s *storage) AuthRequestByID(_ context.Context, id string) (op.AuthRequest, error) {
	entity := oidcAuthRequests.GetByRequestId(id)
	if entity == nil {
		return nil, oidc.ErrInvalidRequest().WithDescription("auth request not found")
	}
	if entity.ExpiresAt.Before(now()) {
		return nil, oidc.ErrInvalidRequest().WithDescription("auth request expired")
	}
	return &authRequest{entity: entity}, nil
}

// AuthRequestByCode consumes an authorization code atomically: the row is
// marked used in a conditional update so concurrent redemption of the same
// code only succeeds once. PKCE/verifier validation happens after this call
// in the library flow; a failed verification therefore also invalidates the
// code, which is safe per RFC 6749 §4.1.2.
func (s *storage) AuthRequestByCode(_ context.Context, code string) (op.AuthRequest, error) {
	entity := oidcAuthRequests.GetByAuthCode(code)
	if entity == nil || !entity.Done || entity.Subject == 0 {
		return nil, oidc.ErrInvalidGrant().WithDescription("invalid code")
	}
	if entity.ExpiresAt.Before(now()) {
		return nil, oidc.ErrInvalidGrant().WithDescription("code expired")
	}
	if affected := oidcAuthRequests.MarkUsed(entity.Id); affected != 1 {
		return nil, oidc.ErrInvalidGrant().WithDescription("code already used")
	}
	return &authRequest{entity: entity}, nil
}

// SaveAuthCode stores the encrypted authorization code on the auth request.
func (s *storage) SaveAuthCode(_ context.Context, authRequestID, code string) error {
	if err := oidcAuthRequests.UpdateAuthCode(authRequestID, code); err != nil {
		return oidc.ErrServerError().WithParent(err)
	}
	return nil
}

// DeleteAuthRequest removes the auth request after a successful token
// exchange (called by the library).
func (s *storage) DeleteAuthRequest(_ context.Context, authRequestID string) error {
	return oidcAuthRequests.DeleteByRequestId(authRequestID)
}

// CreateAccessToken persists an opaque access token row. Only the token ID
// and metadata are stored; the raw bearer token is never persisted.
func (s *storage) CreateAccessToken(_ context.Context, request op.TokenRequest) (string, time.Time, error) {
	sub, err := strconv.ParseUint(request.GetSubject(), 10, 64)
	if err != nil || sub == 0 {
		return "", time.Time{}, oidc.ErrInvalidRequest().WithDescription("invalid subject")
	}
	user, err := users.Get(sub)
	if err != nil || user.Id == 0 {
		return "", time.Time{}, oidc.ErrInvalidRequest().WithDescription("user not found")
	}
	if user.IsFrozen == users.StatusFrozen {
		return "", time.Time{}, oidc.ErrInvalidRequest().WithDescription("user frozen")
	}
	tokenID, err := randomHex(16)
	if err != nil {
		return "", time.Time{}, oidc.ErrServerError().WithParent(err)
	}
	clientID := ""
	if authReq, ok := request.(op.AuthRequest); ok {
		clientID = authReq.GetClientID()
	}
	expiresAt := now().Add(s.cfg.AccessTokenTTL)
	row := &oidcAccessTokens.Entity{
		TokenId:      tokenID,
		Subject:      sub,
		ClientId:     clientID,
		Scopes:       strings.Join(request.GetScopes(), " "),
		TokenVersion: user.TokenVersion,
		ExpiresAt:    expiresAt,
	}
	if err := oidcAccessTokens.Create(row); err != nil {
		return "", time.Time{}, oidc.ErrServerError().WithParent(err)
	}
	return tokenID, expiresAt, nil
}

// CreateAccessAndRefreshTokens is never called: refresh tokens are not
// supported and the grant type is not advertised.
func (s *storage) CreateAccessAndRefreshTokens(context.Context, op.TokenRequest, string) (string, string, time.Time, error) {
	return "", "", time.Time{}, oidc.ErrInvalidGrant().WithDescription("refresh tokens not supported")
}

// TokenRequestByRefreshToken is never called: refresh tokens are not supported.
func (s *storage) TokenRequestByRefreshToken(context.Context, string) (op.RefreshTokenRequest, error) {
	return nil, oidc.ErrInvalidGrant().WithDescription("refresh tokens not supported")
}

// TerminateSession is never called: no end-session endpoint is mounted.
func (s *storage) TerminateSession(context.Context, string, string) error {
	return nil
}

// RevokeToken is never called: no revocation endpoint is mounted.
func (s *storage) RevokeToken(context.Context, string, string, string) *oidc.Error {
	return oidc.ErrInvalidRequest()
}

// GetRefreshTokenInfo is never called: refresh tokens are not supported.
func (s *storage) GetRefreshTokenInfo(context.Context, string, string) (string, string, error) {
	return "", "", oidc.ErrInvalidGrant().WithDescription("refresh tokens not supported")
}

// --- op.Storage ---

// Health reports the storage is ready.
func (s *storage) Health(context.Context) error {
	return nil
}

// SigningKey returns the persistent RS256 signing key.
func (s *storage) SigningKey(context.Context) (op.SigningKey, error) {
	return signingKeyAdapter{km: s.km}, nil
}

// SignatureAlgorithms returns the supported ID token algorithms.
func (s *storage) SignatureAlgorithms(context.Context) ([]jose.SignatureAlgorithm, error) {
	return []jose.SignatureAlgorithm{jose.RS256}, nil
}

// KeySet returns the public key set for JWKS.
func (s *storage) KeySet(context.Context) ([]op.Key, error) {
	return []op.Key{publicKeyAdapter{km: s.km}}, nil
}

// --- adapters ---

type signingKeyAdapter struct{ km *oidcprovider.KeyManager }

func (k signingKeyAdapter) SignatureAlgorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (k signingKeyAdapter) Key() any                                    { return k.km.PrivateKey() }
func (k signingKeyAdapter) ID() string                                  { return k.km.KeyID() }

type publicKeyAdapter struct{ km *oidcprovider.KeyManager }

func (k publicKeyAdapter) ID() string                         { return k.km.KeyID() }
func (k publicKeyAdapter) Algorithm() jose.SignatureAlgorithm { return jose.RS256 }
func (k publicKeyAdapter) Use() string                        { return "sig" }
func (k publicKeyAdapter) Key() any                           { return k.km.PublicKey() }

// fillUserinfo copies user fields according to the granted scopes.
func fillUserinfo(userinfo *oidc.UserInfo, subject string, user users.EntityComplete, scopes []string) {
	userinfo.Subject = subject
	for _, scope := range scopes {
		switch scope {
		case oidc.ScopeProfile:
			userinfo.PreferredUsername = user.Username
			userinfo.Name = user.Nickname
			if userinfo.Name == "" {
				userinfo.Name = user.Username
			}
		case oidc.ScopeEmail:
			userinfo.Email = user.Email
			userinfo.EmailVerified = oidc.Bool(user.IsActivated == users.ActivationSuccess)
		}
	}
}
