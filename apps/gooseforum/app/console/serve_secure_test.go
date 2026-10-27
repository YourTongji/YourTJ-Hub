package console

import (
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

// withEnv restores app.env and server.url after the test so package-level
// viper state does not leak across tests.
func withEnv(t *testing.T, appEnv, serverURL string) {
	t.Helper()
	prevEnv := preferences.GetString("app.env", "production")
	prevURL := preferences.GetString("server.url", "")
	preferences.Set("app.env", appEnv)
	preferences.Set("server.url", serverURL)
	t.Cleanup(func() {
		preferences.Set("app.env", prevEnv)
		preferences.Set("server.url", prevURL)
	})
}

// TestInsecureServerURLMessageFollowsCookieSecure asserts the warning text
// matches the effective setting.CookieSecure(): with Secure on the operator is
// told browsers drop the cookie, with Secure off that credentials travel in
// clear text (CWE-614).
func TestInsecureServerURLMessageFollowsCookieSecure(t *testing.T) {
	secureMsg := insecureServerURLMessage("http://example.com", true)
	plainMsg := insecureServerURLMessage("http://example.com", false)

	if !strings.Contains(secureMsg, "server.cookieSecure") {
		t.Fatalf("secure message must name the cookieSecure override, got %q", secureMsg)
	}
	if !strings.Contains(plainMsg, "CWE-614") {
		t.Fatalf("plaintext message must cite CWE-614, got %q", plainMsg)
	}
	if strings.Contains(plainMsg, "server.cookieSecure") || secureMsg == plainMsg {
		t.Fatalf("messages must differ by the effective Secure flag: secure=%q plain=%q", secureMsg, plainMsg)
	}
	for _, msg := range []string{secureMsg, plainMsg} {
		if !strings.Contains(msg, "http://example.com") {
			t.Fatalf("message must include server.url, got %q", msg)
		}
	}
}

// TestWarnInsecureServerURLGuardsCases focuses on the pure decision function:
// whether the non-https non-local warning should fire. The slog output itself
// is captured indirectly; we assert on an extracted helper so logic is decoupled
// from the logger.
func TestWarnInsecureServerURLGuardsCases(t *testing.T) {
	cases := []struct {
		name      string
		appEnv    string
		serverURL string
		wantWarn  bool
	}{
		{name: "local http localhost no warn", appEnv: "local", serverURL: "http://localhost:5234", wantWarn: false},
		{name: "production https no warn", appEnv: "production", serverURL: "https://example.com", wantWarn: false},
		{name: "production http localhost no warn", appEnv: "production", serverURL: "http://localhost", wantWarn: false},
		{name: "production http 127.0.0.1 no warn", appEnv: "production", serverURL: "http://127.0.0.1:5234", wantWarn: false},
		{name: "production http ::1 no warn", appEnv: "production", serverURL: "http://[::1]:5234", wantWarn: false},
		{name: "production empty url no warn", appEnv: "production", serverURL: "", wantWarn: false},
		{name: "production url without scheme WARN", appEnv: "production", serverURL: "10.0.0.5:5234", wantWarn: true},
		{name: "production unparsable url WARN", appEnv: "production", serverURL: "http://exa mple.com", wantWarn: true},
		{name: "production https with path no warn", appEnv: "production", serverURL: "https://example.com/forum", wantWarn: false},
		{name: "production http example WARN", appEnv: "production", serverURL: "http://example.com", wantWarn: true},
		{name: "production http lan IP WARN", appEnv: "production", serverURL: "http://10.0.0.5:5234", wantWarn: true},
		{name: "staging http example WARN", appEnv: "staging", serverURL: "http://staging.example.com", wantWarn: true},
		{name: "local http example no warn (local bypasses all)", appEnv: "local", serverURL: "http://example.com", wantWarn: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.appEnv, tt.serverURL)
			if got := shouldWarnInsecureServerURL(); got != tt.wantWarn {
				t.Fatalf("shouldWarnInsecureServerURL(env=%q, url=%q) = %v, want %v", tt.appEnv, tt.serverURL, got, tt.wantWarn)
			}
		})
	}
}
