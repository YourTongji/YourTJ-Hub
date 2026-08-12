package api

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// 回归 E：注销账号后必须失效 userInfoCache，使旧 token 立即失效
// （修复前缓存 TTL 2 分钟内 ValidateToken 仍接受旧 token）。
func TestAccountCloseInvalidatesUserInfoCache(t *testing.T) {
	conn := setupBatchDeleteTestDB(t)
	const uid = uint64(948099)
	userservice.InvalidateUserInfoCache(uid)
	_ = conn.Unscoped().Where("id = ?", uid).Delete(&users.EntityComplete{}).Error

	user := users.MakeUser("close-cache-user", "pwd-123", "close@example.com")
	user.Id = uid
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	// 填充 user-info 缓存
	if _, ok := userservice.GetUserInfo(uid); !ok {
		t.Fatal("GetUserInfo should succeed before close")
	}

	resp := AccountClose(component.BetterRequest[AccountCloseReq]{
		UserId: uid,
		Params: AccountCloseReq{Mode: "anonymize", Password: "pwd-123"},
	})
	if resp.Data.Code != component.SUCCESS {
		t.Fatalf("AccountClose failed: %#v", resp)
	}
	// 缓存必须失效：GetUserInfo 应返回 not found
	if _, ok := userservice.GetUserInfo(uid); ok {
		t.Fatal("BUG: user info still cached after account close — old token stays valid within TTL")
	}
}
