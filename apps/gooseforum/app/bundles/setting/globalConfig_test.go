package setting

import (
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/preferences"
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

func TestCookieSecureFailClosed(t *testing.T) {
	cases := []struct {
		name      string
		appEnv    string
		serverURL string
		want      bool
	}{
		// Issue #113: production must force Secure regardless of server.url scheme.
		{name: "production http example", appEnv: "production", serverURL: "http://example.com", want: true},
		{name: "production http localhost", appEnv: "production", serverURL: "http://localhost", want: true},
		{name: "production https", appEnv: "production", serverURL: "https://example.com", want: true},
		{name: "production empty url", appEnv: "production", serverURL: "", want: true},
		{name: "staging http", appEnv: "staging", serverURL: "http://staging.example.com", want: true},
		{name: "unknown env treated as non-local", appEnv: "anything-else", serverURL: "http://x", want: true},
		// Local dev keeps Secure off so 0.0.0.0 / LAN-IP over plain http works.
		{name: "local http localhost", appEnv: "local", serverURL: "http://localhost:5234", want: false},
		{name: "local https", appEnv: "local", serverURL: "https://localhost", want: false},
		{name: "local empty url", appEnv: "local", serverURL: "", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.appEnv, tt.serverURL)
			if got := CookieSecure(); got != tt.want {
				t.Fatalf("CookieSecure(env=%q, url=%q) = %v, want %v", tt.appEnv, tt.serverURL, got, tt.want)
			}
		})
	}
}

func TestCookieSecureDefaultIsProduction(t *testing.T) {
	// When app.env is unset the default must read as production (fail-closed).
	withEnv(t, "production", "http://example.com")
	if !CookieSecure() {
		t.Fatal("default production env must yield Secure=true")
	}
}
