package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/themeservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupSiteThemePublicContractTest 注册公开的 site-theme/tokens 路由并迁移
// page_config 表（数据源），与 route4api.go 的生产注册一致（ginUpNP + 无鉴权）。
func setupSiteThemePublicContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate site-theme public contract tables: %v", err)
	}
	themeservice.ClearCaches()
	t.Cleanup(themeservice.ClearCaches)

	router.GET("/api/site-theme/tokens", ginUpNP(api.GetPublicSiteThemeTokens))
	return conn, router
}

// TestPublicSiteThemeTokensHTTPContract disabled（未配置/未发布）+ enabled（发布态）。
func TestPublicSiteThemeTokensHTTPContract(t *testing.T) {
	t.Run("disabled when no published theme config", func(t *testing.T) {
		_, router := setupSiteThemePublicContractTest(t)
		rec := serveSiteThemeTokens(router)
		if rec.Code != http.StatusOK {
			t.Fatalf("site-theme tokens status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "site-theme-tokens-disabled.json"))
	})

	t.Run("enabled returns published themes and tokens", func(t *testing.T) {
		conn, router := setupSiteThemePublicContractTest(t)
		// 与 fixtures/site-theme-tokens-enabled.json 完全一致（version 3 发布态，
		// gf-light/gf-dark 全 token 合法值，normalize 恒等）。
		persistContractPageConfig(t, conn, pageConfig.SiteTheme, json.RawMessage(contractSiteThemeConfigJSON))
		rec := serveSiteThemeTokens(router)
		if rec.Code != http.StatusOK {
			t.Fatalf("site-theme tokens status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "site-theme-tokens-enabled.json"))
	})
}

// serveSiteThemeTokens 匿名 GET /api/site-theme/tokens。
func serveSiteThemeTokens(router *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/site-theme/tokens", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}
