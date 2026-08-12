// Package filemigrateservice migrates existing SQLite BLOB files to the
// configured object storage provider. It runs either as a taskQueue worker
// (admin panel entry) or as a blocking CLI command.
package filemigrateservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/models/filemodel/filedata"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/service/storageservice"
)

// TaskTypeFileMigrate is the taskQueue type prefix for file migration tasks.
const TaskTypeFileMigrate = "file-migrate"

// ErrProviderNotS3 marks that the active provider is not object storage.
var ErrProviderNotS3 = fmt.Errorf("file migration requires an s3-compatible storage provider")

// Task payload stored in taskQueue.task_json.
type MigrateTask struct {
	LastID            uint64 `json:"lastId"`            // 游标：已处理的最大文件 id
	Total             int64  `json:"total"`             // 迁移前文件总数
	Processed         int64  `json:"processed"`         // 已处理数量
	Failed            int64  `json:"failed"`            // 失败对象数量（同一对象重复失败只计一次）
	ClearAfterMigrate bool   `json:"clearAfterMigrate"` // 成功后是否清空 BLOB
}

const batchSize = 100

// maxConsecutiveFailures aborts the migration after this many Save attempts
// fail without a success in between. It fails fast under a bucket-wide outage
// (every upload fails) and still bounds a single permanently-failing object,
// which is retried once per re-scan round and must not loop forever.
const maxConsecutiveFailures = 50

// CreateMigrateTask validates the active storage provider is object storage
// and enqueues a migration task. Returns the task id.
func CreateMigrateTask(clearAfterMigrate bool) (uint64, error) {
	cfg := storageservice.GetStorageSettings()
	if cfg.Provider != storageservice.ProviderS3 {
		return 0, ErrProviderNotS3
	}
	if err := storageservice.TestConnection(context.Background(), cfg); err != nil {
		return 0, fmt.Errorf("storage connection test failed: %w", err)
	}
	payload := MigrateTask{
		Total:             filedata.CountFiles(),
		ClearAfterMigrate: clearAfterMigrate,
	}
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("encode migrate task: %w", err)
	}
	entity := &taskQueue.Entity{
		Type:     TaskTypeFileMigrate,
		Status:   taskQueue.StatusPending,
		TaskJson: string(taskJSON),
	}
	if err := taskQueue.Create(entity); err != nil {
		return 0, err
	}
	return entity.Id, nil
}

// ListMigrateTasks returns recent migration tasks, newest first.
func ListMigrateTasks(limit int) ([]taskQueue.Entity, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var list []taskQueue.Entity
	if err := taskQueue.QueryByTypeDesc(TaskTypeFileMigrate, limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// RunMigrateTask is the worker handler for a file-migration task.
func RunMigrateTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload MigrateTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return fmt.Errorf("decode migrate task: %w", err)
	}
	processed, failed, err := MigrateFiles(ctx, payload.LastID, payload.ClearAfterMigrate, func(lastID uint64, proc, fail int64) {
		payload.LastID = lastID
		payload.Processed = proc
		payload.Failed = fail
		updateTaskProgress(task.Id, payload)
	})
	if err != nil {
		updateTaskProgress(task.Id, payload)
		return err
	}
	slog.Info("file migration finished", "taskId", task.Id, "processed", processed, "failed", failed)
	return nil
}

// MigrateFiles copies BLOB rows with id > startId to the active provider.
// onProgress (may be nil) is called after each batch with the new cursor.
// Returns processed and failed counts.
func MigrateFiles(ctx context.Context, startID uint64, clearAfterMigrate bool, onProgress func(lastID uint64, processed, failed int64)) (int64, int64, error) {
	return migrateFiles(ctx, storageservice.Current(), filedata.QueryById, filedata.ClearContentByName, startID, clearAfterMigrate, onProgress)
}

