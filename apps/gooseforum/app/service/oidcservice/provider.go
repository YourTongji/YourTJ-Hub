// Package oidcservice implements the forum built-in OIDC provider: a
// standards-compliant authorization code + PKCE (S256) endpoint suite serving
// the forum's own users/accounts as the single identity source for a small
// set of first-party clients (e.g. the course-selection site and the mobile
// app). The forum HS256 JWT is never issued to external clients; external
// clients use opaque access tokens and the userinfo endpoint.
package oidcservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/oidcprovider"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"
)

const (
	// browserBindingCookieName is the HttpOnly cookie binding the authorize
	// start to the callback completion on the same browser. The raw value is
	// never persisted; only its SHA-256 hash is stored with the auth request.
	browserBindingCookieName = "yourtj_oidc_binding"
)

// browserBindingCtxKey carries the SHA-256 hash (hex) of the browser-binding
// cookie value from the authorize handler into CreateAuthRequest.
type browserBindingCtxKey struct{}

var (
	// ErrOIDCDisabled is returned when the built-in provider is not enabled.
	ErrOIDCDisabled = errors.New("oidc: 内建 OIDC Provider 未启用")

	providerMu     sync.Mutex
	providerInst   *op.Provider
	providerCfgKey string
	providerErr    error
)

// configKey returns a stable fingerprint of the effective configuration so
// Provider() rebuilds when settings change (tests configure per-case).
// Secrets and inline private keys are hashed, never included in plaintext in
// the observable key string.
func configKey(cfg Config) string {
	keyPEMHash := ""
	if cfg.KeyPEM != "" {
		sum := sha256.Sum256([]byte(cfg.KeyPEM))
		keyPEMHash = hex.EncodeToString(sum[:8])
	}

	clients := ""
	for _, c := range cfg.Clients {
		secretHash := ""
		if c.Secret != "" {
			sum := sha256.Sum256([]byte(c.Secret))
			secretHash = hex.EncodeToString(sum[:8])
		}
		clients += strings.Join([]string{
			c.ID,
			c.Name,
			secretHash,
			fmt.Sprintf("%v", c.DevMode),
			strings.Join(c.RedirectURIs, ","),
		}, ":") + "|"
	}
	return fmt.Sprintf("%v|%s|%s|%s|%s|%s|%s|%s",
		cfg.Enabled, cfg.Issuer, cfg.KeyFile, keyPEMHash,
		cfg.AccessTokenTTL, cfg.AuthRequestTTL, cfg.IDTokenTTL, clients)
}

// InitOIDC validates and preloads the built-in OIDC provider configuration at
// startup. A missing or disabled configuration only disables the OIDC routes.
func InitOIDC() {
	cfg, err := LoadConfig()
	if err != nil {
		slog.Error("OIDC provider config invalid", "error", err)
		return
	}
	if !cfg.Enabled {
		slog.Info("built-in OIDC provider disabled (oidc.enabled=false)")
		return
	}
	if _, err := Provider(); err != nil {
		slog.Error("OIDC provider init failed", "error", err)
	}
}

// Provider returns the process-wide OIDC provider, rebuilding when the
// effective configuration changes.
func Provider() (*op.Provider, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	key := configKey(cfg)

	providerMu.Lock()
	defer providerMu.Unlock()
	if providerInst != nil && providerCfgKey == key {
		return providerInst, providerErr
	}
	providerInst, providerErr = buildProvider(cfg)
	providerCfgKey = key
	return providerInst, providerErr
}

