package oauthservice

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
)

func TestOIDCResumeTarget(t *testing.T) {
	for _, raw := range []string{"https://evil.test/api/oauth/authorize/callback?id=x", "//evil.test/api/oauth/authorize/callback?id=x", "/settings?id=x", "/api/oauth/authorize/callback", "/api/oauth/authorize/callback?id=x&id=y", "/api/oauth/authorize/callback?id=x&redirect=//evil.test", "/api/oauth/authorize/callback?id=x#bad", "/api/oauth/authorize/callback?id=%0a", "/api/oauth/authorize/callback?id=%zz"} {
		t.Run(raw, func(t *testing.T) {
			if _, err := oidcResumeTarget(raw); err == nil {
				t.Fatal("unsafe target accepted")
			}
		})
	}
	if got, err := oidcResumeTarget("/api/oauth/authorize/callback?id=abc-123"); err != nil || got != "/api/oauth/authorize/callback?id=abc-123" {
		t.Fatalf("valid target: %q %v", got, err)
	}
}

func TestOIDCResumeCookie(t *testing.T) {
	const target = "/api/oauth/authorize/callback?id=abc-123"
	for _, tc := range []string{"success", "state", "provider", "expired", "tampered", "callbackRedirect", "missing"} {
		t.Run(tc, func(t *testing.T) {
			start := httptest.NewRequest(http.MethodGet, "/api/auth/github?provider=github&state=client-controlled&redirect="+url.QueryEscape(target), nil)
			rec := httptest.NewRecorder()
			if err := prepareOIDCResume(rec, start); err != nil {
				t.Fatal(err)
			}
			state := start.URL.Query().Get("state")
			if state == "" || state == "client-controlled" {
				t.Fatal("state must be server-generated")
			}
			cookie := rec.Result().Cookies()[0]
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 600 {
				t.Fatalf("cookie options: %v", cookie)
			}
			callback := httptest.NewRequest(http.MethodGet, "/api/auth/github/callback?provider=github&state="+url.QueryEscape(state), nil)
			if tc == "state" {
				callback.URL.RawQuery = "provider=github&state=wrong"
			}
			if tc == "provider" {
				callback.URL.RawQuery = "provider=google&state=" + url.QueryEscape(state)
			}
			if tc == "callbackRedirect" {
				callback.URL.RawQuery += "&redirect=https://evil.test"
			}
			if tc == "tampered" {
				cookie.Value = "tampered"
			}
			if tc == "expired" {
				session, _ := sessionstore.GetSession().New(start, resumeCookie)
				session.Values["state"] = state
				session.Values["provider"] = "github"
				session.Values["target"] = target
				session.Values["expires"] = time.Now().Add(-time.Minute).Unix()
				saved := httptest.NewRecorder()
				if err := session.Save(start, saved); err != nil {
					t.Fatal(err)
				}
				cookie = saved.Result().Cookies()[0]
			}
			if tc != "missing" {
				callback.AddCookie(cookie)
			}
			result := httptest.NewRecorder()
			got, err := consumeOIDCResume(result, callback)
			switch tc {
			case "success", "callbackRedirect":
				if err != nil || got != target {
					t.Fatalf("resume: %q %v", got, err)
				}
			case "missing":
				if err != nil || got != "" {
					t.Fatalf("ordinary login: %q %v", got, err)
				}
			default:
				if err == nil || got != "" {
					t.Fatal("invalid continuation accepted")
				}
			}
			if tc != "missing" && result.Result().Cookies()[0].MaxAge != -1 {
				t.Fatal("continuation not cleared")
			}
		})
	}
}

func TestOrdinaryOAuthClearsContinuation(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/auth/github?provider=github", nil)
	w := httptest.NewRecorder()
	if err := prepareOIDCResume(w, r); err != nil {
		t.Fatal(err)
	}
	if w.Result().Cookies()[0].MaxAge != -1 {
		t.Fatal("ordinary login retained a stale continuation")
	}
	for _, target := range []string{"/evil", "https://evil.test"} {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/github?provider=github&redirect="+url.QueryEscape(target), nil)
		if err := prepareOIDCResume(httptest.NewRecorder(), req); err == nil {
			t.Fatal("invalid target accepted")
		}
	}
	req := httptest.NewRequest(http.MethodGet, "/api/auth/unknown?provider=unknown&redirect="+url.QueryEscape("/api/oauth/authorize/callback?id=x"), nil)
	if err := prepareOIDCResume(httptest.NewRecorder(), req); err == nil {
		t.Fatal("invalid provider accepted")
	}
}
