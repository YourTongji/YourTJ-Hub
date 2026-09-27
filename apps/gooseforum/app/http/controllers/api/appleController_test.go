package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/appleauthservice"
	"github.com/gin-gonic/gin"
)

func TestAppleExchangeRejectsMalformedAndOversizedCredentials(t *testing.T) {
	for _, body := range []string{`{`, `{}`, `{"authorizationCode":"code","identityToken":"` + strings.Repeat("x", 25000) + `","nonce":"` + strings.Repeat("n", 32) + `"}`} {
		r := gin.New()
		r.POST("/exchange", AppleExchange)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatalf("status=%d", w.Code)
		}
	}
}
func TestAppleExchangeFailsClosedWhenUnconfigured(t *testing.T) {
	r := gin.New()
	r.POST("/exchange", AppleExchange)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/exchange", strings.NewReader(`{"authorizationCode":"code","identityToken":"identity","nonce":"01234567890123456789012345678901"}`)))
	if w.Code != 503 || !strings.Contains(w.Body.String(), "oauth.apple.unavailable") {
		t.Fatalf("status=%d body=%s", w.Code, w.Body)
	}
	if w.Header().Get("New-Token") != "" || w.Header().Get("Set-Cookie") != "" {
		t.Fatal("failure issued a session")
	}
}

func TestInvalidAppleBindingProofDoesNotExpireForumSession(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	appleAuthFailure(c, appleauthservice.ErrInvalidCredential, true)
	if recorder.Code != 400 {
		t.Fatalf("invalid Apple proof must not be forum 401: %d", recorder.Code)
	}
}
