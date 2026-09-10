package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// TestSchemaModelsRegistersPostRevisions 验证 SchemaModels 已注册 post_revisions
// 模型（生产迁移入口依赖该注册，注册丢失则表永远不会创建）。
func TestSchemaModelsRegistersPostRevisions(t *testing.T) {
	for _, model := range SchemaModels() {
		if _, ok := model.(*postRevisions.Entity); ok {
			return
		}
	}
	t.Fatal("SchemaModels() does not include *postRevisions.Entity")
}

// TestPostRevisionsSchemaCreatedOnSQLite 验证全新库上 post_revisions 表可创建，
// 结构完整（append-only 快照所需列）且可正常写入/查询。
func TestPostRevisionsSchemaCreatedOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-post-revisions-create?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.AutoMigrate(&postRevisions.Entity{}); err != nil {
		t.Fatalf("AutoMigrate post_revisions: %v", err)
	}
	if !conn.Migrator().HasTable("post_revisions") {
		t.Fatal("post_revisions table missing after AutoMigrate")
	}
	for _, column := range []string{"id", "post_id", "version", "editor_id", "content", "rendered_html", "process_status", "created_at"} {
		if !conn.Migrator().HasColumn(&postRevisions.Entity{}, column) {
			t.Errorf("post_revisions column %q missing after AutoMigrate", column)
		}
	}

	// 追加语义：同一帖子可写入多个版本（(post_id, version) 由业务层保证单调，
	// 表结构不设唯一约束，验证列可写即可）
	if err := conn.Create(&postRevisions.Entity{PostId: 1, Version: 1, EditorId: 1, Content: "v1", ProcessStatus: 0}).Error; err != nil {
		t.Fatalf("insert revision v1: %v", err)
	}
	if err := conn.Create(&postRevisions.Entity{PostId: 1, Version: 2, EditorId: 2, Content: "v2", ProcessStatus: 2}).Error; err != nil {
		t.Fatalf("insert revision v2: %v", err)
	}
	var count int64
	if err := conn.Model(&postRevisions.Entity{}).Where("post_id = ?", 1).Count(&count).Error; err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if count != 2 {
		t.Fatalf("revision count = %d, want 2", count)
	}
}

// TestPostRevisionsSchemaUpgradeFromLegacySubset 模拟存量实例升级：旧库没有
// post_revisions 表（功能上线前），AutoMigrate 必须自动补齐新表且不破坏旧表
// 与存量数据。
func TestPostRevisionsSchemaUpgradeFromLegacySubset(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-post-revisions-upgrade?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// 旧库：只有 topics + posts（功能上线前的形态），并插入存量数据
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatalf("AutoMigrate legacy subset: %v", err)
	}
	if err := conn.Create(&topics.Entity{Id: 1, Title: "legacy topic", UserId: 1, Status: 1}).Error; err != nil {
		t.Fatalf("insert legacy topic: %v", err)
	}
	if err := conn.Create(&posts.Entity{Id: 1, TopicId: 1, PostNo: 1, UserId: 1, Content: "legacy first post"}).Error; err != nil {
		t.Fatalf("insert legacy post: %v", err)
	}
	if conn.Migrator().HasTable("post_revisions") {
		t.Fatal("precondition failed: legacy schema should not have post_revisions")
	}

	// 部署新二进制：AutoMigrate 补齐 post_revisions
	if err := conn.AutoMigrate(&postRevisions.Entity{}); err != nil {
		t.Fatalf("upgrade AutoMigrate post_revisions: %v", err)
	}
	if !conn.Migrator().HasTable("post_revisions") {
		t.Fatal("post_revisions table missing after upgrade")
	}
	// 旧表数据保留
	var topic topics.Entity
	if err := conn.First(&topic, 1).Error; err != nil || topic.Title != "legacy topic" {
		t.Fatalf("legacy topic lost after upgrade: %+v err=%v", topic, err)
	}
	var post posts.Entity
	if err := conn.First(&post, 1).Error; err != nil || post.Content != "legacy first post" {
		t.Fatalf("legacy post lost after upgrade: %+v err=%v", post, err)
	}
	// 新表可用
	if err := conn.Create(&postRevisions.Entity{PostId: 1, Version: 1, EditorId: 1, Content: "post-upgrade revision"}).Error; err != nil {
		t.Fatalf("insert revision after upgrade: %v", err)
	}
}
