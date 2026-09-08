package dailyStats

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestIncrementPostgreSQLExistingRow(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL daily stats test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}
	if err := db.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("AutoMigrate daily stats on postgres: %v", err)
	}

	day := time.Date(2099, 1, 3, 0, 0, 0, 0, time.UTC)
	dayString := day.Format("2006-01-02")
	if err := db.Where("stat_date = ?", dayString).Delete(&Entity{}).Error; err != nil {
		t.Fatalf("clean daily stats before test: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Where("stat_date = ?", dayString).Delete(&Entity{}).Error
	})

	for _, key := range []StatType{StatTypeRegCount, StatTypeTopicCount, StatTypeReplyCount} {
		if err := db.Table(tableName).Clauses(clause.OnConflict{DoNothing: true}).Create(map[string]any{
			"stat_date":  dayString,
			"stat_key":   string(key),
			"stat_value": int64(0),
		}).Error; err != nil {
			t.Fatalf("preinitialize %s: %v", key, err)
		}
		if err := increment(db.Table(tableName), day, key, 1); err != nil {
			t.Fatalf("first increment(%s): %v", key, err)
		}
		if err := increment(db.Table(tableName), day, key, 1); err != nil {
			t.Fatalf("second increment(%s): %v", key, err)
		}

		var got Entity
		if err := db.Where("stat_date = ? AND stat_key = ?", dayString, key).First(&got).Error; err != nil {
			t.Fatalf("load %s: %v", key, err)
		}
		if got.StatValue != 2 {
			t.Fatalf("%s stat_value = %d, want 2", key, got.StatValue)
		}
	}
}
