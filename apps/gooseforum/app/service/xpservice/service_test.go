package xpservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestApplyRewardIsIdempotentAndDailyBaseDoesNotRequireActivity(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 7)
	if err := db.Create(&userPoints.Entity{UserId: 7, CurrentPoints: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pointsRecord.Entity{UserId: 7, Action: "legacy", PointsChange: 4}).Error; err != nil {
		t.Fatal(err)
	}
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	now := time.Date(2026, 1, 2, 3, 0, 0, 0, time.UTC)

	first, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 7, RuleKey: DailyBaseRuleKey, SourceKey: "daily:7:2026-01-02", OccurredAt: now})
	if err != nil {
		t.Fatalf("daily base award: %v", err)
	}
	if first.Entry.Delta != 5 || first.AlreadyApplied {
		t.Fatalf("first result = %+v, want new +5 entry", first)
	}
	replay, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 7, RuleKey: DailyBaseRuleKey, SourceKey: "daily:7:2026-01-02", OccurredAt: now.Add(time.Minute)})
	if err != nil || !replay.AlreadyApplied {
		t.Fatalf("replay = %+v, %v; want idempotent result", replay, err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 7, RuleKey: DailyBaseRuleKey, SourceKey: "daily:7:second", OccurredAt: now.Add(time.Hour)}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("second daily base award error = %v, want daily count limit", err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 7); err != nil || got != 5 {
		t.Fatalf("current balance = %d, %v; want 5", got, err)
	}
	var entries int64
	if err := db.Model(&xpmodel.Entry{}).Count(&entries).Error; err != nil || entries != 1 {
		t.Fatalf("ledger entries = %d, %v; want exactly one", entries, err)
	}
	var oldPoints userPoints.Entity
	if err := db.Where("user_id = ?", 7).Take(&oldPoints).Error; err != nil || oldPoints.CurrentPoints != 12 {
		t.Fatalf("legacy points balance = %+v, %v; XP must not change it", oldPoints, err)
	}
	var oldRecordCount int64
	if err := db.Model(&pointsRecord.Entity{}).Where("user_id = ?", 7).Count(&oldRecordCount).Error; err != nil || oldRecordCount != 1 {
		t.Fatalf("legacy point record count = %d, %v; want unchanged", oldRecordCount, err)
	}
	var account users.EntityComplete
	if err := db.Where("id = ?", 7).Take(&account).Error; err != nil || account.Prestige != 19 {
		t.Fatalf("legacy prestige = %d, %v; XP must not change it", account.Prestige, err)
	}
	var usage xpmodel.DailyUsage
	if err := db.Where("user_id = ?", 7).Take(&usage).Error; err != nil || usage.PositiveXP != 5 || usage.UsageDate != "2026-01-02" {
		t.Fatalf("daily usage = %+v, %v; want 5 XP for Shanghai date 2026-01-02", usage, err)
	}
}

func TestRewardRequiresCurrentEligibleHumanAccount(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 1)
	if err := db.Create(&users.EntityComplete{Id: 2, Username: "bot", ActorType: users.ActorTypeBot, IsActivated: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&users.EntityComplete{Id: 3, Username: "frozen", IsFrozen: 1, IsActivated: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&users.EntityComplete{Id: 4, Username: "unverified"}).Error; err != nil {
		t.Fatal(err)
	}
	verifiedOnly := pageConfig.SecurityAndRegistration{EnableEmailVerification: true}
	svc := newWithPolicy(db, verifiedOnly)
	for _, id := range []uint64{2, 3, 4} {
		_, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: id, RuleKey: DailyBaseRuleKey, SourceKey: fmt.Sprintf("award:%d", id), OccurredAt: time.Now()})
		if !errors.Is(err, ErrIneligible) {
			t.Errorf("award to ineligible account %d error = %v", id, err)
		}
	}
	withoutVerification := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	if _, err := withoutVerification.ApplyReward(context.Background(), AwardRequest{UserID: 1, RuleKey: DailyBaseRuleKey, SourceKey: "award:1", OccurredAt: time.Now()}); err != nil {
		t.Fatalf("human account should be eligible when verification is disabled at account level: %v", err)
	}
}

