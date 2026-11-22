package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestJWTAuthSessionQueryFailureIsNotUnauthorized(t *testing.T) {
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&users.EntityComplete{}, &userSessions.Entity{}); err != nil {
		t.Fatal(err)
	}
	user := users.MakeUser("jwt-unavailable-test", "password", "jwt-unavailable-test@example.com")
	if err := users.Create(user); err != nil {
		t.Fatal(err)
	}
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, jti, "test", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.GET("/private", JWTAuthCheck, func(c *gin.Context) { c.Status(http.StatusNoContent) })
	request := func(credential string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/private", nil)
		if credential != "" {
			req.Header.Set("Authorization", "Bearer "+credential)
		}
		router.ServeHTTP(rec, req)
		return rec
	}
	const callback = "test:session-unavailable"
	var failingTable atomic.Value
	failingTable.Store("")
	if err := db.Callback().Query().Before("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == failingTable.Load().(string) {
			_ = tx.AddError(errors.New("injected session database error"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	if got := request(token).Code; got != http.StatusNoContent {
		t.Fatalf("live session status = %d", got)
	}
	for _, table := range []string{"user_sessions", "users"} {
		userservice.InvalidateUserInfoCache(user.Id)
		failingTable.Store(table)
		failed := request(token)
		if failed.Code != http.StatusServiceUnavailable ||
			failed.Header().Get("Cache-Control") != "no-store" ||
			failed.Header().Get("New-Token") != "" ||
			!strings.Contains(failed.Body.String(), "common.operation.failed") {
			t.Errorf("%s query must return a non-cacheable 503 without renewal; status=%d", table, failed.Code)
		}
		if got := request("").Code; got != http.StatusUnauthorized {
			t.Errorf("missing session during dependency failure = %d, want 401", got)
		}
		failingTable.Store("")
		if got := request(token).Code; got != http.StatusNoContent {
			t.Fatalf("same session after %s recovery status = %d", table, got)
		}
	}
	if err := sessionservice.RevokeByJti(user.Id, jti); err != nil {
		t.Fatal(err)
	}
	if got := request(token).Code; got != http.StatusUnauthorized {
		t.Errorf("revoked session status = %d, want 401", got)
	}
}
