package users

import (
	"fmt"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Issue #1124: partial updates and the first-save race on PostgreSQL, where
// SELECT ... FOR UPDATE cannot lock a row that does not exist yet.
func TestPrivacySettingsUpsertOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	// One connection keeps the session search_path for every statement.
	sqlDB.SetMaxOpenConns(1)
	schema := fmt.Sprintf("privacy_settings_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}

	self, mutual := AudienceSelf, AudienceMutual
	if _, err := updatePrivacySettings(conn, 1, PrivacySettingsUpdate{ProfileTopics: &self}); err != nil {
		t.Fatal(err)
	}
	got, err := updatePrivacySettings(conn, 1, PrivacySettingsUpdate{FollowersList: &mutual})
	if err != nil {
		t.Fatal(err)
	}
	if got.ProfileTopics != self || got.FollowersList != mutual || got.ProfileLikes != AudienceEveryone {
		t.Fatalf("partial updates merged wrongly: %+v", got)
	}

	competing := false
	if err := conn.Callback().Create().Before("gorm:create").Register("test:privacy_race_pg", func(tx *gorm.DB) {
		if competing || tx.Statement.Table != "user_privacy_settings" {
			return
		}
		competing = true
		if err := tx.Session(&gorm.Session{NewDB: true}).Exec(
			"INSERT INTO user_privacy_settings (user_id, profile_topics, profile_activity, profile_likes, following_list, followers_list, updated_at) VALUES (2, 'everyone', 'everyone', 'self', 'everyone', 'everyone', now())",
		).Error; err != nil {
			t.Errorf("competing insert: %v", err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	saved, err := updatePrivacySettings(conn, 2, PrivacySettingsUpdate{FollowersList: &mutual})
	if err != nil {
		t.Fatal(err)
	}
	if !competing || saved.ProfileLikes != AudienceSelf || saved.FollowersList != mutual {
		t.Fatalf("first-save race lost a field: competing=%v saved=%+v", competing, saved)
	}
}
