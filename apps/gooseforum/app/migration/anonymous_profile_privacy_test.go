package migration

import (
	"fmt"
	"os"
	"testing"
	"time"

	identity "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/anonymousIdentity"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func exerciseAnonymousProfilePrivacyUpgrade(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(&identity.Persona{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropColumn(&identity.Persona{}, "show_content"); err != nil {
		t.Fatal(err)
	}
	when := time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC)
	if err := db.Table("anonymous_personas").Create(map[string]any{"uid": "legacy", "name": "躲进云里的猫", "avatar_seed": "retained-seed", "name_selected_at": when, "name_change_available_at": when.AddDate(1, 0, 0)}).Error; err != nil {
		t.Fatal(err)
	}
	if err := upgradeAnonymousProfilePrivacy(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&identity.Persona{}); err != nil {
		t.Fatal(err)
	}
	var p identity.Persona
	if err := db.First(&p, "uid = ?", "legacy").Error; err != nil || !p.ShowContent || p.AvatarSeed != "retained-seed" || p.Name != "躲进云里的猫" || !p.NameSelectedAt.Equal(when) || !p.NameChangeAvailableAt.Equal(when.AddDate(1, 0, 0)) {
		t.Fatal("legacy persona changed", p, err)
	}
	if err := db.Model(&p).Update("show_content", false).Error; err != nil {
		t.Fatal(err)
	}
	if err := upgradeAnonymousProfilePrivacy(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&identity.Persona{}); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&p, "uid = ?", "legacy").Error; err != nil || p.ShowContent {
		t.Fatal("repeated upgrade reset privacy", p, err)
	}
}

func TestAnonymousProfilePrivacyUpgrade(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	exerciseAnonymousProfilePrivacyUpgrade(t, db)
}

func TestPostgreSQLAnonymousProfilePrivacyUpgrade(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL required")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("persona_privacy_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error; err != nil {
			t.Error(err)
		}
		pool, err := admin.DB()
		if err != nil {
			t.Error(err)
			return
		}
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	exerciseAnonymousProfilePrivacyUpgrade(t, db)
}
