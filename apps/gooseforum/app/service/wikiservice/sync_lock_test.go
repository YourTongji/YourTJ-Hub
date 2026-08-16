package wikiservice

import (
	"errors"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiSyncRuns"
)

// initLocalWikiRepo 创建一个带一个 .md 文件的本地 git 仓库，返回以 file://
// 为源、CloneDir 为空目录的同步配置（首次 syncOnce 真实 clone，后续
// fetch + reset --hard）。
func initLocalWikiRepo(t *testing.T) GitConfig {
	t.Helper()
	src := t.TempDir()
	writeRepoFile(t, src, "docs/page.md", "---\ntitle: 页面\n---\n\n# 标题\n\n正文")
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = src
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-b", "main")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "add", ".")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false", "commit", "-m", "init")
	return GitConfig{
		Enable:   true,
		Repo:     "file://" + src,
		Branch:   "main",
		CloneDir: filepath.Join(t.TempDir(), "clone"),
	}
}

// syncOnceEnterHook 的测试门控状态：每次 syncOnce 进入时计数 +1，然后阻塞
// 在当前 gate 通道上（abort 通道在清理时关闭，防止测试失败时泄漏阻塞）。
var syncTestHookMu sync.Mutex
var syncTestHookGate chan struct{}
var syncTestHookAbort chan struct{}

// installSyncTestHook 安装 syncOnce 入口钩子并返回执行计数。
// 钩子先捕获当前 gate 再计数：调用方观察到计数后即可确定该次 syncOnce
// 已被阻塞在（已捕获的）gate 上，随后切换 gate 不会影响它。
// 默认 gate 为已关闭通道（不阻塞）——未调用 setSyncTestGate 的测试
// （如顺序同步测试）不会被钩子卡住。
// 清理时：先关闭 abort 放行任何被门控卡住的 syncOnce，再等待同步 goroutine
// 完全退出（SyncWithConfig 的 defer 释放锁是最后一步）——失败测试泄漏的
// 锁/运行不会串入下一个测试。
func installSyncTestHook(t *testing.T) *atomic.Int64 {
	t.Helper()
	calls := &atomic.Int64{}
	abort := make(chan struct{})
	open := make(chan struct{})
	close(open)
	syncTestHookMu.Lock()
	syncTestHookGate = open
	syncTestHookAbort = abort
	syncTestHookMu.Unlock()
	h := func() {
		syncTestHookMu.Lock()
		gate := syncTestHookGate
		abort := syncTestHookAbort
		syncTestHookMu.Unlock()
		calls.Add(1)
		select {
		case <-gate:
		case <-abort:
		}
	}
	syncOnceEnterHook.Store(&h)
	t.Cleanup(func() {
		close(abort)
		// 不再接受新的门控，然后等待泄漏的同步 goroutine 完全退出并释放锁
		// （SyncWithConfig 的 defer 以 ReleaseSyncLock 收尾）——失败测试泄漏的
		// 锁/运行不会串入下一个测试。锁空闲即代表已释放。
		syncOnceEnterHook.Store(nil)
		deadline := time.Now().Add(30 * time.Second)
		for {
			if TryAcquireSyncLock() {
				ReleaseSyncLock()
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("sync lock leaked across tests after hook abort")
			}
			time.Sleep(2 * time.Millisecond)
		}
		syncTestHookMu.Lock()
		syncTestHookGate = nil
		syncTestHookAbort = nil
		syncTestHookMu.Unlock()
		syncPending.Store(false)
	})
	return calls
}

// setSyncTestGate 替换钩子当前阻塞的 gate 通道。
func setSyncTestGate(gate chan struct{}) {
	syncTestHookMu.Lock()
	syncTestHookGate = gate
	syncTestHookMu.Unlock()
}

