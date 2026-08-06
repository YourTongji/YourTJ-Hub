package dataservice

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/category"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
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
