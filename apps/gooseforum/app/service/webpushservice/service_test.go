package webpushservice

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushSubscription"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

// setupWebPushTestDB 迁移本组测试需要的表并清空行，保证断言基于干净基线。
// event_notification/task_queue/push_subscriptions 为共享 sqlite 内存连接上的
// 表，包内其他测试也会 AutoMigrate 同一批表，重复迁移幂等无害。
func setupWebPushTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &taskQueue.Entity{}, &pushSubscription.Entity{}); err != nil {
		t.Fatalf("migrate webpush tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&eventNotification.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&pushSubscription.Entity{})
}

// clearVapidKeys 清空测试注入的 VAPID 配置，避免泄漏到其他测试。
func clearVapidKeys(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		preferences.Set("webpush.vapid_public_key", "")
		preferences.Set("webpush.vapid_private_key", "")
	})
}

func TestVapidConfigEnabled(t *testing.T) {
	// base64url 无填充解码：87 chars = 65B（P-256 公钥），43 chars = 32B（私钥），
	// 44 chars = 33B。Enabled 只校验可解码与长度，不做密码学配对验证。
	cases := []struct {
		name string
		pub  string
		priv string
	}{
		{"both empty", "", ""},
		{"public only", strings.Repeat("A", 87), ""},
		{"private only", "", strings.Repeat("B", 43)},
		{"non-base64url public", "%%%", strings.Repeat("B", 43)},
		{"non-base64url private", strings.Repeat("A", 87), "%%%"},
		{"public wrong length", strings.Repeat("A", 43), strings.Repeat("B", 43)},
		{"private wrong length", strings.Repeat("A", 87), strings.Repeat("B", 44)},
	}
	for _, c := range cases {
		cfg := VapidConfig{PublicKey: c.pub, PrivateKey: c.priv}
		if cfg.Enabled() {
			t.Errorf("%s: Enabled() = true, want false", c.name)
		}
	}
}

func TestVapidConfigEnabledWithRealKeys(t *testing.T) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	cfg := VapidConfig{PublicKey: publicKey, PrivateKey: privateKey}
	if !cfg.Enabled() {
		t.Fatal("real VAPID key pair must be enabled")
	}
}

func TestAuthSchemeForEndpoint(t *testing.T) {
	cases := []struct {
		endpoint string
		want     webpush.AuthScheme
	}{
		{"https://fcm.googleapis.com/fcm/send/abc123", webpush.WebPush},
		{"https://updates.push.services.mozilla.com/wpush/v2/gAAAA", webpush.Vapid},
		{"https://push.example.com/sub", webpush.Vapid},
	}
	for _, c := range cases {
		if got := authSchemeFor(c.endpoint); got != c.want {
			t.Errorf("authSchemeFor(%q) = %q, want %q", c.endpoint, got, c.want)
		}
	}
}

func TestEnqueueNotificationDisabledNoTask(t *testing.T) {
	setupWebPushTestDB(t)
	clearVapidKeys(t)
	preferences.Set("webpush.vapid_public_key", "")
	preferences.Set("webpush.vapid_private_key", "")

	EnqueueNotification(1, 9001)
	if tasks := taskQueue.GetPendingTasksByType(TaskTypePush, 10); len(tasks) != 0 {
		t.Fatalf("disabled channel enqueued %d task(s)", len(tasks))
	}
}

func TestEnqueueNotificationEnabledCreatesTask(t *testing.T) {
	setupWebPushTestDB(t)
	clearVapidKeys(t)
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	preferences.Set("webpush.vapid_public_key", publicKey)
	preferences.Set("webpush.vapid_private_key", privateKey)

	EnqueueNotification(7, 9002)
	tasks := taskQueue.GetPendingTasksByType(TaskTypePush, 10)
	if len(tasks) != 1 {
		t.Fatalf("enabled channel produced %d task(s), want 1", len(tasks))
	}
	if got, want := tasks[0].Type, TaskTypePush+"9002"; got != want {
		t.Errorf("task type = %q, want %q", got, want)
	}
	var payload PushTask
	if err := json.Unmarshal([]byte(tasks[0].TaskJson), &payload); err != nil {
		t.Fatalf("unmarshal task json: %v", err)
	}
	if payload.UserId != 7 || payload.NotificationId != 9002 {
		t.Errorf("task payload = %+v, want userId=7 notificationId=9002", payload)
	}
}