// waitSyncOnceCalls 等待 syncOnce 执行计数达到 want（说明对应同步已进入钩子
// 并被阻塞，锁必然已持有）。
func waitSyncOnceCalls(t *testing.T, calls *atomic.Int64, want int64, what string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for calls.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for %s (syncOnce calls=%d, want %d)", what, calls.Load(), want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// TestSyncWithConfigPendingRerunHoldsLock #285 回归：补跑必须持锁执行。
// 场景：A 持锁阻塞 → B 到达置 pending → 放行 A 主运行 → A 的 defer 持锁启动
// 补跑 #2（再次阻塞）→ C 在补跑期间到达：必须返回 ErrSyncAlreadyRunning
// （修复前 defer 先释放锁再补跑，C 会成功获取锁并并发进入 syncOnce）。
// 放行后 C 的 pending 由 A 的排空循环消费为补跑 #3——C 不丢失。
func TestSyncWithConfigPendingRerunHoldsLock(t *testing.T) {
	setupWikiTestDB(t)
	cfg := initLocalWikiRepo(t)

	calls := installSyncTestHook(t)
	g1 := make(chan struct{})
	setSyncTestGate(g1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := SyncWithConfig(cfg, "a"); err != nil {
			t.Errorf("sync A: %v", err)
		}
	}()
	waitSyncOnceCalls(t, calls, 1, "A entered syncOnce")

	// B 到达：A 持锁 → ErrSyncAlreadyRunning，置 pending。
	if _, err := SyncWithConfig(cfg, "b"); !errors.Is(err, ErrSyncAlreadyRunning) {
		t.Fatalf("sync B err=%v, want ErrSyncAlreadyRunning", err)
	}

	// 放行 A 的主运行；A 的 defer 必须持锁启动补跑 #2（阻塞在 g2 上）。
	g2 := make(chan struct{})
	setSyncTestGate(g2)
	close(g1)
	waitSyncOnceCalls(t, calls, 2, "pending rerun entered syncOnce")

	// C 在补跑期间到达：锁必须仍被持有（#285 核心断言），C 不得进入 syncOnce。
	cResult := make(chan error, 1)
	go func() {
		_, err := SyncWithConfig(cfg, "c")
		cResult <- err
	}()
	select {
	case err := <-cResult:
		if !errors.Is(err, ErrSyncAlreadyRunning) {
			t.Fatalf("sync C during pending rerun err=%v, want ErrSyncAlreadyRunning", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sync C did not return ErrSyncAlreadyRunning: lock was released during pending rerun (#285)")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("syncOnce calls=%d after C, want 2 (C must not enter syncOnce)", got)
	}

	// 放行补跑 #2；C 的 pending 由排空循环消费为补跑 #3。
	close(g2)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("sync A (with pending reruns) did not finish")
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("syncOnce calls=%d, want 3 (A + rerun + rerun for C, C not lost)", got)
	}
	runs, err := wikiSyncRuns.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent runs: %v", err)
	}
	if len(runs) != 3 {
		t.Fatalf("sync run rows=%d, want 3", len(runs))
	}
	// 排空结束后锁已释放：下一次同步可正常获取。
	if !TryAcquireSyncLock() {
		t.Fatal("sync lock should be released after pending drain")
	}
	ReleaseSyncLock()
}

// TestSyncWithConfigPendingCoalesced 同一运行期间多个触发到达 → 合并为一次
// 持锁补跑（不丢不并发）：A 阻塞期间 B、C 到达；放行后只补跑一次，
// syncOnce 共执行 2 次。
func TestSyncWithConfigPendingCoalesced(t *testing.T) {
	setupWikiTestDB(t)
	cfg := initLocalWikiRepo(t)

	calls := installSyncTestHook(t)
	g1 := make(chan struct{})
	setSyncTestGate(g1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := SyncWithConfig(cfg, "a"); err != nil {
			t.Errorf("sync A: %v", err)
		}
	}()
	waitSyncOnceCalls(t, calls, 1, "A entered syncOnce")

	for _, trigger := range []string{"b", "c"} {
		if _, err := SyncWithConfig(cfg, trigger); !errors.Is(err, ErrSyncAlreadyRunning) {
			t.Fatalf("sync %s err=%v, want ErrSyncAlreadyRunning", trigger, err)
		}
	}
	close(g1)
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("sync A did not finish")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("syncOnce calls=%d, want 2 (A + one coalesced pending rerun)", got)
	}
	runs, err := wikiSyncRuns.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("sync run rows=%d, want 2", len(runs))
	}
}

