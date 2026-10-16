package users

import (
	"errors"
	"slices"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Profile audiences, ordered from the widest to the narrowest.
const (
	AudienceEveryone  = "everyone"
	AudienceMembers   = "members"
	AudienceFollowers = "followers"
	AudienceMutual    = "mutual"
	AudienceSelf      = "self"
)

var ErrPrivacyAudience = errors.New("invalid privacy audience")

// PrivacySettingsEntity stores which profile aggregates other people may see.
// A missing row means the public defaults that predate the setting, so
// existing accounts need no backfill. It never controls access to the
// underlying topics, replies or relationships.
type PrivacySettingsEntity struct {
	UserID          uint64    `gorm:"primaryKey;autoIncrement:false;not null" json:"-"`
	ProfileTopics   string    `gorm:"type:varchar(16);not null;default:'everyone'" json:"profileTopics"`
	ProfileActivity string    `gorm:"type:varchar(16);not null;default:'everyone'" json:"profileActivity"`
	ProfileLikes    string    `gorm:"type:varchar(16);not null;default:'everyone'" json:"profileLikes"`
	FollowingList   string    `gorm:"type:varchar(16);not null;default:'everyone'" json:"followingList"`
	FollowersList   string    `gorm:"type:varchar(16);not null;default:'everyone'" json:"followersList"`
	UpdatedAt       time.Time `json:"-"`
}

func (PrivacySettingsEntity) TableName() string { return "user_privacy_settings" }

// PrivacySettingsUpdate carries a partial owner update; nil fields keep their value.
type PrivacySettingsUpdate struct {
	ProfileTopics   *string
	ProfileActivity *string
	ProfileLikes    *string
	FollowingList   *string
	FollowersList   *string
}

var allProfileAudiences = []string{AudienceEveryone, AudienceMembers, AudienceFollowers, AudienceMutual, AudienceSelf}

// PrivacyAudiences lists the audiences each setting accepts. Likes and
// relationship lists omit combinations the issue contract does not offer.
var PrivacyAudiences = map[string][]string{
	"profileTopics":   allProfileAudiences,
	"profileActivity": allProfileAudiences,
	"profileLikes":    {AudienceEveryone, AudienceFollowers, AudienceSelf},
	"followingList":   {AudienceEveryone, AudienceFollowers, AudienceMutual, AudienceSelf},
	"followersList":   {AudienceEveryone, AudienceFollowers, AudienceMutual, AudienceSelf},
}

func DefaultPrivacySettings(userID uint64) PrivacySettingsEntity {
	return PrivacySettingsEntity{UserID: userID, ProfileTopics: AudienceEveryone, ProfileActivity: AudienceEveryone, ProfileLikes: AudienceEveryone, FollowingList: AudienceEveryone, FollowersList: AudienceEveryone}
}

// GetPrivacySettings returns stored settings or the public defaults. Storage
// errors are returned so callers can fail closed.
func GetPrivacySettings(userID uint64) (PrivacySettingsEntity, error) {
	settings := DefaultPrivacySettings(userID)
	err := db.Connect().Where("user_id = ?", userID).Take(&settings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultPrivacySettings(userID), nil
	}
	return settings, err
}

// UpdatePrivacySettings validates every supplied audience before writing, then
// applies the partial update atomically on top of the current row.
func UpdatePrivacySettings(userID uint64, update PrivacySettingsUpdate) (PrivacySettingsEntity, error) {
	return updatePrivacySettings(db.Connect(), userID, update)
}

func updatePrivacySettings(conn *gorm.DB, userID uint64, update PrivacySettingsUpdate) (PrivacySettingsEntity, error) {
	fields := []struct {
		key    string
		column string
		value  *string
		apply  func(*PrivacySettingsEntity, string)
	}{
		{"profileTopics", "profile_topics", update.ProfileTopics, func(s *PrivacySettingsEntity, v string) { s.ProfileTopics = v }},
		{"profileActivity", "profile_activity", update.ProfileActivity, func(s *PrivacySettingsEntity, v string) { s.ProfileActivity = v }},
		{"profileLikes", "profile_likes", update.ProfileLikes, func(s *PrivacySettingsEntity, v string) { s.ProfileLikes = v }},
		{"followingList", "following_list", update.FollowingList, func(s *PrivacySettingsEntity, v string) { s.FollowingList = v }},
		{"followersList", "followers_list", update.FollowersList, func(s *PrivacySettingsEntity, v string) { s.FollowersList = v }},
	}
	for _, field := range fields {
		if field.value != nil && !validPrivacyAudience(field.key, *field.value) {
			return PrivacySettingsEntity{}, ErrPrivacyAudience
		}
	}
	var settings PrivacySettingsEntity
	err := conn.Transaction(func(tx *gorm.DB) error {
		settings = DefaultPrivacySettings(userID)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).Take(&settings).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		columns := []string{"updated_at"}
		for _, field := range fields {
			if field.value != nil {
				field.apply(&settings, *field.value)
				columns = append(columns, field.column)
			}
		}
		settings.UserID = userID
		// The row lock serializes existing rows. Two first saves can both miss
		// the row, so the upsert writes only the supplied columns and keeps
		// whatever the other save stored.
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns(columns)}).Create(&settings).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userID).Take(&settings).Error
	})
	return settings, err
}

func validPrivacyAudience(key, audience string) bool {
	return slices.Contains(PrivacyAudiences[key], audience)
}
