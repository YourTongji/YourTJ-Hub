package routes

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCampusRoutesRequireSessionAndCSRF(t *testing.T) {
	db, router := setupHTTPContractTest(t)
	campusRoutes(router)
	for _, r := range []struct{ method, path string }{{"GET", "/api/campus/status"}, {"GET", "/api/campus/data/grades"}, {"GET", "/api/campus/data/profile"}, {"GET", "/api/campus/messages/123"}, {"POST", "/api/campus/tongji/start"}, {"GET", "/api/campus/tongji/callback?state=invalid&code=invalid"}, {"POST", "/api/campus/tongji/confirm"}, {"POST", "/api/campus/tongji/unbind"}} {
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
