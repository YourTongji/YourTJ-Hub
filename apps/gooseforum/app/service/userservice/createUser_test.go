package userservice

import (
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/pointsRecord"
	"github.com/leancodebox/GooseForum/app/models/forum/userPoints"
	"github.com/leancodebox/GooseForum/app/models/forum/userStatistics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

func setupCreateUserTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatalf("migrate user tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&pointsRecord.Entity{})
	conn.Where("1 = 1").Delete(&userPoints.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func TestCreateUserStoresNormalizedLocale(t *testing.T) {
	setupCreateUserTestDB(t)

	user, err := CreateUser("lang-user", "password", "lang@example.com", false, "en-US")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.Locale != "en" {
		t.Fatalf("Locale = %q, want en", user.Locale)
	}
}

func TestCreateUserKeepsLocaleEmptyWhenMissing(t *testing.T) {
	setupCreateUserTestDB(t)

	user, err := CreateUser("empty-locale", "password", "empty@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.Locale != "" {
		t.Fatalf("Locale = %q, want empty", user.Locale)
	}
}

func TestCreateUserRollsBackWhenPointsInitializationFails(t *testing.T) {
	setupCreateUserTestDB(t)
	conn := db.Connect()
	if err := conn.Migrator().DropTable(&userPoints.Entity{}); err != nil {
		t.Fatalf("drop user_points table: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.AutoMigrate(&userPoints.Entity{}); err != nil {
			t.Errorf("restore user_points table: %v", err)
		}
	})

	if _, err := CreateUser("rollback-user", "password", "rollback@example.com", false); err == nil {
		t.Fatal("CreateUser() succeeded without user_points table")
	}
	var count int64
	if err := conn.Model(&users.EntityComplete{}).Where("username = ?", "rollback-user").Count(&count).Error; err != nil {
		t.Fatalf("count rolled back user: %v", err)
	}
	if count != 0 {
		t.Fatalf("rolled back user count = %d, want 0", count)
	}
}

func TestGenerateName(t *testing.T) {
	for range 4 {
		if name := GenerateGooseNickname(); name == "" {
			t.Fatal("expected generated nickname")
		}
	}
}
