package sessionstore

import (
	"testing"

	"github.com/gorilla/sessions"
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

// TestSessionStoreSecureByEnv replaces the old scheme-based sessionCookieSecure
// table: the Secure flag now follows setting.CookieSecure (fail-closed by env),
// not server.url scheme. Covers issue #113.
func TestSessionStoreSecureByEnv(t *testing.T) {
	cases := []struct {
		name      string
		appEnv    string
		serverURL string
		want      bool
	}{
		{name: "production http localhost default", appEnv: "production", serverURL: "http://localhost", want: true},
		{name: "production http example", appEnv: "production", serverURL: "http://example.com", want: true},
		{name: "production https", appEnv: "production", serverURL: "https://example.com", want: true},
		{name: "production empty url", appEnv: "production", serverURL: "", want: true},
		{name: "staging http", appEnv: "staging", serverURL: "http://staging.example.com", want: true},
		{name: "local http localhost", appEnv: "local", serverURL: "http://localhost:5234", want: false},
		{name: "local https", appEnv: "local", serverURL: "https://localhost", want: false},
		{name: "local empty url", appEnv: "local", serverURL: "", want: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.appEnv, tt.serverURL)
			st := sessions.NewCookieStore([]byte("test-key"))
			configureSessionStore(st)
			if st.Options.Secure != tt.want {
				t.Fatalf("Options.Secure = %v (env=%q url=%q), want %v", st.Options.Secure, tt.appEnv, tt.serverURL, tt.want)
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
