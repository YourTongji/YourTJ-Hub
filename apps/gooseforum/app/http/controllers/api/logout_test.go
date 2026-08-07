package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	jwt "github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/sessionservice"
	"github.com/leancodebox/GooseForum/app/service/userservice"
)

func setupLogoutTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
	); err != nil {
		t.Fatalf("migrate logout tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func newLogoutContext(token string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	c.Request = req
	return c, recorder
}

// TestLogoutRevokesSession 验证登出会删除当前会话行，之后原 token 不再映射到有效会话。
func TestLogoutRevokesSession(t *testing.T) {
	setupLogoutTestDB(t)

	user, err := userservice.CreateUser("logoutuser", "password", "logout@example.com", false)
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
	if sessionservice.GetValidByJti(jti) == nil {
		t.Fatal("session row missing before logout")
	}

	c, recorder := newLogoutContext(token)
	Logout(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var res component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Code != component.SUCCESS {
		t.Fatalf("response code = %v, want SUCCESS", res.Code)
	}
	if sessionservice.GetValidByJti(jti) != nil {
		t.Fatal("session row still valid after logout")
	}
}

// TestLogoutWithoutTokenStillClearsCookie 验证未登录的登出请求仅清 cookie 并返回成功。
func TestLogoutWithoutTokenStillClearsCookie(t *testing.T) {
	c, recorder := newLogoutContext("")
	Logout(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var res component.ResultStruct
	if err := json.Unmarshal(recorder.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if res.Code != component.SUCCESS {
		t.Fatalf("response code = %v, want SUCCESS", res.Code)
	}
}
