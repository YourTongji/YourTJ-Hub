// Package backgroundservice hosts long-running task queue workers.
//
// Every worker polls taskQueue rows whose type starts with its own prefix,
// so task types never leak into the wrong handler. Workers are registered
// with the process closer and shut down cleanly on exit.
package backgroundservice

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/closer"
	paniclog "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/recovery"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
)

const (
	batchSize     = 10
	pollInterval  = 5 * time.Second
	retryInterval = 5 * time.Second
	maxRetries    = 3

	// leaseRenewInterval 是任务租约的心跳续约间隔（issue #138）。远小于
	// taskQueue.LeaseDuration（10 分钟），即使进程暂停或 DB 短暂抖动
	// 也不至于丢租约。
	leaseRenewInterval = 30 * time.Second
)

// TaskHandler processes one queued task. Returning an error triggers
// retry/failure bookkeeping on the task row.
type TaskHandler func(ctx context.Context, task *taskQueue.Entity) error

// RunWorker starts a polling worker for tasks whose type starts with
// typePrefix. The worker stops when the process closer fires.
func RunWorker(name, typePrefix string, handler TaskHandler) {
	stopCh := make(chan struct{})
	closer.RegisterPriority(closer.PriorityProducer, func() error {
		close(stopCh)
		return nil
	})
	go func() {
		defer paniclog.Recover(name)
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			// 周期回收租约过期的 Running 任务：崩溃 worker 的任务在
			// LeaseDuration 后回到 Pending，可被其他 worker 重新领取
			// （issue #138）。运行中的 worker 通过心跳续租，不会被误回收。
			if err := taskQueue.RecoverStaleRunning(typePrefix, taskQueue.LeaseDuration); err != nil {
				slog.Error("background: recover stale running tasks failed", "worker", typePrefix, "err", err)
			}
			if !drainTasks(stopCh, typePrefix, handler) {
				return
			}
			select {
			case <-ticker.C:
			case <-stopCh:
				return
			}
		}
	}()
}

func drainTasks(stopCh <-chan struct{}, typePrefix string, handler TaskHandler) bool {
	for {
		select {
		case <-stopCh:
			return false
		default:
		}

		tasks := taskQueue.GetPendingTasksByType(typePrefix, batchSize)
		if len(tasks) == 0 {
			return true
		}
		slog.Debug("background: pulled tasks", "worker", typePrefix, "count", len(tasks))

		for _, task := range tasks {
			if !processTask(stopCh, typePrefix, task, handler) {
				return false
			}
		}
	}
}

func processTask(stopCh <-chan struct{}, typePrefix string, task *taskQueue.Entity, handler TaskHandler) bool {
	// 原子领取（issue #138）：pending/retrying → running 的 CAS 更新，
	// 并发 worker 中只有一个能成功，其余直接跳过，杜绝重复执行外部副作用。
	running, claimed, err := taskQueue.ClaimTask(task.Id)
	if err != nil {
		slog.Error("background: claim task failed", "worker", typePrefix, "id", task.Id, "err", err)
		return true
	}
	if !claimed {
		slog.Debug("background: task already claimed by another worker", "worker", typePrefix, "id", task.Id)
		return true
	}

	// 处理期间心跳续租；租约丢失（任务被回收重领）时取消 ctx，终止 handler。
	guard := NewLeaseGuard(running.ProcessedAt)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	heartbeatDone := StartLeaseHeartbeat(ctx, cancel, running.Id, guard)

	handlerErr := handler(ctx, task)

	// 停止心跳并等待退出，再做一次最终续租拿到权威租约值；之后的所有
	// 状态写入都以该值为 CAS 前置条件（fencing）。若心跳退出瞬间与最后
	// 一次续租交叠，guard 中的租约值可能滞后于 DB，最终续租负责收敛；
	// 续租失败说明租约已被回收（任务被其他 worker 重新领取），跳过全部
	// 终态写入，避免重复执行外部副作用。
	cancel()
	<-heartbeatDone
	lease := guard.Get()
	if ok, renewed, err := taskQueue.RenewLease(task.Id, lease); err != nil {
		slog.Error("background: final lease renewal failed", "worker", typePrefix, "id", task.Id, "err", err)
	} else if ok {
		lease = renewed
	} else {
		slog.Warn("background: task lease lost, skipping terminal write", "worker", typePrefix, "id", task.Id)
		return true
	}

	if handlerErr != nil {
		slog.Error("background: task failed", "worker", typePrefix, "id", task.Id, "type", task.Type, "retryCount", task.RetryCount, "err", handlerErr)
		if task.RetryCount < maxRetries {
			if updateErr := taskQueue.IncrementRetryCountOwned(task.Id, lease); updateErr != nil {
				slog.Error("background: increment retry count failed", "id", task.Id, "err", updateErr)
			}
			if updateErr := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusRetrying, lease, handlerErr); updateErr != nil {
				slog.Error("background: mark task retrying failed", "id", task.Id, "err", updateErr)
			}
			select {
			case <-time.After(retryInterval):
				return true
			case <-stopCh:
				return false
			}
		}
		if updateErr := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusFailed, lease, handlerErr); updateErr != nil {
			slog.Error("background: mark task failed failed", "id", task.Id, "err", updateErr)
		}
		return true
	}

	if err := taskQueue.UpdateStatusOwned(task.Id, taskQueue.StatusSuccess, lease, nil); err != nil {
		slog.Error("background: mark task success failed", "worker", typePrefix, "id", task.Id, "err", err)
	}
	return true
}

// LeaseGuard 跟踪 worker 当前持有的租约值（DB 中最新确认的 processed_at），
// 供续租与终态写入的 CAS 使用。
type LeaseGuard struct {
	mu    sync.Mutex
	lease time.Time
}

// NewLeaseGuard 以领取任务时返回的租约值初始化 guard。
func NewLeaseGuard(lease time.Time) *LeaseGuard {
	return &LeaseGuard{lease: lease}
}

// Get 返回当前已知租约值。
func (g *LeaseGuard) Get() time.Time {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.lease
}

// Set 记录一次成功续租后的新租约值。
func (g *LeaseGuard) Set(lease time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lease = lease
}

// StartLeaseHeartbeat 启动任务续租心跳：每个 leaseRenewInterval 对任务执行
// 一次 CAS 续租。续租失败说明租约已被回收（任务被其他 worker 重新领取），
// 立即取消 ctx 并退出；ctx 被外部取消时也退出。返回的通道在心跳 goroutine
// 退出后关闭，调用方在处理结束后可等待它以固定最终租约值。
func StartLeaseHeartbeat(ctx context.Context, cancel context.CancelFunc, id uint64, guard *LeaseGuard) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(leaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ok, lease, err := taskQueue.RenewLease(id, guard.Get())
				if err != nil {
					slog.Error("background: renew lease failed", "id", id, "err", err)
					continue
				}
				if !ok {
					slog.Warn("background: task lease lost, cancelling handler", "id", id)
					cancel()
					return
				}
				guard.Set(lease)
			case <-ctx.Done():
				return
			}
		}
	}()
	return done
}
