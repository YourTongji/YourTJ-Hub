package routes

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// issueHeadMarkup 是 issue #1244 中站点设置里实际生效的那段整段 <head> payload。
const issueHeadMarkup = "<head>\n" +
	"    <meta charset=\"UTF-8\">\n" +
	"    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n" +
	"    <title>YourTJ社区</title>\n" +
	"    <meta name=\"description\" content=\"同济大学学生社区\">\n" +
	"    <meta name=\"keywords\" content=\"forum,discussion,community,YourTJ,同济大学,YourTJ社区\">\n" +
	"    <meta name=\"robots\" content=\"index, follow\">\n" +
	"    <meta name=\"author\" content=\"YourTJ Community\">\n" +
	"</head>\n"

// issueHeadMarkupSanitized 是 issueHeadMarkup 净化后的期望值：只保留自定义 meta。
const issueHeadMarkupSanitized = "<meta name=\"keywords\" content=\"forum,discussion,community,YourTJ,同济大学,YourTJ社区\"/>\n" +
	"<meta name=\"author\" content=\"YourTJ Community\"/>"

// TestSiteSettingsHeadMarkupStaysInsideHead 断言站点设置注入的 head 片段不会造出
// 嵌套 <head> 或重复的 viewport/title/description/robots（issue #1244），
// 同时自定义 meta 仍然进 head —— 证明净化是结构化剥离，不是整段丢弃。
func TestSiteSettingsHeadMarkupStaysInsideHead(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")
	conn := dbconnect.Connect()
	persistContractPageConfig(t, conn, pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{
		SiteName:      "契约站点",
		ExternalLinks: issueHeadMarkup,
	})
	hotdataserve.ClearSiteSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearSiteSettingsConfigCache)

	for _, page := range []struct {
		path string
		// robots 是页面外壳自己的取值（搜索页 noindex）；为空表示该页不输出 robots。
		robots string
	}{
		{path: "/"},
		{path: "/login"},
		{path: "/search?q=test", robots: "noindex,follow"},
	} {
		body := fetchSSRHTML(t, router, page.path)
		assertTagCount(t, page.path, body, `(?i)<head[ >]`, 1)
		assertTagCount(t, page.path, body, `(?i)<title[ >]`, 1)
		assertTagCount(t, page.path, body, `(?i)<meta[^>]+name="viewport"`, 1)
		// 外壳的 description 按页条件输出，注入的那条（连同取值）必须整体消失。
		if strings.Contains(body, "同济大学学生社区") {
			t.Errorf("GET %s: injected description meta survived", page.path)
		}
		if page.robots == "" {
			assertTagCount(t, page.path, body, `(?i)<meta[^>]+name="robots"`, 0)
		} else {
			assertTagCount(t, page.path, body, `(?i)<meta[^>]+name="robots"`, 1)
			if want := `content="` + page.robots + `"`; !strings.Contains(body, want) {
				t.Errorf("GET %s: robots meta missing %s", page.path, want)
			}
		}
		// 自定义内容保留：净化只剥结构标签和外壳自有标签。
		assertTagCount(t, page.path, body, `(?i)<meta[^>]+name="keywords"`, 1)
		assertTagCount(t, page.path, body, `(?i)<meta[^>]+name="author"`, 1)
		assertTagCount(t, page.path, body, `(?i)<meta[^>]+charset`, 1)
	}
}

// TestSaveSiteSettingsSanitizesHeadMarkup 断言写入路径也净化（issue #1244）：整段
// <head> 在落库前就被剥离，存量脏配置在管理员下次保存站点设置时被清理。
func TestSaveSiteSettingsSanitizesHeadMarkup(t *testing.T) {
	conn, router := setupAdminSiteContractTest(t)
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.SiteSettings).Delete(&pageConfig.Entity{})
	})
	body, err := json.Marshal(map[string]any{"settings": map[string]any{
		"siteName":               "契约站点",
		"externalLinks":          issueHeadMarkup,
		"trustedExternalDomains": []string{"github.com"},
	}})
	if err != nil {
		t.Fatalf("encode save-site-settings body: %v", err)
	}
	serveAdminSiteRaw(t, conn, router, http.MethodPost, "/api/admin/save-site-settings", string(body))

	stored := pageConfig.GetConfigByPageType(pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{})
	if stored.ExternalLinks != issueHeadMarkupSanitized {
		t.Fatalf("stored externalLinks = %q, want %q", stored.ExternalLinks, issueHeadMarkupSanitized)
	}
}

func fetchSSRHTML(t *testing.T, router http.Handler, path string) string {
	t.Helper()
	response := performSecurityHeadersRequest(router, http.MethodGet, path, "text/html")
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s status=%d, want 200 (body=%s)", path, response.Code, response.Body.String())
	}
	return response.Body.String()
}

// assertTagCount 统计响应 HTML 中真实标记的出现次数。payload JSON 里的同名片段
// 由 encoding/json 转义（形如 \u003cmeta），不会计入。
func assertTagCount(t *testing.T, path, body, pattern string, want int) {
	t.Helper()
	if got := len(regexp.MustCompile(pattern).FindAllStringIndex(body, -1)); got != want {
		t.Errorf("GET %s: %d matches of %s, want %d", path, got, pattern, want)
	}
}
