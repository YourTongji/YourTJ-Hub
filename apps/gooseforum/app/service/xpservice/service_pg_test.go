package xpservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestApplyRewardConcurrentWritesOnPostgres(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL XP service test")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	schema := "xp_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error
	})
	conn, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect isolated PostgreSQL schema: %v", err)
	}
	if sqlDB, err := conn.DB(); err == nil {
		sqlDB.SetMaxOpenConns(16)
	}
	models := []any{&users.EntityComplete{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate XP schema on PostgreSQL: %v", err)
	}
	if err := xpmodel.EnsureAppendOnlyGuards(conn); err != nil {
		t.Fatalf("install PostgreSQL XP guards: %v", err)
	}
	if err := EnsureInitialRules(context.Background(), conn); err != nil {
		t.Fatalf("seed initial XP rules on PostgreSQL: %v", err)
	}
	if err := conn.Create(&users.EntityComplete{Username: "xp-pg-" + strconv.FormatInt(time.Now().UnixNano(), 10), ActorType: users.ActorTypeHuman, IsFrozen: users.StatusNormal}).Error; err != nil {
		t.Fatalf("create PostgreSQL XP user: %v", err)
	}
	var account users.EntityComplete
	if err := conn.Order("id DESC").Take(&account).Error; err != nil {
		t.Fatal(err)
	}
	svc := newWithPolicy(conn, pageConfig.SecurityAndRegistration{})
	if !conn.Migrator().HasColumn(&xpmodel.Entry{}, "actor_user_id") || !conn.Migrator().HasColumn(&xpmodel.RuleSet{}, "published_by") || !conn.Migrator().HasTable(&xpmodel.RulePublicationLock{}) {
		t.Fatal("PostgreSQL XP audit or publication-lock schema is incomplete")
	}
	if _, err := svc.PublishRuleSet(testActorContext(account.Id), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: time.Now().UTC(),
		Rules: []xpmodel.RewardRule{{RuleKey: "postgres_concurrent", Enabled: true, RewardXP: 1}},
	}); err != nil {
		t.Fatalf("publish PostgreSQL test rule: %v", err)
	}

	const workers = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := svc.ApplyReward(context.Background(), AwardRequest{
				UserID: account.Id, RuleKey: "postgres_concurrent",
				SourceKey: fmt.Sprintf("pg-concurrent:%d", index), OccurredAt: time.Now().UTC(),
			})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("PostgreSQL concurrent award failed: %v", err)
		}
	}
	if got, err := svc.CurrentBalance(context.Background(), account.Id); err != nil || got != workers {
		t.Fatalf("PostgreSQL XP balance = %d, %v; want %d", got, err, workers)
	}

	const replayWorkers = 16
	startReplay := make(chan struct{})
	var replayWG sync.WaitGroup
	results := make(chan AwardResult, replayWorkers)
	errorsCh := make(chan error, replayWorkers)
	for i := 0; i < replayWorkers; i++ {
		replayWG.Add(1)
		go func() {
			defer replayWG.Done()
			<-startReplay
			result, err := svc.ApplyReward(context.Background(), AwardRequest{
				UserID: account.Id, RuleKey: "postgres_concurrent", SourceKey: "pg-concurrent:shared", OccurredAt: time.Now().UTC(),
			})
			results <- result
			errorsCh <- err
		}()
	}
	close(startReplay)
	replayWG.Wait()
	close(results)
	close(errorsCh)
	created := 0
	for result := range results {
		if !result.AlreadyApplied {
			created++
		}
	}
	for err := range errorsCh {
		if err != nil {
			t.Errorf("PostgreSQL concurrent same-source replay failed: %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("PostgreSQL entries created for same source = %d, want 1", created)
	}
	if got, err := svc.CurrentBalance(context.Background(), account.Id); err != nil || got != workers+1 {
		t.Fatalf("PostgreSQL XP balance after same-source replay = %d, %v; want %d", got, err, workers+1)
	}

	rollbackUserName := "xp-pg-rollback-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := conn.Create(&users.EntityComplete{Username: rollbackUserName, ActorType: users.ActorTypeHuman, IsFrozen: users.StatusNormal}).Error; err != nil {
		t.Fatalf("create PostgreSQL rollback user: %v", err)
	}
	var rollbackUser users.EntityComplete
	if err := conn.Where("username = ?", rollbackUserName).Take(&rollbackUser).Error; err != nil {
		t.Fatal(err)
	}
	limit := int64(5)
	rollbackAt := time.Now().UTC().Add(time.Hour)
	if _, err := svc.PublishRuleSet(testActorContext(account.Id), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, DailyPositiveXPLimit: &limit, EffectiveAt: rollbackAt,
		Rules: []xpmodel.RewardRule{
			{RuleKey: "pg-rollback-base", Enabled: true, RewardXP: 5},
			{RuleKey: "pg-rollback-over", Enabled: true, RewardXP: 1},
		},
	}); err != nil {
		t.Fatalf("publish PostgreSQL rollback rules: %v", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: rollbackUser.Id, RuleKey: "pg-rollback-base", SourceKey: "pg-rollback-base", OccurredAt: rollbackAt.Add(time.Second)}); err != nil {
		t.Fatalf("PostgreSQL base award before rollback check: %v", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: rollbackUser.Id, RuleKey: "pg-rollback-over", SourceKey: "pg-rollback-over", OccurredAt: rollbackAt.Add(2 * time.Second)}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("PostgreSQL quota failure = %v, want ErrRewardLimit", err)
	}
	var rejectedRows int64
	if err := conn.Model(&xpmodel.Entry{}).Where("source_key = ?", "pg-rollback-over").Count(&rejectedRows).Error; err != nil || rejectedRows != 0 {
		t.Fatalf("PostgreSQL rejected award ledger rows = %d, %v; want rollback", rejectedRows, err)
	}
	var rejectedRuleUsage int64
	if err := conn.Model(&xpmodel.DailyRuleUsage{}).Where("user_id = ? AND rule_key = ?", rollbackUser.Id, "pg-rollback-over").Count(&rejectedRuleUsage).Error; err != nil || rejectedRuleUsage != 0 {
		t.Fatalf("PostgreSQL rejected award rule-usage rows = %d, %v; want rollback", rejectedRuleUsage, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), rollbackUser.Id); err != nil || got != 5 {
		t.Fatalf("PostgreSQL rollback balance = %d, %v; want 5", got, err)
	}
	var committedUsage xpmodel.DailyUsage
	if err := conn.Where("user_id = ?", rollbackUser.Id).Take(&committedUsage).Error; err != nil || committedUsage.PositiveXP != 5 {
		t.Fatalf("PostgreSQL rollback daily usage = %+v, %v; want only committed 5 XP", committedUsage, err)
	}

	var entry xpmodel.Entry
	if err := conn.Where("source_key = ?", "pg-concurrent:0").Take(&entry).Error; err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []struct {
		name  string
		query string
		args  []any
	}{
		{name: "ledger update", query: "UPDATE xp_entries SET reason = ? WHERE id = ?", args: []any{"mutated", entry.ID}},
		{name: "ledger delete", query: "DELETE FROM xp_entries WHERE id = ?", args: []any{entry.ID}},
		{name: "ruleset update", query: "UPDATE xp_rule_sets SET enabled = FALSE WHERE version = ?", args: []any{1}},
		{name: "ruleset delete", query: "DELETE FROM xp_rule_sets WHERE version = ?", args: []any{1}},
		{name: "reward rule update", query: "UPDATE xp_reward_rules SET enabled = FALSE WHERE rule_set_version = ? AND rule_key = ?", args: []any{1, DailyBaseRuleKey}},
		{name: "reward rule delete", query: "DELETE FROM xp_reward_rules WHERE rule_set_version = ? AND rule_key = ?", args: []any{1, DailyBaseRuleKey}},
	} {
		if err := conn.Exec(mutation.query, mutation.args...).Error; err == nil || !strings.Contains(strings.ToLower(err.Error()), "append-only") {
			t.Errorf("PostgreSQL %s error = %v; want append-only error", mutation.name, err)
		}
	}
	if err := conn.Model(&xpmodel.Balance{}).Where("user_id = ?", account.Id).Update("current_xp", -1).Error; err == nil {
		t.Fatal("PostgreSQL accepted a negative XP balance")
	}

	var original xpmodel.Entry
	if err := conn.Where("source_key = ?", "pg-concurrent:0").Take(&original).Error; err != nil {
		t.Fatal(err)
	}
	const reversalWorkers = 8
	startReversal := make(chan struct{})
	var reversalWG sync.WaitGroup
	reversalResults := make(chan error, reversalWorkers)
	for i := 0; i < reversalWorkers; i++ {
		reversalWG.Add(1)
		go func(index int) {
			defer reversalWG.Done()
			<-startReversal
			_, err := svc.PostAdjustment(testActorContext(account.Id), AdjustmentRequest{
				UserID: account.Id, SourceKey: fmt.Sprintf("pg-reversal:%d", index), Delta: -1,
				Reason: "concurrent reversal", OccurredAt: time.Now().UTC(), ReversalOfEntryID: original.ID,
			})
			reversalResults <- err
		}(i)
	}
	close(startReversal)
	reversalWG.Wait()
	close(reversalResults)
	reversalSuccesses := 0
	for err := range reversalResults {
		if err == nil {
			reversalSuccesses++
		} else if !errors.Is(err, ErrReversalLimit) {
			t.Errorf("PostgreSQL concurrent reversal error = %v; want success or ErrReversalLimit", err)
		}
	}
	if reversalSuccesses != 1 {
		t.Fatalf("PostgreSQL concurrent reversals applied = %d; want exactly one", reversalSuccesses)
	}
	var reversalCount int64
	if err := conn.Model(&xpmodel.Entry{}).Where("reversal_of_entry_id = ?", original.ID).Count(&reversalCount).Error; err != nil || reversalCount != 1 {
		t.Fatalf("PostgreSQL reversal entries = %d, %v; want one", reversalCount, err)
	}
	var reversal xpmodel.Entry
	if err := conn.Where("reversal_of_entry_id = ?", original.ID).Take(&reversal).Error; err != nil {
		t.Fatal(err)
	}
	if reversal.ActorUserID != account.Id {
		t.Fatalf("PostgreSQL reversal actor = %d; want authenticated user %d", reversal.ActorUserID, account.Id)
	}

	rollbackAt = rollbackAt.Add(2 * time.Hour)
	if err := conn.Exec(`CREATE FUNCTION xp_reject_rule_insert_for_test() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN IF NEW.rule_key = 'reject' THEN RAISE EXCEPTION 'injected XP rule insert failure'; END IF; RETURN NEW; END; $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("CREATE TRIGGER xp_reject_rule_insert_for_test BEFORE INSERT ON xp_reward_rules FOR EACH ROW EXECUTE FUNCTION xp_reject_rule_insert_for_test()").Error; err != nil {
		t.Fatal(err)
	}
	failedVersion, err := svc.PublishRuleSet(testActorContext(account.Id), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: rollbackAt,
		Rules: []xpmodel.RewardRule{
			{RuleKey: "rollback-good", Enabled: true, RewardXP: 1},
			{RuleKey: "reject", Enabled: true, RewardXP: 1},
		},
	})
	if err == nil || failedVersion != 0 {
		t.Fatalf("PostgreSQL failed publication = version %d, %v; want zero version and error", failedVersion, err)
	}
	var partialRulesets, partialRules int64
	if err := conn.Model(&xpmodel.RuleSet{}).Where("version = ?", 4).Count(&partialRulesets).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&xpmodel.RewardRule{}).Where("rule_set_version = ?", 4).Count(&partialRules).Error; err != nil {
		t.Fatal(err)
	}
	if partialRulesets != 0 || partialRules != 0 {
		t.Fatalf("PostgreSQL failed publication left %d rulesets and %d rules", partialRulesets, partialRules)
	}
	if err := conn.Exec("DROP TRIGGER xp_reject_rule_insert_for_test ON xp_reward_rules").Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("DROP FUNCTION xp_reject_rule_insert_for_test()").Error; err != nil {
		t.Fatal(err)
	}
	version, err := svc.PublishRuleSet(testActorContext(account.Id), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: rollbackAt,
		Rules: []xpmodel.RewardRule{{RuleKey: "rollback-good", Enabled: true, RewardXP: 1}},
	})
	if err != nil || version != 4 {
		t.Fatalf("PostgreSQL publication after rollback = version %d, %v; want continuous version 4", version, err)
	}

	const publisherWorkers = 8
	publicationAt := rollbackAt.Add(time.Hour)
	startPublisher := make(chan struct{})
	var publisherWG sync.WaitGroup
	publisherResults := make(chan struct {
		version uint64
		err     error
	}, publisherWorkers)
	for i := 0; i < publisherWorkers; i++ {
		publisherWG.Add(1)
		go func(index int) {
			defer publisherWG.Done()
			<-startPublisher
			version, err := svc.PublishRuleSet(testActorContext(account.Id), RuleSetSpec{
				Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: publicationAt,
				Rules: []xpmodel.RewardRule{{RuleKey: fmt.Sprintf("pg-publication-%d", index), Enabled: true, RewardXP: 1}},
			})
			publisherResults <- struct {
				version uint64
				err     error
			}{version, err}
		}(i)
	}
	close(startPublisher)
	publisherWG.Wait()
	close(publisherResults)
	publicationSuccesses := 0
	for result := range publisherResults {
		if result.err == nil {
			publicationSuccesses++
			continue
		}
		if result.version != 0 || !errors.Is(result.err, ErrInvalidRule) {
			t.Errorf("PostgreSQL concurrent publisher returned version %d, error %v; want zero version and effective-time conflict", result.version, result.err)
		}
	}
	if publicationSuccesses != 1 {
		t.Fatalf("PostgreSQL concurrent publications committed = %d; want one", publicationSuccesses)
	}
	var rulesets []xpmodel.RuleSet
	if err := conn.Order("version").Find(&rulesets).Error; err != nil {
		t.Fatal(err)
	}
	for index, set := range rulesets {
		if set.Version != uint64(index+1) {
			t.Fatalf("PostgreSQL ruleset version at position %d = %d; want %d", index, set.Version, index+1)
		}
		if index == 0 {
			continue
		}
		var count int64
		if err := conn.Model(&xpmodel.RewardRule{}).Where("rule_set_version = ?", set.Version).Count(&count).Error; err != nil || count == 0 {
			t.Fatalf("PostgreSQL published rules version %d has %d rules, %v", set.Version, count, err)
		}
	}
}

func TestDailyBaseSettlementIsIdempotentOnPostgres(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL daily settlement test")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	schema := "xp_daily_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error
	})
	conn, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect isolated PostgreSQL schema: %v", err)
	}
	if sqlDB, err := conn.DB(); err == nil {
		sqlDB.SetMaxOpenConns(12)
	}
	models := []any{&users.EntityComplete{}}
	models = append(models, xpmodel.Models()...)
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate XP schema on PostgreSQL: %v", err)
	}
	if err := xpmodel.EnsureAppendOnlyGuards(conn); err != nil {
		t.Fatalf("install PostgreSQL XP guards: %v", err)
	}
	if err := EnsureInitialRules(context.Background(), conn); err != nil {
		t.Fatalf("seed initial XP rules on PostgreSQL: %v", err)
	}
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	if err := conn.Create(&users.EntityComplete{
		Username:  "xp-pg-daily-" + strconv.FormatInt(time.Now().UnixNano(), 10),
		ActorType: users.ActorTypeHuman, IsFrozen: users.StatusNormal, CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create PostgreSQL daily settlement user: %v", err)
	}
	var account users.EntityComplete
	if err := conn.Order("id DESC").Take(&account).Error; err != nil {
		t.Fatal(err)
	}
	svc := newWithPolicy(conn, pageConfig.SecurityAndRegistration{})
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	stats := make(chan DailySettlementStats, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			result, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
			stats <- result
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	close(stats)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent PostgreSQL daily settlement returned error: %v", err)
		}
	}
	var awarded, replayed int64
	for result := range stats {
		awarded += result.Awarded
		replayed += result.AlreadySettled
	}
	if awarded != 1 || replayed != 1 {
		t.Fatalf("PostgreSQL daily settlement totals awarded=%d already=%d; want one of each", awarded, replayed)
	}
	var count int64
	if err := conn.Model(&xpmodel.Entry{}).Where("user_id = ? AND source_key = ?", account.Id, dailyBaseSourceKey("2026-10-26", account.Id)).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("PostgreSQL daily entries = %d, %v; want one", count, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), account.Id); err != nil || got != 5 {
		t.Fatalf("PostgreSQL XP balance = %d, %v; want 5", got, err)
	}
}

func TestXPAuditColumnsUpgradeExistingPostgresTables(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set; skipping PostgreSQL XP audit upgrade test")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	schema := "xp_audit_upgrade_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error
	})
	conn, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("connect isolated PostgreSQL schema: %v", err)
	}
	if err := conn.AutoMigrate(&legacyXPEntry{}, &legacyXPRuleSet{}); err != nil {
		t.Fatalf("create pre-audit XP schema: %v", err)
	}
	at := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if err := conn.Create(&legacyXPRuleSet{
		Version: 1, Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: at,
	}).Error; err != nil {
		t.Fatalf("seed pre-audit ruleset: %v", err)
	}
	if err := conn.Create(&legacyXPEntry{
		UserID: 90, SourceKey: "legacy:xp", RuleSetVersion: 1, RuleKey: "legacy",
		Delta: 7, OccurredAt: at, CreatedAt: at,
	}).Error; err != nil {
		t.Fatalf("seed pre-audit entry: %v", err)
	}
	if err := conn.AutoMigrate(xpmodel.Models()...); err != nil {
		t.Fatalf("add XP audit fields to PostgreSQL schema: %v", err)
	}
	if err := xpmodel.EnsureAppendOnlyGuards(conn); err != nil {
		t.Fatalf("install PostgreSQL XP guards after additive migration: %v", err)
	}
	var entry xpmodel.Entry
	if err := conn.Where("source_key = ?", "legacy:xp").Take(&entry).Error; err != nil || entry.ActorUserID != 0 || entry.Delta != 7 {
		t.Fatalf("upgraded PostgreSQL entry = %+v, %v; want preserved history and system actor 0", entry, err)
	}
	var ruleset xpmodel.RuleSet
	if err := conn.Where("version = ?", 1).Take(&ruleset).Error; err != nil || ruleset.PublishedBy != 0 {
		t.Fatalf("upgraded PostgreSQL ruleset = %+v, %v; want preserved history and publisher 0", ruleset, err)
	}
}

type legacyXPEntry struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement;not null;column:id"`
	UserID          uint64    `gorm:"not null;column:user_id"`
	SourceKey       string    `gorm:"type:varchar(255);not null;column:source_key"`
	RuleSetVersion  uint64    `gorm:"not null;column:rule_set_version"`
	RuleKey         string    `gorm:"type:varchar(64);not null;column:rule_key"`
	Delta           int64     `gorm:"type:bigint;not null;column:delta"`
	OccurredAt      time.Time `gorm:"not null;column:occurred_at"`
	ReversalOfEntry *uint64   `gorm:"column:reversal_of_entry_id"`
	Reason          string    `gorm:"type:text;not null;default:'';column:reason"`
	CreatedAt       time.Time `gorm:"not null;column:created_at"`
}

func (legacyXPEntry) TableName() string { return "xp_entries" }

type legacyXPRuleSet struct {
	Version              uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:version"`
	Enabled              bool      `gorm:"not null;column:enabled"`
	TimeZone             string    `gorm:"type:varchar(64);not null;column:time_zone"`
	DailyPositiveXPLimit *int64    `gorm:"type:bigint;column:daily_positive_xp_limit"`
	EffectiveAt          time.Time `gorm:"not null;column:effective_at"`
	CreatedAt            time.Time `gorm:"not null;column:created_at"`
}

func (legacyXPRuleSet) TableName() string { return "xp_rule_sets" }
