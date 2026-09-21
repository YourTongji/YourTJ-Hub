package routes

import (
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
)

func setupLinkPreviewContractTest(t *testing.T) http.Handler {
	t.Helper()
	_, router := setupHTTPContractTest(t)
	router.POST(
		"/api/link-previews/resolve",
		middleware.CSRFProtection,
		middleware.JWTAuth,
		middleware.RateLimit(middleware.RateLimitLinkPreview),
		UpLimitedJsonReq(16<<10, api.ResolveLinkPreviews),
	)
	return router
}

func TestResolveLinkPreviewsRejectsOversizedBody(t *testing.T) {
	router := setupLinkPreviewContractTest(t)
	recorder := serveAuthSecurityJSON(
		router,
		http.MethodPost,
		"/api/link-previews/resolve",
		`{"urls":["https://example.com/`+strings.Repeat("x", 17<<10)+`"]}`,
		"",
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d, want 400: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
}

func TestResolveLinkPreviewsHTTPContract(t *testing.T) {
	router := setupLinkPreviewContractTest(t)
	recorder := serveAuthSecurityJSON(
		router,
		http.MethodPost,
		"/api/link-previews/resolve",
		`{"urls":["javascript:alert(1)","//example.com"]}`,
		"",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("link preview status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "link-previews-resolve-success.json"))
}

func TestResolveLinkPreviewsRejectsOversizedBatch(t *testing.T) {
	router := setupLinkPreviewContractTest(t)
	recorder := serveAuthSecurityJSON(
		router,
		http.MethodPost,
		"/api/link-previews/resolve",
		`{"urls":["a","b","c","d","e","f"]}`,
		"",
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("oversized batch status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
}
