package mailservice

import (
	"errors"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
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

// newClaimedEmailTask 创建并原子领取一个邮件任务，返回领取后的实体与
// fencing token，供 outcome 分支测试直接调用 writeEmailTaskOutcome。
func newClaimedEmailTask(t *testing.T, payload string) (task *taskQueue.Entity, running taskQueue.Entity, token string) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task_queue: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	t.Cleanup(func() {
		conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	})

	if err := AddToQueue(EmailTask{To: "nobody@example.com", Type: "activation", Token: "t"}); err != nil {
		t.Fatalf("enqueue task: %v", err)
	}
	// 覆写 payload 为测试需要的形态
	task = taskQueue.GetPendingEmailTasks(10)[0]
	if payload != "" {
		if err := conn.Model(&taskQueue.Entity{}).Where("id = ?", task.Id).Update("task_json", payload).Error; err != nil {
			t.Fatalf("overwrite task_json: %v", err)
		}
		task = taskQueue.GetPendingEmailTasks(10)[0]
	}
	claimed, ok, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !ok {
		t.Fatalf("claim task: claimed=%v err=%v", ok, err)
	}
	return task, claimed, claimed.LeaseToken
}

// TestWriteEmailTaskOutcomeSent 验证 emailOutcomeSent 分支：以 fencing token
// 写 Success，返回不重试。
func TestWriteEmailTaskOutcomeSent(t *testing.T) {
	_, running, token := newClaimedEmailTask(t, "")

	if retrying := writeEmailTaskOutcome(&running, &running, EmailTask{Type: "activation", To: "nobody@example.com"}, emailOutcomeSent, nil, token); retrying {
		t.Fatal("emailOutcomeSent 应返回 retrying=false")
	}
	updated := taskQueue.GetPendingEmailTasks(10)
	if len(updated) != 0 {
		row := mustGetEmailTask(t, running.Id)
		if row.Status != taskQueue.StatusSuccess {
			t.Fatalf("status = %d, want %d (success)", row.Status, taskQueue.StatusSuccess)
		}
	}
}

// TestWriteEmailTaskOutcomeFailedRetriesUntilCap 验证 emailOutcomeFailed 分支
// 的 retry-count fencing 边界（review G3/G4）：RetryCount < MaxRetries 时
// 标记 Retrying 并返回重试；RetryCount 已达 MaxRetries 时直接 Failed。
// 关键：重试上限判断必须用 running.RetryCount（G4），陈旧实体与 running
// 不一致时以 running 为准。
func TestWriteEmailTaskOutcomeFailedRetriesUntilCap(t *testing.T) {
	// 场景 1：running.RetryCount = 1 < MaxRetries(3) → Retrying + retrying=true
	_, running, token := newClaimedEmailTask(t, "")
	running.RetryCount = 1
	retrying := writeEmailTaskOutcome(&running, &running, EmailTask{Type: "activation", To: "nobody@example.com"}, emailOutcomeFailed, errors.New("smtp down"), token)
	if !retrying {
		t.Fatal("RetryCount<MaxRetries 失败应返回 retrying=true")
	}
	row := mustGetEmailTask(t, running.Id)
	if row.Status != taskQueue.StatusRetrying {
		t.Fatalf("status = %d, want %d (retrying)", row.Status, taskQueue.StatusRetrying)
	}

	// 场景 2：running.RetryCount = MaxRetries → Failed + retrying=false
	_, running2, token2 := newClaimedEmailTask(t, "")
	running2.RetryCount = MaxRetries
	retrying2 := writeEmailTaskOutcome(&running2, &running2, EmailTask{Type: "activation", To: "nobody@example.com"}, emailOutcomeFailed, errors.New("smtp down"), token2)
	if retrying2 {
		t.Fatal("RetryCount>=MaxRetries 失败应返回 retrying=false")
	}
	row2 := mustGetEmailTask(t, running2.Id)
	if row2.Status != taskQueue.StatusFailed {
		t.Fatalf("status = %d, want %d (failed)", row2.Status, taskQueue.StatusFailed)
	}

	// 场景 3（G4 回归）：fetch 时陈旧 task.RetryCount=1，但 running（ClaimTask
	// 重读）RetryCount=MaxRetries —— 判断必须用 running，直接 Failed，
	// 不能用陈旧值再重试一次。
	_, running3, token3 := newClaimedEmailTask(t, "")
	staleTask := running3
	staleTask.RetryCount = 1 // 批次拉取时的旧值
	running3.RetryCount = MaxRetries
	retrying3 := writeEmailTaskOutcome(&staleTask, &running3, EmailTask{Type: "activation", To: "nobody@example.com"}, emailOutcomeFailed, errors.New("smtp down"), token3)
	if retrying3 {
		t.Fatal("G4: 陈旧 task.RetryCount 不得触发再重试, running.RetryCount 已到上限应直接 Failed")
	}
	row3 := mustGetEmailTask(t, running3.Id)
	if row3.Status != taskQueue.StatusFailed {
		t.Fatalf("G4: status = %d, want %d (failed)", row3.Status, taskQueue.StatusFailed)
	}
}

func mustGetEmailTask(t *testing.T, id uint64) taskQueue.Entity {
	t.Helper()
	row, err := taskQueue.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID(%d) error = %v", id, err)
	}
	return row
}
