package sessionstore

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gorilla/sessions"
)

// withEnv restores app.env, server.url and server.cookieSecure after the test
// so package-level viper state does not leak across tests. preferences.Set
// cannot unset a key, so an absent cookieSecure is restored as "" — which
// resolves exactly like the unset default "auto".
func withEnv(t *testing.T, appEnv, serverURL, cookieSecure string) {
	t.Helper()
	prevEnv := preferences.GetString("app.env", "production")
	prevURL := preferences.GetString("server.url", "")
	prevCookieSecure := preferences.GetString("server.cookieSecure", "")
	preferences.Set("app.env", appEnv)
	preferences.Set("server.url", serverURL)
	preferences.Set("server.cookieSecure", cookieSecure)
	t.Cleanup(func() {
		preferences.Set("app.env", prevEnv)
		preferences.Set("server.url", prevURL)
		preferences.Set("server.cookieSecure", prevCookieSecure)
	})
}

// TestSessionStoreSecureByEnv replaces the old scheme-based sessionCookieSecure
// table: the Secure flag now follows setting.CookieSecure (auto mode derives it
// from server.url, server.cookieSecure overrides). Covers issues #113 and #1277.
func TestSessionStoreSecureByEnv(t *testing.T) {
	cases := []struct {
		name         string
		appEnv       string
		serverURL    string
		cookieSecure string
		want         bool
	}{
		// Issue #113: loopback is a secure context, Secure stays on.
		{name: "production http localhost default", appEnv: "production", serverURL: "http://localhost", want: true},
		{name: "production https", appEnv: "production", serverURL: "https://example.com", want: true},
		{name: "production empty url", appEnv: "production", serverURL: "", want: true},
		{name: "production url without scheme", appEnv: "production", serverURL: "example.com", want: true},
		// Issue #1277: plain-http public hosts get no Secure flag under auto.
		{name: "production auto http example", appEnv: "production", serverURL: "http://example.com", want: false},
		{name: "production auto http lan ip", appEnv: "production", serverURL: "http://192.168.1.5:5234", want: false},
		{name: "production auto http ipv6 global", appEnv: "production", serverURL: "http://[2001:da8::1]:5234", want: false},
		{name: "staging http", appEnv: "staging", serverURL: "http://staging.example.com", want: false},
		{name: "local http localhost", appEnv: "local", serverURL: "http://localhost:5234", want: false},
		{name: "local https", appEnv: "local", serverURL: "https://localhost", want: false},
		{name: "local empty url", appEnv: "local", serverURL: "", want: false},
		// Explicit server.cookieSecure overrides the auto resolution.
		{name: "explicit on production http lan ip", appEnv: "production", serverURL: "http://192.168.1.5:5234", cookieSecure: "on", want: true},
		{name: "explicit off production https", appEnv: "production", serverURL: "https://example.com", cookieSecure: "off", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.appEnv, tt.serverURL, tt.cookieSecure)
			st := sessions.NewCookieStore([]byte("test-key"))
			configureSessionStore(st)
			if st.Options.Secure != tt.want {
				t.Fatalf("Options.Secure = %v (env=%q url=%q cookieSecure=%q), want %v", st.Options.Secure, tt.appEnv, tt.serverURL, tt.cookieSecure, tt.want)
			}
			if st.Options.Path != "/" {
				t.Fatalf("Path = %q, want /", st.Options.Path)
			}
			if !st.Options.HttpOnly {
				t.Fatalf("HttpOnly = false, want true")
			}
		})
	}
}
