package users

import (
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
)

// issue #678 review 回归：换绑暂存相关的条件更新（CAS）语义。
// 全部走 models 层直连（setupUserIsolationTestDB），聚焦单条 UPDATE 的
// 命中/未命中行为，路由层语义由 email_change_two_phase_test.go 覆盖。

func mustHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := algorithm.MakePassword(password)
	if err != nil {
		t.Fatalf("make password hash: %v", err)
	}
	return hash
}

// 改密成功：写入新哈希、自增 token_version、清空换绑暂存。
func TestApplyPasswordChangeClearsStagingAndBumpsTokenVersion(t *testing.T) {
	setupUserIsolationTestDB(t)
	user := MakeUser("pwd-cas-user", "old-password", "pwd-cas@example.com")
	if err := Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := StagePendingEmail(user.Id, "pwd-cas-new@example.com", time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	newHash := mustHash(t, "new-password")
	if err := ApplyPasswordChange(user.Id, newHash, user.TokenVersion); err != nil {
		t.Fatalf("ApplyPasswordChange: %v", err)
	}

	row, err := Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Password != newHash {
		t.Fatal("password hash must be replaced")
	}
	if row.TokenVersion != user.TokenVersion+1 {
		t.Fatalf("tokenVersion = %d, want %d", row.TokenVersion, user.TokenVersion+1)
	}
	if row.PendingEmail != "" || row.PendingEmailAt != nil {
		t.Fatalf("staging must be cleared by password change, got %q at %v", row.PendingEmail, row.PendingEmailAt)
	}
}

// 改密 CAS 未命中（读取后凭据已被并发变更）：返回 ErrConcurrentPasswordChange，
// 密码与暂存都保持并发变更后的状态，绝不覆盖写入。
func TestApplyPasswordChangeCASMissKeepsConcurrentState(t *testing.T) {
	setupUserIsolationTestDB(t)
	user := MakeUser("pwd-cas-stale", "old-password", "pwd-cas-stale@example.com")
	if err := Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := StagePendingEmail(user.Id, "pwd-cas-stale-new@example.com", time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	// 并发方先行完成一次改密（token_version 已自增）。
	winnerHash := mustHash(t, "winner-password")
	if err := ApplyPasswordChange(user.Id, winnerHash, user.TokenVersion); err != nil {
		t.Fatalf("concurrent password change: %v", err)
	}

	// 本方持过期 expected 重试：必须未命中且不回写任何列。
	staleHash := mustHash(t, "stale-password")
	if err := ApplyPasswordChange(user.Id, staleHash, user.TokenVersion); !errors.Is(err, ErrConcurrentPasswordChange) {
		t.Fatalf("stale ApplyPasswordChange error = %v, want ErrConcurrentPasswordChange", err)
	}

	row, err := Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Password != winnerHash {
		t.Fatal("stale CAS must not overwrite the winner's password hash")
	}
	if row.TokenVersion != user.TokenVersion+1 {
		t.Fatalf("tokenVersion = %d, want %d (only one bump)", row.TokenVersion, user.TokenVersion+1)
	}
}

// 并发的新邮箱确认切换先完成后，旧邮箱激活链接的 CAS 必须未命中：
// 绝不把已切换的 email 回写成旧值。
func TestActivateCurrentEmailCASMissAfterConcurrentSwitch(t *testing.T) {
	setupUserIsolationTestDB(t)
	user := MakeUser("activate-cas", "secret123", "activate-cas@example.com")
	if err := Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	stagedEmail := "activate-cas-new@example.com"
	if err := StagePendingEmail(user.Id, stagedEmail, time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	// 并发方先完成换绑切换。
	if _, err := CompletePendingEmailSwitch(user.Id, stagedEmail, time.Now()); err != nil {
		t.Fatalf("complete switch: %v", err)
	}

	// 旧链接携带读取时状态（旧 email + 仍存在的暂存）重放：CAS 未命中。
	applied, err := ActivateCurrentEmail(user.Id, user.Email, stagedEmail, time.Now())
	if err != nil {
		t.Fatalf("ActivateCurrentEmail: %v", err)
	}
	if applied {
		t.Fatal("ActivateCurrentEmail must miss after a concurrent switch completed")
	}

	row, err := Get(user.Id)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if row.Email != stagedEmail {
		t.Fatalf("email = %q, want switched %q (replayed old link must not revert)", row.Email, stagedEmail)
	}
	if row.PendingEmail != "" {
		t.Fatalf("pending = %q, want empty", row.PendingEmail)
	}
}

// 切换前复查命中（并发注册已把目标邮箱注册为自己的当前 email）：
// 撤销暂存、按 ErrPendingEmailSwitchStale 让步，邮箱留给占用者。
func TestCompletePendingEmailSwitchConcedesWhenTargetHeld(t *testing.T) {
	setupUserIsolationTestDB(t)
	stager := MakeUser("switch-concede", "secret123", "switch-concede@example.com")
	if err := Create(stager); err != nil {
		t.Fatalf("create stager: %v", err)
	}
	target := "switch-concede-target@example.com"
	if err := StagePendingEmail(stager.Id, target, time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}
	owner := MakeUser("switch-owner", "secret123", target)
	if err := Create(owner); err != nil {
		t.Fatalf("create owner: %v", err)
	}

	if _, err := CompletePendingEmailSwitch(stager.Id, target, time.Now()); !errors.Is(err, ErrPendingEmailSwitchStale) {
		t.Fatalf("CompletePendingEmailSwitch error = %v, want ErrPendingEmailSwitchStale", err)
	}

	row, err := Get(stager.Id)
	if err != nil {
		t.Fatalf("get stager: %v", err)
	}
	if row.Email != stager.Email || row.PendingEmail != "" {
		t.Fatalf("stager after concede: email = %q pending = %q, want %q/empty", row.Email, row.PendingEmail, stager.Email)
	}
	ownerRow, err := Get(owner.Id)
	if err != nil {
		t.Fatalf("get owner: %v", err)
	}
	if ownerRow.Email != target {
		t.Fatalf("owner email = %q, want %q (occupier keeps the email)", ownerRow.Email, target)
	}
}

// 占用窗口语义：新鲜暂存占用邮箱（注册侧据此拒绝），过期暂存不再占用，
// 本人暂存被排除（重新暂存同一邮箱 = 再次发起换绑）。
func TestExistEmailOrFreshPendingOccupancyWindow(t *testing.T) {
	setupUserIsolationTestDB(t)
	stager := MakeUser("occupancy-stager", "secret123", "occupancy-stager@example.com")
	if err := Create(stager); err != nil {
		t.Fatalf("create stager: %v", err)
	}
	target := "occupancy-target@example.com"
	if err := StagePendingEmail(stager.Id, target, time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	if !ExistEmailOrFreshPending(target, 0) {
		t.Fatal("fresh staging must occupy the email for others")
	}
	if ExistEmailOrFreshPending(target, stager.Id) {
		t.Fatal("own staging must not block re-staging")
	}

	// 拨出窗口：过期暂存视为放弃，不再占用。
	expired := time.Now().Add(-(PendingEmailWindow + time.Hour))
	if err := builder().Where(pid, stager.Id).
		Updates(map[string]any{"pending_email_at": expired}).Error; err != nil {
		t.Fatalf("age staging: %v", err)
	}
	if ExistEmailOrFreshPending(target, 0) {
		t.Fatal("expired staging must not occupy the email")
	}
}