func TestEnqueueNotificationZeroArgsNoTask(t *testing.T) {
	setupWebPushTestDB(t)
	clearVapidKeys(t)
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	preferences.Set("webpush.vapid_public_key", publicKey)
	preferences.Set("webpush.vapid_private_key", privateKey)

	EnqueueNotification(0, 0)
	EnqueueNotification(0, 9003)
	EnqueueNotification(7, 0)
	if tasks := taskQueue.GetPendingTasksByType(TaskTypePush, 10); len(tasks) != 0 {
		t.Fatalf("zero-arg enqueue produced %d task(s), want 0", len(tasks))
	}
}

// makePushTask 构造一个直接喂给 RunPushTask 的任务行（不落库）。
func makePushTask(t *testing.T, userId uint64, notificationId uint64) *taskQueue.Entity {
	t.Helper()
	raw, err := json.Marshal(PushTask{UserId: userId, NotificationId: notificationId})
	if err != nil {
		t.Fatalf("marshal push task: %v", err)
	}
	return &taskQueue.Entity{Type: TaskTypePush + "0", TaskJson: string(raw)}
}

// RunPushTask 各提前返回分支（真实 HTTP 发送之前 return）不 panic、返回 nil：
// 禁用通道、malformed 负载、参数为 0、通知行不存在、已读通知、无订阅。
func TestRunPushTaskNoopBranches(t *testing.T) {
	setupWebPushTestDB(t)
	clearVapidKeys(t)
	ctx := context.Background()

	// 1. 通道未启用：即使负载正常也直接 no-op。
	preferences.Set("webpush.vapid_public_key", "")
	preferences.Set("webpush.vapid_private_key", "")
	if err := RunPushTask(ctx, makePushTask(t, 1, 9999)); err != nil {
		t.Errorf("disabled-channel task error = %v, want nil", err)
	}

	// 2. malformed TaskJson：不可恢复数据错误，Success 收尾不重试。
	malformed := &taskQueue.Entity{Type: TaskTypePush + "0", TaskJson: "{not-json}"}
	if err := RunPushTask(ctx, malformed); err != nil {
		t.Errorf("malformed task error = %v, want nil", err)
	}

	// 3. 参数为 0。
	if err := RunPushTask(ctx, makePushTask(t, 0, 0)); err != nil {
		t.Errorf("zero-arg task error = %v, want nil", err)
	}

	// 启用通道后，覆盖依赖 DB 状态的提前返回分支。
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	preferences.Set("webpush.vapid_public_key", publicKey)
	preferences.Set("webpush.vapid_private_key", privateKey)

	// 4. 通知行不存在。
	if err := RunPushTask(ctx, makePushTask(t, 1, 424242)); err != nil {
		t.Errorf("missing-notification task error = %v, want nil", err)
	}

	// 5. 已读通知不推送。
	notification := eventNotification.Entity{
		UserId:    1,
		EventType: eventNotification.EventTypeComment,
		IsRead:    true,
	}
	if err := eventNotification.Create(&notification); err != nil {
		t.Fatalf("create read notification: %v", err)
	}
	if err := RunPushTask(ctx, makePushTask(t, 1, notification.Id)); err != nil {
		t.Errorf("read-notification task error = %v, want nil", err)
	}

	// 6. 未读通知但该用户无订阅。
	unread := eventNotification.Entity{
		UserId:    1,
		EventType: eventNotification.EventTypeComment,
		IsRead:    false,
	}
	if err := eventNotification.Create(&unread); err != nil {
		t.Fatalf("create unread notification: %v", err)
	}
	if err := RunPushTask(ctx, makePushTask(t, 1, unread.Id)); err != nil {
		t.Errorf("no-subscription task error = %v, want nil", err)
	}
}

func TestLogConfigStatusNoPanic(t *testing.T) {
	clearVapidKeys(t)
	preferences.Set("webpush.vapid_public_key", "not-base64url")
	preferences.Set("webpush.vapid_private_key", "not-base64url")
	LogConfigStatus()
	preferences.Set("webpush.vapid_public_key", "")
	preferences.Set("webpush.vapid_private_key", "")
	LogConfigStatus()
}

