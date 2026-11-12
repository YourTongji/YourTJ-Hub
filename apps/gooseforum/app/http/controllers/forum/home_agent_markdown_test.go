package forum

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHomeMarkdownNegotiationAndDiscoveryLinks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", Home)

	html := performHome(t, router, "")
	if html.Code != http.StatusOK {
		t.Fatalf("HTML status = %d, want 200: %s", html.Code, html.Body.String())
	}
	if got := html.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("default content-type = %q, want text/html", got)
	}
	if !strings.Contains(html.Body.String(), `id="goose-app"`) {
		t.Fatalf("default body is not the HTML homepage: %s", html.Body.String())
	}
	assertHomepageLLMSLinks(t, html.Header().Get("Link"))

	markdown := performHome(t, router, "text/markdown")
	if markdown.Code != http.StatusOK {
		t.Fatalf("markdown status = %d, want 200: %s", markdown.Code, markdown.Body.String())
	}
	if got := markdown.Header().Get("Content-Type"); !strings.Contains(got, "text/markdown") {
		t.Fatalf("markdown content-type = %q, want text/markdown", got)
	}
	body := markdown.Body.String()
	if strings.Contains(body, `id="goose-app"`) || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("markdown body is still HTML: %s", body)
	}
	if !strings.Contains(body, "# ") {
		t.Fatalf("markdown body = %q, want a markdown heading", body)
	}
	assertHomepageLLMSLinks(t, markdown.Header().Get("Link"))

	equal := performHome(t, router, "text/html, text/markdown")
	if got := equal.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Fatalf("equal-q Accept content-type = %q, want HTML to remain the default", got)
	}

	page := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	page.Header.Set("Accept", "text/markdown")
	page.Header.Set("X-Goose-Page", "true")
	pageRec := httptest.NewRecorder()
	router.ServeHTTP(pageRec, page)
	if got := pageRec.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("page payload content-type = %q, want JSON", got)
	}
	assertHomepageLLMSLinks(t, pageRec.Header().Get("Link"))
}

func performHome(t *testing.T, router http.Handler, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func assertHomepageLLMSLinks(t *testing.T, link string) {
	t.Helper()
	for _, want := range []string{"</llms.txt>", "</llms-full.txt>", `rel="describedby"`} {
		if !strings.Contains(link, want) {
			t.Fatalf("Link = %q, want to contain %q", link, want)
		}
	}
}
