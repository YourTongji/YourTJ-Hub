package users

import (
	"context"
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

// A policy change shares the participant lock with message sends, including the
// first save when there is no preference row to lock yet.
func TestChatPolicyOwnerLockOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := conn.DB()
	pool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = pool.Close() })
	schema := fmt.Sprintf("chat_policy_%d", time.Now().UnixNano())
	if err := conn.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DROP SCHEMA " + schema + " CASCADE") })
	if err := conn.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&EntityComplete{}, &PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&EntityComplete{Id: 1, Username: "policy-owner"}).Error; err != nil {
		t.Fatal(err)
	}
	// A pre-existing profile row upgrades with the public message default.
	if err := conn.Create(&PrivacySettingsEntity{UserID: 1, ProfileTopics: "self"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Migrator().DropColumn(&PrivacySettingsEntity{}, "ChatMessagePolicy"); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}
	got, err := GetPrivacySettingsWithDB(conn, 1)
	if err != nil || got.ChatMessagePolicy != ChatMessageEveryone || got.ProfileTopics != "self" {
		t.Fatalf("upgrade %+v %v", got, err)
	}
	if err := conn.Delete(&PrivacySettingsEntity{}, "user_id = ?", 1).Error; err != nil {
		t.Fatal(err)
	}
	writer, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	writerPool, _ := writer.DB()
	writerPool.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = writerPool.Close() })
	if err := writer.Exec("SET search_path TO " + schema).Error; err != nil {
		t.Fatal(err)
	}
	if err := writer.Exec("SET application_name = 'chat-policy-owner-update'").Error; err != nil {
		t.Fatal(err)
	}
	tx := conn.Begin()
	defer tx.Rollback()
	if err := LockInteractionUserIDs(tx, []uint64{1}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		policy := ChatMessageNobody
		_, err := updatePrivacySettings(writer.WithContext(ctx), 1, PrivacySettingsUpdate{ChatMessagePolicy: &policy})
		done <- err
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	waiting := false
	for !waiting {
		select {
		case err := <-done:
			t.Fatalf("policy save bypassed participant lock: %v", err)
		case <-ctx.Done():
			t.Fatal("policy update never reached lock")
		case <-ticker.C:
			if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM pg_stat_activity WHERE application_name = 'chat-policy-owner-update' AND wait_event_type = 'Lock')").Scan(&waiting).Error; err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got, err = GetPrivacySettingsWithDB(conn, 1)
	if err != nil || got.ChatMessagePolicy != ChatMessageNobody {
		t.Fatalf("save %+v %v", got, err)
	}
}
