package console

import (
	"errors"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/gin-gonic/gin"
)

// newTestServeRuntime builds a serveRuntime bound to an ephemeral port so
// tests do not collide with the running instance or each other. The migration
// function is stubbed by the caller via withMigrate and defaults to success.
func newTestServeRuntime(t *testing.T) *serveRuntime {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral port: %v", err)
	}
	engine := newGinEngine()
	gate := middleware.NewStartupGate()
	engine.Use(gate.Handler)
	engine.GET("/ready", func(c *gin.Context) { c.Status(http.StatusOK) })
	rt := &serveRuntime{
		server:        newHTTPServer(listener.Addr().String(), engine),
		startupGate:   gate,
		listener:      listener,
		quit:          make(chan os.Signal, 1),
		startupDone:   make(chan struct{}),
		migrate:       func() error { return nil },
		startBusiness: func() {},
	}
	t.Cleanup(func() {
		_ = rt.listener.Close()
	})
	return rt
}

// TestServeRuntimeWaitsForQuit verifies the normal shutdown path: an external
// signal wakes wait and the recorded fatal error is nil.
func TestServeRuntimeNormalShutdownReturnsNil(t *testing.T) {
	rt := newTestServeRuntime(t)
	rt.start()

	rt.requestShutdown()
	errCh := make(chan error, 1)
	go func() { errCh <- rt.wait() }()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("wait() after normal shutdown = %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("wait() did not return after requestShutdown")
	}
}

// TestServeRuntimeFatalMigrationSurfaces verifies the acceptance criterion
// "migration failure does not report ready": the gate never completes, the
// server keeps answering 503 until shutdown, and wait() surfaces the fatal
// error for a non-zero exit.
func TestServeRuntimeFatalMigrationSurfaces(t *testing.T) {
	rt := newTestServeRuntime(t)
	boom := errors.New("schema migration failed: simulate")
	rt.migrate = func() error { return boom }
	rt.start()

	assertPendingGate(t, rt)

	rt.requestShutdown()
	errCh := make(chan error, 1)
	go func() { errCh <- rt.wait() }()
	select {
	case err := <-errCh:
		if !errors.Is(err, boom) {
			t.Fatalf("wait() after fatal migration = %v, want %v", err, boom)
		}
		if rt.startupGate.Ready() {
			t.Fatal("startup gate reported ready after a fatal migration failure")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("wait() did not return after fatal shutdown")
	}
}

// TestServeRuntimeFatalMigrationRaceWithQuit 确定性复现 #370 遗留竞态：quit 信号
// 先于启动 goroutine 的 setFatal 到达时，wait() 不得吞掉致命迁移错误。
// migrate 用阻塞桩挂起，确保信号入队时错误尚未记录：
//   - 修复前：wait() 立即返回 nil 错误（致命失败被吞 → 进程会以 0 退出）；
//   - 修复后：wait() 等待启动结果落定（startupDone 屏障），放行后返回 boom。
func TestServeRuntimeFatalMigrationRaceWithQuit(t *testing.T) {
	rt := newTestServeRuntime(t)
	boom := errors.New("schema migration failed: simulate")
	allowMigrate := make(chan struct{})
	rt.migrate = func() error { <-allowMigrate; return boom }
	rt.start()

	// 迁移 goroutine 仍被挂起、setFatal 未执行时信号已入队——最窄窗口。
	rt.requestShutdown()
	errCh := make(chan error, 1)
	go func() { errCh <- rt.wait() }()

	// 修复前：wait() 会立刻读到 nil 并返回，此处直接判负；修复后：wait()
	// 阻塞在 startupDone 上，等待下面放行。
	select {
	case err := <-errCh:
		t.Fatalf("wait() returned before startup settled = %v, want it to wait for the fatal migration outcome", err)
	case <-time.After(1 * time.Second):
	}

	close(allowMigrate)
	select {
	case err := <-errCh:
		if !errors.Is(err, boom) {
			t.Fatalf("wait() after fatal migration = %v, want %v", err, boom)
		}
		if rt.startupGate.Ready() {
			t.Fatal("startup gate reported ready after a fatal migration failure")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("wait() did not return after startup settled")
	}
}

// TestServeRuntimePanickingMigrationSurfaces verifies the panic path: a panic
// inside the migration (e.g. an unreachable database panics in dbconnect) must
// be treated as a fatal startup failure, not swallowed leaving the instance on
// the 503 loading page forever.
func TestServeRuntimePanickingMigrationSurfaces(t *testing.T) {
	rt := newTestServeRuntime(t)
	rt.migrate = func() error { panic("db unreachable") }
	rt.start()

	assertPendingGate(t, rt)

	rt.requestShutdown()
	errCh := make(chan error, 1)
	go func() { errCh <- rt.wait() }()
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("wait() after panicking migration = nil, want a fatal error")
		}
		if rt.startupGate.Ready() {
			t.Fatal("startup gate reported ready after a panicking migration")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("wait() did not return after panicking migration shutdown")
	}
}

// TestServeRuntimeDeferredMigrationBoots verifies the sentinel path: a
// deferred migration (Meilisearch unavailable, or the lock held) must NOT block
// startup — the gate opens and the instance serves.
func TestServeRuntimeDeferredMigrationBoots(t *testing.T) {
	rt := newTestServeRuntime(t)
	rt.migrate = func() error { return migration.ErrRetryLater }
	rt.start()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + rt.listener.Addr().String() + "/ready")
		if err != nil {
			t.Fatalf("get during deferred startup: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("gate status = %d, want 503 while booting", resp.StatusCode)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("gate never opened after a deferred migration")
}

func assertPendingGate(t *testing.T, rt *serveRuntime) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + rt.listener.Addr().String() + "/ready")
	if err != nil {
		t.Fatalf("get during migration: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("gate status during migration = %d, want 503", resp.StatusCode)
	}
	if resp.Header.Get("Retry-After") != "5" {
		t.Fatalf("Retry-After during migration = %q, want 5", resp.Header.Get("Retry-After"))
	}
}
