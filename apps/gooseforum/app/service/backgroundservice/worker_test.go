package backgroundservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
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

// TestClaimTaskAtomicity 验证多 worker 并发领取同一任务时只有一个成功
// （issue #138：原子 claim 取代"查询 pending + 无守卫更新 running"两步分离）。
func TestClaimTaskAtomicity(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	const workers = 8
	var claimed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, ok, err := taskQueue.ClaimTask(task.Id)
			if err != nil {
				t.Errorf("ClaimTask(%d) error = %v", task.Id, err)
				return
			}
			if ok {
				claimed.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := claimed.Load(); got != 1 {
		t.Fatalf("concurrent claims = %d, want exactly 1", got)
	}
	updated := mustGetTask(t, task.Id)
	if updated.Status != taskQueue.StatusRunning {
		t.Fatalf("status after claim = %d, want %d (running)", updated.Status, taskQueue.StatusRunning)
	}
}

// TestProcessTaskConcurrentSingleExecution 验证多个 worker 同时 processTask
// 同一任务时 handler 恰好执行一次（issue #138 的核心回归点：不重复外部副作用）。
func TestProcessTaskConcurrentSingleExecution(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	var runs atomic.Int32
	handler := func(_ context.Context, _ *taskQueue.Entity) error {
		runs.Add(1)
		time.Sleep(20 * time.Millisecond) // 拉长执行窗口，放大竞争
		return nil
	}

	const workers = 8
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			processTask(make(chan struct{}), "export", task, handler)
		}()
	}
	wg.Wait()

	if got := runs.Load(); got != 1 {
		t.Fatalf("handler executed %d times, want exactly 1", got)
	}
	updated := mustGetTask(t, task.Id)
	if updated.Status != taskQueue.StatusSuccess {
		t.Fatalf("status after concurrent processing = %d, want %d (success)", updated.Status, taskQueue.StatusSuccess)
	}
}

// TestLeaseFencing 验证租约 fencing：任务租约过期被回收并重新领取后，
// 旧 worker 的续租与终态写入都必须失败，不能覆盖新持有者（issue #138）。
func TestLeaseFencing(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	running, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !claimed {
		t.Fatalf("first claim failed: claimed=%v err=%v", claimed, err)
	}
	firstToken := running.LeaseToken

	// 模拟旧 worker 崩溃：租约过期 → 回收为 Pending → 新 worker 重新领取
	conn := dbconnect.Connect()
	if err := conn.Exec("UPDATE task_queue SET processed_at = ? WHERE id = ?", time.Now().Add(-20*time.Minute), task.Id).Error; err != nil {
		t.Fatalf("age lease: %v", err)
	}
	if err := taskQueue.RecoverStaleRunning("export", taskQueue.LeaseDuration); err != nil {
		t.Fatalf("recover stale: %v", err)
	}
	if updated := mustGetTask(t, task.Id); updated.Status != taskQueue.StatusPending {
		t.Fatalf("status after recover = %d, want %d (pending)", updated.Status, taskQueue.StatusPending)
	}
	if _, claimed, err := taskQueue.ClaimTask(task.Id); err != nil || !claimed {
		t.Fatalf("second claim failed: claimed=%v err=%v", claimed, err)
	}

	// 旧 worker 续租必须失败（其 fencing token 已被新持有者替换）
	ok, _, _, err := taskQueue.RenewLease(task.Id, firstToken)
	if err != nil {
		t.Fatalf("RenewLease error = %v", err)
	}
	if ok {
		t.Fatal("stale worker renewed lease, want fencing failure")
	}

	// 旧 worker 的终态写入必须被跳过（fencing），新持有者状态不受影响
	if err := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusSuccess, firstToken, nil); err != nil {
		t.Fatalf("UpdateStatusOwned error = %v", err)
	}
	if updated := mustGetTask(t, task.Id); updated.Status != taskQueue.StatusRunning {
		t.Fatalf("fenced write leaked: status = %d, want %d (running)", updated.Status, taskQueue.StatusRunning)
	}
}

