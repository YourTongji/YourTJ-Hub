// Package dataservice provides admin data export/import for users, topics
// and posts. Export runs as a taskQueue background task; import is a
// synchronous, validated JSON import with idempotent skip.
package dataservice

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

// TaskTypeExport is the taskQueue type prefix for export tasks.
const TaskTypeExport = "export"

// AllowedExportTables lists tables that can be exported/imported.
var AllowedExportTables = map[string]bool{"users": true, "topics": true, "posts": true}

// exportDir is where export files are written. Tests override it.
var exportDir = "data/export"

const exportBatchSize = 200

// ExportTask is the payload stored in taskQueue.task_json for export tasks.
type ExportTask struct {
	Tables     []string `json:"tables"`
	Format     string   `json:"format"` // json | csv
	FileName   string   `json:"fileName"`
	Progress   int      `json:"progress"` // 0-100
	ErrorCount int      `json:"errorCount"`
}

// ExportData validates the request and enqueues an export task.
func ExportData(tables []string, format string) (uint64, error) {
	if len(tables) == 0 {
		return 0, fmt.Errorf("至少选择一张表")
	}
	seen := map[string]bool{}
	for _, t := range tables {
		if !AllowedExportTables[t] {
			return 0, fmt.Errorf("不支持的导出表: %s", t)
		}
		if seen[t] {
			return 0, fmt.Errorf("重复的导出表: %s", t)
		}
		seen[t] = true
	}
	if format != "json" && format != "csv" {
		return 0, fmt.Errorf("不支持的导出格式: %s", format)
	}
	payload := ExportTask{Tables: tables, Format: format}
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("编码导出任务失败: %w", err)
	}
	entity := &taskQueue.Entity{
		Type:     TaskTypeExport,
		Status:   taskQueue.StatusPending,
		TaskJson: string(taskJSON),
	}
	if err := taskQueue.Create(entity); err != nil {
		return 0, err
	}
	return entity.Id, nil
}

// ListExportTasks returns recent export tasks, newest first.
func ListExportTasks(limit int) ([]taskQueue.Entity, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var list []taskQueue.Entity
	if err := taskQueue.QueryByTypeDesc(TaskTypeExport, limit).Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ExportFilePath resolves the file path for a finished export task.
// It validates the path stays inside the export directory.
func ExportFilePath(task *taskQueue.Entity) (string, error) {
	var payload ExportTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return "", fmt.Errorf("解码导出任务失败: %w", err)
	}
	if payload.FileName == "" {
		return "", fmt.Errorf("导出文件不存在")
	}
	clean := filepath.Clean(filepath.Join(exportDir, payload.FileName))
	absBase, err := filepath.Abs(exportDir)
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("非法的导出文件路径")
	}
	return absPath, nil
}

// RunExportTask is the background worker handler.
func RunExportTask(ctx context.Context, task *taskQueue.Entity) error {
	var payload ExportTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil {
		return fmt.Errorf("解码导出任务失败: %w", err)
	}
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return fmt.Errorf("创建导出目录失败: %w", err)
	}
	ext := "json"
	if payload.Format == "csv" {
		ext = "csv"
	}
	payload.FileName = fmt.Sprintf("export_%d_%s.%s", task.Id, time.Now().Format("20060102_150405"), ext)
	tmpFile := filepath.Join(exportDir, payload.FileName+".tmp")
	finalFile := filepath.Join(exportDir, payload.FileName)

	file, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建导出文件失败: %w", err)
	}
	defer func() { _ = file.Close() }()

	// 按外键依赖顺序导出：users → topics → posts
	order := []string{"users", "topics", "posts"}
	selected := map[string]bool{}
	for _, t := range payload.Tables {
		selected[t] = true
	}

	var total int64
	for _, table := range order {
		if !selected[table] {
			continue
		}
		total += countTable(table)
	}
	processed := int64(0)

	var writer *csv.Writer
	if payload.Format == "csv" {
		writer = csv.NewWriter(file)
	}
	jsonEncoder := json.NewEncoder(file)
	jsonEncoder.SetEscapeHTML(false)

	// JSON 模式输出与导入兼容的对象结构：{"users":[...],"topics":[...],"posts":[...]}
	firstTable := true
	if payload.Format == "json" {
		if _, err := file.WriteString("{"); err != nil {
			return fmt.Errorf("写入导出 JSON 失败: %w", err)
		}
	}

	for _, table := range order {
		if !selected[table] {
			continue
		}
		if payload.Format == "json" {
			if !firstTable {
				if _, err := file.WriteString(","); err != nil {
					return fmt.Errorf("写入导出 JSON 失败: %w", err)
				}
			}
			firstTable = false
			if _, err := fmt.Fprintf(file, "%q:[", table); err != nil {
				return fmt.Errorf("写入导出 JSON 失败: %w", err)
			}
		}
		count, err := writeExportTable(ctx, file, writer, jsonEncoder, payload.Format, table, func(n int64) {
			processed += n
			if total > 0 {
				payload.Progress = int(processed * 100 / total)
				if payload.Progress > 100 {
					payload.Progress = 100
				}
				updateExportProgress(task.Id, payload)
			}
		})
		if err != nil {
			payload.ErrorCount++
			updateExportProgress(task.Id, payload)
			return err
		}
		processed += count
		if payload.Format == "json" {
			if _, err := file.WriteString("]"); err != nil {
				return fmt.Errorf("写入导出 JSON 失败: %w", err)
			}
		}
	}

	if payload.Format == "json" {
		if _, err := file.WriteString("}"); err != nil {
			return fmt.Errorf("写入导出 JSON 失败: %w", err)
		}
	}

	if writer != nil {
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("写入 CSV 失败: %w", err)
		}
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpFile, finalFile); err != nil {
		return fmt.Errorf("导出文件落盘失败: %w", err)
	}
	payload.Progress = 100
	updateExportProgress(task.Id, payload)
	return nil
}

