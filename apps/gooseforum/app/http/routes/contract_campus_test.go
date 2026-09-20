package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
)

func TestCampusRoutesRequireSessionAndCSRF(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	campusRoutes(router)
	for _, r := range []struct{ method, path string }{{"GET", "/api/campus/status"}, {"GET", "/api/campus/calendar-export"}, {"GET", "/api/campus/data/grades"}, {"GET", "/api/campus/data/profile"}, {"GET", "/api/campus/messages/123"}, {"POST", "/api/campus/tongji/start"}, {"POST", "/api/campus/tongji/confirm"}, {"POST", "/api/campus/tongji/unbind"}} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(r.method, r.path, strings.NewReader(`{}`)))
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("%s: %d", r.path, w.Code)
		}
	}
	u := createHTTPContractUser(t, db, 981271)
	token := contractSessionToken(t, u)
	r := httptest.NewRequest(http.MethodPost, "/api/campus/tongji/start", strings.NewReader(`{"mode":"bind"}`))
	r.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://evil.invalid")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Fatalf("cross-site binding request: %d", w.Code)
	}
}
func TestCampusMobileHandoff(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	router.GET("/api/auth/mobile-web-session", api.MobileWebSession)
	u := createHTTPContractUser(t, db, 981272)
	token := contractSessionToken(t, u)
	r := httptest.NewRequest(http.MethodGet, "/api/auth/mobile-web-session?target=campus", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 303 || w.Header().Get("Location") != "/campus" || !strings.Contains(w.Header().Get("Set-Cookie"), "HttpOnly") {
		t.Fatalf("invalid campus handoff: %d", w.Code)
	}
	r = httptest.NewRequest(http.MethodGet, "/api/auth/mobile-web-session?target=campus", nil)
	r.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatal("cookie-only handoff accepted")
	}
}

func TestCampusCalendarExportDisabledResponseIsPrivate(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	campusRoutes(router)
	t.Setenv("CAMPUS_REDIRECT_URI", "disabled")
	u := createHTTPContractUser(t, db, contractTestID())
	req := httptest.NewRequest(http.MethodGet, "/api/campus/calendar-export", nil)
	req.Header.Set("Authorization", "Bearer "+contractSessionToken(t, u))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable || w.Header().Get("Cache-Control") != "private, no-store" || !strings.Contains(w.Body.String(), "campus.disabled") {
		t.Fatalf("invalid private export failure: status=%d cache=%s", w.Code, w.Header().Get("Cache-Control"))
	}
}

func TestCampusCallbackAlwaysScrubsDeniedOrDisabledRequests(t *testing.T) {
	for _, scenario := range []string{"missing", "invalid", "revoked", "frozen", "pending", "disabled", "limited"} {
		t.Run(scenario, func(t *testing.T) {
			db, router := setupHTTPContractTest(t)
			campusRoutes(router)
			t.Setenv("CAMPUS_CLIENT_ID", "")
			t.Setenv("CAMPUS_REDIRECT_URI", "disabled")
			req := httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?code=sensitive-code&state=sensitive-state", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			if scenario == "invalid" {
				req.Header.Set("Authorization", "Bearer invalid")
			}
			if scenario != "missing" && scenario != "invalid" {
				u := createHTTPContractUser(t, db, contractTestID())
				if scenario == "frozen" {
					db.Model(u).Update("is_frozen", users.StatusFrozen)
				}
				if scenario == "pending" {
					db.Model(u).Update("is_activated", users.ActivationPending)
					cfg := defaultconfig.GetDefaultSecuritySettingsConfig()
					cfg.EnableEmailVerification = true
					persistHTTPContractConfig(t, db, pageConfig.SecuritySettings, cfg)
					hotdataserve.ClearSecuritySettingsConfigCache()
				}
				req.Header.Set("Authorization", "Bearer "+contractSessionToken(t, u))
				if scenario == "revoked" {
					if err := sessionservice.RevokeAllAndInvalidate(u.Id); err != nil {
						t.Fatal(err)
					}
				}
				if scenario == "limited" {
					persistHTTPContractConfig(t, db, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{Enabled: true, Actions: []pageConfig.RateLimitRule{{Action: "campus.authorize", WindowSeconds: 60, LimitPerIp: 1}}})
					hotdataserve.ClearRateLimitConfigCache()
					ratelimit.Default().Allow("campus.authorize:ip:192.0.2.1", 1, time.Minute)
				}
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/campus?authorization=failed" {
				t.Fatalf("callback not scrubbed: status=%d location=%s", w.Code, w.Header().Get("Location"))
			}
			if w.Header().Get("Referrer-Policy") != "no-referrer" || w.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("callback missing privacy headers")
			}
			if strings.Contains(w.Body.String(), "sensitive-") {
				t.Fatal("callback echoed credentials")
			}
		})
	}
}

