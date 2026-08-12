package migration

import (
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"gorm.io/gorm"
)

func TestPointsRecordSourceKeyUpgradeOnSQLite(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE points_record (
		id integer primary key autoincrement,
		user_id integer not null default 0,
		action text not null default '',
		points_change integer not null default 0,
		created_at datetime
	)`).Error; err != nil {
		t.Fatalf("create legacy points_record: %v", err)
	}
	if err := conn.Exec(`INSERT INTO points_record (user_id, action, points_change) VALUES
		(1, 'init', 100), (2, 'init', 100)`).Error; err != nil {
		t.Fatalf("insert legacy points records: %v", err)
	}

	if err := conn.AutoMigrate(&pointsRecord.Entity{}); err != nil {
		t.Fatalf("upgrade points_record: %v", err)
	}
	if !conn.Migrator().HasColumn(&pointsRecord.Entity{}, "source_key") {
		t.Fatal("source_key column missing after SQLite upgrade")
	}
	key := "sqlite-upgrade:unique"
	if err := conn.Create(&pointsRecord.Entity{UserId: 3, Action: "test", SourceKey: &key}).Error; err != nil {
		t.Fatalf("insert source_key after upgrade: %v", err)
	}
	if err := conn.Create(&pointsRecord.Entity{UserId: 4, Action: "test", SourceKey: &key}).Error; !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("duplicate source_key error = %v, want gorm.ErrDuplicatedKey", err)
	}
	var nullCount int64
	if err := conn.Model(&pointsRecord.Entity{}).Where("source_key IS NULL").Count(&nullCount).Error; err != nil {
		t.Fatalf("count legacy NULL source keys: %v", err)
	}
	if nullCount != 2 {
		t.Fatalf("legacy NULL source key count = %d, want 2", nullCount)
	}
}
