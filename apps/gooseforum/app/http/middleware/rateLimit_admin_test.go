package middleware

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/ratelimit"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
)

// TestRateLimitSkipsAdmin 验证管理员豁免：skipAdmin=true（默认）时，
// 管理员用户不受限流，即使超过默认配额也不返回 429。
func TestRateLimitSkipsAdmin(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate rate limit admin tables: %v", err)
	}
	const adminID = uint64(990101)
	const roleID = uint64(990102)
	conn.Unscoped().Delete(&users.EntityComplete{}, adminID)
	conn.Where("role_id = ?", roleID).Delete(&rolePermissionRs.Entity{})
	conn.Create(&users.EntityComplete{Id: adminID, Username: "ratelimit-admin", RoleId: roleID})
	conn.Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.Admin.Id(), Effective: 1})
	// 确保权限缓存从 DB 重载，而不是命中其他测试留下的空缓存。
	permission.InvalidateRole(roleID)

	ratelimit.Default().ResetAll()
	// topic.write 默认 limitPerIp=5；管理员应完全跳过限流。
	for i := 0; i < 7; i++ {
		recorder := rateLimitRecorder(withUser(adminID), RateLimit(RateLimitTopicWrite))
		if recorder.Code != http.StatusOK {
			t.Fatalf("admin attempt %d status = %d, want 200 (skipAdmin)", i+1, recorder.Code)
		}
	}
}

// TestRateLimitDoesNotSkipNonAdmin 验证非管理员用户不受豁免：
// 普通用户（有角色但无 Admin 权限）超出配额仍返回 429。
func TestRateLimitDoesNotSkipNonAdmin(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate rate limit admin tables: %v", err)
	}
	const userID = uint64(990103)
	const roleID = uint64(990104)
	conn.Unscoped().Delete(&users.EntityComplete{}, userID)
	conn.Where("role_id = ?", roleID).Delete(&rolePermissionRs.Entity{})
	conn.Create(&users.EntityComplete{Id: userID, Username: "ratelimit-user", RoleId: roleID})
	// 角色存在但不含 Admin 权限（无 role_permission_rs 记录）。
	permission.InvalidateRole(roleID)

	ratelimit.Default().ResetAll()
	// register 默认 limitPerIp=20、limitPerUser=0（纯 IP 维度）；
	// 非管理员第 21 次应 429。
	for i := 0; i < 20; i++ {
		recorder := rateLimitRecorder(withUser(userID), RateLimit(RateLimitRegister))
		if recorder.Code != http.StatusOK {
			t.Fatalf("user attempt %d status = %d, want 200", i+1, recorder.Code)
		}
	}
	recorder := rateLimitRecorder(withUser(userID), RateLimit(RateLimitRegister))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("user 21st status = %d, want 429", recorder.Code)
	}
}
