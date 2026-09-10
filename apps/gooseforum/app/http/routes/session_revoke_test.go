package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"gorm.io/gorm"
)

// sessionRevokeItem mirrors the wire shape of one session entry returned by
// GET /api/user/sessions, used here only to locate the session IDs to revoke.
type sessionRevokeItem struct {
	ID        uint64 `json:"id"`
	IsCurrent bool   `json:"isCurrent"`
}

func setupSessionRevokeTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.GET("/user/sessions", UpButterReq(api.ListSessions))
	loginAPI.POST("/user/sessions/revoke", UpButterReq(api.RevokeSession))
	return conn, router
}

func serveSessionRevokeJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func sessionIDsByCurrent(t *testing.T, router http.Handler, token string) (current uint64, others []uint64) {
	t.Helper()
	recorder := serveSessionRevokeJSON(router, http.MethodGet, "/api/user/sessions", "", token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list sessions status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Result []sessionRevokeItem `json:"result"`
		Code   int                 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode session list %q: %v", recorder.Body.String(), err)
	}
	for _, item := range envelope.Result {
		if item.IsCurrent {
			current = item.ID
		} else {
			others = append(others, item.ID)
		}
	}
	if current == 0 {
		t.Fatal("session list has no current-session entry")
	}
	return current, others
}

func assertMessageCode(t *testing.T, recorder *httptest.ResponseRecorder, want component.MessageCode) {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (legacy envelope): %s", recorder.Code, recorder.Body.String())
	}
	var envelope struct {
		Code        int    `json:"code"`
		MessageCode string `json:"messageCode"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	if envelope.MessageCode != string(want) {
		t.Fatalf("messageCode = %q, want %q: %s", envelope.MessageCode, want, recorder.Body.String())
	}
}

func TestRevokeSessionValidation(t *testing.T) {
	t.Run("malformed body is a validation failure, not session.notFound", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", "{", token)
		assertMessageCode(t, recorder, component.MessageRequestInvalidParams)
	})

	t.Run("missing id is a validation failure", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", `{}`, token)
		assertMessageCode(t, recorder, component.MessageRequestInvalidParams)
	})

	t.Run("zero id is a validation failure", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", `{"id":0}`, token)
		assertMessageCode(t, recorder, component.MessageRequestInvalidParams)
	})
}

func TestRevokeSessionBehaviorPreserved(t *testing.T) {
	t.Run("unknown session id still reports session.notFound", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", `{"id":999999999}`, token)
		assertMessageCode(t, recorder, component.MessageSessionNotFound)
	})

	t.Run("current session is still not revocable", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		current, _ := sessionIDsByCurrent(t, router, token)
		body, err := json.Marshal(map[string]uint64{"id": current})
		if err != nil {
			t.Fatalf("marshal revoke request: %v", err)
		}
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", string(body), token)
		assertMessageCode(t, recorder, component.MessageSessionCurrentNotRevocable)
	})

	t.Run("valid revoke still succeeds and kills the token", func(t *testing.T) {
		conn, router := setupSessionRevokeTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		otherToken := contractSessionToken(t, user)
		_, others := sessionIDsByCurrent(t, router, token)
		if len(others) != 1 {
			t.Fatalf("non-current session count = %d, want 1", len(others))
		}
		body, err := json.Marshal(map[string]uint64{"id": others[0]})
		if err != nil {
			t.Fatalf("marshal revoke request: %v", err)
		}
		recorder := serveSessionRevokeJSON(router, http.MethodPost, "/api/user/sessions/revoke", string(body), token)
		assertMessageCode(t, recorder, component.MessageSessionRevokeSuccess)

		revoked := serveSessionRevokeJSON(router, http.MethodGet, "/api/user/sessions", "", otherToken)
		if revoked.Code != http.StatusUnauthorized {
			t.Fatalf("revoked token status = %d, want 401", revoked.Code)
		}
	})
}
