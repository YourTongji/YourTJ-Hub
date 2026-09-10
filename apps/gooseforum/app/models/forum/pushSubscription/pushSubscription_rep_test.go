package pushSubscription

import (
	"fmt"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// setupPushSubTestDB 迁移 push_subscriptions 表并清空行（endpoint 全局唯一，
// 共享 sqlite 内存连接上的跨测试残留会污染 Upsert 断言）。
func setupPushSubTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate push_subscriptions: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&Entity{})
}

func TestUpsertCreateAndConvergeOwnership(t *testing.T) {
	setupPushSubTestDB(t)
	const endpoint = "https://push.example.com/sub/abc"

	// 新 endpoint：Upsert 创建一行。
	if err := Upsert(1, endpoint, "p256dh-v1", "auth-v1", "zh"); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	subs := ListByUser(1)
	if len(subs) != 1 {
		t.Fatalf("ListByUser(1) = %d rows, want 1", len(subs))
	}
	if subs[0].Endpoint != endpoint || subs[0].P256dh != "p256dh-v1" || subs[0].Auth != "auth-v1" || subs[0].Lang != "zh" {
		t.Errorf("created row = %#v, want endpoint/p256dh/auth/lang set", subs[0])
	}

	// 同一 endpoint 换用户（换账号登录后 endpoint 不变）：冲突收敛归属到新用户。
	if err := Upsert(2, endpoint, "p256dh-v2", "auth-v2", "en"); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if subs := ListByUser(1); len(subs) != 0 {
		t.Errorf("old user still holds %d subscription(s), want 0", len(subs))
	}
	subs = ListByUser(2)
	if len(subs) != 1 {
		t.Fatalf("ListByUser(2) = %d rows, want 1", len(subs))
	}
	if subs[0].P256dh != "p256dh-v2" || subs[0].Auth != "auth-v2" || subs[0].Lang != "en" {
		t.Errorf("converged row = %#v, want refreshed keys/lang", subs[0])
	}

	// keys 为空（注销旧归属场景）不覆盖已有密钥，仅迁移归属。
	if err := Upsert(3, endpoint, "", "", "ja"); err != nil {
		t.Fatalf("ownership-only upsert: %v", err)
	}
	if subs := ListByUser(2); len(subs) != 0 {
		t.Errorf("user 2 still holds %d subscription(s), want 0", len(subs))
	}
	subs = ListByUser(3)
	if len(subs) != 1 {
		t.Fatalf("ListByUser(3) = %d rows, want 1", len(subs))
	}
	if subs[0].P256dh != "p256dh-v2" || subs[0].Auth != "auth-v2" {
		t.Errorf("ownership-only upsert overwrote keys: %#v", subs[0])
	}
	if subs[0].Lang != "ja" {
		t.Errorf("ownership-only upsert lang = %q, want ja", subs[0].Lang)
	}
}

func TestDeleteByUser(t *testing.T) {
	setupPushSubTestDB(t)
	if err := Upsert(1, "https://push.example.com/u1-a", "k1", "s1", "zh"); err != nil {
		t.Fatalf("upsert u1-a: %v", err)
	}
	if err := Upsert(1, "https://push.example.com/u1-b", "k2", "s2", "zh"); err != nil {
		t.Fatalf("upsert u1-b: %v", err)
	}
	if err := Upsert(2, "https://push.example.com/u2-a", "k3", "s3", "zh"); err != nil {
		t.Fatalf("upsert u2-a: %v", err)
	}

	if err := DeleteByUser(1); err != nil {
		t.Fatalf("DeleteByUser(1): %v", err)
	}
	if subs := ListByUser(1); len(subs) != 0 {
		t.Errorf("user 1 still holds %d subscription(s), want 0", len(subs))
	}
	// 其他用户不受影响。
	if subs := ListByUser(2); len(subs) != 1 {
		t.Errorf("user 2 subscriptions = %d, want 1", len(subs))
	}
}

func TestDeleteByEndpointIdempotent(t *testing.T) {
	setupPushSubTestDB(t)
	const endpoint = "https://push.example.com/stale"
	if err := Upsert(1, endpoint, "k1", "s1", "zh"); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := DeleteByEndpoint(endpoint, 1); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := DeleteByEndpoint(endpoint, 1); err != nil {
		t.Fatalf("second delete (idempotent): %v", err)
	}
	if subs := ListByUser(1); len(subs) != 0 {
		t.Errorf("user 1 still holds %d subscription(s), want 0", len(subs))
	}
	// 不存在的 endpoint 同样静默成功。
	if err := DeleteByEndpoint("https://push.example.com/never-existed", 1); err != nil {
		t.Fatalf("delete missing endpoint: %v", err)
	}
}

