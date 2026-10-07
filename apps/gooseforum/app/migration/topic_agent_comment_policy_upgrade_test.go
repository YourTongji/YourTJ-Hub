package migration

import (
	"fmt"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyTopicSchema struct {
	Id     uint64 `gorm:"primaryKey;column:id;autoIncrement;not null"`
	Title  string `gorm:"column:title;type:varchar(512);not null;default:''"`
	UserId uint64 `gorm:"column:user_id;not null;default:0"`
	Status int8   `gorm:"column:status;not null;default:0"`
}

func (legacyTopicSchema) TableName() string { return "topics" }

// TestTopicAgentCommentPolicyUpgradePreservesLegacyRows 覆盖存量库显式补列：
// 旧 topics 表没有 agent_comment_disabled，升级后列存在、存量行默认允许
// Agent 评论且数据不丢；重复执行保持幂等。
func TestTopicAgentCommentPolicyUpgradePreservesLegacyRows(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&legacyTopicSchema{}); err != nil {
		t.Fatalf("create legacy topics table: %v", err)
	}
	if err := db.Create(&legacyTopicSchema{Id: 801, Title: "legacy topic", UserId: 802, Status: 1}).Error; err != nil {
		t.Fatalf("insert legacy topic: %v", err)
	}
	if db.Migrator().HasColumn("topics", "agent_comment_disabled") {
		t.Fatal("legacy table unexpectedly already has the policy column")
	}

	if err := upgradeTopicAgentCommentPolicy(db); err != nil {
		t.Fatalf("upgrade legacy topics table: %v", err)
	}
	if !db.Migrator().HasColumn(&topics.Entity{}, "agent_comment_disabled") {
		t.Fatal("agent_comment_disabled column missing after upgrade")
	}
	var disabled bool
	if err := db.Raw("SELECT agent_comment_disabled FROM topics WHERE id = ?", 801).Scan(&disabled).Error; err != nil {
		t.Fatalf("read policy flag: %v", err)
	}
	if disabled {
		t.Fatal("legacy topic should default to allowing Agent comments")
	}

	if err := upgradeTopicAgentCommentPolicy(db); err != nil {
		t.Fatalf("second upgrade must stay idempotent: %v", err)
	}
	var title string
	if err := db.Raw("SELECT title FROM topics WHERE id = ?", 801).Scan(&title).Error; err != nil || title != "legacy topic" {
		t.Fatalf("legacy row lost after upgrade: %q %v", title, err)
	}
}
