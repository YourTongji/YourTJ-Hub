package nativepushservice

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// setupNativePushTestDB 迁移本组测试需要的表并清空行，保证断言基于干净基线。
func setupNativePushTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&eventNotification.Entity{}, &taskQueue.Entity{}, &pushDevice.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatalf("migrate nativepush tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&eventNotification.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&pushDevice.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
}

// clearNativePushConfig 清空测试注入的原生推送配置，避免泄漏到其他测试。
func clearNativePushConfig(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		preferences.Set("push.apns.key_path", "")
		preferences.Set("push.apns.key_id", "")
		preferences.Set("push.apns.team_id", "")
		preferences.Set("push.apns.bundle_id", "")
		preferences.Set("push.apns.environment", "")
		preferences.Set("push.fcm.credentials_path", "")
		preferences.Set("push.fcm.project_id", "")
		preferences.Set("push.jpush.app_key", "")
		preferences.Set("push.jpush.master_secret", "")
	})
}

// enableAPNsForTest 注入一份 APNs 配置（文件路径指向不存在位置——enqueue/worker
// 早退分支不需要真实 .p8；发送路径才需要文件存在）。
func enableAPNsForTest() {
	preferences.Set("push.apns.key_path", "/nonexistent/AuthKey.p8")
	preferences.Set("push.apns.key_id", "ABC123DEFG")
	preferences.Set("push.apns.team_id", "DEF123GHIJ")
	preferences.Set("push.apns.bundle_id", "com.yourtj.app")
	preferences.Set("push.apns.environment", "sandbox")
	preferences.Set("push.fcm.credentials_path", "")
	preferences.Set("push.fcm.project_id", "")
}

func TestChannelsDisabledByDefault(t *testing.T) {
	clearNativePushConfig(t)
	preferences.Set("push.apns.key_path", "")
	preferences.Set("push.fcm.credentials_path", "")
	if LoadChannels().Enabled() {
		t.Fatal("empty config must be disabled")
	}
	if LoadAPNsConfig().Enabled() {
		t.Fatal("empty APNs config must be disabled")
	}
	enableAPNsForTest()
	if !LoadChannels().Enabled() {
		t.Fatal("configured APNs channel must be enabled")
	}
	if !LoadAPNsConfig().Enabled() {
		t.Fatal("APNs with all fields set must be enabled")
	}
}

func TestAPNsConfigEnabled(t *testing.T) {
	cases := []struct {
		name   string
		cfg    APNsConfig
		expect bool
	}{
		{"empty", APNsConfig{}, false},
		{"missing key path", APNsConfig{KeyID: "k", TeamID: "t", BundleID: "b", Environment: "sandbox"}, false},
		{"missing key id", APNsConfig{KeyPath: "/k.p8", TeamID: "t", BundleID: "b", Environment: "sandbox"}, false},
		{"missing team id", APNsConfig{KeyPath: "/k.p8", KeyID: "k", BundleID: "b", Environment: "sandbox"}, false},
		{"missing bundle id", APNsConfig{KeyPath: "/k.p8", KeyID: "k", TeamID: "t", Environment: "sandbox"}, false},
		{"bad environment", APNsConfig{KeyPath: "/k.p8", KeyID: "k", TeamID: "t", BundleID: "b", Environment: "staging"}, false},
		{"sandbox", APNsConfig{KeyPath: "/k.p8", KeyID: "k", TeamID: "t", BundleID: "b", Environment: "sandbox"}, true},
		{"production", APNsConfig{KeyPath: "/k.p8", KeyID: "k", TeamID: "t", BundleID: "b", Environment: "production"}, true},
		{"environment case-insensitive", APNsConfig{KeyPath: "/k.p8", KeyID: "k", TeamID: "t", BundleID: "b", Environment: "Production"}, true},
	}
	for _, c := range cases {
		if got := c.cfg.Enabled(); got != c.expect {
			t.Errorf("%s: Enabled() = %v, want %v", c.name, got, c.expect)
		}
	}
}

func TestFCMConfigEnabled(t *testing.T) {
	if (FCMConfig{}).Enabled() {
		t.Error("empty FCM config must be disabled")
	}
	if (FCMConfig{CredentialsPath: "/c.json"}).Enabled() {
		t.Error("missing project id must be disabled")
	}
	if (FCMConfig{ProjectID: "proj"}).Enabled() {
		t.Error("missing credentials path must be disabled")
	}
	if !(FCMConfig{CredentialsPath: "/c.json", ProjectID: "proj"}).Enabled() {
		t.Error("complete FCM config must be enabled")
	}
}

// EnqueueNotification：通道未配置时不产生任务行（dev 不产生 outbox 行）。
func TestEnqueueNotificationDisabledNoTask(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)

	EnqueueNotification(1, 9001)
	if tasks := taskQueue.GetPendingTasksByType(TaskTypeNativePush, 10); len(tasks) != 0 {
		t.Fatalf("disabled channel produced %d task(s), want 0", len(tasks))
	}
}

// EnqueueNotification：参数为 0 时不产生任务行（即使通道已启用）。
func TestEnqueueNotificationZeroArgsNoTask(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)
	enableAPNsForTest()

	EnqueueNotification(0, 0)
	EnqueueNotification(0, 9003)
	EnqueueNotification(7, 0)
	if tasks := taskQueue.GetPendingTasksByType(TaskTypeNativePush, 10); len(tasks) != 0 {
		t.Fatalf("zero-arg enqueue produced %d task(s), want 0", len(tasks))
	}
}