// TestLeaseFencingSameLeaseCollision 构造最坏碰撞场景（review P1）：数据库
// 时间精度截断使新旧 worker 领取的 processed_at 落进同一精度槽位 —— 旧实现
// 把时间戳同时当作租约与 fencing token，此时旧 worker 的 CAS（WHERE
// processed_at = 旧值）会误判匹配新持有者，覆盖状态并重新引入重复外部副作用。
// 修复后 CAS 基于每次领取独立生成的 lease_token（UUID），时间戳碰撞不再
// 影响持有者判定：把新行 processed_at 覆写为与旧租约完全相同（模拟精度截断）
// 后，旧 worker 用旧 token 续租/写终态仍必须失败，任务保持新持有者状态。
func TestLeaseFencingSameLeaseCollision(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// 第一次领取：旧 worker 拿到 token A
	first, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !claimed {
		t.Fatalf("first claim failed: claimed=%v err=%v", claimed, err)
	}
	oldToken := first.LeaseToken
	oldLease := first.ProcessedAt

	// 回收并重新领取：新 worker 拿到 token B（与 A 必然不同）
	conn := dbconnect.Connect()
	if err := conn.Exec("UPDATE task_queue SET processed_at = ? WHERE id = ?", time.Now().Add(-20*time.Minute), task.Id).Error; err != nil {
		t.Fatalf("age lease: %v", err)
	}
	if err := taskQueue.RecoverStaleRunning("export", taskQueue.LeaseDuration); err != nil {
		t.Fatalf("recover stale: %v", err)
	}
	second, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !claimed {
		t.Fatalf("second claim failed: claimed=%v err=%v", claimed, err)
	}
	if second.LeaseToken == "" || second.LeaseToken == oldToken {
		t.Fatalf("second claim token = %q, want non-empty and different from old %q", second.LeaseToken, oldToken)
	}

	// 构造时间戳碰撞：把新持有者的 processed_at 覆写成与旧租约完全相同的值
	// （模拟 PostgreSQL timestamp(6) 等精度截断）。此时若持有者判定依赖
	// 时间戳，旧 worker 会误判仍持有租约；lease_token 保持 B 不变。
	if err := conn.Exec(
		"UPDATE task_queue SET processed_at = ? WHERE id = ?",
		oldLease, task.Id,
	).Error; err != nil {
		t.Fatalf("force lease collision: %v", err)
	}
	// 双保险：确认碰撞构造有效（时间戳维度匹配）且 token 维度不匹配。
	var tsMatch int64
	if err := conn.Raw(
		"SELECT COUNT(*) FROM task_queue WHERE id = ? AND status = ? AND processed_at = ?",
		task.Id, taskQueue.StatusRunning, oldLease,
	).Scan(&tsMatch).Error; err != nil {
		t.Fatalf("verify timestamp collision: %v", err)
	}
	if tsMatch != 1 {
		t.Fatalf("timestamp collision not constructed: processed_at match = %d, want 1", tsMatch)
	}

	// 旧 worker 续租必须失败：时间戳相同（旧实现下会通过）但 lease_token
	// 已更换为 B，CAS 基于 token 判定持有者。
	ok, _, _, err := taskQueue.RenewLease(task.Id, oldToken)
	if err != nil {
		t.Fatalf("RenewLease error = %v", err)
	}
	if ok {
		t.Fatal("stale worker renewed lease despite timestamp collision, want fencing failure")
	}

	// 旧 worker 的终态写入必须被跳过；若持有者判定依赖时间戳，此处会覆盖
	// 新持有者状态，测试失败。
	if err := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusSuccess, oldToken, nil); err != nil {
		t.Fatalf("UpdateStatusOwned error = %v", err)
	}
	if updated := mustGetTask(t, task.Id); updated.Status != taskQueue.StatusRunning {
		t.Fatalf("fenced write leaked: status = %d, want %d (running)", updated.Status, taskQueue.StatusRunning)
	}
}

