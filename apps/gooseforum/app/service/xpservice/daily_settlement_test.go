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
	"gorm.io/gorm"
)

func TestDailyBaseSettlementAwardsFiveWithoutActivityAndLeavesLegacyPointsUntouched(t *testing.T) {
	db := newXPTestDB(t)
	date := "2026-10-26"
	createdAt := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 120, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	if err := db.Create(&userPoints.Entity{UserId: 120, CurrentPoints: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&pointsRecord.Entity{UserId: 120, Action: "legacy", PointsChange: 4}).Error; err != nil {
		t.Fatal(err)
	}
	// The test schema has no activity rows or activity-dependent prerequisite.
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	now := time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC)
	stats, err := svc.settleDailyBase(context.Background(), date, false, 1, now)
	if err != nil {
		t.Fatalf("settle daily base: %v", err)
	}
	if stats.Scanned != 1 || stats.Eligible != 1 || stats.Candidates != 1 || stats.Awarded != 1 || stats.AlreadySettled != 0 {
		t.Fatalf("settlement stats = %+v", stats)
	}

	var entry xpmodel.Entry
	if err := db.Where("user_id = ?", 120).Take(&entry).Error; err != nil {
		t.Fatal(err)
	}
	wantOccurredAt := time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC)
	if entry.Delta != 5 || entry.RuleKey != DailyBaseRuleKey || entry.SourceKey != "xp:daily_base:2026-10-26:user:120" || !entry.OccurredAt.Equal(wantOccurredAt) {
		t.Fatalf("daily entry = %+v; want +5 with stable source key and Shanghai day start", entry)
	}
	if got, err := svc.CurrentBalance(context.Background(), 120); err != nil || got != 5 {
		t.Fatalf("XP balance = %d, %v; want 5", got, err)
	}
	var usage xpmodel.DailyUsage
	if err := db.Where("user_id = ?", 120).Take(&usage).Error; err != nil || usage.UsageDate != date || usage.PositiveXP != 5 {
		t.Fatalf("XP usage = %+v, %v; want +5 for %s", usage, err, date)
	}
	var points userPoints.Entity
	if err := db.Where("user_id = ?", 120).Take(&points).Error; err != nil || points.CurrentPoints != 12 {
		t.Fatalf("legacy points = %+v, %v; want unchanged 12", points, err)
	}
	var recordCount int64
	if err := db.Model(&pointsRecord.Entity{}).Where("user_id = ?", 120).Count(&recordCount).Error; err != nil || recordCount != 1 {
		t.Fatalf("legacy points records = %d, %v; want unchanged one row", recordCount, err)
	}
	var account users.EntityComplete
	if err := db.Where("id = ?", 120).Take(&account).Error; err != nil || account.Prestige != 19 {
		t.Fatalf("legacy prestige = %d, %v; want unchanged 19", account.Prestige, err)
	}
	var seeded xpmodel.RuleSet
	if err := db.Where("version = ?", 1).Take(&seeded).Error; err != nil {
		t.Fatal(err)
	}
	if seeded.DailyPositiveXPLimit != nil {
		t.Fatalf("seeded XP cap = %v; the approved 65 XP cap is not yet active", seeded.DailyPositiveXPLimit)
	}
}

func TestDailyBaseSettlementIsIdempotentAcrossRunsAndDates(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 121, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})

	first, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
	if err != nil || first.Awarded != 1 {
		t.Fatalf("first settlement = %+v, %v", first, err)
	}
	replay, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, time.Date(2026, 10, 26, 2, 0, 0, 0, time.UTC))
	if err != nil || replay.Awarded != 0 || replay.AlreadySettled != 1 {
		t.Fatalf("same-day replay = %+v, %v; want already-settled", replay, err)
	}
	nextDay, err := svc.settleDailyBase(context.Background(), "2026-10-27", false, 1, time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC))
	if err != nil || nextDay.Awarded != 1 {
		t.Fatalf("next-day settlement = %+v, %v; want an independent +5", nextDay, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 121); err != nil || got != 10 {
		t.Fatalf("balance after two dates = %d, %v; want 10", got, err)
	}
	var rows int64
	if err := db.Model(&xpmodel.Entry{}).Where("user_id = ?", 121).Count(&rows).Error; err != nil || rows != 2 {
		t.Fatalf("ledger rows = %d, %v; want one per date", rows, err)
	}
}

