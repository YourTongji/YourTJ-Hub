package forum

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCourseCatalogPageRequestReturnsPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/courses", CourseCatalog)

	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
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
	body := recorder.Body.String()
	if !strings.Contains(body, `"component":"course.index"`) {
		t.Fatalf("expected course.index component in payload: %s", body)
	}
}

func TestCourseCatalogHTMLReturnsNoJSContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/courses", CourseCatalog)

	req := httptest.NewRequest(http.MethodGet, "/courses", nil)
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
}

func TestCourseDetailPageNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/courses/:courseId", CourseDetail)

	req := httptest.NewRequest(http.MethodGet, "/courses/999999", nil)
	req.Header.Set("X-Goose-Page", "true")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