func TestConcurrentRewardsDoNotLoseBalanceOrDuplicateSource(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 9)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	if _, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: time.Now().UTC(),
		Rules: []xpmodel.RewardRule{{RuleKey: "concurrent", Enabled: true, RewardXP: 1}},
	}); err != nil {
		t.Fatalf("publish concurrent rule: %v", err)
	}

	const workers = 24
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			_, err := svc.ApplyReward(context.Background(), AwardRequest{
				UserID: 9, RuleKey: "concurrent", SourceKey: fmt.Sprintf("concurrent:%d", index), OccurredAt: time.Now().UTC(),
			})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent reward failed: %v", err)
		}
	}
	if got, err := svc.CurrentBalance(context.Background(), 9); err != nil || got != workers {
		t.Fatalf("concurrent balance = %d, %v; want %d", got, err, workers)
	}

	var replayWG sync.WaitGroup
	results := make(chan AwardResult, workers)
	errResults := make(chan error, workers)
	for i := 0; i < workers; i++ {
		replayWG.Add(1)
		go func() {
			defer replayWG.Done()
			result, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 9, RuleKey: "concurrent", SourceKey: "one-shared-source", OccurredAt: time.Now().UTC()})
			results <- result
			errResults <- err
		}()
	}
	replayWG.Wait()
	close(results)
	close(errResults)
	created := 0
	for result := range results {
		if !result.AlreadyApplied {
			created++
		}
	}
	for err := range errResults {
		if err != nil {
			t.Errorf("shared source replay failed: %v", err)
		}
	}
	if created != 1 {
		t.Fatalf("new shared-source entries = %d, want 1", created)
	}
	if got, _ := svc.CurrentBalance(context.Background(), 9); got != workers+1 {
		t.Fatalf("shared-source retries changed balance to %d, want %d", got, workers+1)
	}
}

func TestNegativeReversalIsAppendOnlyAndDoesNotRefundQuota(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 11)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	awarded, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 11, RuleKey: DailyBaseRuleKey, SourceKey: "daily:11", OccurredAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	updatedReward := int64(99)
	if _, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: awarded.Entry.OccurredAt.Add(time.Second),
		Rules: []xpmodel.RewardRule{{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: updatedReward}},
	}); err != nil {
		t.Fatalf("publish updated reward rule: %v", err)
	}
	// A valid reversal must still use the original entry, even after the
	// effective reward amount changes in a newer ruleset.
	if err := db.Model(&xpmodel.Balance{}).Where("user_id = ?", 11).Update("current_xp", 0).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PostAdjustment(testActorContext(7), AdjustmentRequest{UserID: 11, SourceKey: "negative-balance", Delta: -1, Reason: "projection guard", ReversalOfEntryID: awarded.Entry.ID}); !errors.Is(err, ErrNegativeBalance) {
		t.Fatalf("negative balance error = %v, want ErrNegativeBalance", err)
	}
	if err := db.Model(&xpmodel.Balance{}).Where("user_id = ?", 11).Update("current_xp", awarded.Entry.Delta).Error; err != nil {
		t.Fatal(err)
	}
	three := int64(3)
	firstReversal, err := svc.PostAdjustment(testActorContext(7), AdjustmentRequest{UserID: 11, SourceKey: "reverse:11:1", Delta: -three, Reason: "partial correction", ReversalOfEntryID: awarded.Entry.ID})
	if err != nil || firstReversal.Entry.Delta != -3 || firstReversal.Entry.RuleSetVersion != awarded.Entry.RuleSetVersion {
		t.Fatalf("first reversal = %+v, %v", firstReversal, err)
	}
	replayedReversal, err := svc.PostAdjustment(testActorContext(7), AdjustmentRequest{UserID: 11, SourceKey: "reverse:11:1", Delta: -three, Reason: "partial correction", ReversalOfEntryID: awarded.Entry.ID})
	if err != nil || !replayedReversal.AlreadyApplied {
		t.Fatalf("replayed reversal = %+v, %v; want idempotent result", replayedReversal, err)
	}
	secondReversal, err := svc.PostAdjustment(testActorContext(7), AdjustmentRequest{UserID: 11, SourceKey: "reverse:11:2", Delta: -2, Reason: "remaining correction", ReversalOfEntryID: awarded.Entry.ID})
	if err != nil || secondReversal.Entry.Delta != -2 {
		t.Fatalf("second reversal = %+v, %v", secondReversal, err)
	}
	if _, err := svc.PostAdjustment(testActorContext(7), AdjustmentRequest{UserID: 11, SourceKey: "reverse:11:3", Delta: -1, Reason: "over correction", ReversalOfEntryID: awarded.Entry.ID}); !errors.Is(err, ErrReversalLimit) {
		t.Fatalf("over reversal error = %v, want ErrReversalLimit", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 11, RuleKey: DailyBaseRuleKey, SourceKey: "daily:11:again", OccurredAt: time.Now().UTC()}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("reward after reversal error = %v, want quota to remain consumed", err)
	}
	if got, _ := svc.CurrentBalance(context.Background(), 11); got != 0 {
		t.Fatalf("balance after full reversal = %d, want 0", got)
	}
	var entries []xpmodel.Entry
	if err := db.Order("id").Find(&entries).Error; err != nil || len(entries) != 3 {
		t.Fatalf("ledger after reversals = %d entries, %v; want original plus two reversals", len(entries), err)
	}
	if err := db.Model(&xpmodel.Entry{}).Where("id = ?", awarded.Entry.ID).Update("reason", "rewritten").Error; !errors.Is(err, xpmodel.ErrImmutableRecord) {
		t.Fatalf("ledger update error = %v, want immutable record error", err)
	}
	if err := db.Delete(&xpmodel.Entry{}, awarded.Entry.ID).Error; !errors.Is(err, xpmodel.ErrImmutableRecord) {
		t.Fatalf("ledger delete error = %v, want immutable record error", err)
	}
	if err := db.Exec("UPDATE xp_entries SET reason = ? WHERE id = ?", "bypass ORM hooks", awarded.Entry.ID).Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct ledger update")
	}
	if err := db.Exec("DELETE FROM xp_entries WHERE id = ?", awarded.Entry.ID).Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct ledger delete")
	}
}

