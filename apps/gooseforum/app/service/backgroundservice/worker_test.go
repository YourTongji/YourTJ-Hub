package backgroundservice

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestGetPendingTasksByTypeHonorsRunAt(t *testing.T) {
	setupWorkerTestDB(t)

	immediate := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusPending, TaskJson: `{"inboxId":1}`}
	if err := taskQueue.Create(immediate); err != nil {
		t.Fatalf("create immediate task: %v", err)
	}
	future := time.Now().Add(5 * time.Minute)
	deferred := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusPending, TaskJson: `{"inboxId":2}`, RunAt: &future}
	if err := taskQueue.Create(deferred); err != nil {
		t.Fatalf("create deferred task: %v", err)
	}
	past := time.Now().Add(-time.Minute)
	due := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusPending, TaskJson: `{"inboxId":3}`, RunAt: &past}
	if err := taskQueue.Create(due); err != nil {
		t.Fatalf("create due task: %v", err)
	}

	tasks := taskQueue.GetPendingTasksByType("agent.webhook", 10)
	got := map[uint64]bool{}
	for _, task := range tasks {
		got[task.Id] = true
	}
	if !got[immediate.Id] {
		t.Fatal("immediate task (nil run_at) must be picked up")
	}
	if got[deferred.Id] {
		t.Fatal("deferred task must not be picked up before run_at")
	}
	if !got[due.Id] {
		t.Fatal("due task (run_at in the past) must be picked up")
	}
}

func TestGetPendingTasksByTypeHonorsRunAtWithRetryingStatus(t *testing.T) {
	setupWorkerTestDB(t)

	future := time.Now().Add(5 * time.Minute)
	deferredRetry := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusRetrying, TaskJson: `{"inboxId":2}`, RunAt: &future}
	if err := taskQueue.Create(deferredRetry); err != nil {
		t.Fatalf("create deferred retry task: %v", err)
	}
	tasks := taskQueue.GetPendingTasksByType("agent.webhook", 10)
	if len(tasks) != 0 {
		t.Fatalf("deferred retrying task must stay invisible: %#v", tasks)
	}
}

func TestUpdateRunAtDefersTask(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusRetrying, TaskJson: `{"inboxId":1}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	future := time.Now().Add(time.Minute)
	if err := taskQueue.UpdateRunAt(task.Id, future); err != nil {
		t.Fatalf("UpdateRunAt: %v", err)
	}
	updated := mustGetTask(t, task.Id)
	if updated.RunAt == nil || !updated.RunAt.Equal(future) {
		t.Fatalf("run_at = %#v, want %v", updated.RunAt, future)
	}
	tasks := taskQueue.GetPendingTasksByType("agent.webhook", 10)
	if len(tasks) != 0 {
		t.Fatalf("deferred task must not be picked up: %#v", tasks)
	}
}

func TestRequeueStaleRunningTasksByType(t *testing.T) {
	setupWorkerTestDB(t)

	staleTime := time.Now().Add(-10 * time.Minute)
	freshTime := time.Now().Add(-time.Minute)
	deferred := time.Now().Add(5 * time.Minute)
	staleAgent := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusRunning, TaskJson: `{}`, ProcessedAt: staleTime, RunAt: &deferred}
	freshAgent := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusRunning, TaskJson: `{}`, ProcessedAt: freshTime}
	staleExport := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusRunning, TaskJson: `{}`, ProcessedAt: staleTime}
	for _, task := range []*taskQueue.Entity{staleAgent, freshAgent, staleExport} {
		if err := taskQueue.Create(task); err != nil {
			t.Fatalf("create task %q: %v", task.Type, err)
		}
	}

	requeueStaleTasks("agent.webhook", 5*time.Minute)

	gotStale := mustGetTask(t, staleAgent.Id)
	if gotStale.Status != taskQueue.StatusRetrying {
		t.Fatalf("stale agent status = %d, want retrying", gotStale.Status)
	}
	if gotStale.RunAt == nil || !gotStale.RunAt.Equal(deferred) {
		t.Fatalf("stale agent run_at = %#v, want preserved %v", gotStale.RunAt, deferred)
	}
	if gotFresh := mustGetTask(t, freshAgent.Id); gotFresh.Status != taskQueue.StatusRunning {
		t.Fatalf("fresh agent status = %d, want running", gotFresh.Status)
	}
	if gotExport := mustGetTask(t, staleExport.Id); gotExport.Status != taskQueue.StatusRunning {
		t.Fatalf("other worker status = %d, want running", gotExport.Status)
	}
}

func TestProcessTaskRecoversHandlerPanic(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "agent.webhook", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	stopCh := make(chan struct{})
	close(stopCh)
	if processTask(stopCh, "agent.webhook", task, func(context.Context, *taskQueue.Entity) error {
		panic("sensitive panic detail")
	}) {
		t.Fatal("processTask returned continue after closed stop channel")
	}

	updated := mustGetTask(t, task.Id)
	if updated.Status != taskQueue.StatusRetrying {
		t.Fatalf("status after panic = %d, want retrying", updated.Status)
	}
	if updated.RetryCount != 1 {
		t.Fatalf("retryCount after panic = %d, want 1", updated.RetryCount)
	}
	if updated.LastError != errTaskHandlerPanicked.Error() {
		t.Fatalf("lastError after panic = %q, want sanitized %q", updated.LastError, errTaskHandlerPanicked)
	}
}
