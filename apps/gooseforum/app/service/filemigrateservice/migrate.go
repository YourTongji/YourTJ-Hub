// Package filemigrateservice migrates existing SQLite BLOB files to the
// configured object storage provider. It runs either as a taskQueue worker
// (admin panel entry) or as a blocking CLI command.
package filemigrateservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
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
	Failed            int64  `json:"failed"`            // 失败数量
	ClearAfterMigrate bool   `json:"clearAfterMigrate"` // 成功后是否清空 BLOB
}

const batchSize = 100

// maxConsecutiveFailures aborts the migration after this many consecutive
// failed files to avoid burning through the whole table on a broken bucket.
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
	// 持有者 fencing token（ClaimTask 领取时生成）：进度写回必须是
	// status=Running AND lease_token=? 的 CAS，任务被回收重领后旧 worker
	// 的进度更新不再命中（fencing，review P1）。
	token := task.LeaseToken
	processed, failed, err := MigrateFiles(ctx, payload.LastID, payload.ClearAfterMigrate, func(lastID uint64, proc, fail int64) {
		payload.LastID = lastID
		payload.Processed = proc
		payload.Failed = fail
		updateTaskProgress(task.Id, token, payload)
	})
	if err != nil {
		updateTaskProgress(task.Id, token, payload)
		return err
	}
	slog.Info("file migration finished", "taskId", task.Id, "processed", processed, "failed", failed)
	return nil
}

// MigrateFiles copies BLOB rows with id > startId to the active provider.
// onProgress (may be nil) is called after each batch with the new cursor.
// Returns processed and failed counts.
func MigrateFiles(ctx context.Context, startID uint64, clearAfterMigrate bool, onProgress func(lastID uint64, processed, failed int64)) (int64, int64, error) {
	provider := storageservice.Current()
	var processed, failed int64
	var lastID = startID
	consecutiveFailures := 0

	for {
		if err := ctx.Err(); err != nil {
			return processed, failed, err
		}
		rows := filedata.QueryById(lastID, batchSize)
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			lastID = row.Id
			if len(row.Data) == 0 {
				continue // 已迁移/已清空
			}
			if err := provider.Save(ctx, row.Name, row.Data, row.Type); err != nil {
				failed++
				consecutiveFailures++
				slog.Error("migrate file failed", "name", row.Name, "err", err)
				if consecutiveFailures >= maxConsecutiveFailures {
					return processed, failed, fmt.Errorf("too many consecutive failures (%d), aborting", consecutiveFailures)
				}
				continue
			}
			consecutiveFailures = 0
			processed++
			if clearAfterMigrate {
				if err := filedata.ClearContentByName(row.Name); err != nil {
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

func updateTaskProgress(taskID uint64, token string, payload MigrateTask) {
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if err := taskQueue.UpdateTaskJsonOwned(taskID, token, string(taskJSON)); err != nil {
		slog.Error("update migrate task progress failed", "taskId", taskID, "err", err)
	}
}

// RecoverStaleTasks 启动时恢复文件迁移 worker 类型前缀下崩溃遗留的 Running 任务。
func RecoverStaleTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeFileMigrate, taskQueue.LeaseDuration)
}