func TestNegativeAdjustmentRequiresOriginalAward(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 12)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	awarded, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 12, RuleKey: DailyBaseRuleKey, SourceKey: "daily:12", OccurredAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}

	_, err = svc.PostAdjustment(testActorContext(7), AdjustmentRequest{
		UserID: 12, SourceKey: "unlinked-reversal", Delta: -1, Reason: "correction without source",
	})
	if !errors.Is(err, ErrInvalidAdjustment) {
		t.Fatalf("unlinked negative adjustment error = %v, want ErrInvalidAdjustment", err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 12); err != nil || got != awarded.Entry.Delta {
		t.Fatalf("balance after rejected unlinked reversal = %d, %v; want %d", got, err, awarded.Entry.Delta)
	}
	var entries int64
	if err := db.Model(&xpmodel.Entry{}).Count(&entries).Error; err != nil || entries != 1 {
		t.Fatalf("ledger rows after rejected unlinked reversal = %d, %v; want only original award", entries, err)
	}

	missingID := uint64(999_999)
	_, err = svc.PostAdjustment(testActorContext(7), AdjustmentRequest{
		UserID: 12, SourceKey: "missing-original", Delta: -1, Reason: "missing source",
		ReversalOfEntryID: missingID,
	})
	if err == nil || errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("missing original reversal error = %v; want a stable domain error", err)
	}
}