// migrateFiles is the testable core of MigrateFiles. The cursor (lastID) only
// advances past objects that were successfully migrated (or were already
// empty). If an object fails, the cursor freezes before it so the next query
// re-fetches the object instead of permanently skipping it; once the object
// recovers, the cursor resumes past it. Save failures reset on any success, so
// a bucket-wide outage aborts quickly (fail-fast) while a single
// permanently-failing object still aborts after maxConsecutiveFailures retries
// rather than looping forever. The returned error surfaces the task as failed
// instead of claiming success over an incomplete migration.
func migrateFiles(
	ctx context.Context,
	provider storageservice.Provider,
	queryByID func(startID uint64, limit int) []*filedata.Entity,
	clearContent func(name string) error,
	startID uint64,
	clearAfterMigrate bool,
	onProgress func(lastID uint64, processed, failed int64),
) (int64, int64, error) {
	var processed, failed int64
	lastID := startID
	consecutiveFailures := 0
	// failedNames tracks distinct objects that failed so the failed counter is
	// not inflated by re-querying the same failing object every batch.
	failedNames := make(map[string]struct{})
	// handled tracks objects already migrated or uploaded in this run. Without
	// it, the frozen window of a stalling cursor would re-upload trailing rows
	// (clearAfterMigrate=false) and inflate processed on every re-scan round.
	handled := make(map[string]struct{})

	for {
		if err := ctx.Err(); err != nil {
			return processed, failed, err
		}
		rows := queryByID(lastID, batchSize)
		if len(rows) == 0 {
			break
		}

		batchFailed := false
		for _, row := range rows {
			if len(row.Data) == 0 {
				// 已迁移/已清空
				if _, ok := handled[row.Name]; !ok {
					handled[row.Name] = struct{}{}
					processed++
				}
				if !batchFailed {
					lastID = row.Id
				}
				continue
			}
			if _, ok := handled[row.Name]; ok {
				// 本 run 已上传（游标冻结窗口内被重复抓取），不再重传或重复计数
				if !batchFailed {
					lastID = row.Id
				}
				continue
			}
			if err := provider.Save(ctx, row.Name, row.Data, row.Type); err != nil {
				if _, seen := failedNames[row.Name]; !seen {
					failedNames[row.Name] = struct{}{}
					failed++
				}
				slog.Error("migrate file failed", "name", row.Name, "err", err)
				// 失败对象保留在游标之后：本批停止推进 lastID，下次查询会重新
				// 抓取该对象，避免失败被永久跳过导致目标文件永久 404。
				consecutiveFailures++
				batchFailed = true
				if consecutiveFailures >= maxConsecutiveFailures {
					return processed, failed, fmt.Errorf("file migration aborted: %d object(s) failed to upload (%s); cursor stuck at id %d",
						failed, sampleFailedNames(failedNames, 3), lastID)
				}
				continue
			}
			consecutiveFailures = 0
			handled[row.Name] = struct{}{}
			processed++
			if !batchFailed {
				lastID = row.Id
			}
			if clearAfterMigrate {
				if err := clearContent(row.Name); err != nil {
					slog.Error("clear migrated blob failed", "name", row.Name, "err", err)
				}
			}
		}

		if onProgress != nil {
			onProgress(lastID, processed, failed)
		}
	}
	return processed, failed, nil
}

// sampleFailedNames returns up to limit failing object names, sorted so the
// error message in task_queue.last_error is stable and actionable.
func sampleFailedNames(names map[string]struct{}, limit int) string {
	if len(names) == 0 {
		return "-"
	}
	list := make([]string, 0, len(names))
	for name := range names {
		list = append(list, name)
	}
	sort.Strings(list)
	if len(list) > limit {
		list = list[:limit]
	}
	return strings.Join(list, ", ")
}

func updateTaskProgress(taskID uint64, payload MigrateTask) {
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := taskQueue.UpdateTaskJson(taskID, string(taskJSON)); err != nil {
		slog.Error("update migrate task progress failed", "taskId", taskID, "err", err)
	}
}

// RecoverStaleTasks 启动时恢复文件迁移 worker 类型前缀下崩溃遗留的 Running 任务。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeFileMigrate, 10*time.Minute)
}
