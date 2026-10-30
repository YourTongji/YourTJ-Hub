package routes

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func TestRegistrationAfterCreateFailureOutcomes(t *testing.T) {
	for _, failure := range []string{"activation token", "queue insert", "session insert"} {
		t.Run(failure, func(t *testing.T) {
			conn, router := setupAccountRecoveryContractTest(t)
			enableContractEmailVerification(t, conn)
			withRouteTestSigningKey(t, "registration-outcome-test-key-2026")
			if failure == "activation token" {
				withRouteTestSigningKey(t, "")
			}
			rejection := errors.New("internal injected registration failure")
			const callback = "test:register_outcome"
			if failure != "activation token" {
				if err := conn.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
					target := (&taskQueue.Entity{}).TableName()
					if failure == "session insert" {
						target = (&userSessions.Entity{}).TableName()
					}
					if tx.Statement.Table == target {
						_ = tx.AddError(rejection)
					}
				}); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })
			}
			username := fmt.Sprintf("outcome_%d", contractTestID())
			email := username + "@example.test"
			body := fmt.Sprintf(`{"userName":%q,"email":%q,"passWord":"password123"}`, username, email)
			recorder := serveAccountRecoveryJSON(router, "/api/register", body)
			response := decodeContractEnvelope(t, recorder)
			user, err := users.GetByEmail(email)
			if err != nil || user.Id == 0 || user.IsActivated != users.ActivationPending {
				t.Fatalf("created account was lost: id=%d err=%v", user.Id, err)
			}
			t.Cleanup(func() {
				conn.Where("user_id = ?", user.Id).Delete(&userSessions.Entity{})
				conn.Where("task_json LIKE ?", "%"+email+"%").Delete(&taskQueue.Entity{})
				conn.Unscoped().Delete(&user)
				userservice.InvalidateUserInfoCache(user.Id)
			})
			if strings.Contains(recorder.Body.String(), rejection.Error()) {
				t.Fatal("internal fault exposed")
			}
			if failure == "session insert" {
				assertRegisterResponseCode(t, recorder, "register-retry-login.json")
				if response.MessageCode != "auth.register.retryLogin" || recorder.Header().Get("New-Token") != "" {
					t.Fatalf("wrong recovery after session failure: %s", recorder.Body.String())
				}
			} else {
				assertRegisterResponseCode(t, recorder, "register-email-retry.json")
				if response.MessageCode != "auth.register.emailRetry" || response.Code != 0 {
					t.Fatalf("mail failure falsely reported ready verification: %s", recorder.Body.String())
				}
				token := recorder.Header().Get("New-Token")
				if token == "" {
					t.Fatal("created account lost its session")
				}
				router.GET("/test/owner", middleware.JWTAuthCheck, func(c *gin.Context) { c.Status(http.StatusNoContent) })
				router.POST("/test/write", middleware.JWTAuthCheck, middleware.CheckWritableAccount, func(c *gin.Context) { c.Status(http.StatusNoContent) })
				for _, request := range []struct {
					method, path string
					want         int
				}{{http.MethodGet, "/test/owner", 204}, {http.MethodPost, "/test/write", 403}} {
					req := httptest.NewRequest(request.method, request.path, nil)
					req.Header.Set("Authorization", "Bearer "+token)
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, req)
					if rec.Code != request.want {
						t.Fatalf("pending session boundary %s = %d: %s", request.path, rec.Code, rec.Body.String())
					}
				}
				var count int64
				if err := conn.Model(&taskQueue.Entity{}).Where("task_json LIKE ?", "%"+email+"%").Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("definite mail rejection inserted a task: %d %v", count, err)
				}
			}
			// A retry must use the existing generic failure, never repeat account creation
			// or reveal whether username or email was the occupied field.
			again := serveAccountRecoveryJSON(router, "/api/register", body)
			if code := decodeContractEnvelope(t, again).MessageCode; code != string(component.MessageAuthRegisterFailed) {
				t.Fatalf("repeat registration: %s", code)
			}
		})
	}
}