func countTable(table string) int64 {
	db := dbconnect.Connect()
	var count int64
	db.Table(table).Count(&count)
	return count
}

// writeExportTable streams one table to the file, returning rows written.
func writeExportTable(ctx context.Context, file *os.File, writer *csv.Writer, encoder *json.Encoder, format, table string, onBatch func(int64)) (int64, error) {
	var lastID uint64
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		rows := fetchExportRows(table, lastID, exportBatchSize)
		if len(rows) == 0 {
			break
		}
		lastID = rows[len(rows)-1].ID
		for i, row := range rows {
			if format == "csv" {
				if err := writeCSVRow(writer, table, row); err != nil {
					return total, err
				}
			} else {
				// JSON 数组内元素间加逗号分隔
				if i > 0 {
					if _, err := file.WriteString(","); err != nil {
						return total, err
					}
				}
				if err := encoder.Encode(row.Fields); err != nil {
					return total, err
				}
			}
		}
		total += int64(len(rows))
		onBatch(int64(len(rows)))
	}
	return total, nil
}

type exportRow struct {
	ID     uint64
	Fields map[string]any
}

func fetchExportRows(table string, lastID uint64, limit int) []exportRow {
	db := dbconnect.Connect()
	switch table {
	case "users":
		var list []users.EntityComplete
		db.Table("users").Where("id > ?", lastID).Order("id asc").Limit(limit).Find(&list)
		rows := make([]exportRow, 0, len(list))
		for _, u := range list {
			rows = append(rows, exportRow{ID: u.Id, Fields: map[string]any{
				"id": u.Id, "username": u.Username, "email": u.Email,
				"nickname": u.Nickname, "bio": u.Bio, "signature": u.Signature,
				"prestige": u.Prestige, "isFrozen": u.IsFrozen, "isActivated": u.IsActivated,
				"roleId": u.RoleId, "avatarUrl": u.AvatarUrl, "website": u.Website,
				"createdAt": u.CreatedAt.Format(time.RFC3339), "updatedAt": u.UpdatedAt.Format(time.RFC3339),
			}})
		}
		return rows
	case "topics":
		var list []topics.Entity
		db.Table("topics").Where("id > ?", lastID).Order("id asc").Limit(limit).Find(&list)
		rows := make([]exportRow, 0, len(list))
		for _, t := range list {
			cats, _ := json.Marshal(t.CategoryIds)
			rows = append(rows, exportRow{ID: t.Id, Fields: map[string]any{
				"id": t.Id, "title": t.Title, "categoryIds": string(cats), "userId": t.UserId,
				"status": t.Status, "processStatus": t.ProcessStatus, "postCount": t.PostCount,
				"replyCount": t.ReplyCount, "excerpt": t.Excerpt, "firstImageUrl": t.FirstImageURL,
				"createdAt": t.CreatedAt.Format(time.RFC3339), "updatedAt": t.UpdatedAt.Format(time.RFC3339),
			}})
		}
		return rows
	case "posts":
		var list []posts.Entity
		db.Table("posts").Where("id > ?", lastID).Order("id asc").Limit(limit).Find(&list)
		rows := make([]exportRow, 0, len(list))
		for _, p := range list {
			rows = append(rows, exportRow{ID: p.Id, Fields: map[string]any{
				"id": p.Id, "topicId": p.TopicId, "postNo": p.PostNo, "userId": p.UserId,
				"replyToPostId": p.ReplyToPostId, "content": p.Content, "processStatus": p.ProcessStatus,
				"createdAt": p.CreatedAt.Format(time.RFC3339), "updatedAt": p.UpdatedAt.Format(time.RFC3339),
			}})
		}
		return rows
	}
	return nil
}

var exportCSVHeaders = map[string][]string{
	"users":  {"id", "username", "email", "nickname", "bio", "signature", "prestige", "isFrozen", "isActivated", "roleId", "avatarUrl", "website", "createdAt", "updatedAt"},
	"topics": {"id", "title", "categoryIds", "userId", "status", "processStatus", "postCount", "replyCount", "excerpt", "firstImageUrl", "createdAt", "updatedAt"},
	"posts":  {"id", "topicId", "postNo", "userId", "replyToPostId", "content", "processStatus", "createdAt", "updatedAt"},
}

func writeCSVRow(writer *csv.Writer, table string, row exportRow) error {
	headers := exportCSVHeaders[table]
	record := make([]string, 0, len(headers))
	for _, h := range headers {
		record = append(record, csvValue(row.Fields[h]))
	}
	return writer.Write(record)
}

func csvValue(v any) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case int64:
		return strconv.FormatInt(val, 10)
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func updateExportProgress(taskID uint64, payload ExportTask) {
	taskJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = taskQueue.UpdateTaskJson(taskID, string(taskJSON))
}

// categoryExists reports whether a category id exists.
func categoryExists(id uint64) bool {
	return category.Get(id).Id != 0
}

// exportRetentionDays 导出文件保留天数，超过后由定时任务清理。
const exportRetentionDays = 7

// CleanupExpiredExports 删除超过保留期的导出文件（任务记录保留）。
func CleanupExpiredExports() {
	entries, err := os.ReadDir(exportDir)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("read export dir failed", "err", err)
		}
		return
	}
	cutoff := time.Now().AddDate(0, 0, -exportRetentionDays)
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(exportDir, entry.Name())
			if err := os.Remove(path); err != nil {
				slog.Warn("remove expired export file failed", "path", path, "err", err)
			} else {
				slog.Info("removed expired export file", "path", path)
			}
		}
	}
}