func TestTouchActiveUpdatesLastActiveAt(t *testing.T) {
	setupPushSubTestDB(t)
	const endpoint = "https://push.example.com/active"
	if err := Upsert(1, endpoint, "k1", "s1", "zh"); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	before := time.Now().Add(-time.Hour)
	// 先落一个旧值，验证 TouchActive 确实刷新（而非首写 no-op）。
	if err := TouchActive(endpoint, before); err != nil {
		t.Fatalf("touch backdated: %v", err)
	}
	now := time.Now()
	if err := TouchActive(endpoint, now); err != nil {
		t.Fatalf("touch active: %v", err)
	}

	subs := ListByUser(1)
	if len(subs) != 1 {
		t.Fatalf("ListByUser(1) = %d rows, want 1", len(subs))
	}
	if !subs[0].LastActiveAt.After(before) {
		t.Errorf("LastActiveAt = %v, want after backdated %v", subs[0].LastActiveAt, before)
	}
}

// DeleteByEndpoint 同时限定 owner：同一 endpoint 已被其他账号接管时，
// 旧账号的删除请求不得误删新归属者的订阅（review P2 越权/竞争防护）。
func TestDeleteByEndpointScopedToOwner(t *testing.T) {
	setupPushSubTestDB(t)
	const endpoint = "https://fcm.googleapis.com/fcm/send/owned"
	if err := Upsert(1, endpoint, "k1", "s1", "zh"); err != nil {
		t.Fatalf("upsert user 1: %v", err)
	}
	// 换账号登录：endpoint 收敛到 user 2（user 1 不再持有）。
	if err := Upsert(2, endpoint, "k2", "s2", "en"); err != nil {
		t.Fatalf("upsert user 2: %v", err)
	}

	// user 1 尝试删除已不属于自己的 endpoint：无匹配行，静默成功且不影响 user 2。
	if err := DeleteByEndpoint(endpoint, 1); err != nil {
		t.Fatalf("foreign-owner delete: %v", err)
	}
	subs := ListByUser(2)
	if len(subs) != 1 {
		t.Fatalf("user 2 subscription was deleted by foreign owner: %d rows, want 1", len(subs))
	}
	if subs[0].P256dh != "k2" {
		t.Errorf("user 2 keys overwritten: %#v", subs[0])
	}

	// user 2（真实归属者）删除成功。
	if err := DeleteByEndpoint(endpoint, 2); err != nil {
		t.Fatalf("owned delete: %v", err)
	}
	if subs := ListByUser(2); len(subs) != 0 {
		t.Errorf("user 2 still holds %d subscription(s), want 0", len(subs))
	}
}

// UpsertCapped 在行数达到上限时按 id 升序淘汰最旧行：worker 串行 fan-out
// 有界（review P1），同 endpoint 刷新与换账号收敛不触发淘汰。
func TestUpsertCappedEvictsOldestWhenAtLimit(t *testing.T) {
	setupPushSubTestDB(t)
	const cap = 3
	// 先写满上限（endpoint 均为合法形状，rep 层不做白名单校验）。
	for i := 1; i <= cap; i++ {
		ep := fmt.Sprintf("https://fcm.googleapis.com/fcm/send/dev-%d", i)
		if err := Upsert(1, ep, "k", "s", "zh"); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	// 新增第 4 条：淘汰最旧（dev-1），保留 dev-2..dev-4。
	evicted, err := UpsertCapped(1, "https://fcm.googleapis.com/fcm/send/dev-4", "k4", "s4", "en", cap)
	if err != nil {
		t.Fatalf("upsert capped: %v", err)
	}
	if evicted != 1 {
		t.Fatalf("evicted = %d, want 1", evicted)
	}
	subs := ListByUser(1)
	if len(subs) != cap {
		t.Fatalf("ListByUser(1) = %d rows, want cap %d", len(subs), cap)
	}
	for _, sub := range subs {
		if sub.Endpoint == "https://fcm.googleapis.com/fcm/send/dev-1" {
			t.Errorf("oldest subscription dev-1 was not evicted: %#v", subs)
		}
	}

	// 已存在 endpoint（本人刷新）：不新增行、不淘汰。
	evicted, err = UpsertCapped(1, "https://fcm.googleapis.com/fcm/send/dev-2", "k2b", "s2b", "ja", cap)
	if err != nil {
		t.Fatalf("refresh capped: %v", err)
	}
	if evicted != 0 {
		t.Fatalf("refresh evicted = %d, want 0", evicted)
	}
	if subs := ListByUser(1); len(subs) != cap {
		t.Errorf("refresh changed row count to %d, want %d", len(subs), cap)
	}
}