// TestSyncWithConfigSequential 无竞争时顺序同步正常：锁正确释放、幂等、
// run 记录 success。
func TestSyncWithConfigSequential(t *testing.T) {
	setupWikiTestDB(t)
	cfg := initLocalWikiRepo(t)

	res, err := SyncWithConfig(cfg, "manual")
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if res.PagesAdded != 1 {
		t.Fatalf("PagesAdded=%d, want 1", res.PagesAdded)
	}
	if _, err := SyncWithConfig(cfg, "manual"); err != nil {
		t.Fatalf("second sync: %v", err)
	}
	runs, err := wikiSyncRuns.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("sync run rows=%d, want 2", len(runs))
	}
	for _, r := range runs {
		if r.Status != wikiSyncRuns.StatusSuccess {
			t.Fatalf("run %d status=%d, want success", r.Id, r.Status)
		}
	}
}

// TestSyncWithConfigFinalHandoffNoLostTrigger review P1 回归：最终排空检查与
// 解锁之间的窗口不得丢触发。
// 场景：A 完成主运行，defer 进入最终交接（持有 syncMu+syncHandoffMu，
// pending 检查已过，即将 ReleaseSyncLock，release 钩子阻塞此处）→ C 此刻
// 到达：TryLock 失败后在 syncHandoffMu 上等待；A 释放锁后 C 取得
// syncHandoffMu 并 TryLock 成功，直接进入正常同步——不返回
// ErrSyncAlreadyRunning、不置 pending 后无人消费（修复前 C 会写 pending
// 返回 ErrSyncAlreadyRunning，且已无持锁者消费该标记，更新 stale 到下次
// 触发）。
func TestSyncWithConfigFinalHandoffNoLostTrigger(t *testing.T) {
	setupWikiTestDB(t)
	cfg := initLocalWikiRepo(t)

	calls := installSyncTestHook(t)

	// release 钩子：阻塞持锁方在「最终检查已过、即将释放锁」的窗口。
	releaseEntered := make(chan struct{})
	releaseGate := make(chan struct{})
	releaseAbort := make(chan struct{})
	var releaseOnce sync.Once
	rh := func() {
		releaseOnce.Do(func() { close(releaseEntered) })
		select {
		case <-releaseGate:
		case <-releaseAbort:
		}
	}
	syncReleaseHook.Store(&rh)
	// 清理顺序（LIFO）：先放行 release 钩子，再等 installSyncTestHook 的锁
	// 释放等待——失败测试泄漏的锁/运行不会串入下一个测试。
	t.Cleanup(func() {
		close(releaseAbort)
		syncReleaseHook.Store(nil)
	})

	g1 := make(chan struct{})
	setSyncTestGate(g1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, err := SyncWithConfig(cfg, "a"); err != nil {
			t.Errorf("sync A: %v", err)
		}
	}()
	waitSyncOnceCalls(t, calls, 1, "A entered syncOnce")

	// 放行 A 主运行；A 的 defer 进入最终交接并阻塞在 release 钩子上。
	close(g1)
	select {
	case <-releaseEntered:
	case <-time.After(10 * time.Second):
		t.Fatal("A did not reach final handoff (release hook)")
	}

	// C 此刻到达：TryLock 失败 → 在 syncHandoffMu 上等待 A 释放锁。
	cDone := make(chan struct{})
	var cErr error
	go func() {
		defer close(cDone)
		_, cErr = SyncWithConfig(cfg, "c")
	}()

	// C 必须等待交接（锁未释放、不得以 ErrSyncAlreadyRunning 提前返回——
	// 那正是修复前的丢触发窗口）。
	select {
	case <-cDone:
		t.Fatalf("sync C returned before lock release: err=%v (must wait for handoff)", cErr)
	case <-time.After(200 * time.Millisecond):
	}

	// 放行 A：释放锁 → C 取得 syncHandoffMu + TryLock 成功 → 直接进入同步。
	close(releaseGate)
	select {
	case <-cDone:
	case <-time.After(30 * time.Second):
		t.Fatal("sync C did not complete after lock release")
	}
	if cErr != nil {
		t.Fatalf("sync C err=%v, want nil (final handoff must not lose the trigger)", cErr)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("sync A did not finish")
	}
	// A + C 各一次 syncOnce；C 直接同步（非 pending 补跑）。
	if got := calls.Load(); got != 2 {
		t.Fatalf("syncOnce calls=%d, want 2 (A + C direct)", got)
	}
	runs, err := wikiSyncRuns.ListRecent(10)
	if err != nil {
		t.Fatalf("list recent runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("sync run rows=%d, want 2 (A + C)", len(runs))
	}
}
