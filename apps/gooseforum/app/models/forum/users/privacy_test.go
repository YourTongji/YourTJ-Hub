package users

import (
	"errors"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
)

func TestPrivacySettingsDefaultsAndPartialUpdates(t *testing.T) {
	setupUserIsolationTestDB(t)
	if err := db.Connect().AutoMigrate(&PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}
	const owner = uint64(4242)
	// Existing accounts have no row and keep the public pre-release behavior.
	got, err := GetPrivacySettings(owner)
	if err != nil || got != DefaultPrivacySettings(owner) {
		t.Fatalf("defaults = %+v, %v", got, err)
	}
	self, mutual := AudienceSelf, AudienceMutual
	if _, err := UpdatePrivacySettings(owner, PrivacySettingsUpdate{ProfileTopics: &self}); err != nil {
		t.Fatal(err)
	}
	if _, err := UpdatePrivacySettings(owner, PrivacySettingsUpdate{FollowersList: &mutual}); err != nil {
		t.Fatal(err)
	}
	got, _ = GetPrivacySettings(owner)
	if got.ProfileTopics != self || got.FollowersList != mutual || got.ProfileActivity != AudienceEveryone {
		t.Fatalf("partial updates merged wrongly: %+v", got)
	}
	// One invalid field rejects the whole update.
	members, everyone := AudienceMembers, AudienceEveryone
	if _, err := UpdatePrivacySettings(owner, PrivacySettingsUpdate{ProfileTopics: &everyone, ProfileLikes: &members}); !errors.Is(err, ErrPrivacyAudience) {
		t.Fatalf("invalid likes audience: %v", err)
	}
	if after, _ := GetPrivacySettings(owner); after != got {
		t.Fatalf("rejected update changed storage: %+v", after)
	}
	var rows int64
	db.Connect().Model(&PrivacySettingsEntity{}).Where("user_id = ?", owner).Count(&rows)
	if rows != 1 {
		t.Fatalf("rows = %d", rows)
	}
}

// Two first saves can both find no row. The later upsert must only write its
// own fields, or it resets the other save to the default.
func TestPrivacySettingsConcurrentFirstSavesKeepBothFields(t *testing.T) {
	setupUserIsolationTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&PrivacySettingsEntity{}); err != nil {
		t.Fatal(err)
	}
	const owner = uint64(4343)
	competing := false
	// Simulate the other writer committing between our read and our upsert.
	err := conn.Callback().Create().Before("gorm:create").Register("test:privacy_race", func(tx *gorm.DB) {
		if competing || tx.Statement.Table != "user_privacy_settings" {
			return
		}
		competing = true
		if err := tx.Session(&gorm.Session{NewDB: true}).Exec(
			"INSERT INTO user_privacy_settings (user_id, profile_topics, profile_activity, profile_likes, following_list, followers_list, updated_at) VALUES (?, 'everyone', 'everyone', 'self', 'everyone', 'everyone', CURRENT_TIMESTAMP)",
			owner,
		).Error; err != nil {
			t.Errorf("competing insert: %v", err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove("test:privacy_race") })

	mutual := AudienceMutual
	saved, err := UpdatePrivacySettings(owner, PrivacySettingsUpdate{FollowersList: &mutual})
	if err != nil {
		t.Fatal(err)
	}
	if !competing {
		t.Fatal("race was not simulated")
	}
	got, _ := GetPrivacySettings(owner)
	if got.FollowersList != mutual || got.ProfileLikes != AudienceSelf {
		t.Fatalf("stored %+v, want followers mutual and likes self", got)
	}
	if saved != got {
		t.Fatalf("response %+v differs from storage %+v", saved, got)
	}
}