func TestDailyBaseSettlementUsesShanghaiDateAndNeverPaysBeforeRegistration(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 26, 15, 59, 59, 0, time.UTC) // 23:59:59 Shanghai.
	createSettlementUser(t, db, 122, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	now := time.Date(2026, 10, 26, 16, 5, 0, 0, time.UTC) // 00:05 on October 27 in Shanghai.

	prior, err := svc.settleDailyBase(context.Background(), "2026-10-25", false, 1, now)
	if err != nil || prior.Scanned != 0 || prior.Awarded != 0 {
		t.Fatalf("pre-registration date settlement = %+v, %v; want no scanned account or award", prior, err)
	}
	today, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, now)
	if err != nil || today.Awarded != 1 {
		t.Fatalf("registration-date settlement = %+v, %v; want one award", today, err)
	}
	var entry xpmodel.Entry
	if err := db.Where("user_id = ?", 122).Take(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if !entry.OccurredAt.Equal(createdAt) || entry.OccurredAt.In(mustShanghai(t)).Format(DailySettlementDateLayout) != "2026-10-26" {
		t.Fatalf("registration-day OccurredAt = %s; want account creation time on Shanghai date 2026-10-26", entry.OccurredAt)
	}
}

func TestDailyBaseSettlementUsesEffectiveRuleTimeForEligibleAccount(t *testing.T) {
	db := newXPTestDB(t)
	date := "2026-11-26"
	createdAt := time.Date(2026, 11, 25, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 123, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	dayStart, err := ParseDailySettlementDate(date)
	if err != nil {
		t.Fatal(err)
	}
	effectiveAt := dayStart.Add(12 * time.Hour)
	five := int64(5)
	one := int64(1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	if _, err := svc.PublishRuleSet(testActorContext(10), RuleSetSpec{
		Enabled: false, TimeZone: ShanghaiTimeZone, EffectiveAt: dayStart,
		Rules: []xpmodel.RewardRule{{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: five, MaxPerEventXP: &five, DailyCountLimit: &one}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PublishRuleSet(testActorContext(10), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: effectiveAt,
		Rules: []xpmodel.RewardRule{{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: five, MaxPerEventXP: &five, DailyCountLimit: &one}},
	}); err != nil {
		t.Fatal(err)
	}
	stats, err := svc.settleDailyBase(context.Background(), date, false, 1, time.Date(2026, 11, 27, 1, 0, 0, 0, time.UTC))
	if err != nil || stats.Awarded != 1 {
		t.Fatalf("settlement = %+v, %v; want one award", stats, err)
	}
	var entry xpmodel.Entry
	if err := db.Where("user_id = ?", 123).Take(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if !entry.OccurredAt.Equal(effectiveAt) || entry.OccurredAt.In(mustShanghai(t)).Format(DailySettlementDateLayout) != date {
		t.Fatalf("OccurredAt = %s; want rule effective time %s on target date", entry.OccurredAt, effectiveAt)
	}
}

func TestDailyBaseSettlementHonorsCurrentQualificationAndDoesNotRequireSSO(t *testing.T) {
	db := newXPTestDB(t)
	date := "2026-10-26"
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	activatedAt := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 130, createdAt, &activatedAt, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 131, createdAt, &activatedAt, users.ActorTypeBot, users.StatusNormal, 1)
	createSettlementUser(t, db, 132, createdAt, &activatedAt, users.ActorTypeHuman, users.StatusFrozen, 1)
	createSettlementUser(t, db, 133, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 0)
	createSettlementUser(t, db, 134, createdAt, &activatedAt, users.ActorTypeHuman, users.StatusNormal, 1)
	deleted := users.EntityComplete{Id: 134}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	// No Tongji campus binding table or binding row is present in this XP test
	// schema. Existing EligibleForXP behavior does not require SSO linkage.
	createSettlementUser(t, db, 135, createdAt, &activatedAt, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
	stats, err := svc.settleDailyBase(context.Background(), date, false, 2, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("settle eligible accounts: %v", err)
	}
	if stats.Scanned != 6 || stats.Eligible != 2 || stats.Awarded != 2 || stats.Skipped != 4 {
		t.Fatalf("qualification stats = %+v; want two eligible humans and four skipped", stats)
	}
	for _, id := range []uint64{130, 135} {
		if got, err := svc.CurrentBalance(context.Background(), id); err != nil || got != 5 {
			t.Errorf("user %d XP = %d, %v; want 5", id, got, err)
		}
	}
	for _, id := range []uint64{131, 132, 133, 134} {
		if got, err := svc.CurrentBalance(context.Background(), id); err != nil || got != 0 {
			t.Errorf("ineligible user %d XP = %d, %v; want 0", id, got, err)
		}
	}
}

func TestHistoricalSettlementSkipsUnverifiableActivationAndReportsStateLimits(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	verifiedBeforeDate := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	verifiedAfterDate := time.Date(2026, 10, 25, 17, 0, 0, 0, time.UTC) // October 26 Shanghai, after the October 25 day.
	createSettlementUser(t, db, 140, createdAt, &verifiedBeforeDate, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 141, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 142, createdAt, &verifiedAfterDate, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
	stats, err := svc.settleDailyBase(context.Background(), "2026-10-25", false, 2, time.Date(2026, 10, 26, 3, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("historical settlement: %v", err)
	}
	if !stats.Historical || len(stats.Warnings) == 0 || stats.Awarded != 1 || stats.UnverifiableHistory != 1 || stats.Skipped != 2 {
		t.Fatalf("historical stats = %+v; want one award, one unknown activation, and warning", stats)
	}
	if got, _ := svc.CurrentBalance(context.Background(), 140); got != 5 {
		t.Fatalf("verified account XP = %d; want 5", got)
	}
	for _, id := range []uint64{141, 142} {
		if got, _ := svc.CurrentBalance(context.Background(), id); got != 0 {
			t.Errorf("historically unqualified user %d XP = %d; want 0", id, got)
		}
	}
}

func TestDailyBaseDryRunCountsCandidatesAndSettledUsersWithoutXPWrites(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 150, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 151, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	now := time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC)
	occurredAt := time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC)
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 150, RuleKey: DailyBaseRuleKey, SourceKey: dailyBaseSourceKey("2026-10-26", 150), OccurredAt: occurredAt}); err != nil {
		t.Fatal(err)
	}
	var entriesBefore, balancesBefore, usageBefore int64
	if err := db.Model(&xpmodel.Entry{}).Count(&entriesBefore).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&xpmodel.Balance{}).Count(&balancesBefore).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&xpmodel.DailyUsage{}).Count(&usageBefore).Error; err != nil {
		t.Fatal(err)
	}
	stats, err := svc.settleDailyBase(context.Background(), "2026-10-26", true, 1, now)
	if err != nil || stats.DryRun != true || stats.Candidates != 1 || stats.AlreadySettled != 1 || stats.Awarded != 0 {
		t.Fatalf("dry-run stats = %+v, %v", stats, err)
	}
	var entriesAfter, balancesAfter, usageAfter int64
	_ = db.Model(&xpmodel.Entry{}).Count(&entriesAfter).Error
	_ = db.Model(&xpmodel.Balance{}).Count(&balancesAfter).Error
	_ = db.Model(&xpmodel.DailyUsage{}).Count(&usageAfter).Error
	if entriesAfter != entriesBefore || balancesAfter != balancesBefore || usageAfter != usageBefore {
		t.Fatalf("dry-run changed XP writes: entries %d→%d balances %d→%d usage %d→%d", entriesBefore, entriesAfter, balancesBefore, balancesAfter, usageBefore, usageAfter)
	}
}

func TestDailyBaseServiceRequiresAndEnforcesApplyBoundary(t *testing.T) {
	db := newXPTestDB(t)
	createSettlementUser(t, db, 155, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC), nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	for _, tc := range []struct {
		name      string
		startDate string
		wantErr   error
	}{
		{name: "missing boundary", startDate: "", wantErr: ErrMissingSettlementBoundary},
		{name: "before boundary", startDate: "2026-10-26", wantErr: ErrBeforeSettlementBoundary},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stats, err := svc.SettleDailyBase(context.Background(), "2026-10-25", tc.startDate, false, 10)
			if !errors.Is(err, tc.wantErr) || stats.Failed != 1 {
				t.Fatalf("apply result = %+v, %v; want failed boundary error %v", stats, err, tc.wantErr)
			}
		})
	}
	if got, err := svc.CurrentBalance(context.Background(), 155); err != nil || got != 0 {
		t.Fatalf("boundary rejection wrote XP: balance=%d err=%v", got, err)
	}
}

