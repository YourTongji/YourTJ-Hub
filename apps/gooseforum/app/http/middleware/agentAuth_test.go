package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
)

func setupAgentAuthTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &agents.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatalf("migrate agent auth tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

// agentAuthRouter registers the middleware in front of a handler that echoes
// the resolved userId.
func agentAuthRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(AgentAuth)
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"userId": c.GetUint64("userId")})
	})
	return router
}

func agentRequest(router http.Handler, authorization string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestAgentAuthSetsUserIdAndTouchesLastUsed(t *testing.T) {
	setupAgentAuthTestDB(t)
	result, err := agentservice.Create(agentservice.CreateParams{Username: "auth-ok-agent"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	rec := agentRequest(agentAuthRouter(), "Bearer "+result.Token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		UserId uint64 `json:"userId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	if body.UserId != result.Agent.UserId {
		t.Fatalf("userId = %d, want %d", body.UserId, result.Agent.UserId)
	}
	stored := agents.GetByUserID(result.Agent.UserId)
	if stored == nil || stored.LastUsedAt == nil {
		t.Fatal("last_used_at should be updated after a successful resolve")
	}
}

// TestAgentAuthFailureMatrix verifies every rejected credential resolves to
// the byte-identical 401 envelope with messageCode auth.required and never
// leaks token material.
func TestAgentAuthFailureMatrix(t *testing.T) {
	setupAgentAuthTestDB(t)
	result, err := agentservice.Create(agentservice.CreateParams{Username: "auth-fail-agent"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	token := result.Token

	// Wrong random tail sharing the same prefix: same-length token whose hash
	// cannot match.
	wrongTail := token[:len("agt_")+8] + strings.Repeat("Z", len(token)-len("agt_")-8)

	// A valid token for a second agent, then disabled.
	other, err := agentservice.Create(agentservice.CreateParams{Username: "auth-disabled-agent"})
	if err != nil {
		t.Fatalf("create disabled agent: %v", err)
	}
	if err := agentservice.Disable(other.Agent.UserId); err != nil {
		t.Fatalf("disable agent: %v", err)
	}
	disabledToken := other.Token

	// A valid token whose bot user is frozen.
	frozen, err := agentservice.Create(agentservice.CreateParams{Username: "auth-frozen-agent"})
	if err != nil {
		t.Fatalf("create frozen agent: %v", err)
	}
	if err := db.Connect().Model(&users.EntityComplete{}).
		Where("id = ?", frozen.User.Id).
		Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatalf("freeze bot: %v", err)
	}
	frozenToken := frozen.Token

	// A valid agent token whose row points at a human user exercises the
	// non-bot rejection after prefix and hash verification.
	human := users.EntityComplete{Username: "auth-human", IsActivated: users.ActivationSuccess}
	if err := db.Connect().Create(&human).Error; err != nil {
		t.Fatalf("create human: %v", err)
	}
	humanToken, err := agentservice.GenerateToken()
	if err != nil {
		t.Fatalf("generate human-linked token: %v", err)
	}
	if err := db.Connect().Create(&agents.Entity{
		UserId:      human.Id,
		TokenPrefix: humanToken.Prefix,
		TokenHash:   humanToken.Hash,
		Enabled:     agents.StatusEnabled,
	}).Error; err != nil {
		t.Fatalf("create human-linked agent row: %v", err)
	}

	// A valid token whose agents row is deleted (soft-deleted row path).
	deleted, err := agentservice.Create(agentservice.CreateParams{Username: "auth-deleted-agent"})
	if err != nil {
		t.Fatalf("create deleted agent: %v", err)
	}
	if err := db.Connect().Where("user_id = ?", deleted.Agent.UserId).Delete(&agents.Entity{}).Error; err != nil {
		t.Fatalf("delete agent row: %v", err)
	}
	deletedToken := deleted.Token

	cases := []struct {
		name  string
		authz string
	}{
		{"missing header", ""},
		{"malformed scheme", "Basic " + token},
		{"empty bearer", "Bearer "},
		{"bearer without prefix", "Bearer not-an-agent-token"},
		{"unknown token", "Bearer " + strings.Repeat("agt_unknown_", 4)},
		{"wrong hash tail", "Bearer " + wrongTail},
		{"disabled agent", "Bearer " + disabledToken},
		{"frozen bot", "Bearer " + frozenToken},
		{"deleted agent row", "Bearer " + deletedToken},
		{"non-bot user", "Bearer " + humanToken.Token},
	}

	router := agentAuthRouter()
	canonical := ""
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := agentRequest(router, tc.authz)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			body := rec.Body.String()
			var envelope struct {
				Result      any    `json:"result"`
				Code        int    `json:"code"`
				MessageCode string `json:"messageCode"`
			}
			if err := json.Unmarshal([]byte(body), &envelope); err != nil {
				t.Fatalf("decode 401 envelope %q: %v", body, err)
			}
			if envelope.Code != 1 || envelope.MessageCode != "auth.required" {
				t.Fatalf("envelope = %#v, want code=1 messageCode=auth.required", envelope)
			}
			if strings.Contains(body, token) || strings.Contains(body, "agt_") {
				t.Fatalf("401 body leaks token material: %q", body)
			}
			if canonical == "" {
				canonical = body
			} else if body != canonical {
				t.Fatalf("401 body differs from canonical:\n got: %s\nwant: %s", body, canonical)
			}
		})
	}
}

// TestAgentAuthIgnoresCookiesAndFallbackCredentials verifies that a human
// session cookie or other credential sources never authenticate an Agent
// request.
func TestAgentAuthIgnoresCookiesAndFallbackCredentials(t *testing.T) {
	setupAgentAuthTestDB(t)
	result, err := agentservice.Create(agentservice.CreateParams{Username: "auth-cookie-agent"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "access_token", Value: result.Token})
	rec := httptest.NewRecorder()
	agentAuthRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-only status = %d, want 401", rec.Code)
	}

	// Lowercase scheme is not the canonical Bearer form and must be rejected.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer "+result.Token)
	rec = httptest.NewRecorder()
	agentAuthRouter().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("lowercase bearer status = %d, want 401", rec.Code)
	}
}