// ValidateEndpoint 白名单校验：仅接受 https + 推送服务白名单 host；
// http / IP 字面量 / localhost / 未知 host / userinfo / DNS 后缀伪装
// 一律拒绝（review P1 存储型 SSRF 防护）。
func TestValidateEndpoint(t *testing.T) {
	valid := []string{
		"https://fcm.googleapis.com/fcm/send/abc123",
		"https://updates.push.services.mozilla.com/wpush/v2/gAAAA",
		"https://web.push.apple.com/xyz",
		"https://android.googleapis.com/gcm/send/abc",
		"https://foo.fcm.googleapis.com/fcm/send/abc",
	}
	for _, ep := range valid {
		if err := ValidateEndpoint(ep); err != nil {
			t.Errorf("ValidateEndpoint(%q) = %v, want nil", ep, err)
		}
	}
	invalid := []string{
		"",                                       // 空
		"not-a-url",                              // 非 URL
		"http://fcm.googleapis.com/fcm/send/abc", // 非 https
		"https://push.example.com/sub",           // 非白名单 host
		"https://localhost/sub",                  // localhost
		"https://127.0.0.1/sub",                  // IP 字面量（回环）
		"https://10.0.0.8/sub",                   // IP 字面量（私网）
		"https://user:pass@fcm.googleapis.com/x", // userinfo
		"https://fcm.googleapis.com.evil.com/x",  // DNS 后缀伪装
	}
	for _, ep := range invalid {
		if err := ValidateEndpoint(ep); err == nil {
			t.Errorf("ValidateEndpoint(%q) = nil, want error", ep)
		}
	}
}

// RunPushTask 在 worker 关闭/租约取消（ctx 已取消）时返回 ctx.Err()：
// 任务保持可重试（taskQueue 置 Retrying），由下个进程续投而不是被标记
// Success 吞掉未发送订阅（review P2）。
func TestRunPushTaskReturnsContextErrorWhenCancelled(t *testing.T) {
	setupWebPushTestDB(t)
	clearVapidKeys(t)
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatalf("GenerateVAPIDKeys: %v", err)
	}
	preferences.Set("webpush.vapid_public_key", publicKey)
	preferences.Set("webpush.vapid_private_key", privateKey)

	notification := eventNotification.Entity{
		UserId:    1,
		EventType: eventNotification.EventTypeComment,
		IsRead:    false,
	}
	if err := eventNotification.Create(&notification); err != nil {
		t.Fatalf("create notification: %v", err)
	}
	if err := pushSubscription.Upsert(1, "https://fcm.googleapis.com/fcm/send/cancel",
		strings.Repeat("A", 87), strings.Repeat("B", 22), "zh"); err != nil {
		t.Fatalf("seed subscription: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 任务处理前已取消：模拟 worker 关闭/租约取消
	err = RunPushTask(ctx, makePushTask(t, 1, notification.Id))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunPushTask with cancelled ctx error = %v, want context.Canceled", err)
	}
}

// CleanupTerminalTasks 只清理早于保留期的终态（Success/Failed）行；
// 非终态（Pending）与保留期内的行保留（review P2 task_queue 无界增长防护）。
func TestCleanupTerminalTasks(t *testing.T) {
	setupWebPushTestDB(t)
	cutoff := time.Now()
	old := cutoff.Add(-8 * 24 * time.Hour)
	makeTask := func(status uint8, processedAt time.Time) *taskQueue.Entity {
		return &taskQueue.Entity{Type: TaskTypePush + "cleanup", Status: status, TaskJson: `{}`, ProcessedAt: processedAt}
	}
	oldSuccess := makeTask(taskQueue.StatusSuccess, old)
	if err := taskQueue.Create(oldSuccess); err != nil {
		t.Fatalf("create old success task: %v", err)
	}
	oldFailed := makeTask(taskQueue.StatusFailed, old)
	if err := taskQueue.Create(oldFailed); err != nil {
		t.Fatalf("create old failed task: %v", err)
	}
	fresh := makeTask(taskQueue.StatusSuccess, cutoff.Add(time.Minute))
	if err := taskQueue.Create(fresh); err != nil {
		t.Fatalf("create fresh task: %v", err)
	}
	pending := makeTask(taskQueue.StatusPending, old) // 非终态：即使很旧也保留
	if err := taskQueue.Create(pending); err != nil {
		t.Fatalf("create pending task: %v", err)
	}

	removed, err := CleanupTerminalTasks(cutoff, 100)
	if err != nil {
		t.Fatalf("CleanupTerminalTasks: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2 (old success + old failed; fresh/pending kept)", removed)
	}
	tasks := taskQueue.GetPendingTasksByType(TaskTypePush, 10)
	if len(tasks) != 1 || tasks[0].Id != pending.Id {
		t.Errorf("pending tasks after cleanup = %+v, want only pending id %d", tasks, pending.Id)
	}
}
