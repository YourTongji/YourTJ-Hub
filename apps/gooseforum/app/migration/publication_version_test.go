package migration

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userBadges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
)

// A deployment already at v30 must still run the legacy pending adoption.
func TestVersionedMigrationRunsPublicationAdoptionFrom30(t *testing.T) {
	conn := dbconnect.Connect()
	models := []any{&topics.Entity{}, &pageConfig.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &badges.Entity{}, &userBadges.Entity{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(30); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("migration stopped at %d; want %d", got, pageConfig.AppMigrationVersion)
	}
	var dailyRule xpmodel.RewardRule
	if err := conn.Where("rule_set_version = ? AND rule_key = ?", 1, "daily_base").Take(&dailyRule).Error; err != nil {
		t.Fatalf("initial daily XP rule missing after migration: %v", err)
	}
	if dailyRule.RewardXP != 5 || dailyRule.DailyCountLimit == nil || *dailyRule.DailyCountLimit != 1 {
		t.Fatalf("initial daily XP rule = %+v; want confirmed +5 XP once per day", dailyRule)
	}
}

func TestVersionedMigrationFrom33SeedsInitialXPRule(t *testing.T) {
	conn := dbconnect.Connect()
	models := []any{&topics.Entity{}, &pageConfig.Entity{}, &posts.Entity{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(33); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("migration from v33 stopped at %d; want %d", got, pageConfig.AppMigrationVersion)
	}
	var dailyRule xpmodel.RewardRule
	if err := conn.Where("rule_set_version = ? AND rule_key = ?", 1, "daily_base").Take(&dailyRule).Error; err != nil {
		t.Fatalf("initial daily XP rule missing after v33 upgrade: %v", err)
	}
	if dailyRule.RewardXP != 5 {
		t.Fatalf("initial daily XP rule grants %d; want confirmed +5 XP", dailyRule.RewardXP)
	}
}

func TestVersionedMigrationFrom34SeedsInitialXPRule(t *testing.T) {
	conn := dbconnect.Connect()
	models := []any{&topics.Entity{}, &pageConfig.Entity{}, &posts.Entity{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(34); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("migration from v34 stopped at %d; want %d", got, pageConfig.AppMigrationVersion)
	}
	var dailyRule xpmodel.RewardRule
	if err := conn.Where("rule_set_version = ? AND rule_key = ?", 1, "daily_base").Take(&dailyRule).Error; err != nil {
		t.Fatalf("initial daily XP rule missing after v34 upgrade: %v", err)
	}
	if dailyRule.RewardXP != 5 {
		t.Fatalf("initial daily XP rule grants %d; want confirmed +5 XP", dailyRule.RewardXP)
	}
}

func TestVersionedMigrationFrom31RunsRobotBadgesThenXPAndIsIdempotent(t *testing.T) {
	conn := dbconnect.Connect()
	models := []any{&topics.Entity{}, &pageConfig.Entity{}, &posts.Entity{}, &users.EntityComplete{}, &badges.Entity{}, &userBadges.Entity{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := pageConfig.SyncMigrationVersion(31); err != nil {
		t.Fatal(err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatal(err)
	}
	if got := pageConfig.GetMigrationVersion(); got != pageConfig.AppMigrationVersion {
		t.Fatalf("migration from v31 stopped at %d; want %d", got, pageConfig.AppMigrationVersion)
	}
	var ruleCount int64
	if err := conn.Model(&xpmodel.RewardRule{}).Where("rule_set_version = ? AND rule_key = ?", 1, "daily_base").Count(&ruleCount).Error; err != nil || ruleCount != 1 {
		t.Fatalf("daily base rules after v31 upgrade = %d, %v; want one", ruleCount, err)
	}
	if err := runVersionedDataMigrations(); err != nil {
		t.Fatalf("repeat versioned migration: %v", err)
	}
	var rulesets, rules int64
	if err := conn.Model(&xpmodel.RuleSet{}).Count(&rulesets).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&xpmodel.RewardRule{}).Count(&rules).Error; err != nil {
		t.Fatal(err)
	}
	if rulesets != 1 || rules != 1 {
		t.Fatalf("XP defaults after repeated migration = %d rulesets, %d rules; want one of each", rulesets, rules)
	}
}
