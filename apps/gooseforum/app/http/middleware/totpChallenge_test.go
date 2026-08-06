package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

func requestWithTotpChallenge(token string) (*httptest.ResponseRecorder, uint64) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TOTPChallengeAuth)
	var gotUserID uint64
	router.GET("/", func(c *gin.Context) {
		gotUserID = c.GetUint64("userId")
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, request)
	return recorder, gotUserID
}

func TestTotpChallengeAuthAcceptsChallengeToken(t *testing.T) {
	setupSessionAuthTestDB(t)
	user, err := userservice.CreateUser("totpaccept", "password", "totpaccept@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	token, err := jwt.CreateChallengeToken(user.Id, user.TokenVersion, jwt.PurposeTotpChallenge, 5*time.Minute)
	if err != nil {
		t.Fatalf("create challenge token: %v", err)
	}
	recorder, userID := requestWithTotpChallenge(token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if userID != user.Id {
		t.Fatalf("userId = %d, want %d", userID, user.Id)
	}
}

func TestTotpChallengeAuthRejectsSessionToken(t *testing.T) {
	setupSessionAuthTestDB(t)
	user, err := userservice.CreateUser("totpsession", "password", "totpsession@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	token, _, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("create session token: %v", err)
	}
	recorder, _ := requestWithTotpChallenge(token)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTotpChallengeAuthRejectsExpiredToken(t *testing.T) {
	setupSessionAuthTestDB(t)
	user, err := userservice.CreateUser("totpexpired", "password", "totpexpired@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	token, err := jwt.CreateChallengeToken(user.Id, user.TokenVersion, jwt.PurposeTotpChallenge, -1*time.Minute)
	if err != nil {
		t.Fatalf("create expired challenge token: %v", err)
	}
	recorder, _ := requestWithTotpChallenge(token)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTotpChallengeAuthRejectsStaleTokenVersion(t *testing.T) {
	setupSessionAuthTestDB(t)
	user, err := userservice.CreateUser("totpstale", "password", "totpstale@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	// 用错误的 TokenVersion 签发 challenge token，模拟改密/退出所有设备后。
	token, err := jwt.CreateChallengeToken(user.Id, user.TokenVersion+99, jwt.PurposeTotpChallenge, 5*time.Minute)
	if err != nil {
		t.Fatalf("create stale challenge token: %v", err)
	}
	recorder, _ := requestWithTotpChallenge(token)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestTotpChallengeAuthRejectsMissingToken(t *testing.T) {
	recorder, _ := requestWithTotpChallenge("")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
