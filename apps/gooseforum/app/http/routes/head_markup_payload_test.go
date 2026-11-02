package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// Custom head markup belongs to server rendering, not the client page protocol.
// Keep both paths under the real router so omitting the duplicate JSON cannot
// accidentally remove the administrator's configured HTML head elements.
func TestSiteSettingsHeadMarkupIsServerOnly(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")
	persistContractPageConfig(t, dbconnect.Connect(), pageConfig.SiteSettings, pageConfig.SiteSettingsConfig{
		SiteName:      "Head payload test",
		ExternalLinks: `<meta name="d28-head-only" content="preserved"/>`,
	})
	hotdataserve.ClearSiteSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearSiteSettingsConfigCache)

	for _, path := range []string{"/", "/login", "/search?q=test"} {
		t.Run(path, func(t *testing.T) {
			html := fetchSSRHTML(t, router, path)
			headEnd := strings.Index(html, "</head>")
			if headEnd < 0 {
				t.Fatal("HTML has no head closing tag")
			}
			assertTagCount(t, path, html[:headEnd], `<meta name="d28-head-only" content="preserved"/>`, 1)
			payload := regexp.MustCompile(`(?s)<script id="goose-payload" type="application/json">(.*?)</script>`).FindStringSubmatch(html)
			if len(payload) != 2 {
				t.Fatal("HTML has no embedded page payload")
			}
			assertNoClientHeadMarkup(t, []byte(payload[1]))

			request := httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil)
			request.Header.Set("X-Goose-Page", "true")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("page JSON status=%d, want 200", response.Code)
			}
			assertNoClientHeadMarkup(t, response.Body.Bytes())
		})
	}
}

func assertNoClientHeadMarkup(t *testing.T, data []byte) {
	t.Helper()
	var payload struct {
		Layout struct {
			Site map[string]json.RawMessage `json:"site"`
		} `json:"layout"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode page JSON: %v", err)
	}
	if string(payload.Layout.Site["name"]) != `"Head payload test"` {
		t.Fatal("expected public site metadata is missing")
	}
	if _, present := payload.Layout.Site["externalLinks"]; present {
		t.Error("server-only head markup is duplicated in client site payload")
	}
	if strings.Contains(string(data), "d28-head-only") {
		t.Error("custom head markup leaked into client JSON")
	}
}
