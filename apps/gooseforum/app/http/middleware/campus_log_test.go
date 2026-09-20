package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCampusCallbackLogQueryExcludesCredentials(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?code=secret&state=private", nil)
	if logQuery(r.URL) != "" {
		t.Fatal("authorization callback query may enter logs")
	}
	r = httptest.NewRequest(http.MethodGet, "/search?q=hello", nil)
	if logQuery(r.URL) != "q=hello" {
		t.Fatal("ordinary query removed")
	}
}
