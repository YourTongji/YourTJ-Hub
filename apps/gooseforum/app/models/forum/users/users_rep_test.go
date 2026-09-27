package users

import (
	"testing"
	"time"
)

// issue #702 回归：验证禁用分支的即时邮箱切换（UpdateEmailVerificationDisabled）
// 必须是定向列更新——全行 Save 会把读取时快照的 password/token_version 回写，
// 静默回滚读-写间隙内提交的并发改密 CAS，并让已吊销的会话随 token_version
// 回滚重新生效。本用例复现该交错并断言凭据列不被触碰。
func TestUpdateEmailVerificationDisabledPreservesConcurrentPasswordChange(t *testing.T) {
	setupUserIsolationTestDB(t)
	user := MakeUser("email-cas-user", "old-password", "email-cas@example.com")
	if err := Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := StagePendingEmail(user.Id, "email-cas-staged@example.com", time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	// 并发改密先提交：CAS 命中，token_version 已自增。
	newHash := mustHash(t, "new-password")
	if err := ApplyPasswordChange(user.Id, newHash, user.TokenVersion); err != nil {
		t.Fatalf("concurrent password change: %v", err)
	}

	changedAt := time.Now()
	if err := UpdateEmailVerificationDisabled(user.Id, "email-cas-new@example.com", changedAt); err != nil {
		t.Fatalf("UpdateEmailVerificationDisabled: %v", err)
	}

	row, err := Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Email != "email-cas-new@example.com" {
		t.Fatalf("email = %q, want new address", row.Email)
	}
	if row.Password != newHash {
		t.Fatal("email update must not roll back the concurrent password hash (issue #702)")
	}
	if row.TokenVersion != user.TokenVersion+1 {
		t.Fatalf("tokenVersion = %d, want %d; rolling it back re-activates revoked sessions", row.TokenVersion, user.TokenVersion+1)
	}
	if row.IsActivated != ActivationPending {
		t.Fatalf("isActivated = %d, want %d (ActivationPending)", row.IsActivated, ActivationPending)
	}
	if row.ActivatedAt != nil {
		t.Fatalf("activatedAt = %v, want nil", row.ActivatedAt)
	}
	if row.PendingEmail != "" || row.PendingEmailAt != nil {
		t.Fatalf("staging must be cleared, got %q at %v", row.PendingEmail, row.PendingEmailAt)
	}
	if row.EmailChangedAt == nil || row.EmailChangedAt.Sub(changedAt).Abs() > time.Second {
		t.Fatalf("emailChangedAt = %v, want ~%v", row.EmailChangedAt, changedAt)
	}
	if row.UpdatedAt.Sub(changedAt).Abs() > time.Second {
		t.Fatalf("updatedAt = %v, want ~%v (email change must stay observable)", row.UpdatedAt, changedAt)
	}
}

func TestUpdateFieldsPreservesConcurrentPasswordChange(t *testing.T) {
	setupUserIsolationTestDB(t)
	user := MakeUser("fields-cas-user", "old-password", "fields-cas@example.com")
	if err := Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	newHash := mustHash(t, "new-password")
	if err := ApplyPasswordChange(user.Id, newHash, user.TokenVersion); err != nil {
		t.Fatalf("concurrent password change: %v", err)
	}
	if err := UpdateFields(user.Id, map[string]any{"username": "fields-cas-renamed"}); err != nil {
		t.Fatalf("update fields: %v", err)
	}
	row, err := Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Username != "fields-cas-renamed" || row.Password != newHash || row.TokenVersion != user.TokenVersion+1 {
		t.Fatalf("targeted update changed unrelated credentials: username=%q tokenVersion=%d", row.Username, row.TokenVersion)
	}
}

// GetActiveUserIdsByRoleIds 只返回角色下未注销（软删）的用户：
// builder() 走 .Table() 原生查询不含软删 scope，必须显式过滤 deleted_at，
// 否则告警（issue #855）会错误地发给已注销账号。
func TestGetActiveUserIdsByRoleIds(t *testing.T) {
	setupUserIsolationTestDB(t)

	manager := MakeUser("active-manager", "secret123", "active-manager@example.com")
	manager.RoleId = 8001
	if err := Create(manager); err != nil {
		t.Fatalf("create manager: %v", err)
	}
	closed := MakeUser("closed-manager", "secret123", "closed-manager@example.com")
	closed.RoleId = 8001
	if err := Create(closed); err != nil {
		t.Fatalf("create closed manager: %v", err)
	}
	if err := builder().Delete(&EntityComplete{}, "id = ?", closed.Id).Error; err != nil {
		t.Fatalf("soft delete closed manager: %v", err)
	}
	plain := MakeUser("plain-user", "secret123", "plain-user@example.com")
	plain.RoleId = 8002
	if err := Create(plain); err != nil {
		t.Fatalf("create plain user: %v", err)
	}

	got := GetActiveUserIdsByRoleIds([]uint64{8001})
	if len(got) != 1 || got[0] != manager.Id {
		t.Fatalf("GetActiveUserIdsByRoleIds([8001]) = %v, want [%d] only (soft-deleted excluded)", got, manager.Id)
	}

	if got := GetActiveUserIdsByRoleIds([]uint64{8002}); len(got) != 1 || got[0] != plain.Id {
		t.Fatalf("GetActiveUserIdsByRoleIds([8002]) = %v, want [%d]", got, plain.Id)
	}

	if got := GetActiveUserIdsByRoleIds([]uint64{9999}); len(got) != 0 {
		t.Fatalf("GetActiveUserIdsByRoleIds([9999]) = %v, want empty", got)
	}
	if got := GetActiveUserIdsByRoleIds(nil); got != nil {
		t.Fatalf("GetActiveUserIdsByRoleIds(nil) = %v, want nil", got)
	}
}
