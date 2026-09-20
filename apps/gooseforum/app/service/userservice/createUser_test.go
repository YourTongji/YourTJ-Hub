package userservice

import (
	"errors"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
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

// 注册事务内占用复查（issue #678 review P2）：目标邮箱命中他账号仍在
// 窗口内的换绑暂存时，注册必须整体回滚并按占用失败，绝不与暂存双占。
func TestCreateUserRollsBackWhenEmailFreshlyStaged(t *testing.T) {
	setupCreateUserTestDB(t)
	stager := users.MakeUser("staged-email-stager", "password", "staged-stager@example.com")
	if err := users.Create(stager); err != nil {
		t.Fatalf("create stager: %v", err)
	}
	stagedEmail := "staged-target@example.com"
	if err := users.StagePendingEmail(stager.Id, stagedEmail, time.Now()); err != nil {
		t.Fatalf("stage pending email: %v", err)
	}

	if _, err := CreateUser("staged-register", "password", stagedEmail, false); !errors.Is(err, users.ErrEmailOccupied) {
		t.Fatalf("CreateUser error = %v, want users.ErrEmailOccupied", err)
	}

	conn := db.Connect()
	var count int64
	if err := conn.Model(&users.EntityComplete{}).Where("username = ?", "staged-register").Count(&count).Error; err != nil {
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

func TestCreateUserWithQuotaRejectsWithoutInitializingPoints(t *testing.T) {
	setupCreateUserTestDB(t)
	if user, err := CreateUserWithQuota("quota-user", "password", "quota@example.test", true, 0); !errors.Is(err, users.ErrSignupQuota) || user != nil {
		t.Fatalf("quota denial: user=%v error=%v", user, err)
	}
	for _, model := range []any{&userPoints.Entity{}, &pointsRecord.Entity{}, &userStatistics.Entity{}} {
		var count int64
		if err := db.Connect().Model(model).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("%T rows=%d error=%v", model, count, err)
		}
	}
}
