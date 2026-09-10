package oidcservice

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProviderHintOnlyRewritesForumLogin(t *testing.T) {
	callback := "/api/oauth/authorize/callback?id=bound-request"
	login := "/login?redirect=" + url.QueryEscape(callback)
	for _, tc := range []struct{ hint, location, want string }{
		{"github", login, "/api/auth/github?redirect=" + url.QueryEscape(callback)},
		{"google", login, "/api/auth/google?redirect=" + url.QueryEscape(callback)},
		{"https://evil.test", login, login},
		{"", login, login},
		{"github", "https://evil.test/login?redirect=x", "https://evil.test/login?redirect=x"},
		{"github", "/login?redirect=//evil.test", "/login?redirect=//evil.test"},
		{"google", "/api/oauth/authorize/callback?id=already-authenticated", "/api/oauth/authorize/callback?id=already-authenticated"},
	} {
		t.Run(tc.hint+tc.location, func(t *testing.T) {
			h := withProviderHint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Set-Cookie", "binding=kept; HttpOnly; Secure")
				http.Redirect(w, r, tc.location, http.StatusFound)
			}))
			request := httptest.NewRequest(http.MethodGet, "/authorize?login_hint="+url.QueryEscape(tc.hint), nil)
			response := httptest.NewRecorder()
			h.ServeHTTP(response, request)
			if got := response.Header().Get("Location"); got != tc.want {
				t.Fatalf("Location = %q, want %q", got, tc.want)
			}
			if response.Header().Get("Set-Cookie") == "" {
				t.Fatal("browser binding cookie lost")
			}
		})
	}
}

func TestSocialHintAuthorizeFlowKeepsBrowserBindingAndPKCE(t *testing.T) {
	for _, provider := range []string{"google", "github"} {
		t.Run(provider, func(t *testing.T) {
			issuer := "https://forum.example.com/api/oauth"
			setupProviderConfig(t, issuer, defaultClients())
			h, err := Router()
			if err != nil {
				t.Fatal(err)
			}
			cookie, _ := loginCookie(t, "hintuser"+provider)
			verifier, _ := pkcePair(t)
			requestID, binding, location := startAuthorize(t, h, authorizeURL(issuer, "web-client", "https://example.com/callback", "hint-state", "hint-nonce", verifier)+"&login_hint="+provider)
			parsed, err := url.Parse(location)
			if err != nil || parsed.Path != "/api/auth/"+provider {
				t.Fatalf("provider redirect: %q", location)
			}
			code := completeAuthorize(t, h, issuer, requestID, cookie, binding)
			rec, body := exchangeToken(t, h, issuer+"/token", code, "https://example.com/callback", "web-client", "web-secret", verifier)
			if rec.Code != http.StatusOK || body["id_token"] == "" {
				t.Fatalf("exchange status=%d body=%v", rec.Code, body)
			}
		})
	}
}
