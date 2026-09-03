package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// csrfTestRouter builds a gin engine with CSRFProtection mounted in front of a
// POST handler, mirroring the production write-group order (CSRF runs before
// the authentication middleware). The handler records that it ran.
func csrfTestRouter() (*gin.Engine, *bool) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	reached := false
	router.POST("/write", CSRFProtection, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	router.GET("/read", CSRFProtection, func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.OPTIONS("/write", CSRFProtection, func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	return router, &reached
}

// csrfRequest builds a POST request with an optional access_token cookie,
// optional Authorization header, and optional Origin/Referer.
func csrfRequest(t *testing.T, cookie, authorization, origin, referer string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "http://forum.example.test/write", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "access_token", Value: cookie})
	}
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	if origin != "" {
		request.Header.Set("Origin", origin)
	}
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	return request
}

func performCsrf(router http.Handler, request *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func assertCsrfAllowed(t *testing.T, recorder *httptest.ResponseRecorder, reached *bool) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", recorder.Code, recorder.Body.String())
	}
	if !*reached {
		t.Fatal("handler did not run")
	}
}

func assertCsrfRejected(t *testing.T, recorder *httptest.ResponseRecorder, reached *bool) {
	t.Helper()
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %s)", recorder.Code, recorder.Body.String())
	}
	if *reached {
		t.Fatal("handler ran despite CSRF rejection")
	}
	var envelope struct {
		Code        int    `json:"code"`
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode CSRF rejection body %q: %v", recorder.Body.String(), err)
	}
	if envelope.MessageCode != "auth.csrf.rejected" {
		t.Fatalf("messageCode = %q, want auth.csrf.rejected", envelope.MessageCode)
	}
}

func TestCSRFProtectionSameOriginCookiePostAllowed(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "http://forum.example.test", "")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionDefaultPortNormalized(t *testing.T) {
	router, reached := csrfTestRouter()
	// Browsers omit the default port from Origin while Host keeps it implicit.
	request := csrfRequest(t, "session-token", "", "http://forum.example.test:80", "")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionHttpsViaForwardedProto(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "https://forum.example.test", "")
	request.Header.Set("X-Forwarded-Proto", "https")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionCrossSiteOriginRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "http://evil.example.test", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionSchemeMismatchRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	// https origin against a plain-http request must not pass.
	request := csrfRequest(t, "session-token", "", "https://forum.example.test", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionSubdomainOriginRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "http://sub.forum.example.test", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionMissingOriginAndRefererRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionRefererFallbackAllowed(t *testing.T) {
	router, reached := csrfTestRouter()
	// Origin-less legacy browser POST carrying a same-site Referer.
	request := csrfRequest(t, "session-token", "", "", "http://forum.example.test/publish")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionCrossSiteRefererRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "", "http://evil.example.test/x")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionMalformedOriginRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "not a url", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionNullOriginRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	// Sandboxed iframes (srcdoc/data/blob) and some redirect flows send
	// `Origin: null`; it never matches a real site origin, so it must be
	// rejected fail-closed.
	request := csrfRequest(t, "session-token", "", "null", "")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionMalformedRefererRejected(t *testing.T) {
	router, reached := csrfTestRouter()
	// Malformed/unparseable Referer values (broken IPv6 bracket, host-less
	// scheme) must not accidentally pass the fallback check.
	for _, referer := range []string{"http://[::1", "http://", "not a url"} {
		request := csrfRequest(t, "session-token", "", "", referer)
		assertCsrfRejected(t, performCsrf(router, request), reached)
	}
}

func TestCSRFProtectionInvalidForwardedProtoIgnored(t *testing.T) {
	// An invalid X-Forwarded-Proto value is ignored: the connection scheme is
	// kept, so the plain-http site origin still matches and the request is
	// not spuriously rejected.
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "", "http://forum.example.test", "")
	request.Header.Set("X-Forwarded-Proto", "ftp")
	assertCsrfAllowed(t, performCsrf(router, request), reached)

	// ...and it must not widen the allowed set either: an https Origin against
	// a plain-http request with a garbage forwarded-proto stays rejected.
	*reached = false
	request = csrfRequest(t, "session-token", "", "https://forum.example.test", "")
	request.Header.Set("X-Forwarded-Proto", "garbage")
	assertCsrfRejected(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionForwardedProtoFirstHopWins(t *testing.T) {
	router, reached := csrfTestRouter()
	// Multi-hop proxies send a comma list; only the first hop is trusted.
	request := csrfRequest(t, "session-token", "", "https://forum.example.test", "")
	request.Header.Set("X-Forwarded-Proto", "https, http")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionHeadExemptFromOriginCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.HEAD("/read", CSRFProtection, func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodHead, "http://forum.example.test/read", nil)
	request.AddCookie(&http.Cookie{Name: "access_token", Value: "session-token"})
	request.Header.Set("Origin", "http://evil.example.test")
	recorder := performCsrf(router, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d, want 200", recorder.Code)
	}
}

func TestCSRFProtectionGetExemptFromOriginCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/read", CSRFProtection, func(c *gin.Context) { c.Status(http.StatusOK) })

	request := httptest.NewRequest(http.MethodGet, "http://forum.example.test/read", nil)
	request.AddCookie(&http.Cookie{Name: "access_token", Value: "session-token"})
	request.Header.Set("Origin", "http://evil.example.test")
	recorder := performCsrf(router, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want 200", recorder.Code)
	}
}

func TestCSRFProtectionBearerExemptEvenWithCookieAndForeignOrigin(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "session-token", "Bearer api-client-token", "http://evil.example.test", "")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionNoCookieAnonymousPostAllowed(t *testing.T) {
	router, reached := csrfTestRouter()
	request := csrfRequest(t, "", "", "", "")
	assertCsrfAllowed(t, performCsrf(router, request), reached)
}

func TestCSRFProtectionOptionsExemptFromOriginCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.OPTIONS("/write", CSRFProtection, func(c *gin.Context) { c.Status(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodOptions, "http://forum.example.test/write", nil)
	request.AddCookie(&http.Cookie{Name: "access_token", Value: "session-token"})
	request.Header.Set("Origin", "http://evil.example.test")
	request.Header.Set("Access-Control-Request-Method", "POST")
	recorder := performCsrf(router, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d, want 204", recorder.Code)
	}
}
