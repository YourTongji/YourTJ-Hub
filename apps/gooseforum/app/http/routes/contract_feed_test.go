package routes

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFeedRoutesAuthPermissionBodyAndDisabledAcceptance(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(feed.Models()...); err != nil {
		t.Fatal(err)
	}
	router.POST("/api/forum/feed/events", middleware.CSRFProtection, middleware.JWTAuthCheck, api.FeedEvents)
	router.GET("/api/admin/feed/summary", middleware.JWTAuthCheck, middleware.CheckPermission(permission.Admin), UpButterReq(api.FeedSummary))
	assertInteractionUnauthenticated(t, router, "/api/forum/feed/events", `{"patches":[]}`, "auth-required.json")
	for _, path := range []string{"/api/admin/feed/summary"} {
		r := httptest.NewRecorder()
		router.ServeHTTP(r, httptest.NewRequest(http.MethodGet, path, nil))
		if r.Code != 401 {
			t.Fatalf("unauthorized admin=%d", r.Code)
		}
	}
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/feed/summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(r, req)
	if r.Code != 403 {
		t.Fatalf("non-admin read raw status=%d: %s", r.Code, r.Body.String())
	}
	preferences.Set("feed.metrics.enabled", false)
	good := serveJSON(router, "/api/forum/feed/events", `{"patches":[]}`, token)
	if good.Code != http.StatusOK {
		t.Fatal(good.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, good), contractFixture(t, "feed-events-success.json"))
	oversized := serveJSON(router, "/api/forum/feed/events", `{"patches":[],"unknown":"`+strings.Repeat("x", 32768)+`"}`, token)
	if oversized.Code != 400 {
		t.Fatalf("oversized=%d", oversized.Code)
	}
}