func buildProvider(cfg Config) (*op.Provider, error) {
	if !cfg.Enabled {
		return nil, ErrOIDCDisabled
	}
	km, err := oidcprovider.Load()
	if err != nil {
		return nil, err
	}
	cryptoKey := deriveCryptoKey()
	st := newStorage(cfg, km)

	opts := []op.Option{}
	if strings.HasPrefix(cfg.Issuer, "http://") {
		// local development (http://localhost:port) requires an insecure issuer
		opts = append(opts, op.WithAllowInsecure())
	}
	p, err := op.NewProvider(
		&op.Config{
			CryptoKey:      cryptoKey,
			CryptoKeyId:    "oidc-opaque-token",
			CodeMethodS256: true,
			SupportedScopes: []string{
				oidc.ScopeOpenID,
				oidc.ScopeProfile,
				oidc.ScopeEmail,
			},
		},
		st,
		op.StaticIssuer(cfg.Issuer),
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// DiscoveryConfig returns the advertised provider metadata. Only the features
// actually implemented are declared: authorization code, PKCE S256, RS256,
// client_secret_basic / none.
func DiscoveryConfig() (*oidc.DiscoveryConfiguration, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return nil, ErrOIDCDisabled
	}
	return &oidc.DiscoveryConfiguration{
		Issuer:                            cfg.Issuer,
		AuthorizationEndpoint:             cfg.Issuer + "/authorize",
		TokenEndpoint:                     cfg.Issuer + "/token",
		UserinfoEndpoint:                  cfg.Issuer + "/userinfo",
		JwksURI:                           cfg.Issuer + "/keys",
		ScopesSupported:                   []string{oidc.ScopeOpenID, oidc.ScopeProfile, oidc.ScopeEmail},
		ResponseTypesSupported:            []string{string(oidc.ResponseTypeCode)},
		ResponseModesSupported:            []string{"query"},
		GrantTypesSupported:               []oidc.GrantType{oidc.GrantTypeCode},
		SubjectTypesSupported:             []string{"public"},
		IDTokenSigningAlgValuesSupported:  []string{string(jose.RS256)},
		TokenEndpointAuthMethodsSupported: []oidc.AuthMethod{oidc.AuthMethodNone, oidc.AuthMethodBasic},
		CodeChallengeMethodsSupported:     []oidc.CodeChallengeMethod{oidc.CodeChallengeMethodS256},
		ClaimsSupported: []string{
			"sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "azp",
			"preferred_username", "name", "email", "email_verified",
		},
	}, nil
}

// Router returns an http.Handler mounting only the implemented OIDC endpoints
// under the issuer path. When the provider is disabled it returns nil and the
// caller must not register any route.
func Router() (http.Handler, error) {
	provider, err := Provider()
	if err != nil {
		return nil, err
	}
	// 库的 CreateRouter 通过 issuer interceptor 把 issuer 注入 context
	// （CreateIDToken / discovery 依赖 IssuerFromContext）；手动挂载时
	// 需要同样注入，否则 id_token 的 iss claim 为空。
	withIssuer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := op.ContextWithIssuer(r.Context(), provider.IssuerFromRequest(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	mux := http.NewServeMux()
	// discovery is served by our own config (not the library default) so the
	// advertised surface matches exactly what is implemented. Endpoints are
	// registered with the full issuer path so the handler is self-contained.
	mux.Handle(issuerPath+"/.well-known/openid-configuration", withIssuer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg, err := DiscoveryConfig()
		if err != nil {
			http.Error(w, "oidc disabled", http.StatusNotFound)
			return
		}
		op.Discover(w, cfg)
	})))
	mux.Handle(issuerPath+"/authorize", withIssuer(withBrowserBinding(provider, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op.Authorize(w, r, provider)
	}))))
	mux.Handle(issuerPath+"/authorize/callback", withIssuer(authorizeCallbackBridge(provider)))
	mux.Handle(issuerPath+"/token", withIssuer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		op.Exchange(w, r, provider)
	})))
	mux.Handle(issuerPath+"/userinfo", withIssuer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op.Userinfo(w, r, provider)
	})))
	mux.Handle(issuerPath+"/keys", withIssuer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		op.Keys(w, r, provider.Storage())
	})))
	return mux, nil
}

