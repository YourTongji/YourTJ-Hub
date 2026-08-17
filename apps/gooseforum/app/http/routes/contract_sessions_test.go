package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// contractSessionItem mirrors the wire shape of one session entry returned by
// GET /api/user/sessions (SessionVO in sessionController.go).
type contractSessionItem struct {
	ID        uint64 `json:"id"`
	IPMasked  string `json:"ipMasked"`
	UserAgent string `json:"userAgent"`
	CreatedAt int64  `json:"createdAt"`
	ExpiresAt int64  `json:"expiresAt"`
	IsCurrent bool   `json:"isCurrent"`
}

func setupSessionContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	router.POST("/api/logout", api.Logout)
	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.GET("/user/sessions", UpButterReq(api.ListSessions))
	loginAPI.POST("/user/sessions/revoke", UpButterReq(api.RevokeSession))
	loginAPI.POST("/user/sessions/revoke-all", UpButterReq(api.RevokeAllSessions))
	return conn, router
}

func serveSessionJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// decodeSessionList decodes the success result of GET /api/user/sessions and
// checks that each entry carries exactly the keys pinned by the fixture.
func decodeSessionList(t *testing.T, result json.RawMessage) []contractSessionItem {
	t.Helper()
	var rawItems []map[string]any
	if err := json.Unmarshal(result, &rawItems); err != nil {
		t.Fatalf("decode session list result %q: %v", result, err)
	}
	fixture := contractFixture(t, "sessions-list-success.json")
	var fixtureItems []map[string]any
	if err := json.Unmarshal(fixture.Result, &fixtureItems); err != nil || len(fixtureItems) == 0 {
		t.Fatalf("decode sessions-list fixture result %q: %v", fixture.Result, err)
	}
	fixtureKeys := make([]string, 0, len(fixtureItems[0]))
	for key := range fixtureItems[0] {
		fixtureKeys = append(fixtureKeys, key)
	}
	var items []contractSessionItem
	decoder := json.NewDecoder(bytes.NewReader(result))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&items); err != nil {
		t.Fatalf("session entries drifted from the pinned shape %q: %v", result, err)
	}
	for index, rawItem := range rawItems {
		actualKeys := make([]string, 0, len(rawItem))
		for key := range rawItem {
			actualKeys = append(actualKeys, key)
		}
		if !reflect.DeepEqual(sortedStrings(actualKeys), sortedStrings(fixtureKeys)) {
			t.Fatalf("session entry %d keys = %v, want fixture keys %v", index, actualKeys, fixtureKeys)
		}
	}
	return items
}

func sortedStrings(values []string) []string {
	sorted := make([]string, len(values))
	copy(sorted, values)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	return sorted
}

func currentSessionItem(t *testing.T, items []contractSessionItem) contractSessionItem {
	t.Helper()
	for _, item := range items {
		if item.IsCurrent {
			return item
		}
	}
	t.Fatal("session list has no current-session entry")
	return contractSessionItem{}
}

func TestLogoutHTTPContract(t *testing.T) {
	t.Run("authenticated logout revokes the session and clears the cookie", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		recorder := serveSessionJSON(router, http.MethodPost, "/api/logout", "", token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("logout status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "logout-success.json"))
		if cookie := recorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "access_token=") {
			t.Fatalf("logout response missing access_token clearing cookie: %q", cookie)
		}

		revoked := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", token)
		if revoked.Code != http.StatusUnauthorized {
			t.Fatalf("revoked token status = %d, want 401", revoked.Code)
		}
	})

	t.Run("logout without a session stays an idempotent success", func(t *testing.T) {
		_, router := setupSessionContractTest(t)
		recorder := serveSessionJSON(router, http.MethodPost, "/api/logout", "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("anonymous logout status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "logout-success.json"))
		if cookie := recorder.Header().Get("Set-Cookie"); !strings.Contains(cookie, "access_token=") {
			t.Fatalf("anonymous logout response missing access_token clearing cookie: %q", cookie)
		}
	})

	t.Run("logout twice with the same bearer stays an idempotent success", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		first := serveSessionJSON(router, http.MethodPost, "/api/logout", "", token)
		if first.Code != http.StatusOK {
			t.Fatalf("first logout status = %d, want 200", first.Code)
		}
		second := serveSessionJSON(router, http.MethodPost, "/api/logout", "", token)
		if second.Code != http.StatusOK {
			t.Fatalf("second logout status = %d, want 200: %s", second.Code, second.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, second), contractFixture(t, "logout-success.json"))
		if cookie := second.Header().Get("Set-Cookie"); !strings.Contains(cookie, "access_token=") {
			t.Fatalf("second logout response missing access_token clearing cookie: %q", cookie)
		}
	})
}

