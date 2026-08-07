package backgroundservice

import (
	"context"
	"errors"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
)

func setupWorkerTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate task_queue: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
}

func TestGetPendingTasksByTypeIsolation(t *testing.T) {
	setupWorkerTestDB(t)

	emailTask := &taskQueue.Entity{Type: "email.activation", Status: taskQueue.StatusPending, TaskJson: `{"to":"a@b.c"}`}
	exportTask := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	migrateTask := &taskQueue.Entity{Type: "file-migrate", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(emailTask); err != nil {
		t.Fatalf("create email task: %v", err)
	}
	if err := taskQueue.Create(exportTask); err != nil {
		t.Fatalf("create export task: %v", err)
	}
	if err := taskQueue.Create(migrateTask); err != nil {
		t.Fatalf("create migrate task: %v", err)
	}

	exportTasks := taskQueue.GetPendingTasksByType("export", 10)
	if len(exportTasks) != 1 || exportTasks[0].Id != exportTask.Id {
		t.Fatalf("GetPendingTasksByType(export) = %d tasks, want 1 (id %d)", len(exportTasks), exportTask.Id)
	}

	migrateTasks := taskQueue.GetPendingTasksByType("file-migrate", 10)
	if len(migrateTasks) != 1 || migrateTasks[0].Id != migrateTask.Id {
		t.Fatalf("GetPendingTasksByType(file-migrate) = %d tasks, want 1", len(migrateTasks))
	}

	// 邮件 worker：email.* 前缀任务 + 存量无前缀任务都匹配，其他类型不泄漏
	emailTasks := taskQueue.GetPendingEmailTasks(10)
	for _, task := range emailTasks {
		if task.Type != emailTask.Type {
			t.Fatalf("GetPendingEmailTasks leaked non-email task type %q", task.Type)
		}
	}

	// 带前缀的旧邮件任务（如 legacy type 无 "."）也应匹配
	legacy := &taskQueue.Entity{Type: "activation", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(legacy); err != nil {
		t.Fatalf("create legacy task: %v", err)
	}
	emailTasks2 := taskQueue.GetPendingEmailTasks(10)
	foundLegacy := false
	for _, task := range emailTasks2 {
		if task.Id == legacy.Id {
			foundLegacy = true
		}
	}
	if !foundLegacy {
		t.Fatal("GetPendingEmailTasks did not match legacy un-prefixed task")
	}
}

func TestProcessTaskRetryThenFail(t *testing.T) {
	setupWorkerTestDB(t)

	// 用 error 类型的 handler 先失败一次（重试），再成功
	failOnce := errors.New("boom")
	attempts := 0
	handler := func(_ context.Context, _ *taskQueue.Entity) error {
		attempts++
		if attempts == 1 {
			return failOnce
		}
		return nil
	}

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 第一次：失败 → 进入重试
	if !processTask(make(chan struct{}), "export", task, handler) {
		t.Fatal("processTask returned stop=true on first attempt, want continue")
	}
	updated := mustGetTask(t, task.Id)
	if updated.Status != taskQueue.StatusRetrying {
		t.Fatalf("status after first failure = %d, want %d (retrying)", updated.Status, taskQueue.StatusRetrying)
	}
	if updated.RetryCount != 1 {
		t.Fatalf("retryCount after first failure = %d, want 1", updated.RetryCount)
	}

	// 第二次：成功 → 标记成功
	if !processTask(make(chan struct{}), "export", task, handler) {
		t.Fatal("processTask returned stop=true on success, want continue")
	}
	updated = mustGetTask(t, task.Id)
	if updated.Status != taskQueue.StatusSuccess {
		t.Fatalf("status after success = %d, want %d (success)", updated.Status, taskQueue.StatusSuccess)
	}
}

func mustGetTask(t *testing.T, id uint64) taskQueue.Entity {
	t.Helper()
	task, err := taskQueue.GetByID(id)
	if err != nil {
		t.Fatalf("GetByID(%d) error = %v", id, err)
	}
	return task
}
