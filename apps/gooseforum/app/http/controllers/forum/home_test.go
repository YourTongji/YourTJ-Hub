package forum

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

func TestHomePageRequestReturnsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", Home)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Goose-Page", "true")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") == "" {
		t.Fatal("expected JSON content type")
	}
	if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("expected page payload response to avoid browser cache, got %q", got)
	}
}

func TestHomeHTMLReturnsNoJSContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", Home)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `id="goose-app"`) {
		t.Fatalf("expected app mount point in HTML: %s", body)
	}
	if !strings.Contains(body, `id="goose-payload"`) {
		t.Fatalf("expected initial payload in HTML: %s", body)
	}
	if !strings.Contains(body, `<noscript>`) {
		t.Fatalf("expected noscript fallback in HTML: %s", body)
	}
	if strings.Contains(body, `goose-seo-content`) {
		t.Fatalf("expected no hidden SEO duplicate in HTML: %s", body)
	}
}

func TestResourceEntryUsesViteInLocalMode(t *testing.T) {
	if got := viteDevServerFor("", false); got != "http://localhost:3010" {
		t.Fatalf("expected local mode to default to Vite dev server, got %q", got)
	}
	if got := viteDevServerFor("http://127.0.0.1:4173", true); got != "http://127.0.0.1:4173" {
		t.Fatalf("expected explicit dev server to win, got %q", got)
	}
	if got := viteDevServerFor("", true); got != "" {
		t.Fatalf("expected production mode without override to use manifest, got %q", got)
	}
}

func TestResourceEntryUsesConfiguredDevServer(t *testing.T) {
	previousEnv := preferences.GetString("app.env", "production")
	previousDevServer := preferences.GetString("resource.devServer", "")
	previousDevBase := preferences.GetString("resource.devBase", "/assets/")
	t.Cleanup(func() {
		preferences.Set("app.env", previousEnv)
		preferences.Set("resource.devServer", previousDevServer)
		preferences.Set("resource.devBase", previousDevBase)
	})

	preferences.Set("app.env", "local")
	preferences.Set("resource.devServer", "http://127.0.0.1:4173")
	preferences.Set("resource.devBase", "/dev-assets/")

	html := string(resourceEntry("src/site/main.ts"))
	if !strings.Contains(html, `/dev-assets/@vite/client`) {
		t.Fatalf("expected resource entry to include Vite client: %s", html)
	}
	if !strings.Contains(html, `/dev-assets/src/site/main.ts`) {
		t.Fatalf("expected resource entry to include Vite entry: %s", html)
	}
	if strings.Contains(html, `http://127.0.0.1:4173`) {
		t.Fatalf("expected same-origin resource paths instead of the Vite server URL: %s", html)
	}
}
