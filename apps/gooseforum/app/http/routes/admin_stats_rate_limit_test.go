package routes

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

func TestAdminStatsShareRateLimitDespiteAdminExemption(t *testing.T) {
	conn, router := setupAdminStatsTest(t)
	cfg := defaultconfig.GetDefaultRateLimitConfig()
	cfg.Enabled, cfg.SkipAdmin = true, true
	cfg.Actions = []pageConfig.RateLimitRule{{Action: "admin.stats", WindowSeconds: 60, LimitPerUser: 1}}
	persistHTTPContractConfig(t, conn, pageConfig.RateLimitSettings, cfg)
	hotdataserve.ClearRateLimitConfigCache()
	ratelimit.Default().ResetAll()
	t.Cleanup(func() { hotdataserve.ClearRateLimitConfigCache(); ratelimit.Default().ResetAll() })
	admin := createContractAdmin(t, conn)
	token := contractSessionToken(t, admin)
	first := serveJSON(router, statsSummaryPath, `{}`, token)
	if first.Code != http.StatusOK {
		t.Fatalf("first = %d, want 200: %s", first.Code, first.Body.String())
	}
	for _, path := range []string{statsSummaryPath, statsTrendPath, statsRankingPath, statsDetailPath} {
		recorder := serveJSON(router, path, `{}`, token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Errorf("%s after shared quota = %d, want 429: %s", path, recorder.Code, recorder.Body.String())
			continue
		}
		response := decodeContractEnvelope(t, recorder)
		assertRetryAfter(t, recorder, response, "admin.stats")
		fixture := contractFixture(t, "admin-stats-rate-limited.json")
		fixture.Params["retryAfterSeconds"] = response.Params["retryAfterSeconds"]
		assertFixtureEnvelope(t, response, fixture)
	}
	// Authorization is checked before quota. A denied user must not consume a
	// future administrator's budget, and another administrator has a distinct one.
	next := createHTTPContractUser(t, conn, contractTestID())
	nextToken := contractSessionToken(t, next)
	if rec := serveJSON(router, statsSummaryPath, `{}`, nextToken); rec.Code != http.StatusForbidden {
		t.Fatalf("unprivileged = %d, want 403", rec.Code)
	}
	grantContractPermission(t, conn, next.Id, permission.Admin)
	userservice.InvalidateUserInfoCache(next.Id)
	if rec := serveJSON(router, statsSummaryPath, `{}`, nextToken); rec.Code != http.StatusOK {
		t.Fatalf("independent administrator = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}
