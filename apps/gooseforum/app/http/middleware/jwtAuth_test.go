package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	jwt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpChallenges"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

func setupSessionAuthTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
		&userTotpChallenges.Entity{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userTotpChallenges.Entity{})
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&pointsRecord.Entity{})
	conn.Where("1 = 1").Delete(&userPoints.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func requestWithJWT(token string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthCheck)
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestJWTAuthRejectsTokenWithoutSessionRecord(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("sessionless", "password", "sessionless@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	// Token signed with a jti that has no user_sessions row.
	token, _, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken() error = %v", err)
	}
	recorder := requestWithJWT(token)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no session row)", recorder.Code)
	}
}

func TestJWTAuthRejectsRevokedSession(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("revoked", "password", "revoked@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	token, jti, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken() error = %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("sessionservice.Create() error = %v", err)
	}

	// Before revocation the token is accepted.
	if recorder := requestWithJWT(token); recorder.Code != http.StatusOK {
		t.Fatalf("pre-revoke status = %d, want 200", recorder.Code)
	}

	// After revocation the token is rejected immediately.
	if err := sessionservice.RevokeByID(user.Id, sessionservice.GetValidByJti(jti).Id); err != nil {
		t.Fatalf("RevokeByID() error = %v", err)
	}
	if recorder := requestWithJWT(token); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("post-revoke status = %d, want 401", recorder.Code)
	}
}

func TestJWTAuthRejectsChallengeToken(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("challenge", "password", "challenge@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	challenge, err := jwt.CreateChallengeToken(user.Id, user.TokenVersion, jwt.PurposeTotpChallenge, 5*time.Minute)
	if err != nil {
		t.Fatalf("CreateChallengeToken() error = %v", err)
	}
	recorder := requestWithJWT(challenge)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (challenge token has no session)", recorder.Code)
	}
}
