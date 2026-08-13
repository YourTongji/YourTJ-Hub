package dataservice

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicCategoryIndex"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserStat"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
)

func setupDataTestDB(t *testing.T) {
	t.Helper()
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&postRevisions.Entity{},
		&taskQueue.Entity{},
		&category.Entity{},
		&topicCategoryIndex.Entity{},
		&topicUserStat.Entity{},
	); err != nil {
		t.Fatalf("migrate data tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&postRevisions.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&taskQueue.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&category.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&topicCategoryIndex.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&topicUserStat.Entity{})
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

	// 直接构造导出任务并运行（真实 worker 路径：先原子领取拿到 fencing
	// token，进度/文件名的写回是 status=Running AND lease_token=? 的 CAS）
	taskEntity := &taskQueue.Entity{
		Type:     TaskTypeExport,
		Status:   taskQueue.StatusPending,
		TaskJson: `{"tables":["users","topics","posts"],"format":"json"}`,
	}
	if err := taskQueue.Create(taskEntity); err != nil {
		t.Fatalf("create export task: %v", err)
	}
	running, claimed, err := taskQueue.ClaimTask(taskEntity.Id)
	if err != nil || !claimed {
		t.Fatalf("claim export task: claimed=%v err=%v", claimed, err)
	}
	if err := RunExportTask(context.Background(), &running); err != nil {
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
	// 真实 worker 路径：先原子领取（拿到 fencing token），再运行 handler。
	// 进度写回是 status=Running AND lease_token=? 的 CAS，未领取的任务
	// 没有 token，进度不会落库（与线上行为一致）。
	running, claimed, err := taskQueue.ClaimTask(taskEntity.Id)
	if err != nil || !claimed {
		t.Fatalf("claim export task: claimed=%v err=%v", claimed, err)
	}
	if err := RunExportTask(context.Background(), &running); err != nil {
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

// TestExportImportRoundTripPreservesTopicInvariants 验证导出→导入空库后
// 话题 invariants（首末帖指针、post_seq、计数、发帖人、分类索引）与源库一致，
// 且导入后可继续回复（issue #135 核心回归）。
func TestExportImportRoundTripPreservesTopicInvariants(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	// 源库数据：1 个分类、2 个用户、1 个话题（2 条回复）
	cat := category.Entity{Name: "往返分类"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	author := users.EntityComplete{Username: "author", Email: "author@example.com"}
	if err := users.Create(&author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	replier := users.EntityComplete{Username: "replier", Email: "replier@example.com"}
	if err := users.Create(&replier); err != nil {
		t.Fatalf("create replier: %v", err)
	}
	now := time.Now().UTC().Add(-time.Hour)
	topic := topics.Entity{
		Title:         "往返主题",
		CategoryIds:   []uint64{cat.Id},
		UserId:        author.Id,
		Status:        1,
		PostCount:     2,
		ReplyCount:    1,
		PostSeq:       2,
		Posters:       []topics.Poster{{UserID: author.Id}, {UserID: replier.Id}},
		Excerpt:       "摘要",
		FirstImageURL: "",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatalf("create topic: %v", err)
	}
	firstPost := posts.Entity{
		TopicId: topic.Id, PostNo: 1, UserId: author.Id,
		Content: "首帖", CreatedAt: now, UpdatedAt: now,
	}
	if err := conn.Create(&firstPost).Error; err != nil {
		t.Fatalf("create first post: %v", err)
	}
	secondPost := posts.Entity{
		TopicId: topic.Id, PostNo: 2, UserId: replier.Id,
		Content: "回复", CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute),
	}
	if err := conn.Create(&secondPost).Error; err != nil {
		t.Fatalf("create second post: %v", err)
	}
	lastPosted := now.Add(time.Minute)
	topic.FirstPostId = firstPost.Id
	topic.LastPostId = secondPost.Id
	topic.LastPostedAt = &lastPosted
	if err := conn.Model(&topics.Entity{}).Where("id = ?", topic.Id).Updates(map[string]any{
		"first_post_id":  firstPost.Id,
		"last_post_id":   secondPost.Id,
		"last_posted_at": lastPosted,
	}).Error; err != nil {
		t.Fatalf("update topic invariants: %v", err)
	}
	// 分类索引 + 参与者统计
	if err := conn.Create(&topicCategoryIndex.Entity{TopicId: topic.Id, CategoryId: cat.Id, Effective: 1}).Error; err != nil {
		t.Fatalf("create topic category index: %v", err)
	}
	if err := conn.Create(&topicUserStat.Entity{TopicId: topic.Id, UserId: replier.Id, ReplyCount: 1, LastReplyAt: lastPosted}).Error; err != nil {
		t.Fatalf("create topic user stat: %v", err)
	}

	// 1) 全量导出
	taskEntity := &taskQueue.Entity{
		Type:     TaskTypeExport,
		Status:   taskQueue.StatusPending,
		TaskJson: `{"tables":["users","topics","posts","topicCategoryIndex","topicUserStat"],"format":"json"}`,
	}
	if err := taskQueue.Create(taskEntity); err != nil {
		t.Fatalf("create export task: %v", err)
	}
	// 真实 worker 路径：先原子领取拿到 fencing token 再运行 handler，
	// 进度/文件名的写回是 status=Running AND lease_token=? 的 CAS
	running, claimed, err := taskQueue.ClaimTask(taskEntity.Id)
	if err != nil || !claimed {
		t.Fatalf("claim export task: claimed=%v err=%v", claimed, err)
	}
	if err := RunExportTask(context.Background(), &running); err != nil {
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
	exported, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}

	// 导出文件应包含 invariants 字段
	var doc map[string][]map[string]any
	if err := json.Unmarshal(exported, &doc); err != nil {
		t.Fatalf("export is not valid JSON: %v", err)
	}
	if len(doc["topics"]) != 1 || len(doc["topicCategoryIndex"]) != 1 || len(doc["topicUserStat"]) != 1 {
		t.Fatalf("export tables = topics %d tci %d tus %d, want 1/1/1",
			len(doc["topics"]), len(doc["topicCategoryIndex"]), len(doc["topicUserStat"]))
	}
	topicRow := doc["topics"][0]
	if rowUint64(topicRow, "postSeq") != 2 || rowUint64(topicRow, "firstPostId") != firstPost.Id || rowUint64(topicRow, "lastPostId") != secondPost.Id {
		t.Fatalf("export topic invariants = postSeq %d first %d last %d, want 2/%d/%d",
			rowUint64(topicRow, "postSeq"), rowUint64(topicRow, "firstPostId"), rowUint64(topicRow, "lastPostId"),
			firstPost.Id, secondPost.Id)
	}
	if rowString(topicRow, "lastPostedAt") == "" {
		t.Fatal("export topic lastPostedAt missing")
	}
	if rowUint64(topicRow, "postCount") != 2 || rowUint64(topicRow, "replyCount") != 1 {
		t.Fatalf("export topic counts = post %d reply %d, want 2/1",
			rowUint64(topicRow, "postCount"), rowUint64(topicRow, "replyCount"))
	}

	// 2) 清空库后导入（fresh-DB round-trip）
	conn.Unscoped().Where("1 = 1").Delete(&topicUserStat.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&topicCategoryIndex.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&topics.Entity{})
	conn.Unscoped().Where("1 = 1").Delete(&users.EntityComplete{})
	conn.Unscoped().Where("1 = 1").Delete(&category.Entity{})
	// 分类保留（导入 topics 时校验分类存在）
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("recreate category: %v", err)
	}

	report, err := ImportData(context.Background(), exported, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("ImportData() failed = %d, want 0 (errors: %+v)", report.Failed, report.Errors)
	}

	// 3) 校验导入后话题 invariants 与源库一致
	var gotTopic topics.Entity
	if err := conn.First(&gotTopic, topic.Id).Error; err != nil {
		t.Fatalf("imported topic missing: %v", err)
	}
	if gotTopic.PostSeq != 2 || gotTopic.FirstPostId != firstPost.Id || gotTopic.LastPostId != secondPost.Id {
		t.Fatalf("imported topic invariants = postSeq %d first %d last %d, want 2/%d/%d",
			gotTopic.PostSeq, gotTopic.FirstPostId, gotTopic.LastPostId, firstPost.Id, secondPost.Id)
	}
	if gotTopic.PostCount != 2 || gotTopic.ReplyCount != 1 {
		t.Fatalf("imported topic counts = post %d reply %d, want 2/1", gotTopic.PostCount, gotTopic.ReplyCount)
	}
	if gotTopic.LastPostedAt == nil || !gotTopic.LastPostedAt.Equal(lastPosted) {
		t.Fatalf("imported topic lastPostedAt = %v, want %v", gotTopic.LastPostedAt, lastPosted)
	}
	// 发帖人列表恢复
	if len(gotTopic.Posters) != 2 || gotTopic.Posters[0].UserID != author.Id || gotTopic.Posters[1].UserID != replier.Id {
		t.Fatalf("imported topic posters = %#v, want [%d %d]", gotTopic.Posters, author.Id, replier.Id)
	}
	// 首帖/末帖指针可解析
	var gotFirst posts.Entity
	if err := conn.First(&gotFirst, gotTopic.FirstPostId).Error; err != nil {
		t.Fatalf("first post pointer broken: %v", err)
	}
	// 分类索引与参与者统计已恢复
	var tciCount, tusCount int64
	conn.Model(&topicCategoryIndex.Entity{}).Where("topic_id = ?", topic.Id).Count(&tciCount)
	conn.Model(&topicUserStat.Entity{}).Where("topic_id = ?", topic.Id).Count(&tusCount)
	if tciCount != 1 || tusCount != 1 {
		t.Fatalf("imported derived tables = tci %d tus %d, want 1/1", tciCount, tusCount)
	}

	// 4) 继续回复：下一次 post_no 应为 3（post_seq 未归零，不与首帖冲突）
	var postEntity posts.Entity
	postEntity.TopicId = topic.Id
	postEntity.UserId = author.Id
	if err := postservice.CreateTopicPost(&postEntity, gotTopic); err != nil {
		t.Fatalf("CreateTopicPost() error = %v", err)
	}
	if postEntity.PostNo != 3 {
		t.Fatalf("next post post_no = %d, want 3", postEntity.PostNo)
	}
}

// TestImportLegacyExportBackfillsInvariants 验证旧格式导出（无 invariants 字段）
// 导入后仍能从 posts 推导补齐指针/计数/序列（issue #135 兼容路径）。
func TestImportLegacyExportBackfillsInvariants(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	cat := category.Entity{Name: "旧格式分类"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	user := users.EntityComplete{Username: "legacy", Email: "legacy@example.com"}
	if err := users.Create(&user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 旧格式：topics 无 postSeq/firstPostId/lastPostId 等字段
	jsonData := []byte(`{
	  "users": [{"id": ` + jsonUint64(user.Id) + `, "username": "legacy", "email": "legacy@example.com"}],
	  "topics": [{"id": 5001, "title": "旧主题", "userId": ` + jsonUint64(user.Id) + `, "categoryIds": "[` + jsonUint64(cat.Id) + `]"}],
	  "posts": [
	    {"id": 6001, "topicId": 5001, "userId": ` + jsonUint64(user.Id) + `, "content": "首帖", "postNo": 1},
	    {"id": 6002, "topicId": 5001, "userId": ` + jsonUint64(user.Id) + `, "content": "回复", "postNo": 2}
	  ]
	}`)
	report, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("ImportData() failed = %d, want 0 (errors: %+v)", report.Failed, report.Errors)
	}

	var topic topics.Entity
	if err := conn.First(&topic, 5001).Error; err != nil {
		t.Fatalf("imported topic missing: %v", err)
	}
	if topic.PostSeq != 2 || topic.FirstPostId != 6001 || topic.LastPostId != 6002 {
		t.Fatalf("backfilled invariants = postSeq %d first %d last %d, want 2/6001/6002",
			topic.PostSeq, topic.FirstPostId, topic.LastPostId)
	}
	if topic.PostCount != 2 || topic.ReplyCount != 1 {
		t.Fatalf("backfilled counts = post %d reply %d, want 2/1", topic.PostCount, topic.ReplyCount)
	}
	// 分类索引补齐
	var tciCount int64
	conn.Model(&topicCategoryIndex.Entity{}).Where("topic_id = ?", 5001).Count(&tciCount)
	if tciCount != 1 {
		t.Fatalf("backfilled category index count = %d, want 1", tciCount)
	}
}

// TestImportDefaultUIPathRebuildsTopicUserStats 覆盖默认 UI 导出路径
// （仅 users/topics/posts 三表，无 topicUserStat）：新格式 topics 自带完整
// invariants（无需回填指针），但 topic_user_stat 未导出——导入后必须从
// posts 重建参与者统计（PR #160 review, warning 2）。
func TestImportDefaultUIPathRebuildsTopicUserStats(t *testing.T) {
	setupDataTestDB(t)
	withTempExportDir(t)

	conn := dbconnect.Connect()
	// 数据：1 分类、2 用户、1 话题（2 帖）
	cat := category.Entity{Name: "UI路径分类"}
	if err := conn.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	author := users.EntityComplete{Username: "ui-author", Email: "ui-author@example.com"}
	if err := users.Create(&author); err != nil {
		t.Fatalf("create author: %v", err)
	}
	replier := users.EntityComplete{Username: "ui-replier", Email: "ui-replier@example.com"}
	if err := users.Create(&replier); err != nil {
		t.Fatalf("create replier: %v", err)
	}
	now := time.Now().UTC().Add(-time.Hour)

	// 新格式 topics：postSeq/firstPostId/lastPostId 等 invariants 完整（无需回填）
	jsonData := []byte(`{
	  "users": [
	    {"id": ` + jsonUint64(author.Id) + `, "username": "ui-author", "email": "ui-author@example.com"},
	    {"id": ` + jsonUint64(replier.Id) + `, "username": "ui-replier", "email": "ui-replier@example.com"}
	  ],
	  "topics": [{
	    "id": 9001, "title": "UI 路径话题", "userId": ` + jsonUint64(author.Id) + `,
	    "categoryIds": "[` + jsonUint64(cat.Id) + `]", "status": 1, "postCount": 2, "replyCount": 1,
	    "postSeq": 2, "firstPostId": 9002, "lastPostId": 9003,
	    "lastPostedAt": "` + now.Add(time.Minute).Format(time.RFC3339Nano) + `",
	    "posters": "[{\"user_id\":` + jsonUint64(author.Id) + `},{\"user_id\":` + jsonUint64(replier.Id) + `}]"
	  }],
	  "posts": [
	    {"id": 9002, "topicId": 9001, "userId": ` + jsonUint64(author.Id) + `, "content": "首帖", "postNo": 1},
	    {"id": 9003, "topicId": 9001, "userId": ` + jsonUint64(replier.Id) + `, "content": "回复", "postNo": 2}
	  ]
	}`)

	report, err := ImportData(context.Background(), jsonData, "json")
	if err != nil {
		t.Fatalf("ImportData() error = %v", err)
	}
	if report.Failed != 0 {
		t.Fatalf("ImportData() failed = %d, want 0 (errors: %+v)", report.Failed, report.Errors)
	}

	// topics 非回填分支：指针/计数保持导出精确值
	var topic topics.Entity
	if err := conn.First(&topic, 9001).Error; err != nil {
		t.Fatalf("imported topic missing: %v", err)
	}
	if topic.PostSeq != 2 || topic.FirstPostId != 9002 || topic.LastPostId != 9003 {
		t.Fatalf("invariants = postSeq %d first %d last %d, want 2/9002/9003",
			topic.PostSeq, topic.FirstPostId, topic.LastPostId)
	}
	// 参与者统计被重建（warning 2 核心断言）：replier 的回复计数恢复
	var stat topicUserStat.Entity
	if err := conn.Where("topic_id = ? AND user_id = ?", 9001, replier.Id).First(&stat).Error; err != nil {
		t.Fatalf("topic_user_stat not rebuilt for replier: %v", err)
	}
	if stat.ReplyCount != 1 {
		t.Fatalf("rebuilt reply count = %d, want 1", stat.ReplyCount)
	}
	// 分类索引补齐
	var tciCount int64
	conn.Model(&topicCategoryIndex.Entity{}).Where("topic_id = ?", 9001).Count(&tciCount)
	if tciCount != 1 {
		t.Fatalf("category index count = %d, want 1", tciCount)
	}
}
