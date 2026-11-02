package setting

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
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

func TestCookieSecureFailClosed(t *testing.T) {
	cases := []struct {
		name         string
		appEnv       string
		serverURL    string
		cookieSecure string
		want         bool
	}{
		// CWE-614 bottom line: https, empty and unparsable URLs stay Secure.
		{name: "production https", appEnv: "production", serverURL: "https://example.com", want: true},
		{name: "staging https", appEnv: "staging", serverURL: "https://staging.example.com", want: true},
		{name: "production empty url", appEnv: "production", serverURL: "", want: true},
		{name: "production url without scheme", appEnv: "production", serverURL: "example.com", want: true},
		{name: "production unparsable url", appEnv: "production", serverURL: "http://exa mple.com", want: true},
		{name: "production protocol-relative url", appEnv: "production", serverURL: "//example.com", want: true},
		{name: "production ftp scheme", appEnv: "production", serverURL: "ftp://example.com", want: true},
		{name: "production uppercase https scheme", appEnv: "production", serverURL: "HTTPS://example.com", want: true},
		{name: "production https with path and userinfo", appEnv: "production", serverURL: "https://user:pass@example.com/forum", want: true},
		{name: "production http lan with path and userinfo", appEnv: "production", serverURL: "http://user@192.168.1.5:5234/forum", want: false},
		// Issue #113: loopback hosts are a secure context, Secure must stay on.
		{name: "production http localhost", appEnv: "production", serverURL: "http://localhost", want: true},
		{name: "production http localhost with port", appEnv: "production", serverURL: "http://localhost:5234", want: true},
		{name: "production http 127.0.0.1", appEnv: "production", serverURL: "http://127.0.0.1:5234", want: true},
		{name: "production http ipv6 loopback", appEnv: "production", serverURL: "http://[::1]:5234", want: true},
		// Issue #1277: a public plain-http host cannot receive Secure cookies,
		// so auto resolves to false instead of breaking web login.
		{name: "production auto http example", appEnv: "production", serverURL: "http://example.com", want: false},
		{name: "production auto http lan ip", appEnv: "production", serverURL: "http://192.168.1.5:5234", want: false},
		{name: "production auto http ipv6 global", appEnv: "production", serverURL: "http://[2001:da8::1]:5234", want: false},
		{name: "staging http", appEnv: "staging", serverURL: "http://staging.example.com", want: false},
		{name: "unknown env treated as non-local", appEnv: "anything-else", serverURL: "http://x", want: false},
		// Local dev keeps Secure off so 0.0.0.0 / LAN-IP over plain http works.
		{name: "local http localhost", appEnv: "local", serverURL: "http://localhost:5234", want: false},
		{name: "local https", appEnv: "local", serverURL: "https://localhost", want: false},
		{name: "local empty url", appEnv: "local", serverURL: "", want: false},
		// server.cookieSecure overrides everything, including local.
		{name: "explicit on production http lan ip", appEnv: "production", serverURL: "http://192.168.1.5:5234", cookieSecure: "on", want: true},
		{name: "explicit on local", appEnv: "local", serverURL: "http://localhost:5234", cookieSecure: "on", want: true},
		{name: "explicit on uppercase", appEnv: "production", serverURL: "http://example.com", cookieSecure: "ON", want: true},
		{name: "explicit off production https", appEnv: "production", serverURL: "https://example.com", cookieSecure: "off", want: false},
		{name: "explicit off production empty url", appEnv: "production", serverURL: "", cookieSecure: "off", want: false},
		// Unrecognized values fall back to auto.
		{name: "unknown value http lan ip", appEnv: "production", serverURL: "http://192.168.1.5:5234", cookieSecure: "sometimes", want: false},
		{name: "unknown value https", appEnv: "production", serverURL: "https://example.com", cookieSecure: "sometimes", want: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.appEnv, tt.serverURL, tt.cookieSecure)
			if got := CookieSecure(); got != tt.want {
				t.Fatalf("CookieSecure(env=%q, url=%q, cookieSecure=%q) = %v, want %v", tt.appEnv, tt.serverURL, tt.cookieSecure, got, tt.want)
			}
		})
	}
}

func TestCookieSecureEmptyEnvIsNotLocal(t *testing.T) {
	// An unset/empty app.env is not "local", so the auto resolution applies
	// and an https server.url stays Secure (fail-closed).
	withEnv(t, "", "https://example.com", "")
	if !CookieSecure() {
		t.Fatal("empty app.env must not count as local; https server.url must yield Secure=true")
	}
	withEnv(t, "", "http://192.168.1.5:5234", "")
	if CookieSecure() {
		t.Fatal("empty app.env + plain-http non-loopback host must yield Secure=false under auto")
	}
}

func TestIsLoopbackHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{host: "localhost", want: true},
		{host: "LocalHost", want: true},
		{host: "127.0.0.1", want: true},
		{host: "::1", want: true},
		{host: "example.com", want: false},
		{host: "192.168.1.5", want: false},
		{host: "2001:da8::1", want: false},
		{host: "", want: false},
	}
	for _, tt := range cases {
		t.Run(tt.host, func(t *testing.T) {
			if got := IsLoopbackHost(tt.host); got != tt.want {
				t.Fatalf("IsLoopbackHost(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}
