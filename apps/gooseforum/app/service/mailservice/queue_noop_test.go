package mailservice

import (
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
)

// TestProcessEmailTaskNoopSilentlySucceeds 验证 noop 类型任务被静默消费：
// 等时化 dummy 任务（账号枚举防护 #124）直接成功返回，不发送任何邮件。
func TestProcessEmailTaskNoopSilentlySucceeds(t *testing.T) {
	err := processEmailTask(EmailTask{To: "nobody@example.com", Type: "noop"})
	if err != nil {
		t.Fatalf("processEmailTask(noop) 应返回 nil，实际返回错误: %v", err)
	}
}

// TestProcessEmailTaskUnknownTypeReturnsError 验证未知类型仍返回错误，
// 确保 noop 只静默消费 dummy 任务，不会误吞真实邮件类型。
func TestProcessEmailTaskUnknownTypeReturnsError(t *testing.T) {
	err := processEmailTask(EmailTask{To: "nobody@example.com", Type: "bogus"})
	if err == nil {
		t.Fatal("processEmailTask(bogus) 应返回错误，实际返回 nil")
	}
}

// TestPendingNoopTaskDeletedAfterConsumption 验证邮件 worker 消费 noop 任务后
// 直接删除行（不保留 Success 状态、不打"发送成功"日志），避免未认证请求通过
// 未知邮箱路径让 task_queue 无界增长（账号枚举防护 #124 的副作用收敛）。
func TestPendingNoopTaskDeletedAfterConsumption(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task_queue: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	})

	if err := AddToQueue(EmailTask{To: "nobody@example.com", Type: "noop"}); err != nil {
		t.Fatalf("enqueue noop: %v", err)
	}

	if !processPendingEmailTasks(make(chan struct{})) {
		t.Fatal("processPendingEmailTasks 应处理完所有任务并返回 true")
	}

	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count task_queue: %v", err)
	}
	if count != 0 {
		t.Fatalf("noop 任务应被删除而非保留为 Success，task_queue 剩余 %d 行", count)
	}
}

// TestProcessClaimedEmailTaskNoopDeleted 验证重构后的单任务处理入口
// processClaimedEmailTask（review P1：提取独立函数以持有 defer cancel()）：
// noop 任务被原子领取、处理、并以租约 fencing 删除行，且不触发停止。
func TestProcessClaimedEmailTaskNoopDeleted(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task_queue: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	})

	if err := AddToQueue(EmailTask{To: "nobody@example.com", Type: "noop"}); err != nil {
		t.Fatalf("enqueue noop: %v", err)
	}
	task := taskQueue.GetPendingEmailTasks(10)
	if len(task) != 1 {
		t.Fatalf("pending email tasks = %d, want 1", len(task))
	}

	if stop := processClaimedEmailTask(make(chan struct{}), task[0]); stop {
		t.Fatal("processClaimedEmailTask 返回 stop=true，want false（队列未关闭）")
	}

	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Count(&count).Error; err != nil {
		t.Fatalf("count task_queue: %v", err)
	}
	if count != 0 {
		t.Fatalf("noop 任务应被删除，task_queue 剩余 %d 行", count)
	}
}
