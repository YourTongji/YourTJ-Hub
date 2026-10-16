package routes

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func TestPushPreferencesHTTPContract(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	response := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "push-preferences-defaults.json"))
	response = serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", `{"chat":false}`, token)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var envelope struct {
		Code   int `json:"code"`
		Result struct {
			Chat        bool `json:"chat"`
			System      bool `json:"system"`
			DefaultOnly bool `json:"systemDefaultOnly"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Code != 0 || envelope.Result.Chat || !envelope.Result.System || !envelope.Result.DefaultOnly {
		t.Fatalf("partial override changed selective defaults: %+v", envelope)
	}
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", contractSessionToken(t, other))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "push-preferences-defaults.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/push/preferences", "", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status %d", response.Code)
	}
}

func TestPushPreferencesPUTParsingHTTPContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"malformed", `{"chat":`},
		{"wrong type", `{"chat":"false"}`},
		{"unknown field", `{"chat":true,"bogus":1}`},
		{"trailing second value", `{"chat":true}{"system":false}`},
		{"trailing garbage", `{"chat":true} x`},
		{"null body", `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, router := setupNotificationChatContractTest(t)
			user := createHTTPContractUser(t, conn, contractTestID())
			recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", tc.body, contractSessionToken(t, user))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
		})
	}

	t.Run("legal trailing whitespace stays 200", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", "{\"chat\":false}\n \t", contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("oversize body returns 400", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		// Content-Length is used directly (unlike the chunked path exercised by
		// the forum post parser), so a >4KiB body is rejected before decoding.
		oversize := `{"chat":` + strings.Repeat("0", 5<<10) + `}`
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", oversize, contractSessionToken(t, user))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "parse-failed.json"))
	})
}

func TestPushPreferencesPUTFrozenAndRateLimitHTTPContract(t *testing.T) {
	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatal(err)
		}
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", `{"chat":false}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
	})
	t.Run("rate limited returns 429 with retry metadata", func(t *testing.T) {
		conn, router := setupNotificationChatContractTest(t)
		restrictContractRateLimit(t, conn, middleware.RateLimitInteract)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		for attempt := 0; attempt < 5; attempt++ {
			recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", `{"chat":false}`, token)
			if recorder.Code != http.StatusOK {
				t.Fatalf("attempt %d status = %d, want 200: %s", attempt+1, recorder.Code, recorder.Body.String())
			}
		}
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/forum/push/preferences", `{"chat":false}`, token)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("rate limited status = %d, want 429: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		assertFixtureEnvelope(t, response, contractFixture(t, "account-close-rate-limited.json"))
		assertRetryAfter(t, recorder, response, middleware.RateLimitInteract)
	})
}