// TestLeaseFencingPayloadWriteRejectedAfterReclaim 验证进度 payload 写入的
// fencing（review P1）：UpdateTaskJsonOwned 的 CAS 谓词是
// status=Running AND lease_token=token。worker A 租约过期被回收、任务被
// worker B 重新领取后，A 用旧 token 写 task_json 必须不命中（0 行受影响），
// B 的游标/文件名不被覆盖；A 用新 token（B 的）则正常写入。
func TestLeaseFencingPayloadWriteRejectedAfterReclaim(t *testing.T) {
	setupWorkerTestDB(t)

	task := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{"progress":0}`}
	if err := taskQueue.Create(task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	// worker A 领取，持有 token A
	first, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !claimed {
		t.Fatalf("first claim failed: claimed=%v err=%v", claimed, err)
	}
	oldToken := first.LeaseToken

	// A 的租约过期 → 回收为 Pending → B 重新领取（token B）
	conn := dbconnect.Connect()
	if err := conn.Exec("UPDATE task_queue SET processed_at = ? WHERE id = ?", time.Now().Add(-20*time.Minute), task.Id).Error; err != nil {
		t.Fatalf("age lease: %v", err)
	}
	if err := taskQueue.RecoverStaleRunning("export", taskQueue.LeaseDuration); err != nil {
		t.Fatalf("recover stale: %v", err)
	}
	second, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil || !claimed {
		t.Fatalf("second claim failed: claimed=%v err=%v", claimed, err)
	}
	newToken := second.LeaseToken
	if newToken == "" || newToken == oldToken {
		t.Fatalf("second claim token = %q, want non-empty and different from old %q", newToken, oldToken)
	}

	// B 先写入自己的进度（正常命中）
	if err := taskQueue.UpdateTaskJsonOwned(task.Id, newToken, `{"progress":50,"fileName":"export_b.csv"}`); err != nil {
		t.Fatalf("new owner progress write error = %v", err)
	}
	if updated := mustGetTask(t, task.Id); !strings.Contains(updated.TaskJson, "export_b.csv") {
		t.Fatalf("new owner progress not persisted: task_json = %s", updated.TaskJson)
	}

	// 旧 owner A 用旧 token 写进度必须不命中（0 行受影响），B 的 payload 保留
	if err := taskQueue.UpdateTaskJsonOwned(task.Id, oldToken, `{"progress":100,"fileName":"export_a.csv"}`); err != nil {
		t.Fatalf("stale owner progress write error = %v", err)
	}
	if updated := mustGetTask(t, task.Id); !strings.Contains(updated.TaskJson, "export_b.csv") {
		t.Fatalf("stale owner overwrote new owner payload: task_json = %s", updated.TaskJson)
	}

	// 旧 owner A 用旧 token 在任务终态后写进度同样必须不命中
	if err := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusSuccess, newToken, nil); err != nil {
		t.Fatalf("UpdateStatusOwned error = %v", err)
	}
	if err := taskQueue.UpdateTaskJsonOwned(task.Id, oldToken, `{"progress":100}`); err != nil {
		t.Fatalf("stale owner post-terminal progress write error = %v", err)
	}
	if updated := mustGetTask(t, task.Id); updated.Status != taskQueue.StatusSuccess {
		t.Fatalf("fenced payload write flipped terminal state: status = %d, want %d", updated.Status, taskQueue.StatusSuccess)
	}
}

// TestRecoverStaleRunningOnlyReclaimsExpiredLeases 验证过期租约回收只命中
// 崩溃残留（processed_at 超时），运行中（心跳续租中的）任务不被误回收。
func TestRecoverStaleRunningOnlyReclaimsExpiredLeases(t *testing.T) {
	setupWorkerTestDB(t)

	fresh := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	stale := &taskQueue.Entity{Type: "export", Status: taskQueue.StatusPending, TaskJson: `{}`}
	if err := taskQueue.Create(fresh); err != nil {
		t.Fatalf("create fresh task: %v", err)
	}
	if err := taskQueue.Create(stale); err != nil {
		t.Fatalf("create stale task: %v", err)
	}
	if _, claimed, err := taskQueue.ClaimTask(fresh.Id); err != nil || !claimed {
		t.Fatalf("claim fresh: claimed=%v err=%v", claimed, err)
	}
	if _, claimed, err := taskQueue.ClaimTask(stale.Id); err != nil || !claimed {
		t.Fatalf("claim stale: claimed=%v err=%v", claimed, err)
	}

	conn := dbconnect.Connect()
	if err := conn.Exec("UPDATE task_queue SET processed_at = ? WHERE id = ?", time.Now().Add(-20*time.Minute), stale.Id).Error; err != nil {
		t.Fatalf("age stale lease: %v", err)
	}

	if err := taskQueue.RecoverStaleRunning("export", taskQueue.LeaseDuration); err != nil {
		t.Fatalf("recover stale: %v", err)
	}

	if updated := mustGetTask(t, stale.Id); updated.Status != taskQueue.StatusPending {
		t.Fatalf("stale task status = %d, want %d (pending)", updated.Status, taskQueue.StatusPending)
	}
	if updated := mustGetTask(t, fresh.Id); updated.Status != taskQueue.StatusRunning {
		t.Fatalf("fresh task wrongly reclaimed: status = %d, want %d (running)", updated.Status, taskQueue.StatusRunning)
	}
}

// TestRecoverStaleEmailTasksReclaimsExpiredLegacyLeases 验证邮件回收与领取侧
// 使用同一类型谓词（review P1）：存量无前缀 activation/reset_password 行
// 崩溃（租约过期）后同样能被回收为 Pending，不会永久卡在 Running；
// 非邮件类型（export）与租约未过期的邮件行不被误回收。
func TestRecoverStaleEmailTasksReclaimsExpiredLegacyLeases(t *testing.T) {
	setupWorkerTestDB(t)

	createTask := func(typ string) *taskQueue.Entity {
		t.Helper()
		task := &taskQueue.Entity{Type: typ, Status: taskQueue.StatusPending, TaskJson: `{}`}
		if err := taskQueue.Create(task); err != nil {
			t.Fatalf("create %q task: %v", typ, err)
		}
		return task
	}

	prefixed := createTask("email.activation")
	legacyActivation := createTask("activation")
	legacyReset := createTask("reset_password")
	nonEmail := createTask("export")
	freshEmail := createTask("email.reset_password")

	for _, task := range []*taskQueue.Entity{prefixed, legacyActivation, legacyReset, nonEmail, freshEmail} {
		if _, claimed, err := taskQueue.ClaimTask(task.Id); err != nil || !claimed {
			t.Fatalf("claim %q (id %d): claimed=%v err=%v", task.Type, task.Id, claimed, err)
		}
	}

	conn := dbconnect.Connect()
	age := func(task *taskQueue.Entity) {
		t.Helper()
		if err := conn.Exec("UPDATE task_queue SET processed_at = ? WHERE id = ?", time.Now().Add(-20*time.Minute), task.Id).Error; err != nil {
			t.Fatalf("age lease of %q: %v", task.Type, err)
		}
	}
	// 模拟崩溃残留：三个邮件行租约过期；export 与 freshEmail 保持新鲜
	age(prefixed)
	age(legacyActivation)
	age(legacyReset)

	if err := taskQueue.RecoverStaleEmailTasks(taskQueue.LeaseDuration); err != nil {
		t.Fatalf("recover stale email tasks: %v", err)
	}

	for _, task := range []*taskQueue.Entity{prefixed, legacyActivation, legacyReset} {
		if updated := mustGetTask(t, task.Id); updated.Status != taskQueue.StatusPending {
			t.Fatalf("expired %q task status = %d, want %d (pending)", task.Type, updated.Status, taskQueue.StatusPending)
		}
	}
	if updated := mustGetTask(t, nonEmail.Id); updated.Status != taskQueue.StatusRunning {
		t.Fatalf("non-email %q task wrongly reclaimed: status = %d, want %d (running)", nonEmail.Type, updated.Status, taskQueue.StatusRunning)
	}
	if updated := mustGetTask(t, freshEmail.Id); updated.Status != taskQueue.StatusRunning {
		t.Fatalf("fresh %q task wrongly reclaimed: status = %d, want %d (running)", freshEmail.Type, updated.Status, taskQueue.StatusRunning)
	}
}
