package migration

import (
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func assertTopicAgentRepliesUpgrade(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.AutoMigrate(&topics.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Migrator().DropColumn(&topics.Entity{}, "agent_replies_disabled"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("INSERT INTO topics (id, title) VALUES (1051, 'legacy topic')").Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := conn.AutoMigrate(&topics.Entity{}); err != nil {
			t.Fatal(err)
		}
	}
	var topic topics.Entity
	if err := conn.First(&topic, 1051).Error; err != nil {
		t.Fatal(err)
	}
	if topic.AgentRepliesDisabled || topic.Title != "legacy topic" {
		t.Fatalf("legacy upgrade = %+v", topic)
	}
	if err := topics.UpdateAgentRepliesDisabledTx(conn, 1051, true); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&topics.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.First(&topic, 1051).Error; err != nil || !topic.AgentRepliesDisabled {
		t.Fatalf("migration reset policy: %v %+v", err, topic)
	}
}

func TestTopicAgentRepliesSchemaUpgrade(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:topic-agent-replies-upgrade?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	assertTopicAgentRepliesUpgrade(t, conn)
}

func TestTopicAgentRepliesSchemaUpgradePostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("DROP SCHEMA public CASCADE; CREATE SCHEMA public").Error; err != nil {
		t.Fatal(err)
	}
	assertTopicAgentRepliesUpgrade(t, conn)
}