// EnqueueNotification：通道启用时产生任务行。
func TestEnqueueNotificationEnabledCreatesTask(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)
	enableAPNsForTest()

	EnqueueNotification(1, 9001)
	tasks := taskQueue.GetPendingTasksByType(TaskTypeNativePush, 10)
	if len(tasks) != 1 {
		t.Fatalf("enabled channel produced %d task(s), want 1", len(tasks))
	}
	var payload PushTask
	if err := json.Unmarshal([]byte(tasks[0].TaskJson), &payload); err != nil {
		t.Fatalf("decode task payload: %v", err)
	}
	if payload.UserId != 1 || payload.NotificationId != 9001 {
		t.Fatalf("task payload = %+v, want userId 1 notificationId 9001", payload)
	}
}

// makeNativePushTask 构造一个直接喂给 RunPushTask 的任务行（不落库）。
func makeNativePushTask(t *testing.T, userId uint64, notificationId uint64) *taskQueue.Entity {
	t.Helper()
	raw, err := json.Marshal(PushTask{UserId: userId, NotificationId: notificationId})
	if err != nil {
		t.Fatalf("marshal push task: %v", err)
	}
	return &taskQueue.Entity{Type: TaskTypeNativePush + "0", TaskJson: string(raw)}
}

// RunPushTask 各提前返回分支（真实 HTTP 发送之前 return）不 panic、返回 nil：
// 禁用通道、malformed 负载、参数为 0、通知行不存在、已读通知、无设备。
func TestRunPushTaskNoopBranches(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)
	ctx := context.Background()

	// 1. 通道未启用：即使负载正常也直接 no-op。
	if err := RunPushTask(ctx, makeNativePushTask(t, 1, 9999)); err != nil {
		t.Errorf("disabled-channel task error = %v, want nil", err)
	}

	// 2. malformed TaskJson：不可恢复数据错误，Success 收尾不重试。
	malformed := &taskQueue.Entity{Type: TaskTypeNativePush + "0", TaskJson: "{not-json}"}
	if err := RunPushTask(ctx, malformed); err != nil {
		t.Errorf("malformed task error = %v, want nil", err)
	}

	// 3. 参数为 0。
	if err := RunPushTask(ctx, makeNativePushTask(t, 0, 0)); err != nil {
		t.Errorf("zero-arg task error = %v, want nil", err)
	}

	// 启用通道后，覆盖依赖 DB 状态的提前返回分支。
	enableAPNsForTest()

	// 4. 通知行不存在。
	if err := RunPushTask(ctx, makeNativePushTask(t, 1, 424242)); err != nil {
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
	if err := RunPushTask(ctx, makeNativePushTask(t, 1, notification.Id)); err != nil {
		t.Errorf("read-notification task error = %v, want nil", err)
	}

	// 6. 未读通知但该用户无设备注册。
	unread := eventNotification.Entity{
		UserId:    1,
		EventType: eventNotification.EventTypeComment,
		IsRead:    false,
	}
	if err := eventNotification.Create(&unread); err != nil {
		t.Fatalf("create unread notification: %v", err)
	}
	if err := RunPushTask(ctx, makeNativePushTask(t, 1, unread.Id)); err != nil {
		t.Errorf("no-device task error = %v, want nil", err)
	}
}

func TestLogConfigStatusNoPanic(t *testing.T) {
	clearNativePushConfig(t)
	LogConfigStatus()
	enableAPNsForTest()
	LogConfigStatus()
}

func TestCleanupTerminalTasks(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)

	old := time.Now().Add(-48 * time.Hour)
	fresh := time.Now()
	// CleanupTerminalTasks 按终态 + ProcessedAt（处理时间）清理：用 ProcessedAt
	// 定位旧行（webpush 同款语义），CreatedAt 只是创建时间不参与保留判定。
	tasks := []*taskQueue.Entity{
		{Type: TaskTypeNativePush + "1", Status: taskQueue.StatusSuccess, TaskJson: "{}", ProcessedAt: old},
		{Type: TaskTypeNativePush + "2", Status: taskQueue.StatusFailed, TaskJson: "{}", ProcessedAt: old},
		{Type: TaskTypeNativePush + "3", Status: taskQueue.StatusPending, TaskJson: "{}", ProcessedAt: old},
		{Type: TaskTypeNativePush + "4", Status: taskQueue.StatusSuccess, TaskJson: "{}", ProcessedAt: fresh},
	}
	for _, task := range tasks {
		if err := taskQueue.Create(task); err != nil {
			t.Fatalf("create task: %v", err)
		}
	}
	removed, err := CleanupTerminalTasks(time.Now().Add(-24*time.Hour), 100)
	if err != nil {
		t.Fatalf("CleanupTerminalTasks: %v", err)
	}
	if removed != 2 {
		t.Fatalf("removed = %d, want 2 (old terminal rows only; fresh/pending kept)", removed)
	}
	remaining := taskQueue.GetPendingTasksByType(TaskTypeNativePush, 10)
	if len(remaining) != 1 || remaining[0].Status != taskQueue.StatusPending {
		t.Fatalf("remaining = %+v, want only the pending row", remaining)
	}
}