func TestListSessionsHTTPContract(t *testing.T) {
	t.Run("success lists live sessions and marks the current one", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		contractSessionToken(t, user)

		recorder := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("list sessions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("list sessions code = %d, want 0: %s", response.Code, recorder.Body.String())
		}
		items := decodeSessionList(t, response.Result)
		if len(items) != 2 {
			t.Fatalf("session count = %d, want 2", len(items))
		}
		current := currentSessionItem(t, items)
		if current.ID == 0 {
			t.Fatal("current session id = 0, want positive")
		}
		if current.IPMasked != "127.0.0.*" {
			t.Fatalf("current session ipMasked = %q, want masked IPv4", current.IPMasked)
		}
		if current.UserAgent != "contract-test" {
			t.Fatalf("current session userAgent = %q, want contract-test", current.UserAgent)
		}
		if current.CreatedAt <= 0 || current.ExpiresAt <= current.CreatedAt {
			t.Fatalf("session timestamps createdAt = %d, expiresAt = %d, want positive and increasing",
				current.CreatedAt, current.ExpiresAt)
		}
		currentCount := 0
		for _, item := range items {
			if item.IsCurrent {
				currentCount++
			}
		}
		if currentCount != 1 {
			t.Fatalf("isCurrent count = %d, want exactly 1", currentCount)
		}
	})

	t.Run("excludes expired rows and never exposes unparseable addresses", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		if err := sessionservice.Create(user.Id, "contract-malformed-ip", "contract-test", "not-an-ip"); err != nil {
			t.Fatalf("create malformed-ip session: %v", err)
		}
		if err := sessionservice.Create(user.Id, "contract-expired-session", "contract-test", "203.0.113.9"); err != nil {
			t.Fatalf("create expired session: %v", err)
		}
		if err := userSessions.UpdateExpiresAtByJti("contract-expired-session", time.Now().Add(-time.Hour)); err != nil {
			t.Fatalf("expire session: %v", err)
		}

		recorder := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("list sessions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		items := decodeSessionList(t, decodeContractEnvelope(t, recorder).Result)
		if len(items) != 2 {
			t.Fatalf("session count = %d, want 2 live rows", len(items))
		}
		for _, item := range items {
			if item.ID == 0 {
				t.Fatal("session id = 0, want positive")
			}
			if item.IPMasked == "not-an-ip" {
				t.Fatal("session list exposed an unparseable raw IP")
			}
		}
		var malformedIPFound bool
		for _, item := range items {
			if item.IPMasked == "" {
				malformedIPFound = true
			}
		}
		if !malformedIPFound {
			t.Fatal("session list did not include the malformed-IP session as an empty mask")
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupSessionContractTest(t)
		recorder := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})
}

func TestRevokeSessionHTTPContract(t *testing.T) {
	t.Run("success revokes the other session", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		otherToken := contractSessionToken(t, user)

		listRecorder := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", token)
		items := decodeSessionList(t, decodeContractEnvelope(t, listRecorder).Result)
		var otherID uint64
		for _, item := range items {
			if !item.IsCurrent {
				otherID = item.ID
			}
		}
		if otherID == 0 {
			t.Fatal("no non-current session to revoke")
		}

		body, err := json.Marshal(map[string]uint64{"id": otherID})
		if err != nil {
			t.Fatalf("marshal revoke request: %v", err)
		}
		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke", string(body), token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("revoke status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "sessions-revoke-success.json"))

		revoked := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", otherToken)
		if revoked.Code != http.StatusUnauthorized {
			t.Fatalf("revoked token status = %d, want 401", revoked.Code)
		}
	})

	t.Run("current session is not revocable", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		listRecorder := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", token)
		current := currentSessionItem(t, decodeSessionList(t, decodeContractEnvelope(t, listRecorder).Result))
		body, err := json.Marshal(map[string]uint64{"id": current.ID})
		if err != nil {
			t.Fatalf("marshal revoke request: %v", err)
		}
		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke", string(body), token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("revoke-current status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "sessions-revoke-current.json"))
	})

	t.Run("unknown session id reports not found", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke", `{"id":999999999}`, token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("revoke-missing status = %d, want 200", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "sessions-revoke-not-found.json"))
	})

	t.Run("malformed, missing, or zero id is a validation failure", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		for name, body := range map[string]string{
			"malformed JSON": "{",
			"missing id":     `{}`,
			"zero id":        `{"id":0}`,
		} {
			recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke", body, token)
			if recorder.Code != http.StatusOK {
				t.Fatalf("%s: status = %d, want 200", name, recorder.Code)
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "invalid-params.json"))
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupSessionContractTest(t)
		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke", `{"id":1}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})
}

func TestRevokeAllSessionsHTTPContract(t *testing.T) {
	t.Run("success invalidates every token of the account", func(t *testing.T) {
		conn, router := setupSessionContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		otherToken := contractSessionToken(t, user)

		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke-all", "", token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("revoke-all status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "sessions-revoke-all-success.json"))

		// The contract promises token version invalidation as the second layer; pin it
		// directly, since deleted session rows alone already make tokens 401.
		reloaded, err := users.Get(user.Id)
		if err != nil {
			t.Fatalf("reload user after revoke-all: %v", err)
		}
		if reloaded.TokenVersion != user.TokenVersion+1 {
			t.Fatalf("tokenVersion = %d, want %d (second-layer invalidation)", reloaded.TokenVersion, user.TokenVersion+1)
		}

		for name, revokedToken := range map[string]string{"current": token, "other": otherToken} {
			revoked := serveSessionJSON(router, http.MethodGet, "/api/user/sessions", "", revokedToken)
			if revoked.Code != http.StatusUnauthorized {
				t.Fatalf("%s token after revoke-all status = %d, want 401", name, revoked.Code)
			}
		}
	})

	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupSessionContractTest(t)
		recorder := serveSessionJSON(router, http.MethodPost, "/api/user/sessions/revoke-all", "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401", recorder.Code)
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})
}