func TestCampusDoesNotConsumeLoginOrCatalogBudgets(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	campusRoutes(router)
	t.Setenv("CAMPUS_REDIRECT_URI", "disabled")
	u := createHTTPContractUser(t, db, contractTestID())
	token := contractSessionToken(t, u)
	for _, action := range []string{middleware.RateLimitLogin, middleware.RateLimitCourseCatalog} {
		for range 130 {
			ratelimit.Default().Allow(action+":ip:192.0.2.1", 200, time.Minute)
		}
	}
	for _, path := range []string{"/api/campus/calendar-export", "/api/campus/data/calendar", "/api/campus/messages/123", "/api/campus/tongji/start"} {
		method := "GET"
		if strings.HasSuffix(path, "start") {
			method = "POST"
		}
		req := httptest.NewRequest(method, path, strings.NewReader(`{"mode":"bind"}`))
		req.RemoteAddr = "192.0.2.1:1234"
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code == http.StatusTooManyRequests {
			t.Fatalf("campus shares other domain limit: %s", path)
		}
	}
}

func TestCampusReadQuotaIsPerUserBehindSharedNAT(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	campusRoutes(router)
	t.Setenv("CAMPUS_REDIRECT_URI", "disabled")
	persistHTTPContractConfig(t, db, pageConfig.RateLimitSettings, pageConfig.RateLimitConfig{Enabled: true, Actions: []pageConfig.RateLimitRule{{Action: "campus.read", WindowSeconds: 60, LimitPerUser: 1}}})
	hotdataserve.ClearRateLimitConfigCache()
	first := contractSessionToken(t, createHTTPContractUser(t, db, contractTestID()))
	second := contractSessionToken(t, createHTTPContractUser(t, db, contractTestID()))
	for i, token := range []string{first, first, second} {
		req := httptest.NewRequest(http.MethodGet, "/api/campus/data/calendar", nil)
		req.RemoteAddr = "192.0.2.7:1234"
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if (w.Code == http.StatusTooManyRequests) != (i == 1) {
			t.Fatalf("request %d quota: %d", i, w.Code)
		}
	}
}

func TestTongjiLoginCallbackDoesNotRequireForumSessionAndScrubsFailure(t *testing.T) {
	_, router := setupHTTPContractTest(t)
	campusRoutes(router)
	t.Setenv("CAMPUS_REDIRECT_URI", "disabled")
	for _, query := range []string{"state=login.test&code=private-code", "state=login.test&error=access_denied"} {
		req := httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?"+query, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != 303 || rec.Header().Get("Location") != "/login?tongjiNotice=unavailable" {
			t.Fatalf("login callback: %d %s", rec.Code, rec.Header().Get("Location"))
		}
		if rec.Header().Get("Cache-Control") != "private, no-store" || rec.Header().Get("Referrer-Policy") != "no-referrer" || strings.Contains(rec.Body.String(), "private-code") {
			t.Fatal("callback privacy failure")
		}
	}
}
