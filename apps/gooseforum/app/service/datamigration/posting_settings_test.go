package datamigration

import (
	"encoding/json"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAddDefaultTopicLimit(t *testing.T) {
	updated, changed, err := addDefaultTopicLimit(`{"textControl":{"maxPostLength":50000}}`, 10)
	if err != nil || !changed {
		t.Fatalf("addDefaultTopicLimit() = %q, %v, %v", updated, changed, err)
	}
	var value map[string]map[string]int
	if err := json.Unmarshal([]byte(updated), &value); err != nil {
		t.Fatalf("decode updated config: %v", err)
	}
	if value["textControl"]["maxDailyTopicsPerUser"] != 10 {
		t.Fatalf("maxDailyTopicsPerUser = %d, want 10", value["textControl"]["maxDailyTopicsPerUser"])
	}
	if value["textControl"]["maxPostLength"] != 50000 {
		t.Fatalf("maxPostLength = %d, want 50000 (existing keys preserved)", value["textControl"]["maxPostLength"])
	}
}

func TestAddDefaultTopicLimitPreservesExplicitValue(t *testing.T) {
	input := `{"textControl":{"maxDailyTopicsPerUser":0}}`
	updated, changed, err := addDefaultTopicLimit(input, 10)
	if err != nil || changed || updated != input {
		t.Fatalf("addDefaultTopicLimit() = %q, %v, %v", updated, changed, err)
	}
}

func TestAddDefaultTopicLimitHandlesMalformedConfig(t *testing.T) {
	if _, _, err := addDefaultTopicLimit(`{not-json`, 10); err == nil {
		t.Fatal("addDefaultTopicLimit() = nil error for malformed config")
	}
}

func openPostingSettingsTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// cache=shared 的进程内 DB 在包内测试间持久存在，连接名必须唯一，
	// 否则后续测试建表会撞已有表（admin_secret_plaintext_test.go 同约定）。
	conn, err := gorm.Open(sqlite.Open("file:migration-posting-settings-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE page_config (
		id integer primary key autoincrement,
		page_type varchar(128) not null default '',
		config text,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create page_config: %v", err)
	}
	return conn
}

func TestEnsurePostingSettingsTopicLimitAddsMissingKey(t *testing.T) {
	conn := openPostingSettingsTestDB(t)
	seedLegacyPageConfig(t, conn, "postingSettings", `{"textControl":{"maxPostLength":50000},"uploadControl":{}}`)

	result := EnsurePostingSettingsTopicLimitWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("migration failed: %+v", result)
	}
	if !result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v, want updated only", result.Updated, result.Skipped)
	}

	config := readPageConfig(t, conn, "postingSettings")
	textControl, ok := config["textControl"].(map[string]any)
	if !ok {
		t.Fatalf("textControl missing in migrated config: %#v", config)
	}
	if got := int(textControl["maxDailyTopicsPerUser"].(float64)); got != 10 {
		t.Fatalf("maxDailyTopicsPerUser = %d, want 10", got)
	}
}

func TestEnsurePostingSettingsTopicLimitSkipsExplicitValue(t *testing.T) {
	conn := openPostingSettingsTestDB(t)
	seedLegacyPageConfig(t, conn, "postingSettings", `{"textControl":{"maxDailyTopicsPerUser":0}}`)

	result := EnsurePostingSettingsTopicLimitWithDB(conn)
	if result.Failed > 0 || result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want no-op", result.Updated, result.Skipped, result.Failed)
	}

	config := readPageConfig(t, conn, "postingSettings")
	textControl, ok := config["textControl"].(map[string]any)
	if !ok {
		t.Fatalf("textControl missing in migrated config: %#v", config)
	}
	if got := int(textControl["maxDailyTopicsPerUser"].(float64)); got != 0 {
		t.Fatalf("maxDailyTopicsPerUser = %d, want preserved 0", got)
	}
}

func TestEnsurePostingSettingsTopicLimitSkipsWhenNoRow(t *testing.T) {
	conn := openPostingSettingsTestDB(t)
	result := EnsurePostingSettingsTopicLimitWithDB(conn)
	if result.Failed > 0 || result.Updated || !result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want skipped", result.Updated, result.Skipped, result.Failed)
	}
}