func TestDailyBaseSettlementResumesAfterInterruptionWithoutDuplicateAwards(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 160, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 161, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	date := "2026-10-26"
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{
		UserID: 160, RuleKey: DailyBaseRuleKey, SourceKey: dailyBaseSourceKey(date, 160),
		OccurredAt: time.Date(2026, 10, 26, 0, 0, 0, 0, mustShanghai(t)),
	}); err != nil {
		t.Fatal(err)
	}
	stats, err := svc.settleDailyBase(context.Background(), date, false, 1, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
	if err != nil || stats.AlreadySettled != 1 || stats.Awarded != 1 {
		t.Fatalf("resumed settlement = %+v, %v; want one replay and one remaining award", stats, err)
	}
	for _, id := range []uint64{160, 161} {
		if got, err := svc.CurrentBalance(context.Background(), id); err != nil || got != 5 {
			t.Errorf("user %d balance = %d, %v; want exactly 5", id, got, err)
		}
	}
}

func TestDailyBaseSettlementRecoversAfterDatabaseFailureMidRun(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 162, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 163, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	if err := db.Exec(`CREATE TRIGGER fail_daily_base_user_163 BEFORE INSERT ON xp_entries
		WHEN NEW.user_id = 163 BEGIN SELECT RAISE(ABORT, 'injected settlement failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	now := time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC)
	partial, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, now)
	if err == nil || partial.Awarded != 1 || partial.Failed != 1 {
		t.Fatalf("failed run = %+v, %v; want first award committed and second failure reported", partial, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 162); err != nil || got != 5 {
		t.Fatalf("first account after partial failure = %d, %v; want committed +5", got, err)
	}
	if got, err := svc.CurrentBalance(context.Background(), 163); err != nil || got != 0 {
		t.Fatalf("failing account after rollback = %d, %v; want 0", got, err)
	}
	if err := db.Exec("DROP TRIGGER fail_daily_base_user_163").Error; err != nil {
		t.Fatal(err)
	}
	recovered, err := svc.settleDailyBase(context.Background(), "2026-10-26", false, 1, now)
	if err != nil || recovered.Awarded != 1 || recovered.AlreadySettled != 1 {
		t.Fatalf("retry after failure = %+v, %v; want one new award and one existing award", recovered, err)
	}
	for _, id := range []uint64{162, 163} {
		if got, err := svc.CurrentBalance(context.Background(), id); err != nil || got != 5 {
			t.Errorf("account %d after recovery = %d, %v; want exactly +5", id, got, err)
		}
	}
}

func TestConcurrentDailyBaseSettlementRequestsDoNotDoubleAward(t *testing.T) {
	db := newXPTestDB(t)
	createdAt := time.Date(2026, 10, 24, 16, 0, 0, 0, time.UTC)
	createSettlementUser(t, db, 170, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
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
			t.Errorf("concurrent settlement returned error: %v", err)
		}
	}
	var created, replayed int64
	for report := range stats {
		created += report.Awarded
		replayed += report.AlreadySettled
	}
	if created != 1 || replayed != 1 {
		t.Fatalf("concurrent stats awarded=%d already=%d; want one of each", created, replayed)
	}
	var entries int64
	if err := db.Model(&xpmodel.Entry{}).Where("user_id = ? AND source_key = ?", 170, dailyBaseSourceKey("2026-10-26", 170)).Count(&entries).Error; err != nil || entries != 1 {
		t.Fatalf("daily source entries = %d, %v; want exactly one", entries, err)
	}
}

func TestDailyBaseSettlementErrorsOnUnavailableRuleDatabaseAndCancelledContext(t *testing.T) {
	t.Run("unavailable rule", func(t *testing.T) {
		db := newXPTestDB(t)
		date := time.Date(2026, 10, 26, 0, 0, 0, 0, mustShanghai(t))
		if _, err := newWithPolicy(db, pageConfig.SecurityAndRegistration{}).PublishRuleSet(testActorContext(10), RuleSetSpec{
			Enabled: false, TimeZone: ShanghaiTimeZone, EffectiveAt: date,
			Rules: []xpmodel.RewardRule{{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: 5}},
		}); err != nil {
			t.Fatal(err)
		}
		createSettlementUser(t, db, 180, date.Add(-24*time.Hour), nil, users.ActorTypeHuman, users.StatusNormal, 1)
		stats, err := newWithPolicy(db, pageConfig.SecurityAndRegistration{}).settleDailyBase(context.Background(), "2026-10-26", false, 10, date.Add(time.Hour))
		if !errors.Is(err, ErrRuleUnavailable) || stats.Failed == 0 {
			t.Fatalf("unavailable rule result = %+v, %v; want located ErrRuleUnavailable", stats, err)
		}
	})

	t.Run("database failure", func(t *testing.T) {
		db := newXPTestDB(t)
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatal(err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatal(err)
		}
		stats, err := newWithPolicy(db, pageConfig.SecurityAndRegistration{}).settleDailyBase(context.Background(), "2026-10-26", false, 10, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
		if err == nil || stats.Failed == 0 {
			t.Fatalf("closed-database result = %+v, %v; want visible failure", stats, err)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		db := newXPTestDB(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		stats, err := newWithPolicy(db, pageConfig.SecurityAndRegistration{}).settleDailyBase(ctx, "2026-10-26", false, 10, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
		if !errors.Is(err, context.Canceled) || stats.Failed == 0 {
			t.Fatalf("cancelled result = %+v, %v; want context.Canceled", stats, err)
		}
	})
}

func TestPublished65XPDailyCapRemainsEnforcedButIsNotSeeded(t *testing.T) {
	db := newXPTestDB(t)
	var initial xpmodel.RuleSet
	if err := db.Where("version = ?", 1).Take(&initial).Error; err != nil {
		t.Fatal(err)
	}
	if initial.DailyPositiveXPLimit != nil {
		t.Fatalf("initial cap = %v; want no active 65 XP cap", initial.DailyPositiveXPLimit)
	}
	createXPUser(t, db, 190)
	svc := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	start := time.Date(2026, 10, 26, 0, 0, 0, 0, mustShanghai(t))
	cap := int64(65)
	if _, err := svc.PublishRuleSet(testActorContext(10), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: start, DailyPositiveXPLimit: &cap,
		Rules: []xpmodel.RewardRule{
			{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: 5, DailyCountLimit: ptrInt64(1)},
			{RuleKey: "cap-test", Enabled: true, RewardXP: 60},
			{RuleKey: "cap-over", Enabled: true, RewardXP: 1},
		},
	}); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		rule   string
		source string
	}{{DailyBaseRuleKey, "cap-test:base"}, {"cap-test", "cap-test:sixty"}} {
		if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 190, RuleKey: item.rule, SourceKey: item.source, OccurredAt: start.Add(time.Minute)}); err != nil {
			t.Fatalf("apply %s: %v", item.rule, err)
		}
	}
	if _, err := svc.ApplyReward(context.Background(), AwardRequest{UserID: 190, RuleKey: "cap-over", SourceKey: "cap-test:over", OccurredAt: start.Add(2 * time.Minute)}); !errors.Is(err, ErrRewardLimit) {
		t.Fatalf("award over published 65 XP cap = %v; want ErrRewardLimit", err)
	}
}

func TestParseDailySettlementDateRequiresExactShanghaiCalendarDate(t *testing.T) {
	for _, input := range []string{"2026-10-26", "2026-02-30", "2026-2-6", "2026-10-26T00:00:00+08:00", " 2026-10-26"} {
		date, err := ParseDailySettlementDate(input)
		if input == "2026-10-26" {
			if err != nil || date.Hour() != 0 || date.Location().String() != ShanghaiTimeZone {
				t.Fatalf("parse %q = %v, %v; want Shanghai midnight", input, date, err)
			}
			continue
		}
		if !errors.Is(err, ErrInvalidSettlementDate) {
			t.Errorf("parse %q error = %v; want ErrInvalidSettlementDate", input, err)
		}
	}
}

func createSettlementUser(t *testing.T, db *gorm.DB, id uint64, createdAt time.Time, activatedAt *time.Time, actorType, frozen int8, activated int8) {
	t.Helper()
	user := users.EntityComplete{
		Id: id, Username: fmt.Sprintf("xp-settlement-user-%d", id), ActorType: actorType,
		IsFrozen: frozen, IsActivated: activated, CreatedAt: createdAt, ActivatedAt: activatedAt,
		Prestige: 19,
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create settlement user %d: %v", id, err)
	}
}

func mustShanghai(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation(ShanghaiTimeZone)
	if err != nil {
		t.Fatal(err)
	}
	return location
}
