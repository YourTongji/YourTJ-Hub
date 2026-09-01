package dataservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"gorm.io/gorm"
)

// MaxImportSize bounds both the multipart request and the staged task file.
const MaxImportSize = 50 << 20

// TaskTypeImport identifies asynchronous data import tasks.
const TaskTypeImport = "import"

var importDir = "data/import"

// ImportTask is the small task payload. The import body stays in a mode-0600
// staging file instead of being copied into task_queue.task_json.
type ImportTask struct {
	FileName string `json:"fileName"`
	SHA256   string `json:"sha256"`
	Format   string `json:"format"`
}

// SetImportDirForTest redirects staged imports for tests.
func SetImportDirForTest(dir string) func() {
	old := importDir
	importDir = dir
	return func() { importDir = old }
}

// EnqueueImport validates and stages a JSON import, then creates an idempotent
// task. A repeated identical body reuses an existing active/success/failed
// task, so a client retry cannot create duplicate imports. Failed tasks keep
// their staging file and must be replayed explicitly.
func EnqueueImport(ctx context.Context, data []byte, format string) (*ImportReport, error) {
	if format != "json" {
		return nil, fmt.Errorf("导入仅支持 JSON 格式")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("导入文件为空")
	}
	if len(data) > MaxImportSize {
		return nil, fmt.Errorf("导入文件超过 %d MB 限制", MaxImportSize>>20)
	}
	parsed, err := parseImportJSON(data)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	digest := sha256.Sum256(data)
	hexDigest := hex.EncodeToString(digest[:])
	payload := ImportTask{
		FileName: hexDigest + ".json",
		SHA256:   hexDigest,
		Format:   format,
	}
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("编码导入任务失败: %w", err)
	}
	path := filepath.Join(importDir, payload.FileName)
	if err := stageImportFile(path, data); err != nil {
		return nil, err
	}

	var task taskQueue.Entity
	txErr := dbconnect.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing taskQueue.Entity
		err := tx.Where("type = ? AND task_json = ? AND status IN ?", TaskTypeImport, string(taskJSON),
			[]int{taskQueue.StatusPending, taskQueue.StatusRunning, taskQueue.StatusRetrying, taskQueue.StatusSuccess, taskQueue.StatusFailed}).
			Order("id desc").First(&existing).Error
		if err == nil {
			task = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		task = taskQueue.Entity{Type: TaskTypeImport, Status: taskQueue.StatusPending, TaskJson: string(taskJSON)}
		return taskQueue.CreateTx(tx, &task)
	})
	if txErr != nil {
		_ = os.Remove(path)
		return nil, txErr
	}

	status := importTaskStatus(task.Status)
	return &ImportReport{
		Status:   status,
		TaskID:   task.Id,
		Errors:   []ImportError{},
		Imported: importTableNames(parsed),
	}, nil
}

func stageImportFile(path string, data []byte) error {
	if err := os.MkdirAll(importDir, 0o700); err != nil {
		return fmt.Errorf("创建导入暂存目录失败: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("创建导入暂存文件失败: %w", err)
	}
	defer func() { _ = file.Close() }()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("设置导入暂存文件权限失败: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("写入导入暂存文件失败: %w", err)
	}
	return file.Sync()
}

// RunImportTask consumes a staged import under the worker lease. The source
// file is retained on failure so an operator can replay the task after fixing
// the underlying issue; successful imports remove it after the DB transaction.
func RunImportTask(ctx context.Context, task *taskQueue.Entity) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if task == nil {
		return errors.New("import task is nil")
	}
	var payload ImportTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return fmt.Errorf("解码导入任务失败: %w", err)
	}
	path, err := importFilePath(payload.FileName)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("打开导入暂存文件失败: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(file, MaxImportSize+1))
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("读取导入暂存文件失败: %w", readErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭导入暂存文件失败: %w", closeErr)
	}
	if len(data) > MaxImportSize {
		return fmt.Errorf("导入暂存文件超过 %d MB 限制", MaxImportSize>>20)
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != payload.SHA256 {
		return errors.New("导入暂存文件校验失败")
	}
	report, err := ImportData(ctx, data, payload.Format)
	if err != nil {
		return err
	}
	if report.Failed > 0 {
		return fmt.Errorf("导入存在 %d 个失败行，任务保留以便重放", report.Failed)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除已完成导入暂存文件失败: %w", err)
	}
	return nil
}

func importFilePath(fileName string) (string, error) {
	if fileName == "" || filepath.Base(fileName) != fileName {
		return "", errors.New("非法的导入暂存文件路径")
	}
	base, err := filepath.Abs(importDir)
	if err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(importDir, fileName))
	if err != nil {
		return "", err
	}
	if path != base && !isWithinDirectory(base, path) {
		return "", errors.New("非法的导入暂存文件路径")
	}
	return path, nil
}

func isWithinDirectory(base, path string) bool {
	rel, err := filepath.Rel(base, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// ReplayImportTask puts a failed import back into the pending state without
// replacing its retained, checksum-addressed staging file.
func ReplayImportTask(id uint64) (taskQueue.Entity, error) {
	var task taskQueue.Entity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND type = ?", id, TaskTypeImport).First(&task).Error; err != nil {
			return err
		}
		if task.Status != taskQueue.StatusFailed {
			return fmt.Errorf("导入任务 %d 当前不可重放", id)
		}
		var payload ImportTask
		if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
			return fmt.Errorf("解码导入任务失败: %w", err)
		}
		path, err := importFilePath(payload.FileName)
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("导入暂存文件不可重放: %w", err)
		}
		result := tx.Model(&taskQueue.Entity{}).
			Where("id = ? AND type = ? AND status = ?", id, TaskTypeImport, taskQueue.StatusFailed).
			Updates(map[string]any{
				"status":       taskQueue.StatusPending,
				"retry_count":  0,
				"last_error":   "",
				"processed_at": nil,
				"lease_token":  "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("导入任务 %d 状态已变化", id)
		}
		return tx.Where("id = ?", id).First(&task).Error
	})
	return task, err
}

func importTaskStatus(status uint8) string {
	switch status {
	case taskQueue.StatusPending:
		return "pending"
	case taskQueue.StatusRunning:
		return "running"
	case taskQueue.StatusRetrying:
		return "retrying"
	case taskQueue.StatusSuccess:
		return "success"
	case taskQueue.StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// ListImportTasks returns recent import task rows for the admin status view.
func ListImportTasks(limit int) ([]taskQueue.Entity, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var list []taskQueue.Entity
	if err := taskQueue.QueryByTypeDesc(TaskTypeImport, limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// RecoverImportTasks restores expired import leases after a process restart.
func RecoverImportTasks() error {
	return taskQueue.RecoverStaleRunning(TaskTypeImport, taskQueue.LeaseDuration)
}

func importTableNames(parsed map[string][]map[string]any) []string {
	result := make([]string, 0, len(parsed))
	for _, table := range []string{"users", "topics", "posts", "postRevisions", "topicCategoryIndex", "topicUserStat"} {
		if _, ok := parsed[table]; ok {
			result = append(result, table)
		}
	}
	return result
}
