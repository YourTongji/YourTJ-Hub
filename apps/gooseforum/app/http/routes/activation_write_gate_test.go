package routes

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupActivationWriteGateContractTest 在共享 harness（setupHTTPContractTest，
// 已含 topics/write 路由）之上补齐 posts/create 与恢复路径
// set-user-email / resend-activation-email，中间件链与 route4api.go 一致。
func setupActivationWriteGateContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate activation gate task queue: %v", err)
	}

	loginAPI := router.Group("/api").Use(middleware.JWTAuthCheck)
	loginAPI.POST("/set-user-email", middleware.CheckWritableAccountAllowPendingActivation, middleware.RateLimit(middleware.RateLimitEmailChange), UpButterReq(api.EditUserEmail))
	loginAPI.POST("/resend-activation-email", middleware.CheckWritableAccountAllowPendingActivation, UpButterReq(api.ResendActivationEmail))

	forumLoginAPI := router.Group("/api/forum").Use(middleware.JWTAuthCheck)
	forumLoginAPI.POST("/posts/create", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPostCreate), UpButterReq(api.CreatePost))
	return conn, router
}

// createPendingContractUser 创建已激活用户后把激活状态改为待激活，等价于
// 邮箱验证开启时注册即发会话的 pending 账号（密码注册与 OAuth 未验证邮箱
// 落库后的同一状态）。
func createPendingContractUser(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(user).Update("is_activated", users.ActivationPending).Error; err != nil {
		t.Fatalf("mark contract user pending activation: %v", err)
	}
	user.IsActivated = users.ActivationPending
	return user
}

// TestActivationWriteGateBlocksPendingContentWrites 复现 issue #404 顶层行为：
// 邮箱验证开启时，密码注册/OAuth 存量 pending 会话写 topics/write、posts/create
// 必须被 CheckWritableAccount 稳定 403 拒绝（permission.emailRequired）。
func TestActivationWriteGateBlocksPendingContentWrites(t *testing.T) {
	t.Run("topics/write blocked for pending user", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingContractUser(t, conn)
		token := contractSessionToken(t, user)

		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, token)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("pending topics/write status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.MessageCode != string(component.MessagePermissionEmailRequired) {
			t.Fatalf("messageCode = %q, want %q", envelope.MessageCode, component.MessagePermissionEmailRequired)
		}
		if string(envelope.Result) != "null" {
			t.Fatalf("result = %s, want null (no account info leak)", string(envelope.Result))
		}
		if envelope.Params["actionCode"] != "write" {
			t.Fatalf("params.actionCode = %#v, want write", envelope.Params["actionCode"])
		}

		// 错误响应稳定：重复请求返回相同的 messageCode/params。
		second := serveJSON(router, "/api/forum/topics/write", `{}`, token)
		if second.Code != http.StatusForbidden {
			t.Fatalf("second pending topics/write status = %d, want 403: %s", second.Code, second.Body.String())
		}
		secondEnvelope := decodeContractEnvelope(t, second)
		if secondEnvelope.MessageCode != envelope.MessageCode || fmt.Sprint(secondEnvelope.Params) != fmt.Sprint(envelope.Params) {
			t.Fatalf("pending gate response not stable: first=%s second=%s", recorder.Body.String(), second.Body.String())
		}
	})

	t.Run("posts/create blocked for pending user", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingContractUser(t, conn)
		token := contractSessionToken(t, user)

		recorder := serveJSON(router, "/api/forum/posts/create", `{}`, token)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("pending posts/create status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.MessageCode != string(component.MessagePermissionEmailRequired) {
			t.Fatalf("messageCode = %q, want %q", envelope.MessageCode, component.MessagePermissionEmailRequired)
		}
	})
}

