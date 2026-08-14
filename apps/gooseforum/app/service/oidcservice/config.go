package oidcservice

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

const (
	// issuerPath is the fixed mount path of the built-in OIDC provider.
	issuerPath = "/api/oauth"

	defaultAccessTokenTTL = 3600 * time.Second
	defaultAuthRequestTTL = 10 * time.Minute
	defaultIDTokenTTL     = 3600 * time.Second
	defaultKeyFile        = "./storage/oidc/signing_key.pem"
)

// ClientConfig is one configured OIDC client (first-party services only).
// Empty Secret means a public client using PKCE with auth method `none`;
// a non-empty Secret means a confidential client using client_secret_basic.
type ClientConfig struct {
	ID               string
	Name             string
	Secret           string
	RedirectURIs     []string
	RedirectURIGlobs []string
	DevMode          bool
}

// Config is the effective provider configuration.
type Config struct {
	Enabled        bool
	Issuer         string
	KeyFile        string
	KeyPEM         string
	AccessTokenTTL time.Duration
	AuthRequestTTL time.Duration
	IDTokenTTL     time.Duration
	Clients        []ClientConfig
}

// LoadConfig reads and validates the oidc.* settings. Validation fails
// closed: an invalid issuer (http on non-loopback, wrong path, query or
// fragment) disables the provider entirely.
func LoadConfig() (Config, error) {
	cfg := Config{
		Enabled:        preferences.GetBool("oidc.enabled", false),
		KeyFile:        preferences.GetString("oidc.signing_key_file", defaultKeyFile),
		KeyPEM:         preferences.GetString("oidc.signing_key", ""),
		AccessTokenTTL: time.Duration(preferences.GetInt64("oidc.access_token_ttl", int64(defaultAccessTokenTTL/time.Second))) * time.Second,
		AuthRequestTTL: time.Duration(preferences.GetInt64("oidc.auth_request_ttl", int64(defaultAuthRequestTTL/time.Second))) * time.Second,
		IDTokenTTL:     time.Duration(preferences.GetInt64("oidc.id_token_ttl", int64(defaultIDTokenTTL/time.Second))) * time.Second,
	}
	if cfg.AccessTokenTTL <= 0 {
		cfg.AccessTokenTTL = defaultAccessTokenTTL
	}
	if cfg.AuthRequestTTL <= 0 {
		cfg.AuthRequestTTL = defaultAuthRequestTTL
	}
	if cfg.IDTokenTTL <= 0 {
		cfg.IDTokenTTL = defaultIDTokenTTL
	}
	cfg.Issuer = strings.TrimSpace(preferences.GetString("oidc.issuer", ""))
	if cfg.Issuer == "" {
		siteURL := strings.TrimSpace(hotdataserve.GetSiteSettingsConfigCache().SiteUrl)
		if siteURL == "" {
			siteURL = preferences.GetString("server.url", "http://localhost")
		}
		siteURL = addLoopbackPort(siteURL, preferences.GetInt("server.port", 5234))
		cfg.Issuer = strings.TrimSuffix(siteURL, "/") + issuerPath
	}
	if err := validateIssuer(cfg.Issuer); err != nil {
		return cfg, err
	}

	clients, err := loadClients()
	if err != nil {
		return cfg, err
	}
	cfg.Clients = clients
	return cfg, nil
}

func addLoopbackPort(rawURL string, port int) string {
	if port <= 0 {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" || u.Port() != "" {
		return rawURL
	}
	host := u.Hostname()
	if host != "localhost" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return rawURL
		}
	}
	u.Host = net.JoinHostPort(host, strconv.Itoa(port))
	return u.String()
}

// validateIssuer enforces the issuer shape:
//   - path must be exactly /api/oauth (no query, no fragment);
//   - http is only allowed for loopback hosts (localhost / 127.0.0.1 / ::1);
//   - anything else must be https.
func validateIssuer(issuer string) error {
	u, err := url.Parse(issuer)
	if err != nil || u.Host == "" {
		return fmt.Errorf("oidc: issuer 无效: %q", issuer)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("oidc: issuer 不允许 query/fragment: %q", issuer)
	}
	if u.Path != issuerPath {
		return fmt.Errorf("oidc: issuer path 必须为 %q，实际 %q", issuerPath, u.Path)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		host := u.Hostname()
		if host == "localhost" {
			return nil
		}
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			return nil
		}
		return fmt.Errorf("oidc: http issuer 仅允许 loopback（localhost/127.0.0.1/::1）: %q", issuer)
	default:
		return fmt.Errorf("oidc: issuer scheme 必须为 https 或 loopback http: %q", issuer)
	}
}

// loadClients parses the [[oidc.clients]] table from config.toml.
func loadClients() ([]ClientConfig, error) {
	raw := preferences.GetRaw("oidc.clients")
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		// viper sometimes yields []map[string]any directly for TOML arrays.
		if maps, ok := raw.([]map[string]any); ok {
			items = make([]any, len(maps))
			for i, m := range maps {
				items[i] = m
			}
		} else {
			return nil, fmt.Errorf("oidc: oidc.clients 配置格式错误")
		}
	}

	clients := make([]ClientConfig, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("oidc: oidc.clients[%d] 配置格式错误", i)
		}
		cfg := ClientConfig{
			ID:      stringValue(m["id"]),
			Name:    stringValue(m["name"]),
			Secret:  stringValue(m["secret"]),
			DevMode: boolValue(m["dev_mode"]),
		}
		if redirects, ok := m["redirect_uris"].([]any); ok {
			for _, r := range redirects {
				if s, ok := r.(string); ok && s != "" {
					cfg.RedirectURIs = append(cfg.RedirectURIs, s)
				}
			}
		}
		// redirect_uris_globs: doublestar 通配模式。zitadel/oidc 库在精确匹配
		// 失败后按 glob 匹配; pattern 在加载时校验, 非法即拒绝整个配置
		// (fail-closed), 避免把坏 pattern 留给运行时。
		if globs, ok := m["redirect_uris_globs"].([]any); ok {
			for _, g := range globs {
				if s, ok := g.(string); ok && s != "" {
					if !doublestar.ValidatePattern(s) {
						return nil, fmt.Errorf("oidc: oidc.clients[%d] redirect_uris_globs 无效 pattern: %q", i, s)
					}
					cfg.RedirectURIGlobs = append(cfg.RedirectURIGlobs, s)
				}
			}
		}
		if cfg.ID == "" || (len(cfg.RedirectURIs) == 0 && len(cfg.RedirectURIGlobs) == 0) {
			return nil, fmt.Errorf("oidc: oidc.clients[%d] 缺少 id 或 redirect_uris/redirect_uris_globs", i)
		}
		clients = append(clients, cfg)
	}
	return clients, nil
}

func stringValue(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func boolValue(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