func TestAdjustmentCannotReverseAnotherAccountsAward(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 20)
	createXPUser(t, db, 21)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	awarded, err := svc.ApplyReward(context.Background(), AwardRequest{
		UserID: 20, RuleKey: DailyBaseRuleKey, SourceKey: "daily:20", OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.PostAdjustment(testActorContext(7), AdjustmentRequest{
		UserID: 21, SourceKey: "cross-account-reversal", Delta: -1, Reason: "wrong account",
		ReversalOfEntryID: awarded.Entry.ID,
	})
	if !errors.Is(err, ErrReversalLimit) {
		t.Fatalf("cross-account reversal error = %v; want ErrReversalLimit", err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 20); err != nil || got != awarded.Entry.Delta {
		t.Fatalf("source account balance after rejected reversal = %d, %v; want %d", got, err, awarded.Entry.Delta)
	}
	if got, err := svc.CurrentBalance(context.Background(), 21); err != nil || got != 0 {
		t.Fatalf("requesting account balance after rejected reversal = %d, %v; want 0", got, err)
	}
	var rejectedEntries int64
	if err := db.Model(&xpmodel.Entry{}).Where("source_key = ?", "cross-account-reversal").Count(&rejectedEntries).Error; err != nil || rejectedEntries != 0 {
		t.Fatalf("cross-account reversal rows = %d, %v; want none", rejectedEntries, err)
	}
}

func TestPublishRuleSetTrimsKeysBeforeDuplicateValidation(t *testing.T) {
	db := newXPTestDB(t)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	version, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		Rules: []xpmodel.RewardRule{
			{RuleKey: "bonus", Enabled: true, RewardXP: 1},
			{RuleKey: " bonus ", Enabled: true, RewardXP: 2},
		},
	})
	if !errors.Is(err, ErrInvalidRule) || version != 0 {
		t.Fatalf("trimmed duplicate publication = version %d, %v; want zero version and ErrInvalidRule", version, err)
	}
	var versions int64
	if err := db.Model(&xpmodel.RuleSet{}).Where("version > 1").Count(&versions).Error; err != nil || versions != 0 {
		t.Fatalf("versions created by rejected duplicate rules = %d, %v; want none", versions, err)
	}
}

