package routes

import (
	"encoding/json"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/feedservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

func TestFeedSessionRoutesAuthBoundsAndDisabledRefresh(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(append(feed.Models(), &users.BlockEntity{})...); err != nil {
		t.Fatal(err)
	}
	router.POST("/api/forum/feed/refresh", middleware.CSRFProtection, middleware.JWTAuthCheck, forum.FeedRefresh)
	router.POST("/api/forum/feed/reconcile", middleware.CSRFProtection, middleware.JWTAuthCheck, forum.FeedReconcile)
	for _, path := range []string{"/api/forum/feed/refresh", "/api/forum/feed/reconcile"} {
		assertInteractionUnauthenticated(t, router, path, `{}`, "auth-required.json")
	}
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	oldConfig := feedconfig.Current()
	t.Cleanup(func() {
		preferences.Set("feed.for_you.enabled", oldConfig.Enabled)
		preferences.Set("ranking.enabled", oldConfig.Ranking)
	})
	preferences.Set("feed.for_you.enabled", false)
	preferences.Set("ranking.enabled", false)
	for _, body := range []string{`{"topicIds":[]}`, `{"topicIds":[1,1]}`, `{"topicIds":[0]}`} {
		if response := serveJSON(router, "/api/forum/feed/reconcile", body, token); response.Code != 400 {
			t.Fatalf("invalid IDs accepted: %s", response.Body.String())
		}
	}
	response := serveJSON(router, "/api/forum/feed/reconcile", `{"topicIds":[9999999]}`, token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var result struct {
		Code   int                       `json:"code"`
		Result forum.FeedSessionResponse `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Result.ViewerID != user.Id || result.Result.Available || len(result.Result.Topics) != 0 || len(result.Result.RemovedIDs) != 1 || result.Result.RemovedIDs[0] != 9999999 || result.Result.Pagination != nil {
		t.Fatal("invalid reconcile response")
	}
	response = serveJSON(router, "/api/forum/feed/refresh", `{}`, token)
	if response.Code != 503 {
		t.Fatalf("disabled explicit refresh silently replaced batch: %s", response.Body.String())
	}
	response = serveJSON(router, "/api/forum/feed/refresh", `{"seenPatches":[{"proof":"forged","seen":{"0":1000}}]}`, token)
	if response.Code != 400 {
		t.Fatal("forged refresh claim accepted")
	}
	response = serveJSON(router, "/api/forum/feed/refresh", `{"replaceSnapshotId":"`+strings.Repeat("x", 129)+`"}`, token)
	if response.Code != 400 {
		t.Fatal("oversized replacement ID accepted")
	}
}

func TestFeedSeenACKIsDurableWithMetricsDisabled(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(feed.Models()...); err != nil {
		t.Fatal(err)
	}
	router.POST("/api/forum/feed/events", middleware.CSRFProtection, middleware.JWTAuthCheck, api.FeedEvents)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	preferences.Set("feed.metrics.enabled", false)
	proofs := feedservice.MintSeenProof(user.Id, []uint64{999888}, time.Now().Add(-time.Minute))
	time.Sleep(1010 * time.Millisecond)
	body, err := json.Marshal(map[string]any{"patches": []any{}, "seenPatches": []feedservice.SeenPatch{{Proof: proofs[0].Token, Seen: map[int]int64{0: 1000}}}})
	if err != nil {
		t.Fatal(err)
	}
	response := serveJSON(router, "/api/forum/feed/events", string(body), token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "feed-seen-success.json"))
	var state feed.SeenState
	if err := conn.First(&state, "user_id = ? AND topic_id = ?", user.Id, 999888).Error; err != nil {
		t.Fatal("ACK preceded durable state", err)
	}
	t.Cleanup(func() { conn.Delete(&state); conn.Delete(&feed.Owner{}, "user_id = ?", user.Id) })
}

func TestEmptyFeedRefreshReturnsArrayFields(t *testing.T) {
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&users.BlockEntity{}, &userFollow.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(feed.Models()...); err != nil {
		t.Fatal(err)
	}
	router.POST("/api/forum/feed/refresh", middleware.CSRFProtection, middleware.JWTAuthCheck, forum.FeedRefresh)
	user := createHTTPContractUser(t, conn, contractTestID())
	previous := feedconfig.Current()
	t.Cleanup(func() {
		preferences.Set("ranking.enabled", previous.Ranking)
		preferences.Set("feed.for_you.enabled", previous.Enabled)
		preferences.Set("feed.metrics.enabled", previous.Metrics)
	})
	preferences.Set("ranking.enabled", true)
	preferences.Set("feed.for_you.enabled", true)
	preferences.Set("feed.metrics.enabled", false)
	feedconfig.SetRankReady(true)
	t.Cleanup(func() { feedconfig.SetRankReady(false) })
	response := serveJSON(router, "/api/forum/feed/refresh", `{"seenPatches":[]}`, contractSessionToken(t, user))
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	envelope := decodeContractEnvelope(t, response)
	var result map[string]any
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"topics", "seenProofs", "removedIds"} {
		if _, ok := result[key].([]any); !ok {
			t.Fatalf("%s must serialize as an array: %s", key, response.Body.String())
		}
	}
	if result["viewerId"] != float64(user.Id) || result["snapshotId"] == "" || result["available"] != true || result["seenConfirmed"] != true || envelope.Code != 0 {
		t.Fatalf("invalid empty-batch envelope: %s", response.Body.String())
	}
}
