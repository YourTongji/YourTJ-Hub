package routes

import (
	"encoding/json"
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

func TestSavedRateLimitSettingsExposeNewBudgetAndPreserveDisabledOverride(t *testing.T) {
	for _, disable := range []bool{false, true} {
		name := "legacy-missing-budget"
		if disable {
			name = "explicitly-disabled-budget"
		}
		t.Run(name, func(t *testing.T) {
			conn, router := setupAdminSiteContractTest(t)
			cfg := pageConfig.RateLimitConfig{Enabled: false, SkipAdmin: true, Actions: []pageConfig.RateLimitRule{{Action: "topic.write", WindowSeconds: 19, LimitPerUser: 2}}}
			if disable {
				cfg.Actions = append(cfg.Actions, pageConfig.RateLimitRule{Action: "admin.stats", WindowSeconds: 31, LimitPerUser: 0})
			}
			persistContractPageConfig(t, conn, pageConfig.RateLimitSettings, cfg)
			hotdataserve.ClearRateLimitConfigCache()
			t.Cleanup(hotdataserve.ClearRateLimitConfigCache)
			recorder := serveAdminSiteRaw(t, conn, router, http.MethodGet, "/api/admin/rate-limit-settings", "")
			var response struct {
				Code   int                        `json:"code"`
				Result pageConfig.RateLimitConfig `json:"result"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != 0 || response.Result.Enabled || !response.Result.SkipAdmin {
				t.Fatal("stored global switches changed")
			}
			foundStats, foundCustom := false, false
			for _, rule := range response.Result.Actions {
				switch rule.Action {
				case "admin.stats":
					foundStats = true
					if disable {
						if rule.WindowSeconds != 31 || rule.LimitPerUser != 0 {
							t.Fatal("explicit disabled budget overwritten")
						}
					} else if rule.WindowSeconds != 60 || rule.LimitPerUser != 60 {
						t.Fatal("new default budget differs from enforced policy")
					}
				case "topic.write":
					foundCustom = true
					if rule.WindowSeconds != 19 || rule.LimitPerUser != 2 {
						t.Fatal("stored custom budget overwritten")
					}
				}
			}
			if !foundStats || !foundCustom {
				t.Fatal("administrator cannot see effective new/custom budgets")
			}
		})
	}
}
