package passkeyservice

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var appleTeamIDPattern = regexp.MustCompile(`^[A-Z0-9]{10}$`)

// Config is the explicitly configured relying-party trust boundary. Android
// opaque origins are derived from configured signing-certificate SHA-256
// fingerprints so the allowlist and Digital Asset Links document cannot drift.
type Config struct {
	Enabled             bool
	RPID                string
	DisplayName         string
	WebOrigins          []string
	AndroidOrigins      []string
	AppleTeamID         string
	AppleBundleID       string
	AndroidPackageName  string
	AndroidFingerprints []string
	LocalEnvironment    bool
}

func LoadConfig() (Config, error) {
	cfg := Config{
		Enabled:             preferences.GetBool("passkeys.enabled", false),
		RPID:                strings.TrimSpace(preferences.GetString("passkeys.rp_id", "")),
		DisplayName:         strings.TrimSpace(preferences.GetString("passkeys.display_name", "YourTJ")),
		WebOrigins:          cleanStrings(preferences.GetStringSlice("passkeys.web_origins")),
		AppleTeamID:         strings.TrimSpace(preferences.GetString("passkeys.apple_team_id", "")),
		AppleBundleID:       strings.TrimSpace(preferences.GetString("passkeys.ios_bundle_id", "tj.yourtj.forumApp")),
		AndroidPackageName:  strings.TrimSpace(preferences.GetString("passkeys.android_package_name", "tj.yourtj.forum_app")),
		AndroidFingerprints: cleanStrings(preferences.GetStringSlice("passkeys.android_sha256_cert_fingerprints")),
		LocalEnvironment:    strings.EqualFold(preferences.GetString("app.env", "production"), "local"),
	}
	for _, fingerprint := range cfg.AndroidFingerprints {
		origin, err := androidOrigin(fingerprint)
		if err != nil {
			if cfg.Enabled {
				return cfg, fmt.Errorf("invalid Android signing fingerprint: %w", err)
			}
			continue
		}
		cfg.AndroidOrigins = append(cfg.AndroidOrigins, origin)
	}
	if !cfg.Enabled {
		return cfg, nil
	}
	if cfg.RPID == "" || cfg.DisplayName == "" || len(cfg.WebOrigins) == 0 {
		return cfg, errors.New("passkey RP ID, display name, and at least one explicit web origin are required")
	}
	if !appleTeamIDPattern.MatchString(cfg.AppleTeamID) || cfg.AppleBundleID == "" {
		return cfg, errors.New("passkey Apple Team ID and iOS bundle ID are required")
	}
	if cfg.AndroidPackageName == "" || len(cfg.AndroidOrigins) == 0 {
		return cfg, errors.New("passkey Android package name and signing-certificate fingerprints are required")
	}
	for _, origin := range cfg.WebOrigins {
		if err := validateWebOrigin(cfg.RPID, origin, cfg.LocalEnvironment); err != nil {
			return cfg, err
		}
	}
	if _, err := cfg.newWebAuthn(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (c Config) NewWebAuthn() (*webauthn.WebAuthn, error) {
	if !c.Enabled {
		return nil, errors.New("passkeys are disabled")
	}
	return c.newWebAuthn()
}

func (c Config) newWebAuthn() (*webauthn.WebAuthn, error) {
	if c.RPID == "" || c.DisplayName == "" || len(c.WebOrigins) == 0 {
		return nil, errors.New("incomplete WebAuthn relying-party configuration")
	}
	return webauthn.New(&webauthn.Config{
		RPID:                  c.RPID,
		RPDisplayName:         c.DisplayName,
		RPOrigins:             append([]string(nil), c.WebOrigins...),
		RPOpaqueOrigins:       append([]string(nil), c.AndroidOrigins...),
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Registration: webauthn.TimeoutConfig{Enforce: true},
			Login:        webauthn.TimeoutConfig{Enforce: true},
		},
	})
}

func (c Config) AppleAppID() string {
	if c.AppleTeamID == "" || c.AppleBundleID == "" {
		return ""
	}
	return c.AppleTeamID + "." + c.AppleBundleID
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func validateWebOrigin(rpID, raw string, local bool) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("invalid passkey web origin %q", raw)
	}
	if u.Scheme != "https" && (!local || u.Scheme != "http" || !isLoopbackHost(u.Hostname())) {
		return fmt.Errorf("passkey origin %q must use HTTPS outside local development", raw)
	}
	if host := strings.ToLower(u.Hostname()); host != strings.ToLower(rpID) && !strings.HasSuffix(host, "."+strings.ToLower(rpID)) {
		return fmt.Errorf("passkey origin %q is outside RP ID %q", raw, rpID)
	}
	return nil
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func androidOrigin(fingerprint string) (string, error) {
	compact := strings.ReplaceAll(strings.TrimSpace(fingerprint), ":", "")
	if len(compact) != 64 {
		return "", errors.New("SHA-256 fingerprint must contain 32 bytes")
	}
	decoded, err := hex.DecodeString(compact)
	if err != nil || len(decoded) != 32 {
		return "", errors.New("SHA-256 fingerprint is not valid hexadecimal")
	}
	return "android:apk-key-hash:" + base64.RawURLEncoding.EncodeToString(decoded), nil
}
