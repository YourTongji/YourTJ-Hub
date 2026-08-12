package dataservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func setupDataTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&taskQueue.Entity{},
		&category.Entity{},
	); err != nil {
		t.Fatalf("migrate data tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&category.Entity{})
}

func withTempExportDir(t *testing.T) {
	t.Helper()
	old := exportDir
	exportDir = t.TempDir()
	t.Cleanup(func() { exportDir = old })
}

func TestExportDataValidation(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	// 空表 → 报错
	if _, err := ExportData(nil, "json"); err == nil {
		t.Fatal("ExportData(nil) error = nil, want error")
	}
	// 未知表 → 报错
	if _, err := ExportData([]string{"unknown"}, "json"); err == nil {
		t.Fatal("ExportData(unknown) error = nil, want error")
	}
	// 未知格式 → 报错
	if _, err := ExportData([]string{"users"}, "xml"); err == nil {
		t.Fatal("ExportData(xml) error = nil, want error")
	}
	// 合法请求 → 创建任务
	taskID, err := ExportData([]string{"users"}, "json")
	if err != nil {
		t.Fatalf("ExportData() error = %v", err)
	}
	if taskID == 0 {
		t.Fatal("ExportData() taskID = 0, want non-zero")
	}
}

func TestExportRunAndFile(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	user := users.EntityComplete{Username: "exporter", Email: "exporter@example.com", Nickname: "导出测试"}
	if err := users.Create(&user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	topic := topics.Entity{Title: "导出主题", UserId: user.Id, Status: 1}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 1, UserId: user.Id, Content: "导出正文"}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	// 直接构造导出任务并运行
	taskEntity := &taskQueue.Entity{
		Type:     TaskTypeExport,
		Status:   taskQueue.StatusPending,
		TaskJson: `{"tables":["users","topics","posts"],"format":"json"}`,
	}
	if err := taskQueue.Create(taskEntity); err != nil {
		t.Fatalf("create export task: %v", err)
	}
	if err := RunExportTask(context.Background(), taskEntity); err != nil {
		t.Fatalf("RunExportTask() error = %v", err)
	}

	// 重新读取任务（RunExportTask 通过 DB 更新 FileName/进度）
	reloaded, err := taskQueue.GetByID(taskEntity.Id)
	if err != nil {
		t.Fatalf("reload export task: %v", err)
	}

	// 文件应存在且包含数据（无 BLOB/密码）
	path, err := ExportFilePath(&reloaded)
	if err != nil {
		t.Fatalf("ExportFilePath() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("export file missing: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("export file empty")
	}
	// 导出为与导入兼容的对象结构：{"users":[...],"topics":[...],"posts":[...]}
	var doc map[string][]map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("export is not JSON object: %v", err)
	}
	foundUser := false
	for _, row := range doc["users"] {
		if row["username"] == "exporter" {
			foundUser = true
			if _, hasPwd := row["password"]; hasPwd {
				t.Fatal("export leaked password field")
			}
		}
	}
	if !foundUser {
		t.Fatal("export file missing user row")
	}
	if len(doc["topics"]) != 1 || len(doc["posts"]) != 1 {
		t.Fatalf("export tables count = topics %d posts %d, want 1/1", len(doc["topics"]), len(doc["posts"]))
	}
}

func TestExportFilePathTraversal(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	task := &taskQueue.Entity{Id: 1, TaskJson: `{"fileName":"../../etc/passwd"}`}
	if _, err := ExportFilePath(task); err == nil {
		t.Fatal("ExportFilePath() with traversal error = nil, want error")
	}
}

func TestImportDataValidationAndIdempotency(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	cat := category.Entity{Name: "测试分类"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	user := users.EntityComplete{Username: "importer", Email: "importer@example.com"}
	if err := users.Create(&user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	jsonData := []byte(`[
	  {"table":"topics","id":1,"title":"导入主题","userId":` + jsonUint64(user.Id) + `,"categoryIds":"[` + jsonUint64(cat.Id) + `]"},
	  {"table":"posts","id":1,"topicId":1,"userId":` + jsonUint64(user.Id) + `,"content":"导入正文","postNo":1}
	]`)

	report, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("ImportData() failed = %d, want 0 (errors: %+v)", report.Failed, report.Errors)
	}
	if report.Success != 2 {
		t.Fatalf("ImportData() success = %d, want 2", report.Success)
	}

	// 再次导入相同数据 → 幂等跳过
	report2, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() second error = %v", err)
	}
	if report2.Skipped != 2 {
		t.Fatalf("ImportData() second skipped = %d, want 2", report2.Skipped)
	}
}

func TestImportDataForeignKeyFailure(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	// topic 引用不存在的 user
	jsonData := []byte(`[
	  {"table":"topics","id":99,"title":"孤儿主题","userId":99999}
	]`)
	report, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 1 {
		t.Fatalf("ImportData() failed = %d, want 1", report.Failed)
	}
	if len(report.Errors) != 1 || report.Errors[0].Table != "topics" {
		t.Fatalf("ImportData() errors = %+v, want one topics error", report.Errors)
	}
}

func TestImportDataRejectsCSV(t *testing.T) {
	if _, err := ImportData(context.Background(), []byte("a,b"), "csv"); err == nil {
		t.Fatal("ImportData(csv) error = nil, want error")
	}
}

func jsonUint64(v uint64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

var _ = filepath.Join // keep import used if assertions change

// TestExportMultiBatchJSONValid 验证跨批次（>exportBatchSize）导出时
// JSON 仍然合法（B1 回归：批次间逗号缺失曾导致 }{ 非法 JSON）。
func TestExportMultiBatchJSONValid(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	// 插入超过一个批次的行数（batchSize=200）
	for i := 0; i < exportBatchSize+25; i++ {
		user := users.EntityComplete{
			Username: fmt.Sprintf("batch_%d", i),
			Email:    fmt.Sprintf("batch_%d@example.com", i),
		}
		if err := conn.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
	}

	taskEntity := &taskQueue.Entity{
		Type:     TaskTypeExport,
		Status:   taskQueue.StatusPending,
		TaskJson: `{"tables":["users"],"format":"json"}`,
	}
	if err := taskQueue.Create(taskEntity); err != nil {
		t.Fatalf("create export task: %v", err)
	}
	if err := RunExportTask(context.Background(), taskEntity); err != nil {
		t.Fatalf("RunExportTask() error = %v", err)
	}
	reloaded, err := taskQueue.GetByID(taskEntity.Id)
	if err != nil {
		t.Fatalf("reload export task: %v", err)
	}
	path, err := ExportFilePath(&reloaded)
	if err != nil {
		t.Fatalf("ExportFilePath() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	var doc map[string][]map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("multi-batch export is invalid JSON: %v", err)
	}
	if len(doc["users"]) != exportBatchSize+25 {
		t.Fatalf("export users count = %d, want %d", len(doc["users"]), exportBatchSize+25)
	}
}

// TestImportPreservesOriginalIDs 验证导出→导入往返保留原始 id（B2 回归）：
// users 按导出 id 导入后，topics/posts 的外键 userId/topicId 仍然有效。
func TestImportPreservesOriginalIDs(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	cat := category.Entity{Name: "往返分类"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	// 先导入带显式 id 的 users/topics/posts（模拟导出文件）
	jsonData := []byte(`{
	  "users": [{"id": 1001, "username": "roundtrip", "email": "roundtrip@example.com"}],
	  "topics": [{"id": 2001, "title": "往返主题", "userId": 1001, "categoryIds": "[` + jsonUint64(cat.Id) + `]"}],
	  "posts": [{"id": 3001, "topicId": 2001, "userId": 1001, "content": "往返正文", "postNo": 1}]
	}`)
	report, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("ImportData() failed = %d, want 0 (errors: %+v)", report.Failed, report.Errors)
	}
	if report.Success != 3 {
		t.Fatalf("ImportData() success = %d, want 3", report.Success)
	}

	// 验证外键关系存在（topic.userId=1001, post.topicId=2001）
	var topic topics.Entity
	if err := conn.First(&topic, 2001).Error; err != nil {
		t.Fatalf("topic 2001 missing: %v", err)
	}
	if topic.UserId != 1001 {
		t.Fatalf("topic.UserId = %d, want 1001", topic.UserId)
	}
	var post posts.Entity
	if err := conn.First(&post, 3001).Error; err != nil {
		t.Fatalf("post 3001 missing: %v", err)
	}
	if post.TopicId != 2001 || post.UserId != 1001 {
		t.Fatalf("post FK = topic %d user %d, want 2001/1001", post.TopicId, post.UserId)
	}

	// 再次导入相同数据 → 全部跳过（幂等，按 id 识别）
	report2, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() second error = %v", err)
	}
	if report2.Skipped != 3 {
		t.Fatalf("ImportData() second skipped = %d, want 3", report2.Skipped)
	}
}

var _ = fmt.Sprintf // keep fmt import used if assertions change