// TestActivationWriteGateAllowsRecoveryForPendingUsers 恢复路径显式放行：
// pending 用户仍可 resend-activation-email 与 set-user-email（改邮箱继续恢复）。
func TestActivationWriteGateAllowsRecoveryForPendingUsers(t *testing.T) {
	t.Run("resend-activation-email allowed for pending user", func(t *testing.T) {
		useContractTempKV(t)
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingContractUser(t, conn)

		recorder := serveJSON(router, "/api/resend-activation-email", `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("pending resend status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.MessageCode != "auth.activation.resendSuccess" {
			t.Fatalf("messageCode = %q, want auth.activation.resendSuccess", envelope.MessageCode)
		}
	})

	t.Run("set-user-email allowed for pending user", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createPendingContractUser(t, conn)
		body := fmt.Sprintf(`{"email":"pending-recover-%d@example.test","password":"secret123"}`, user.Id)

		recorder := serveJSON(router, "/api/set-user-email", body, contractSessionToken(t, user))
		if recorder.Code != http.StatusOK {
			t.Fatalf("pending set-user-email status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.MessageCode != "user.updateSuccess" {
			t.Fatalf("messageCode = %q, want user.updateSuccess", envelope.MessageCode)
		}
	})
}

// TestActivationWriteGateRevokesWriteAfterRePending 改邮箱把账号重新置为
// pending（或会话存续期间账号被置为 pending）后，既有会话写请求立即被拒。
func TestActivationWriteGateRevokesWriteAfterRePending(t *testing.T) {
	t.Run("email change re-pending immediately blocks write", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		body := fmt.Sprintf(`{"email":"re-pending-%d@example.test","password":"secret123"}`, user.Id)

		change := serveJSON(router, "/api/set-user-email", body, token)
		if change.Code != http.StatusOK {
			t.Fatalf("set-user-email status = %d, want 200: %s", change.Code, change.Body.String())
		}
		if got := decodeContractEnvelope(t, change).MessageCode; got != "user.updateSuccess" {
			t.Fatalf("set-user-email messageCode = %q, want user.updateSuccess", got)
		}

		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, token)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("post-change topics/write status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.MessageCode != string(component.MessagePermissionEmailRequired) {
			t.Fatalf("messageCode = %q, want %q", envelope.MessageCode, component.MessagePermissionEmailRequired)
		}
	})

	t.Run("existing session blocked once account turns pending", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)
		// 会话签发时账号已激活；随后账号回到待激活状态（与改邮箱/激活状态
		// 变更同构），该会话不得继续写。
		if err := conn.Model(user).Update("is_activated", users.ActivationPending).Error; err != nil {
			t.Fatalf("mark user pending: %v", err)
		}

		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, token)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("pending-session topics/write status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		if got := decodeContractEnvelope(t, recorder).MessageCode; got != string(component.MessagePermissionEmailRequired) {
			t.Fatalf("messageCode = %q, want %q", got, component.MessagePermissionEmailRequired)
		}
	})
}

// TestActivationWriteGateNoRegression 门禁外行为零回归：已激活用户正常写；
// 关闭邮箱验证时 pending 用户写入不被新增判定拦截。
func TestActivationWriteGateNoRegression(t *testing.T) {
	t.Run("activated user can write with verification enabled", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		enableContractEmailVerification(t, conn)
		user := createHTTPContractUser(t, conn, contractTestID())

		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, contractSessionToken(t, user))
		if recorder.Code == http.StatusForbidden {
			t.Fatalf("activated topics/write blocked with 403: %s", recorder.Body.String())
		}
		if got := decodeContractEnvelope(t, recorder).MessageCode; got == string(component.MessagePermissionEmailRequired) {
			t.Fatal("activated user must not receive permission.emailRequired")
		}
	})

	t.Run("pending user can write when verification disabled", func(t *testing.T) {
		conn, router := setupActivationWriteGateContractTest(t)
		user := createPendingContractUser(t, conn)

		recorder := serveJSON(router, "/api/forum/topics/write", `{}`, contractSessionToken(t, user))
		if recorder.Code == http.StatusForbidden {
			t.Fatalf("verification-disabled pending topics/write blocked with 403: %s", recorder.Body.String())
		}
		if got := decodeContractEnvelope(t, recorder).MessageCode; got == string(component.MessagePermissionEmailRequired) {
			t.Fatal("verification disabled must not emit permission.emailRequired")
		}
	})
}
