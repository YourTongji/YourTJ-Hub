package pushDevice

import (
	"fmt"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

// setupPushDeviceTestDB 迁移 push_device 表并清空行（token 全局唯一，
// 共享 sqlite 内存连接上的跨测试残留会污染 Upsert 断言）。
func setupPushDeviceTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate push_device: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&Entity{})
}

func TestUpsertCreateAndConvergeOwnership(t *testing.T) {
	setupPushDeviceTestDB(t)
	now := time.Now()
	const token = "ios-device-token-abc"

	// 新 token：Upsert 创建一行。
	if err := Upsert(1, PlatformIOS, token, now); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	devices := ListByUser(1)
	if len(devices) != 1 {
		t.Fatalf("ListByUser(1) = %d rows, want 1", len(devices))
	}
	if devices[0].Token != token || devices[0].Platform != PlatformIOS {
		t.Errorf("created row = %#v, want token/platform set", devices[0])
	}

	// 同一 token 换用户（换账号登录后 token 不变）：冲突收敛归属到新用户，
	// 并刷新最近注册时间。
	later := now.Add(time.Hour)
	if err := Upsert(2, PlatformAndroid, token, later); err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if devices := ListByUser(1); len(devices) != 0 {
		t.Errorf("old user still holds %d device(s), want 0", len(devices))
	}
	devices = ListByUser(2)
	if len(devices) != 1 {
		t.Fatalf("ListByUser(2) = %d rows, want 1", len(devices))
	}
	if devices[0].Platform != PlatformAndroid {
		t.Errorf("converged row platform = %q, want android", devices[0].Platform)
	}
	if !devices[0].LastRegisteredAt.Equal(later) {
		t.Errorf("LastRegisteredAt = %v, want %v", devices[0].LastRegisteredAt, later)
	}
}

func TestDeleteByUser(t *testing.T) {
	setupPushDeviceTestDB(t)
	now := time.Now()
	if err := Upsert(1, PlatformIOS, "u1-a", now); err != nil {
		t.Fatalf("upsert u1-a: %v", err)
	}
	if err := Upsert(1, PlatformAndroid, "u1-b", now); err != nil {
		t.Fatalf("upsert u1-b: %v", err)
	}
	if err := Upsert(2, PlatformIOS, "u2-a", now); err != nil {
		t.Fatalf("upsert u2-a: %v", err)
	}

	if err := DeleteByUser(1); err != nil {
		t.Fatalf("DeleteByUser(1): %v", err)
	}
	if devices := ListByUser(1); len(devices) != 0 {
		t.Errorf("user 1 still holds %d device(s), want 0", len(devices))
	}
	// 其他用户不受影响。
	if devices := ListByUser(2); len(devices) != 1 {
		t.Errorf("user 2 devices = %d, want 1", len(devices))
	}
}

func TestDeleteByTokenIdempotentAndOwnerScoped(t *testing.T) {
	setupPushDeviceTestDB(t)
	now := time.Now()
	const token = "ios-device-token-stale"

	// owner 删除成功且幂等。
	if err := Upsert(1, PlatformIOS, token, now); err != nil {
		t.Fatalf("upsert user 1: %v", err)
	}
	if err := DeleteByToken(token, 1); err != nil {
		t.Fatalf("first delete: %v", err)
	}
	if err := DeleteByToken(token, 1); err != nil {
		t.Fatalf("second delete (idempotent): %v", err)
	}
	if devices := ListByUser(1); len(devices) != 0 {
		t.Errorf("user 1 still holds %d device(s), want 0", len(devices))
	}

	// 换账号登录：token 收敛到 user 2。user 1 尝试删除已不属于自己的 token：
	// 无匹配行，静默成功且不影响 user 2（越权/竞争防护，与 pushSubscription 同款）。
	if err := Upsert(2, PlatformAndroid, token, now); err != nil {
		t.Fatalf("upsert user 2: %v", err)
	}
	if err := DeleteByToken(token, 1); err != nil {
		t.Fatalf("foreign-owner delete: %v", err)
	}
	if devices := ListByUser(2); len(devices) != 1 {
		t.Fatalf("user 2 device was deleted by foreign owner: %d rows, want 1", len(devices))
	}
	if err := DeleteByToken(token, 2); err != nil {
		t.Fatalf("owned delete: %v", err)
	}
	if devices := ListByUser(2); len(devices) != 0 {
		t.Errorf("user 2 still holds %d device(s), want 0", len(devices))
	}
}

// UpsertCapped 在行数达到上限时按 id 升序淘汰最旧行：worker 串行 fan-out
// 有界（与 pushSubscription.UpsertCapped 同口径），同 token 刷新与换账号
// 收敛不触发淘汰。
func TestUpsertCappedEvictsOldestWhenAtLimit(t *testing.T) {
	setupPushDeviceTestDB(t)
	const cap = 3
	now := time.Now()
	// 先写满上限（token 均为合法形状，rep 层不做推送服务校验）。
	for i := 1; i <= cap; i++ {
		token := fmt.Sprintf("ios-device-token-%d", i)
		if err := Upsert(1, PlatformIOS, token, now); err != nil {
			t.Fatalf("seed %d: %v", i, err)
		}
	}
	// 新增第 4 台：淘汰最旧（dev-1），保留 dev-2..dev-4。
	evicted, err := UpsertCapped(1, PlatformIOS, "ios-device-token-4", cap, now)
	if err != nil {
		t.Fatalf("upsert capped: %v", err)
	}
	if evicted != 1 {
		t.Fatalf("evicted = %d, want 1", evicted)
	}
	devices := ListByUser(1)
	if len(devices) != cap {
		t.Fatalf("ListByUser(1) = %d rows, want cap %d", len(devices), cap)
	}
	for _, dev := range devices {
		if dev.Token == "ios-device-token-1" {
			t.Errorf("oldest device dev-1 was not evicted: %#v", devices)
		}
	}

	// 已存在 token（本人刷新）：不新增行、不淘汰。
	evicted, err = UpsertCapped(1, PlatformAndroid, "ios-device-token-2", cap, now)
	if err != nil {
		t.Fatalf("refresh capped: %v", err)
	}
	if evicted != 0 {
		t.Fatalf("refresh evicted = %d, want 0", evicted)
	}
	if devices := ListByUser(1); len(devices) != cap {
		t.Errorf("refresh changed row count to %d, want %d", len(devices), cap)
	}
}
