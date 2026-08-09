// Package backgroundservice hosts long-running task queue workers.
//
// Every worker polls taskQueue rows whose type starts with its own prefix,
// so task types never leak into the wrong handler. Workers are registered
// with the process closer and shut down cleanly on exit.
package backgroundservice

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/closer"
	paniclog "github.com/leancodebox/GooseForum/app/bundles/recovery"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
)

const (
	batchSize     = 10
	pollInterval  = 5 * time.Second
	retryInterval = 5 * time.Second
	maxRetries    = 3
)

var errTaskHandlerPanicked = errors.New("task handler panicked")

// TaskHandler processes one queued task. Returning an error triggers
// retry/failure bookkeeping on the task row.
type TaskHandler func(ctx context.Context, task *taskQueue.Entity) error

// RunWorker starts a polling worker for tasks whose type starts with
// typePrefix. The worker stops when the process closer fires.
func RunWorker(name, typePrefix string, handler TaskHandler) {
	runWorker(name, typePrefix, 0, handler)
}

// RunWorkerWithStaleRecovery additionally requeues tasks that stayed running
// beyond staleAfter. It is intended for bounded handlers whose durable work
// must survive a process crash after the task was claimed.
func RunWorkerWithStaleRecovery(name, typePrefix string, staleAfter time.Duration, handler TaskHandler) {
	runWorker(name, typePrefix, staleAfter, handler)
}

func runWorker(name, typePrefix string, staleAfter time.Duration, handler TaskHandler) {
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
			if staleAfter > 0 {
				requeueStaleTasks(typePrefix, staleAfter)
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

func requeueStaleTasks(typePrefix string, staleAfter time.Duration) {
	count, err := taskQueue.RequeueStaleRunningByType(typePrefix, time.Now().Add(-staleAfter))
	if err != nil {
		slog.Error("background: stale task recovery failed", "worker", typePrefix, "err", err)
		return
	}
	if count > 0 {
		slog.Warn("background: requeued stale running tasks", "worker", typePrefix, "count", count)
	}
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
	if err := taskQueue.UpdateStatus(task.Id, taskQueue.StatusRunning, nil); err != nil {
		slog.Error("background: mark task running failed", "worker", typePrefix, "id", task.Id, "err", err)
		return true
	}

	if err := invokeTaskHandler(handler, task); err != nil {
		slog.Error("background: task failed", "worker", typePrefix, "id", task.Id, "type", task.Type, "retryCount", task.RetryCount, "err", err)
		if task.RetryCount < maxRetries {
			if updateErr := taskQueue.IncrementRetryCount(task.Id); updateErr != nil {
				slog.Error("background: increment retry count failed", "id", task.Id, "err", updateErr)
			}
			if updateErr := taskQueue.UpdateStatus(task.Id, taskQueue.StatusRetrying, err); updateErr != nil {
				slog.Error("background: mark task retrying failed", "id", task.Id, "err", updateErr)
			}
			select {
			case <-time.After(retryInterval):
				return true
			case <-stopCh:
				return false
			}
		}
		if updateErr := taskQueue.UpdateStatus(task.Id, taskQueue.StatusFailed, err); updateErr != nil {
			slog.Error("background: mark task failed failed", "id", task.Id, "err", updateErr)
		}
		return true
	}

	if err := taskQueue.UpdateStatus(task.Id, taskQueue.StatusSuccess, nil); err != nil {
		slog.Error("background: mark task success failed", "worker", typePrefix, "id", task.Id, "err", err)
	}
	return true
}

func invokeTaskHandler(handler TaskHandler, task *taskQueue.Entity) (err error) {
	defer func() {
		if recover() != nil {
			err = errTaskHandlerPanicked
		}
	}()
	return handler(context.Background(), task)
}
