package routes

import (
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
)

// TestResetPasswordBindsTokenVersion 是 issue #106 point 2 的核心保护：重置令牌
// 绑定签发时的 token_version，每次成功重置 SetPassword 都会自增 token_version，
// 因此旧重置链接在账户被重置后立即失效，无法重放接管账户。
func TestResetPasswordBindsTokenVersion(t *testing.T) {
	setupEmailChangeTestDB(t)
	withRouteTestSigningKey(t, "route-test-signing-key-bind")
	const userID = uint64(9107)
	createEmailChangeUser(t, userID, "reset-bind", "original-password-123")

	// MakeUser 已经 SetPassword 一次，所以新用户 token_version=1；签发时绑定它。
	user0, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user before reset: %v", err)
	}
	token, err := tokenservice.GeneratePasswordResetToken(userID, "reset-bind@example.com", user0.TokenVersion)
	if err != nil {
		t.Fatalf("generate reset token: %v", err)
	}

	// 1. 首次重置：成功。
	res := api.ResetPassword(component.BetterRequest[api.ResetPasswordReq]{
		Params: api.ResetPasswordReq{Token: token, NewPassword: "brand-new-password-1"},
	})
	if res.Code != http.StatusOK || res.Data.Code != component.SUCCESS {
		t.Fatalf("first reset should succeed, got %#v", res)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user after reset: %v", err)
	}
	// SetPassword 把 token_version 再自增 1，旧令牌（绑定旧版本）必须不再有效。
	if user.TokenVersion != user0.TokenVersion+1 {
		t.Fatalf("token_version after reset = %d, want %d (SetPassword must bump it)", user.TokenVersion, user0.TokenVersion+1)
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, "brand-new-password-1"); err != nil {
		t.Fatalf("password must be the new one after reset: %v", err)
	}

	// 2. 重放同一令牌：必须被拒（claims.TokenVersion=旧 != 当前 user.TokenVersion）。
	resReplay := api.ResetPassword(component.BetterRequest[api.ResetPasswordReq]{
		Params: api.ResetPasswordReq{Token: token, NewPassword: "replay-password-2"},
	})
	if resReplay.Code != http.StatusOK || resReplay.Data.Code == component.SUCCESS {
		t.Fatalf("replaying old reset token must be rejected, got %#v", resReplay)
	}
	user, err = users.Get(userID)
	if err != nil {
		t.Fatalf("get user after replay: %v", err)
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, "replay-password-2"); err == nil {
		t.Fatal("password must NOT be changed via replayed reset token (token_version binding)")
	}
}

// TestResetPasswordRejectsTokenWithWrongVersion 验证 token_version 绑定本身即可
// 阻断伪造：即便令牌签名/邮箱匹配，绑定的版本号必须等于用户当前 token_version。
func TestResetPasswordRejectsTokenWithWrongVersion(t *testing.T) {
	setupEmailChangeTestDB(t)
	withRouteTestSigningKey(t, "route-test-signing-key-wrong")
	const userID = uint64(9108)
	createEmailChangeUser(t, userID, "reset-wrong", "original-password-456")

	// 用户当前 token_version=1（MakeUser 已 SetPassword），故意签发绑定错误版本号 5 的令牌。
	token, err := tokenservice.GeneratePasswordResetToken(userID, "reset-wrong@example.com", 5)
	if err != nil {
		t.Fatalf("generate reset token: %v", err)
	}

	res := api.ResetPassword(component.BetterRequest[api.ResetPasswordReq]{
		Params: api.ResetPasswordReq{Token: token, NewPassword: "should-not-apply-789"},
	})
	if res.Code != http.StatusOK || res.Data.Code == component.SUCCESS {
		t.Fatalf("reset with wrong token_version must be rejected, got %#v", res)
	}

	user, err := users.Get(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, "should-not-apply-789"); err == nil {
		t.Fatal("password must NOT be changed when token_version binding mismatches")
	}
}

func TestResetPasswordRejectsRevokedSessionForEmptyEmail(t *testing.T) {
	conn, router := setupAccountRecoveryContractTest(t)
	cleanAccountRecoveryTables(t, conn)
	const key = "reset-revoked-session-test-key"
	withRouteTestSigningKey(t, key)
	user := users.MakeUser("reset-empty-email", "original-password-123", "")
	user.Id = 9199
	if err := users.Create(user); err != nil {
		t.Fatal(err)
	}
	claims := jwtopt.CustomClaims{UserId: user.Id, TokenVersion: user.TokenVersion, Jti: "revoked-reset-test", RegisteredClaims: jwtopt.GetBaseRegisteredClaims(time.Hour)}
	token, err := jwtopt.NewJWT([]byte(key)).CreateToken(claims)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, claims.Jti, "test", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.RevokeByJti(user.Id, claims.Jti); err != nil {
		t.Fatal(err)
	}
	if sessionservice.GetValidByJti(claims.Jti) != nil {
		t.Fatal("session was not revoked")
	}
	recorder := serveAccountRecoveryJSON(router, "/api/reset-password", fmt.Sprintf(`{"token":%q,"newPassword":"replacement-password-123"}`, token))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "reset-password-token-invalid.json"))
	after, err := users.Get(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if after.Password != user.Password || after.TokenVersion != user.TokenVersion {
		t.Fatal("rejected reset changed account")
	}
}