func TestPublishRuleSetFailureRollsBackAndReturnsZeroVersion(t *testing.T) {
	db := newXPTestDB(t)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	if err := db.Exec(`CREATE TRIGGER reject_xp_rule BEFORE INSERT ON xp_reward_rules
		WHEN NEW.rule_key = 'reject'
		BEGIN SELECT RAISE(ABORT, 'injected rule insert failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	effectiveAt := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	version, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: effectiveAt,
		Rules: []xpmodel.RewardRule{
			{RuleKey: "good", Enabled: true, RewardXP: 1},
			{RuleKey: "reject", Enabled: true, RewardXP: 1},
		},
	})
	if err == nil || version != 0 {
		t.Fatalf("failed publication = version %d, %v; want zero version and error", version, err)
	}
	var partialSets, partialRules int64
	if err := db.Model(&xpmodel.RuleSet{}).Where("version = ?", 2).Count(&partialSets).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&xpmodel.RewardRule{}).Where("rule_set_version = ?", 2).Count(&partialRules).Error; err != nil {
		t.Fatal(err)
	}
	if partialSets != 0 || partialRules != 0 {
		t.Fatalf("failed publication left %d rulesets and %d rules", partialSets, partialRules)
	}
	if err := db.Exec("DROP TRIGGER reject_xp_rule").Error; err != nil {
		t.Fatal(err)
	}
	version, err = svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: effectiveAt,
		Rules: []xpmodel.RewardRule{{RuleKey: "good", Enabled: true, RewardXP: 1}},
	})
	if err != nil || version != 2 {
		t.Fatalf("publication after rollback = version %d, %v; want continuous version 2", version, err)
	}
}

func TestConcurrentSQLiteRuleSetPublishingIsSerialized(t *testing.T) {
	db := newXPTestDB(t)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	const workers = 8
	baseTime := time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC)
	start := make(chan struct{})
	type publishResult struct {
		version uint64
		err     error
	}
	results := make(chan publishResult, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			version, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
				Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: baseTime.Add(time.Duration(index+1) * time.Second),
				Rules: []xpmodel.RewardRule{{RuleKey: fmt.Sprintf("concurrent-%d", index), Enabled: true, RewardXP: 1}},
			})
			results <- publishResult{version: version, err: err}
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	var successes int
	for result := range results {
		if result.err == nil {
			successes++
			continue
		}
		if result.version != 0 {
			t.Errorf("failed publisher returned version %d with %v", result.version, result.err)
		}
		if !errors.Is(result.err, ErrInvalidRule) {
			t.Errorf("SQLite concurrent publication error = %v; want ordered conflict or success", result.err)
		}
	}
	if successes == 0 {
		t.Fatal("no SQLite concurrent publication succeeded")
	}
	var sets []xpmodel.RuleSet
	if err := db.Order("version").Find(&sets).Error; err != nil {
		t.Fatal(err)
	}
	if len(sets) != successes+1 {
		t.Fatalf("SQLite committed rulesets = %d; initial plus %d successful publications expected", len(sets), successes)
	}
	for index, set := range sets {
		if set.Version != uint64(index+1) {
			t.Fatalf("SQLite ruleset version at position %d = %d; want continuous version %d", index, set.Version, index+1)
		}
		if index == 0 {
			continue
		}
		var rules int64
		if err := db.Model(&xpmodel.RewardRule{}).Where("rule_set_version = ?", set.Version).Count(&rules).Error; err != nil || rules != 1 {
			t.Fatalf("SQLite rules in version %d = %d, %v; want complete one-rule set", set.Version, rules, err)
		}
	}
	version, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: baseTime.Add(time.Hour),
		Rules: []xpmodel.RewardRule{{RuleKey: "after-concurrency", Enabled: true, RewardXP: 1}},
	})
	if err != nil || version != uint64(len(sets)+1) {
		t.Fatalf("SQLite next version after concurrency = %d, %v; want %d", version, err, len(sets)+1)
	}
}

func TestEffectiveAtUniqueConstraintConflictIsClassified(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "PostgreSQL named unique index",
			err:  errors.New(`duplicated key not allowed: ERROR: duplicate key value violates unique constraint "idx_xp_rule_sets_effective_at" (SQLSTATE 23505)`),
			want: true,
		},
		{
			name: "SQLite effective-at column",
			err:  errors.New("UNIQUE constraint failed: xp_rule_sets.effective_at"),
			want: true,
		},
		{
			name: "other unique constraint",
			err:  errors.New(`duplicated key not allowed: ERROR: duplicate key value violates unique constraint "xp_rule_sets_pkey" (SQLSTATE 23505)`),
			want: false,
		},
		{name: "unrelated database error", err: errors.New("connection reset"), want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isEffectiveAtUniqueConflict(test.err); got != test.want {
				t.Fatalf("isEffectiveAtUniqueConflict(%v) = %t; want %t", test.err, got, test.want)
			}
		})
	}
}

func TestXPDatabaseSchemaContainsAuditSubjects(t *testing.T) {
	db := newXPTestDB(t)
	for _, field := range []struct {
		model any
		name  string
	}{
		{model: &xpmodel.Entry{}, name: "actor_user_id"},
		{model: &xpmodel.RuleSet{}, name: "published_by"},
	} {
		if !db.Migrator().HasColumn(field.model, field.name) {
			t.Errorf("XP schema is missing audit column %s", field.name)
		}
	}
}

func TestSQLiteUpgradeAddsAuditColumnsWithoutLosingXPHistory(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 88)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	award, err := svc.ApplyReward(context.Background(), AwardRequest{
		UserID: 88, RuleKey: DailyBaseRuleKey, SourceKey: "upgrade:audit", OccurredAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropColumn(&xpmodel.Entry{}, "actor_user_id"); err != nil {
		t.Fatalf("simulate existing XP entry schema: %v", err)
	}
	if err := db.Migrator().DropColumn(&xpmodel.RuleSet{}, "published_by"); err != nil {
		t.Fatalf("simulate existing XP ruleset schema: %v", err)
	}
	if err := db.AutoMigrate(&xpmodel.Entry{}, &xpmodel.RuleSet{}); err != nil {
		t.Fatalf("add XP audit columns: %v", err)
	}
	if err := xpmodel.EnsureAppendOnlyGuards(db); err != nil {
		t.Fatalf("reinstall SQLite XP guards after additive migration: %v", err)
	}
	var entry xpmodel.Entry
	if err := db.Where("id = ?", award.Entry.ID).Take(&entry).Error; err != nil || entry.ActorUserID != 0 {
		t.Fatalf("upgraded XP entry = %+v, %v; want preserved row with system actor 0", entry, err)
	}
	var ruleset xpmodel.RuleSet
	if err := db.Where("version = ?", 1).Take(&ruleset).Error; err != nil || ruleset.PublishedBy != 0 {
		t.Fatalf("upgraded XP ruleset = %+v, %v; want preserved row with system publisher 0", ruleset, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 88); err != nil || got != award.Entry.Delta {
		t.Fatalf("balance after additive audit migration = %d, %v; want %d", got, err, award.Entry.Delta)
	}
}

func TestOperationAuditActorsAreDerivedFromTrustedContext(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 89)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	occurredAt := time.Date(2026, 4, 4, 8, 0, 0, 0, time.UTC)
	award, err := svc.ApplyReward(context.Background(), AwardRequest{
		UserID: 89, RuleKey: DailyBaseRuleKey, SourceKey: "audit:award", OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if award.Entry.ActorUserID != 0 {
		t.Fatalf("automated reward actor = %d; want system actor 0", award.Entry.ActorUserID)
	}
	if _, err := svc.PublishRuleSet(context.Background(), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: occurredAt.Add(time.Hour),
		Rules: []xpmodel.RewardRule{{RuleKey: "audit-rule", Enabled: true, RewardXP: 1}},
	}); !errors.Is(err, ErrAuditActorRequired) {
		t.Fatalf("rule publication without authenticated actor = %v; want ErrAuditActorRequired", err)
	}
	version, err := svc.PublishRuleSet(testActorContext(89), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: occurredAt.Add(time.Hour),
		Rules: []xpmodel.RewardRule{{RuleKey: "audit-rule", Enabled: true, RewardXP: 1}},
	})
	if err != nil || version != 2 {
		t.Fatalf("publish audited rule version = %d, %v; want version 2", version, err)
	}
	var published xpmodel.RuleSet
	if err := db.Where("version = ?", version).Take(&published).Error; err != nil || published.PublishedBy != 89 {
		t.Fatalf("published_by = %d, %v; want authenticated user 89", published.PublishedBy, err)
	}

	request := AdjustmentRequest{
		UserID: 89, SourceKey: "audit:manual-reversal", Delta: -1, Reason: "manual correction",
		OccurredAt: occurredAt.Add(time.Hour), ReversalOfEntryID: award.Entry.ID,
	}
	if _, err := svc.PostAdjustment(context.Background(), request); !errors.Is(err, ErrAuditActorRequired) {
		t.Fatalf("manual reversal without authenticated actor = %v; want ErrAuditActorRequired", err)
	}
	manual, err := svc.PostAdjustment(testActorContext(89), request)
	if err != nil || manual.Entry.ActorUserID != 89 {
		t.Fatalf("manual reversal actor = %+v, %v; want actor 89", manual.Entry, err)
	}
	system, err := svc.PostSystemAdjustment(context.Background(), AdjustmentRequest{
		UserID: 89, SourceKey: "audit:system-reversal", Delta: -1, Reason: "automatic correction",
		OccurredAt: occurredAt.Add(2 * time.Hour), ReversalOfEntryID: award.Entry.ID,
	})
	if err != nil || system.Entry.ActorUserID != 0 {
		t.Fatalf("system reversal actor = %+v, %v; want actor 0", system.Entry, err)
	}
}

func TestRewardErrorTerminalContract(t *testing.T) {
	for _, terminal := range []error{
		ErrIneligible, ErrRuleUnavailable, ErrRewardLimit, ErrInvalidRule,
		ErrInvalidAdjustment, ErrSourceKeyConflict, ErrNegativeBalance,
		ErrReversalLimit, ErrOriginalAwardNotFound, ErrAuditActorRequired,
	} {
		if !IsTerminal(fmt.Errorf("settlement rejected: %w", terminal)) {
			t.Errorf("IsTerminal(%v) = false; want true", terminal)
		}
	}
	if IsTerminal(errors.New("database connection reset")) || IsTerminal(gorm.ErrRecordNotFound) {
		t.Fatal("unclassified infrastructure errors must remain retryable")
	}
}

func TestDailyQuotaUsesShanghaiNaturalDayFromUTCEventTime(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 90)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	beforeBoundary := time.Date(2026, 5, 1, 15, 59, 59, 0, time.UTC) // Shanghai 23:59:59.
	afterBoundary := time.Date(2026, 5, 1, 16, 0, 0, 0, time.UTC)    // Shanghai 00:00:00 on May 2.
	for _, event := range []struct {
		source string
		at     time.Time
	}{{"utc-before-boundary", beforeBoundary}, {"utc-after-boundary", afterBoundary}} {
		if _, err := svc.ApplyReward(context.Background(), AwardRequest{
			UserID: 90, RuleKey: DailyBaseRuleKey, SourceKey: event.source, OccurredAt: event.at,
		}); err != nil {
			t.Fatalf("apply daily reward for event %s: %v", event.source, err)
		}
	}
	var usage []xpmodel.DailyUsage
	if err := db.Where("user_id = ?", 90).Order("usage_date").Find(&usage).Error; err != nil {
		t.Fatal(err)
	}
	if len(usage) != 2 || usage[0].UsageDate != "2026-05-01" || usage[1].UsageDate != "2026-05-02" {
		t.Fatalf("daily usage dates = %+v; want Shanghai dates May 1 and May 2", usage)
	}
}

func TestRuleVersionsEnablementLimitsAndShanghaiDay(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 13)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	start := time.Date(2026, 3, 1, 15, 59, 0, 0, time.UTC) // Shanghai 23:59
	cap := int64(8)
	version, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: start.Add(time.Minute), DailyPositiveXPLimit: &cap,
		Rules: []xpmodel.RewardRule{
			{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: 5, DailyCountLimit: ptrInt64(1)},
			{RuleKey: "bonus", Enabled: true, RewardXP: 6, MaxPerEventXP: ptrInt64(3)},
		},
	})
	if err != nil || version != 2 {
		t.Fatalf("published ruleset version = %d, %v; want version 2", version, err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: DailyBaseRuleKey, SourceKey: "boundary:base", OccurredAt: start.Add(2 * time.Minute)}); err != nil {
		t.Fatalf("base reward: %v", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: "bonus", SourceKey: "boundary:bonus", OccurredAt: start.Add(2 * time.Minute)}); err != nil {
		t.Fatalf("clamped bonus reward: %v", err)
	}
	if got, _ := svc.CurrentBalance(context.Background(), 13); got != 8 {
		t.Fatalf("balance after per-event cap = %d, want 8", got)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: "bonus", SourceKey: "boundary:over-cap", OccurredAt: start.Add(3 * time.Minute)}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("daily aggregate cap error = %v, want ErrRewardLimit", err)
	}
	var rolledBack int64
	if err := db.Model(&xpmodel.Entry{}).Where("source_key = ?", "boundary:over-cap").Count(&rolledBack).Error; err != nil || rolledBack != 0 {
		t.Fatalf("over-cap ledger rows = %d, %v; rejected award must roll back", rolledBack, err)
	}
	// Both rejected awards roll their ledger inserts back; the same Shanghai
	// calendar day remains capped even when UTC advances past 16:00.
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: DailyBaseRuleKey, SourceKey: "boundary:same-day-again", OccurredAt: start.Add(3 * time.Minute)}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("same-day second base award error = %v, want limit", err)
	}
	nextDay := time.Date(2026, 3, 2, 16, 1, 0, 0, time.UTC)
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: DailyBaseRuleKey, SourceKey: "boundary:next-shanghai-day", OccurredAt: nextDay}); err != nil {
		t.Fatalf("next Shanghai day award: %v", err)
	}

	disabledAt := nextDay.Add(time.Hour)
	if _, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{Enabled: false, TimeZone: ShanghaiTimeZone, EffectiveAt: disabledAt, Rules: []xpmodel.RewardRule{{RuleKey: "bonus", Enabled: true, RewardXP: 1}}}); err != nil {
		t.Fatalf("publish disabled version: %v", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: "bonus", SourceKey: "disabled:bonus", OccurredAt: disabledAt.Add(time.Second)}); !errors.Is(err, ErrRuleUnavailable) {
		t.Fatalf("disabled ruleset error = %v, want ErrRuleUnavailable", err)
	}
	if err := db.Model(&xpmodel.RuleSet{}).Where("version = ?", version).Update("enabled", false).Error; !errors.Is(err, xpmodel.ErrImmutableRecord) {
		t.Fatalf("published ruleset update error = %v, want immutable record error", err)
	}
	if _, err := svc.PublishRuleSet(testActorContext(7), RuleSetSpec{Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: disabledAt.Add(2 * time.Second), Rules: []xpmodel.RewardRule{{RuleKey: "bonus", Enabled: false, RewardXP: 1}}}); err != nil {
		t.Fatalf("publish version with disabled reward rule: %v", err)
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 13, RuleKey: "bonus", SourceKey: "disabled:rule", OccurredAt: disabledAt.Add(3 * time.Second)}); !errors.Is(err, ErrRuleUnavailable) {
		t.Fatalf("disabled reward rule error = %v, want ErrRuleUnavailable", err)
	}
	if err := db.Exec("UPDATE xp_reward_rules SET enabled = TRUE WHERE rule_set_version = ? AND rule_key = ?", version, "bonus").Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct reward-rule update")
	}
	if err := db.Exec("DELETE FROM xp_reward_rules WHERE rule_set_version = ? AND rule_key = ?", version, "bonus").Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct reward-rule delete")
	}
	if err := db.Exec("UPDATE xp_rule_sets SET enabled = FALSE WHERE version = ?", version).Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct ruleset update")
	}
	if err := db.Exec("DELETE FROM xp_rule_sets WHERE version = ?", version).Error; err == nil {
		t.Fatal("SQLite append-only trigger accepted a direct ruleset delete")
	}
}

func TestEligibleForXPCriteriaAndDynamicLevel(t *testing.T) {
	verified := pageConfig.SecurityAndRegistration{EnableEmailVerification: true}
	base := users.EntityComplete{Id: 1, ActorType: users.ActorTypeHuman, IsFrozen: users.StatusNormal, IsActivated: 1}
	if !EligibleForXP(base, verified) {
		t.Fatal("activated human account should be eligible")
	}
	if EligibleForXP(users.EntityComplete{Id: 2, ActorType: users.ActorTypeHuman}, verified) {
		t.Fatal("unactivated account should be ineligible when email verification is enabled")
	}
	bot := base
	bot.ActorType = users.ActorTypeBot
	if EligibleForXP(bot, verified) {
		t.Fatal("bot account should be ineligible")
	}
	frozen := base
	frozen.IsFrozen = 1
	if EligibleForXP(frozen, verified) {
		t.Fatal("frozen account should be ineligible")
	}
	deleted := base
	deleted.DeletedAt = gorm.DeletedAt{Valid: true, Time: time.Now()}
	if EligibleForXP(deleted, verified) {
		t.Fatal("deleted account should be ineligible")
	}
	if got := LevelForXP(30_000, false); got != LevelSeedling {
		t.Fatalf("ineligible account level = %v, want Lv0", got)
	}
}

func TestPublishedRulesAndBalancesStartAtZeroAndAreAppendOnly(t *testing.T) {
	db := newXPTestDB(t)
	createXPUser(t, db, 17)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	if got, err := svc.CurrentBalance(context.Background(), 17); err != nil || got != 0 {
		t.Fatalf("initial XP = %d, %v; want zero", got, err)
	}
	if got, err := svc.CurrentLevel(context.Background(), 17); err != nil || got != LevelGreenTea {
		t.Fatalf("initial level = %v, %v; want Lv1", got, err)
	}
	if err := db.Create(&xpmodel.Balance{UserID: 17}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&xpmodel.Balance{}).Where("user_id = ?", 17).Update("current_xp", -1).Error; err == nil {
		t.Fatal("database accepted a negative XP balance")
	}
	var rule xpmodel.RewardRule
	if err := db.Where("rule_set_version = ? AND rule_key = ?", 1, DailyBaseRuleKey).Take(&rule).Error; err != nil {
		t.Fatal(err)
	}
	if rule.RewardXP != 5 || rule.DailyCountLimit == nil || *rule.DailyCountLimit != 1 {
		t.Fatalf("seeded daily rule = %+v, want adjustable confirmed +5 and one/day", rule)
	}
}

func newXPTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:xpservice-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(12)
	}
	models := []any{&users.EntityComplete{}, &userPoints.Entity{}, &pointsRecord.Entity{}}
	models = append(models, xpmodel.Models()...)
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate XP test models: %v", err)
	}
	if err := xpmodel.EnsureAppendOnlyGuards(db); err != nil {
		t.Fatalf("install SQLite XP guards: %v", err)
	}
	if err := EnsureInitialRules(context.Background(), db); err != nil {
		t.Fatalf("seed initial XP rules: %v", err)
	}
	return db
}

func createXPUser(t *testing.T, db *gorm.DB, id uint64) {
	t.Helper()
	if err := db.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("xp-user-%d", id), ActorType: users.ActorTypeHuman, IsFrozen: users.StatusNormal, Prestige: 19}).Error; err != nil {
		t.Fatalf("create XP user %d: %v", id, err)
	}
}

func ptrInt64(value int64) *int64 { return &value }

func testActorContext(userID uint64) context.Context {
	return WithAuthenticatedActor(context.Background(), userID)
}
