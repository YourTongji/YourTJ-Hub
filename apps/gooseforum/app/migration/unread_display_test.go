package migration

import (
	"os"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type legacyUnreadStyleUser struct {
	ID           uint64 `gorm:"primaryKey"`
	Username     string
	Email        string
	Password     string
	TokenVersion uint64
}

func (legacyUnreadStyleUser) TableName() string { return "users" }

func assertUnreadStyleUpgrade(t *testing.T, conn *gorm.DB) {
	t.Helper()
	if err := conn.AutoMigrate(&legacyUnreadStyleUser{}); err != nil {
		t.Fatal(err)
	}
	legacy := legacyUnreadStyleUser{ID: 77, Username: "legacy-badge", Email: "legacy-badge@example.test", Password: "existing-hash", TokenVersion: 9}
	if err := conn.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	var upgraded users.EntityComplete
	if err := conn.First(&upgraded, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if upgraded.UnreadBadgeStyle != users.UnreadBadgeNumber || upgraded.Password != legacy.Password || upgraded.TokenVersion != legacy.TokenVersion {
		t.Fatalf("upgrade changed default or credentials: style=%q version=%d", upgraded.UnreadBadgeStyle, upgraded.TokenVersion)
	}
	if err := conn.Model(&upgraded).Update("unread_badge_style", users.UnreadBadgeDot).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.First(&upgraded, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if upgraded.UnreadBadgeStyle != users.UnreadBadgeDot {
		t.Fatal("repeat migration reset the explicit preference")
	}
}

func TestUnreadBadgeStyleUpgradeOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:unread-style-upgrade?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	assertUnreadStyleUpgrade(t, conn)
}

func TestUnreadBadgeStyleUpgradeOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`).Error; err != nil {
		t.Fatal(err)
	}
	assertUnreadStyleUpgrade(t, conn)
}
