package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupExternalLinkPolicyContractTest 注册公开的 external-link-policy 路由并迁移
// page_config 表（数据源），与 route4api.go 的生产注册一致（ginUpNP + 无鉴权）。
func setupExternalLinkPolicyContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate external-link-policy contract tables: %v", err)
	}
	hotdataserve.ClearSiteSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearSiteSettingsConfigCache)

	router.GET("/api/external-link-policy", ginUpNP(api.GetExternalLinkPolicy))
	return conn, router
}

// TestPublicExternalLinkPolicyHTTPContract 覆盖公开白名单下发的两种形态：
// 已保存配置（fixture 契约）与升级前存量行 / 未配置（内置默认清单兜底）。
func TestPublicExternalLinkPolicyHTTPContract(t *testing.T) {
	t.Run("stored whitelist is served to anonymous clients", func(t *testing.T) {
		conn, router := setupExternalLinkPolicyContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{
			SiteName:               "契约站点",
			TrustedExternalDomains: []string{"github.com", "bilibili.com"},
		})
		rec := serveExternalLinkPolicy(router)
		if rec.Code != http.StatusOK {
			t.Fatalf("external-link-policy status = %d, want 200: %s", rec.Code, rec.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, rec), contractFixture(t, "external-link-policy-success.json"))
	})

	t.Run("legacy row without the field falls back to the built-in whitelist", func(t *testing.T) {
		conn, router := setupExternalLinkPolicyContractTest(t)
		// 升级前落库的站点设置行没有 trustedExternalDomains 键（issue #1200）：
		// 缺失按内置默认清单补齐，而不是「显式清空」的空数组。
		persistContractPageConfig(t, conn, pageConfig.SiteSettings, json.RawMessage(`{"siteName":"旧站点"}`))
		domains := decodeExternalLinkPolicyDomains(t, serveExternalLinkPolicy(router))
		if want := defaultconfig.GetDefaultSiteSettingsConfig().TrustedExternalDomains; !slices.Equal(domains, want) {
			t.Fatalf("trustedDomains = %#v, want the built-in default %#v", domains, want)
		}
	})

	t.Run("unconfigured site serves the built-in whitelist", func(t *testing.T) {
		_, router := setupExternalLinkPolicyContractTest(t)
		domains := decodeExternalLinkPolicyDomains(t, serveExternalLinkPolicy(router))
		if want := defaultconfig.GetDefaultSiteSettingsConfig().TrustedExternalDomains; !slices.Equal(domains, want) {
			t.Fatalf("trustedDomains = %#v, want the built-in default %#v", domains, want)
		}
	})
}

// serveExternalLinkPolicy 匿名 GET /api/external-link-policy。
func serveExternalLinkPolicy(router *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/api/external-link-policy", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

// decodeExternalLinkPolicyDomains 解出公开下发的白名单并断言成功信封。
func decodeExternalLinkPolicyDomains(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("external-link-policy status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Code   int `json:"code"`
		Result struct {
			TrustedDomains []string `json:"trustedDomains"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode external-link-policy envelope %s: %v", rec.Body.String(), err)
	}
	if envelope.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", envelope.Code, rec.Body.String())
	}
	return envelope.Result.TrustedDomains
}