// withBrowserBinding ensures a browser-binding cookie exists before the
// authorize handler runs, and passes its SHA-256 hash to CreateAuthRequest
// via the request context. The cookie is HttpOnly + SameSite=Lax, scoped to
// the issuer path, and Secure when the issuer is https. The raw cookie value
// is a random per-browser token, never the client-controlled request id.
func withBrowserBinding(provider *op.Provider, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(browserBindingCookieName)
		if err != nil || cookie.Value == "" || len(cookie.Value) > 128 {
			value, err := randomHex(32)
			if err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			issuer := provider.IssuerFromRequest(r)
			secure := strings.HasPrefix(issuer, "https://")
			// The cookie must stay valid at least as long as the pending
			// authorization request it binds, so its lifetime follows the
			// configured auth request TTL of this provider's storage.
			ttl := defaultAuthRequestTTL
			if st, ok := provider.Storage().(*storage); ok {
				ttl = st.browserBindingTTL()
			}
			http.SetCookie(w, &http.Cookie{
				Name:     browserBindingCookieName,
				Value:    value,
				Path:     issuerPath,
				MaxAge:   int(ttl.Seconds()),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				Secure:   secure,
			})
			sum := sha256.Sum256([]byte(value))
			ctx := contextWithBrowserBinding(r.Context(), hex.EncodeToString(sum[:]))
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		sum := sha256.Sum256([]byte(cookie.Value))
		ctx := contextWithBrowserBinding(r.Context(), hex.EncodeToString(sum[:]))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// contextWithBrowserBinding returns a context carrying the binding hash.
func contextWithBrowserBinding(ctx context.Context, hash string) context.Context {
	return context.WithValue(ctx, browserBindingCtxKey{}, hash)
}

// browserBindingHashFromContext reads the binding hash set by
// withBrowserBinding. Empty means no binding was established.
func browserBindingHashFromContext(ctx context.Context) string {
	hash, _ := ctx.Value(browserBindingCtxKey{}).(string)
	return hash
}

// browserBindingHashOf returns the SHA-256 hex hash of a cookie value.
func browserBindingHashOf(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// authorizeCallbackBridge completes the login bridge: it requires an
// authenticated forum session and a browser-binding cookie matching the
// authorize start, marks the in-flight authorization request as done for
// that user, then hands over to the library callback handler which mints the
// single-use authorization code. Unauthenticated users are sent through the
// forum login page and return to this callback afterwards. The callback
// target is built from the server-side request ID only, so no user-controlled
// redirect target is ever followed.
func authorizeCallbackBridge(provider *op.Provider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		userID, err := AuthenticateRequest(r)
		if err != nil {
			callback := "/api/oauth/authorize/callback?id=" + url.QueryEscape(id)
			http.Redirect(w, r, "/login?redirect="+url.QueryEscape(callback), http.StatusFound)
			return
		}
		if id != "" {
			// Browser binding: the callback must carry the same cookie that
			// started the authorize request. Missing/mismatched binding is a
			// hard 400; the request is never completed and no client redirect
			// happens.
			cookie, err := r.Cookie(browserBindingCookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, "browser binding missing", http.StatusBadRequest)
				return
			}
			bindingHash := browserBindingHashOf(cookie.Value)
			if err := CompleteLogin(id, userID, time.Now(), bindingHash); err != nil {
				// Binding mismatch, already completed, expired or unknown; the
				// request must not be completed for this browser.
				slog.Warn("oidc login bridge rejected", "error", err)
				http.Error(w, "login bridge rejected", http.StatusBadRequest)
				return
			}
		}
		op.AuthorizeCallback(w, r, provider)
	}
}

// fallbackCryptoKey is the process-sticky random fallback used by
// deriveCryptoKey when app.signingKey is unset. It is generated once per
// process from secure randomness and never persisted, so provider rebuilds
// do not re-randomize the opaque-token encryption key mid-process. A process
// restart rotates it, which is acceptable because the missing signing key is
// a misconfiguration and the encrypted tokens are short-lived.
var (
	fallbackCryptoKeyMu  sync.Mutex
	fallbackCryptoKey    [32]byte
	fallbackCryptoKeySet bool
)

// deriveCryptoKey derives the opaque-access-token encryption key from the
// forum signing key using a domain-separated SHA-256. It never reuses
// app.signingKey directly. When no signing key is configured, a secure
// random key is generated once and kept for the rest of the process.
//
// The random fallback is defensive only: serve already refuses to boot on an
// empty/weak app.signingKey (issue #106), so this branch is reachable solely
// from tests or a non-serve entrypoint and does not create a forgeable-token
// surface in a runnable deploy.
func deriveCryptoKey() [32]byte {
	signingKey := preferences.GetString("app.signingKey", "")
	if signingKey != "" {
		return sha256.Sum256([]byte("yourtj-oidc-opaque-token|" + signingKey))
	}
	fallbackCryptoKeyMu.Lock()
	defer fallbackCryptoKeyMu.Unlock()
	if !fallbackCryptoKeySet {
		b, err := algorithm.GenerateRandomBytes(32)
		if err != nil {
			panic(fmt.Sprintf("oidcservice: derive crypto key: %v", err))
		}
		copy(fallbackCryptoKey[:], b)
		fallbackCryptoKeySet = true
	}
	return fallbackCryptoKey
}

func randomHex(size int) (string, error) {
	b, err := algorithm.GenerateRandomBytes(size)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// now is a small indirection for deterministic tests.
var now = time.Now
