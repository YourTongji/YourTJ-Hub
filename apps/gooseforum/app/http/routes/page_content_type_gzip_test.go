package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHTMLPageContentTypeUnderGzip guards the homepage HTML document against
// the compression path dropping Content-Type (browser renders the page as raw
// text). renderPageWithStatus streams template output without an explicit
// Content-Type; with the view-route gzip middleware enabled the response
// headers are committed by the compression writer before Go's sniffing can
// fill Content-Type in, so browsers receive Content-Encoding: gzip with no
// Content-Type and, combined with X-Content-Type-Options: nosniff, display
// the decoded HTML as text/plain. The renderer must set Content-Type itself.
func TestHTMLPageContentTypeUnderGzip(t *testing.T) {
	router := securityHeadersTestRouter(t, "production")

	for _, path := range []string{"/", "/login", "/search", "/links"} {
		request := httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil)
		request.Header.Set("Accept", "text/html")
		request.Header.Set("Accept-Encoding", "gzip")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d, want 200 (body=%s)", path, recorder.Code, recorder.Body.String())
		}
		if got := recorder.Header().Get("Content-Encoding"); got != "gzip" {
			t.Fatalf("GET %s expected the gzip path to be exercised, got Content-Encoding %q", path, got)
		}
		if got := recorder.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Fatalf("GET %s with Accept-Encoding: gzip returned Content-Type %q, want text/html; charset=utf-8", path, got)
		}
	}
}
