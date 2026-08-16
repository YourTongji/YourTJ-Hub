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
		Repo:     "file://" + src,
		Branch:   "main",
		CloneDir: filepath.Join(t.TempDir(), "clone"),
	}
}

// syncOnceEnterHook 的测试门控状态：每次 syncOnce 进入时计数 +1，然后阻塞
// 在当前 gate 通道上（abort 通道在清理时关闭，防止测试失败时泄漏阻塞）。
var (
	syncTestHookMu   sync.Mutex
	syncTestHookGate chan struct{}
)

// installSyncTestHook 安装 syncOnce 入口钩子并返回执行计数。
func installSyncTestHook(t *testing.T) *atomic.Int64 {
	t.Helper()
	calls := &atomic.Int64{}
	abort := make(chan struct{})
	syncOnceEnterHook = func() {
		calls.Add(1)
		syncTestHookMu.Lock()
		gate := syncTestHookGate
		syncTestHookMu.Unlock()
		select {
		case <-gate:
		case <-abort:
		}
	}
	t.Cleanup(func() {
		close(abort)
		syncOnceEnterHook = nil
		syncTestHookGate = nil
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
	runs := wikiSyncRuns.ListRecent(10)
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
	runs := wikiSyncRuns.ListRecent(10)
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
	runs := wikiSyncRuns.ListRecent(10)
	if len(runs) != 2 {
		t.Fatalf("sync run rows=%d, want 2", len(runs))
	}
	for _, r := range runs {
		if r.Status != wikiSyncRuns.StatusSuccess {
			t.Fatalf("run %d status=%d, want success", r.Id, r.Status)
		}
	}
}
