package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupAdminSearchTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	old := preferences.GetBool("meilisearch.maintenance_enabled", false)
	preferences.Set("meilisearch.maintenance_enabled", true)
	t.Cleanup(func() { preferences.Set("meilisearch.maintenance_enabled", old) })
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&taskQueue.Entity{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("type = ?", taskQueue.SearchMaintenanceType).Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	admin := router.Group("/api/admin", middleware.CSRFProtection, middleware.JWTAuthCheck, middleware.CheckWritableAccount, middleware.CheckPermission(permission.SiteManager))
	admin.GET("/search/indexes", UpButterReq(api.GetSearchMaintenance))
	admin.POST("/search/maintenance", UpButterReq(api.CreateSearchMaintenance))
	return conn, router
}

func TestAdminSearchContract(t *testing.T) {
	for _, route := range []struct{ method, path string }{{http.MethodGet, "/api/admin/search/indexes"}, {http.MethodPost, "/api/admin/search/maintenance"}} {
		t.Run(route.path+" guards", func(t *testing.T) {
			conn, router := setupAdminSearchTest(t)
			res := serveAuthSecurityJSON(router, route.method, route.path, `{}`, "")
			if res.Code != 401 {
				t.Fatalf("unauthenticated %d", res.Code)
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "auth-required.json"))
			user := createHTTPContractUser(t, conn, contractTestID())
			res = serveAuthSecurityJSON(router, route.method, route.path, `{}`, contractSessionToken(t, user))
			if res.Code != 403 {
				t.Fatalf("permission %d", res.Code)
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "admin-ai-summary-settings-permission-denied.json"))
			manager := createContractSiteManager(t, conn)
			if err := conn.Model(manager).Update("is_frozen", users.StatusFrozen).Error; err != nil {
				t.Fatal(err)
			}
			res = serveAuthSecurityJSON(router, route.method, route.path, `{}`, contractSessionToken(t, manager))
			if res.Code != 403 {
				t.Fatalf("frozen %d", res.Code)
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "account-frozen.json"))
		})
	}
	t.Run("disabled engine and validation", func(t *testing.T) {
		conn, router := setupAdminSearchTest(t)
		manager := createContractSiteManager(t, conn)
		token := contractSessionToken(t, manager)
		res := serveAuthSecurityJSON(router, "GET", "/api/admin/search/indexes", "", token)
		if res.Code != 200 {
			t.Fatalf("status %d %s", res.Code, res.Body.String())
		}
		actual := decodeContractEnvelope(t, res)
		expected := contractFixture(t, "admin-search-status.json")
		actualStatus := searchContractResult(t, actual)
		expectedStatus := searchContractResult(t, expected)
		expectedStatus["maintenanceEnabled"] = true
		if actualStatus["configured"] != (meiliconnect.GetClient() != nil) {
			t.Fatal("incorrect configured state")
		}
		// Engine observations depend on the optional local test configuration.
		for _, key := range []string{"configured", "available", "engineVersion"} {
			expectedStatus[key] = actualStatus[key]
		}
		for i, item := range actualStatus["indexes"].([]any) {
			for _, key := range []string{"exists", "documents", "indexing"} {
				expectedStatus["indexes"].([]any)[i].(map[string]any)[key] = item.(map[string]any)[key]
			}
		}
		expected.Result, _ = json.Marshal(expectedStatus)
		assertFixtureEnvelope(t, actual, expected)
		for _, body := range []string{`{}`, `{"index":"not-ours","action":"rebuild"}`, `{"index":"all","action":"delete"}`} {
			res = serveAuthSecurityJSON(router, "POST", "/api/admin/search/maintenance", body, token)
			assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "admin-search-invalid.json"))
		}
		res = serveAuthSecurityJSON(router, "POST", "/api/admin/search/maintenance", `{"index":"all","action":"check"}`, token)
		if meiliconnect.GetClient() == nil {
			if res.Code != 503 {
				t.Fatalf("unconfigured %d", res.Code)
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "admin-search-unavailable.json"))
		} else {
			if res.Code != 200 {
				t.Fatalf("submission %d %s", res.Code, res.Body.String())
			}
			actual := decodeContractEnvelope(t, res)
			fixture := contractFixture(t, "admin-search-submission.json")
			job := searchContractResult(t, actual)["job"].(map[string]any)
			fixtureResult := searchContractResult(t, fixture)
			expectedJob := fixtureResult["job"].(map[string]any)
			if job["requestedBy"] != float64(manager.Id) {
				t.Fatal("actor not audited")
			}
			for _, key := range []string{"id", "createdAt", "updatedAt", "requestedBy"} {
				expectedJob[key] = job[key]
			}
			fixture.Result, _ = json.Marshal(fixtureResult)
			assertFixtureEnvelope(t, actual, fixture)
			res = serveAuthSecurityJSON(router, "POST", "/api/admin/search/maintenance", `{"index":"courses","action":"rebuild"}`, token)
			duplicate := searchContractResult(t, decodeContractEnvelope(t, res))
			if duplicate["created"] != false || duplicate["job"].(map[string]any)["id"] != job["id"] {
				t.Fatal("duplicate created another task")
			}
		}
	})
	t.Run("cookie CSRF", func(t *testing.T) {
		conn, router := setupAdminSearchTest(t)
		manager := createContractSiteManager(t, conn)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/search/maintenance", strings.NewReader(`{"index":"all","action":"rebuild"}`))
		req.AddCookie(&http.Cookie{Name: "access_token", Value: contractSessionToken(t, manager)})
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "https://other.example")
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatalf("CSRF %d %s", res.Code, res.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, res), contractFixture(t, "csrf-rejected.json"))
	})
}

func searchContractResult(t *testing.T, envelope contractEnvelope) map[string]any {
	t.Helper()
	var result map[string]any
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatal(err)
	}
	return result
}
