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

func TestDailyBaseSettlementSkipsUserWithoutCoveringWindowAndContinues(t *testing.T) {
	db := newXPTestDB(t)
	date := "2026-11-26"
	dayStart, err := ParseDailySettlementDate(date)
	if err != nil {
		t.Fatal(err)
	}
	service := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
	publishDailyBaseRule(t, service, dayStart.Add(-time.Hour), true)
	publishDailyBaseRule(t, service, dayStart.Add(10*time.Hour), false)

	eligibleAtNine := dayStart.Add(9 * time.Hour)
	eligibleAtEleven := dayStart.Add(11 * time.Hour)
	createSettlementUser(t, db, 124, dayStart.Add(-time.Hour), &eligibleAtNine, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 125, dayStart.Add(-time.Hour), &eligibleAtEleven, users.ActorTypeHuman, users.StatusNormal, 1)
	createSettlementUser(t, db, 126, dayStart.Add(-time.Hour), &eligibleAtNine, users.ActorTypeHuman, users.StatusNormal, 1)

	stats, err := service.settleDailyBase(context.Background(), date, false, 1, dayStart.AddDate(0, 0, 1).Add(time.Hour))
	if err != nil {
		t.Fatalf("settlement stopped on a user outside all active windows: %v", err)
	}
	if stats.Scanned != 3 || stats.Eligible != 3 || stats.Candidates != 2 || stats.Awarded != 2 || stats.Skipped != 1 || stats.Failed != 0 {
		t.Fatalf("settlement stats = %+v; want two awards, one skipped, and no failure", stats)
	}
	for _, id := range []uint64{124, 126} {
		if got, err := service.CurrentBalance(context.Background(), id); err != nil || got != 5 {
			t.Errorf("user %d balance = %d, %v; want 5", id, got, err)
		}
	}
	if got, err := service.CurrentBalance(context.Background(), 125); err != nil || got != 0 {
		t.Errorf("post-disable eligible user balance = %d, %v; want 0", got, err)
	}
}

func TestDailyBaseSettlementRuleWindowsUseHalfOpenEffectiveIntervals(t *testing.T) {
	tests := []struct {
		name    string
		publish func(t *testing.T, service *Service, dayStart time.Time)
		users   []struct {
			id       uint64
			eligible time.Duration
		}
		wantAward map[uint64]time.Duration
	}{
		{
			name: "prior enabled version covered by midnight disable is unavailable",
			publish: func(t *testing.T, service *Service, dayStart time.Time) {
				publishDailyBaseRule(t, service, dayStart.Add(-time.Hour), true)
				publishDailyBaseRule(t, service, dayStart, false)
			},
		},
		{
			name: "prior enabled version remains valid until 10",
			publish: func(t *testing.T, service *Service, dayStart time.Time) {
				publishDailyBaseRule(t, service, dayStart.Add(-time.Hour), true)
				publishDailyBaseRule(t, service, dayStart.Add(10*time.Hour), false)
			},
			users: []struct {
				id       uint64
				eligible time.Duration
			}{{201, 9 * time.Hour}, {202, 10 * time.Hour}},
			wantAward: map[uint64]time.Duration{201: 9 * time.Hour},
		},
		{
			name: "10 to 16 enablement interval only",
			publish: func(t *testing.T, service *Service, dayStart time.Time) {
				publishDailyBaseRule(t, service, dayStart, false)
				publishDailyBaseRule(t, service, dayStart.Add(10*time.Hour), true)
				publishDailyBaseRule(t, service, dayStart.Add(16*time.Hour), false)
			},
			users: []struct {
				id       uint64
				eligible time.Duration
			}{{203, 9 * time.Hour}, {204, 10 * time.Hour}, {205, 12 * time.Hour}, {206, 16 * time.Hour}},
			wantAward: map[uint64]time.Duration{203: 10 * time.Hour, 204: 10 * time.Hour, 205: 12 * time.Hour},
		},
		{
			name: "alternating versions select each effective window",
			publish: func(t *testing.T, service *Service, dayStart time.Time) {
				publishDailyBaseRule(t, service, dayStart, false)
				publishDailyBaseRule(t, service, dayStart.Add(6*time.Hour), true)
				publishDailyBaseRule(t, service, dayStart.Add(8*time.Hour), false)
				publishDailyBaseRule(t, service, dayStart.Add(12*time.Hour), true)
				publishDailyBaseRule(t, service, dayStart.Add(16*time.Hour), false)
			},
			users: []struct {
				id       uint64
				eligible time.Duration
			}{{207, 5 * time.Hour}, {208, 7 * time.Hour}, {209, 10 * time.Hour}, {210, 13 * time.Hour}, {211, 17 * time.Hour}},
			wantAward: map[uint64]time.Duration{207: 6 * time.Hour, 208: 7 * time.Hour, 209: 12 * time.Hour, 210: 13 * time.Hour},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := newXPTestDB(t)
			date := "2026-11-26"
			dayStart, err := ParseDailySettlementDate(date)
			if err != nil {
				t.Fatal(err)
			}
			service := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
			tc.publish(t, service, dayStart)
			for _, user := range tc.users {
				activation := dayStart.Add(user.eligible)
				createSettlementUser(t, db, user.id, dayStart.Add(-time.Hour), &activation, users.ActorTypeHuman, users.StatusNormal, 1)
			}
			stats, err := service.settleDailyBase(context.Background(), date, false, 2, dayStart.AddDate(0, 0, 1).Add(time.Hour))
			if len(tc.users) == 0 {
				if !errors.Is(err, ErrRuleUnavailable) || stats.Failed == 0 {
					t.Fatalf("settlement = %+v, %v; want date-level ErrRuleUnavailable", stats, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("settlement: %v", err)
			}
			if stats.Awarded != int64(len(tc.wantAward)) || stats.Skipped != int64(len(tc.users)-len(tc.wantAward)) || stats.Failed != 0 {
				t.Fatalf("settlement stats = %+v; want %d awards, %d skips", stats, len(tc.wantAward), len(tc.users)-len(tc.wantAward))
			}
			for _, user := range tc.users {
				_, wantAward := tc.wantAward[user.id]
				var entry xpmodel.Entry
				err := db.Where("user_id = ?", user.id).Take(&entry).Error
				if wantAward {
					if err != nil {
						t.Errorf("user %d expected award: %v", user.id, err)
						continue
					}
					wantAt := dayStart.Add(tc.wantAward[user.id])
					if !entry.OccurredAt.Equal(wantAt) {
						t.Errorf("user %d OccurredAt = %s; want %s", user.id, entry.OccurredAt, wantAt)
					}
				} else if !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Errorf("user %d should not have an entry, got %+v / %v", user.id, entry, err)
				}
			}
		})
	}
}

func TestDailyBaseSettlementTodayDoesNotUseFutureWindowsAndHonorsBoundaries(t *testing.T) {
	t.Run("future enablement is unavailable before its exact effective time", func(t *testing.T) {
		db := newXPTestDB(t)
		date := "2026-11-26"
		dayStart, _ := ParseDailySettlementDate(date)
		service := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
		publishDailyBaseRule(t, service, dayStart, false)
		publishDailyBaseRule(t, service, dayStart.Add(12*time.Hour), true)
		stats, err := service.settleDailyBase(context.Background(), date, true, 10, dayStart.Add(9*time.Hour))
		if !errors.Is(err, ErrRuleUnavailable) || stats.Failed == 0 {
			t.Fatalf("pre-enable settlement = %+v, %v; want date-level ErrRuleUnavailable", stats, err)
		}
	})

	t.Run("effective time is inclusive at the current instant", func(t *testing.T) {
		db := newXPTestDB(t)
		date := "2026-11-26"
		dayStart, _ := ParseDailySettlementDate(date)
		service := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
		enableAt := dayStart.Add(9 * time.Hour)
		endAt := dayStart.Add(10 * time.Hour)
		publishDailyBaseRule(t, service, dayStart, false)
		publishDailyBaseRule(t, service, enableAt, true)
		publishDailyBaseRule(t, service, endAt, false)
		createSettlementUser(t, db, 220, dayStart.Add(-time.Hour), nil, users.ActorTypeHuman, users.StatusNormal, 1)
		stats, err := service.settleDailyBase(context.Background(), date, false, 10, enableAt)
		if err != nil {
			t.Fatalf("settle at effective-time boundary: %v", err)
		}
		if stats.Awarded != 1 {
			t.Fatalf("boundary stats = %+v; want one award", stats)
		}
		var entry xpmodel.Entry
		if err := db.Where("user_id = ?", 220).Take(&entry).Error; err != nil {
			t.Fatal(err)
		}
		if !entry.OccurredAt.Equal(enableAt) || entry.OccurredAt.After(enableAt) {
			t.Fatalf("OccurredAt = %s; want exact effective time %s and not later than now", entry.OccurredAt, enableAt)
		}
	})

	t.Run("account eligible after an elapsed window is skipped", func(t *testing.T) {
		db := newXPTestDB(t)
		date := "2026-11-26"
		dayStart, _ := ParseDailySettlementDate(date)
		service := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
		publishDailyBaseRule(t, service, dayStart, true)
		publishDailyBaseRule(t, service, dayStart.Add(10*time.Hour), false)
		eligibleAtNine := dayStart.Add(9 * time.Hour)
		eligibleAtEleven := dayStart.Add(11 * time.Hour)
		createSettlementUser(t, db, 223, dayStart.Add(-time.Hour), &eligibleAtNine, users.ActorTypeHuman, users.StatusNormal, 1)
		createSettlementUser(t, db, 224, dayStart.Add(-time.Hour), &eligibleAtEleven, users.ActorTypeHuman, users.StatusNormal, 1)
		stats, err := service.settleDailyBase(context.Background(), date, false, 10, dayStart.Add(12*time.Hour))
		if err != nil || stats.Eligible != 2 || stats.Awarded != 1 || stats.Skipped != 1 || stats.Failed != 0 {
			t.Fatalf("current-date elapsed-window result = %+v, %v; want one award, one skip, no failure", stats, err)
		}
		var entry xpmodel.Entry
		if err := db.Where("user_id = ?", 223).Take(&entry).Error; err != nil || !entry.OccurredAt.Equal(eligibleAtNine) || entry.OccurredAt.After(dayStart.Add(12*time.Hour)) {
			t.Fatalf("pre-disable eligible entry = %+v, %v; want event at activation and no later than now", entry, err)
		}
		if err := db.Where("user_id = ?", 224).Take(&entry).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("post-disable eligible user received entry %+v, %v", entry, err)
		}
	})

	t.Run("interval end is exclusive for historical eligibility", func(t *testing.T) {
		db := newXPTestDB(t)
		date := "2026-11-26"
		dayStart, _ := ParseDailySettlementDate(date)
		service := newWithPolicy(db, pageConfig.SecurityAndRegistration{EnableEmailVerification: true})
		enableAt := dayStart.Add(9 * time.Hour)
		endAt := dayStart.Add(10 * time.Hour)
		publishDailyBaseRule(t, service, enableAt, true)
		publishDailyBaseRule(t, service, endAt, false)
		activationAtStart := enableAt
		activationAtEnd := endAt
		createSettlementUser(t, db, 221, dayStart.Add(-time.Hour), &activationAtStart, users.ActorTypeHuman, users.StatusNormal, 1)
		createSettlementUser(t, db, 222, dayStart.Add(-time.Hour), &activationAtEnd, users.ActorTypeHuman, users.StatusNormal, 1)
		stats, err := service.settleDailyBase(context.Background(), date, false, 10, dayStart.AddDate(0, 0, 1))
		if err != nil {
			t.Fatalf("historical boundary settlement: %v", err)
		}
		if stats.Awarded != 1 || stats.Skipped != 1 {
			t.Fatalf("historical boundary stats = %+v; want one award and one skip", stats)
		}
		var entry xpmodel.Entry
		if err := db.Where("user_id = ?", 221).Take(&entry).Error; err != nil || !entry.OccurredAt.Equal(enableAt) {
			t.Fatalf("user at inclusive effective time entry = %+v, %v; want OccurredAt %s", entry, err, enableAt)
		}
		if err := db.Where("user_id = ?", 222).Take(&entry).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("user at exclusive interval end received entry %+v, %v", entry, err)
		}
	})
}

func TestDailyBaseSettlementCandidatesArePreQuotaCounts(t *testing.T) {
	db := newXPTestDB(t)
	date := "2026-11-26"
	dayStart, err := ParseDailySettlementDate(date)
	if err != nil {
		t.Fatal(err)
	}
	service := newWithPolicy(db, pageConfig.SecurityAndRegistration{})
	cap := int64(5)
	five := int64(5)
	if _, err := service.PublishRuleSet(testActorContext(10), RuleSetSpec{
		Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: dayStart, DailyPositiveXPLimit: &cap,
		Rules: []xpmodel.RewardRule{
			{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: five, MaxPerEventXP: &five},
			{RuleKey: "quota-consumer", Enabled: true, RewardXP: five, MaxPerEventXP: &five},
		},
	}); err != nil {
		t.Fatal(err)
	}
	createSettlementUser(t, db, 230, dayStart.Add(-time.Hour), nil, users.ActorTypeHuman, users.StatusNormal, 1)
	if _, err := service.ApplyReward(context.Background(), AwardRequest{
		UserID: 230, RuleKey: "quota-consumer", SourceKey: "daily-base-quota-test:consumer",
		OccurredAt: dayStart.Add(time.Hour).UTC(),
	}); err != nil {
		t.Fatalf("consume daily XP quota: %v", err)
	}
	now := dayStart.AddDate(0, 0, 1)

	dryRun, err := service.settleDailyBase(context.Background(), date, true, 10, now)
	if err != nil || dryRun.Candidates != 1 || dryRun.Awarded != 0 || dryRun.Skipped != 0 {
		t.Fatalf("dry-run quota preview = %+v, %v; want one pre-quota candidate and no award", dryRun, err)
	}

	apply, err := service.settleDailyBase(context.Background(), date, false, 10, now)
	if err != nil || apply.Eligible != 1 || apply.Candidates != 1 || apply.Awarded != 0 || apply.Skipped != 1 || apply.Failed != 0 {
		t.Fatalf("quota-rejected apply = %+v, %v; want candidate and skipped overlap", apply, err)
	}
	if got, err := service.CurrentBalance(context.Background(), 230); err != nil || got != 5 {
		t.Fatalf("balance after quota rejection = %d, %v; want only the quota-consuming +5", got, err)
	}
}

func publishDailyBaseRule(t *testing.T, service *Service, effectiveAt time.Time, enabled bool) {
	t.Helper()
	five := int64(5)
	if _, err := service.PublishRuleSet(testActorContext(10), RuleSetSpec{
		Enabled: enabled, TimeZone: ShanghaiTimeZone, EffectiveAt: effectiveAt,
		Rules: []xpmodel.RewardRule{{RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: five, MaxPerEventXP: &five}},
	}); err != nil {
		t.Fatalf("publish daily_base rule at %s (enabled=%t): %v", effectiveAt, enabled, err)
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

	t.Run("source key conflict", func(t *testing.T) {
		db := newXPTestDB(t)
		date := "2026-10-26"
		createdAt := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
		createSettlementUser(t, db, 181, createdAt, nil, users.ActorTypeHuman, users.StatusNormal, 1)
		entry := xpmodel.Entry{
			UserID: 181, SourceKey: dailyBaseSourceKey(date, 181), RuleKey: "different-rule",
			Delta: 5, OccurredAt: time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC),
		}
		if err := db.Create(&entry).Error; err != nil {
			t.Fatal(err)
		}
		stats, err := newWithPolicy(db, pageConfig.SecurityAndRegistration{}).settleDailyBase(context.Background(), date, false, 10, time.Date(2026, 10, 26, 1, 0, 0, 0, time.UTC))
		if !errors.Is(err, ErrSourceKeyConflict) || stats.Failed != 1 {
			t.Fatalf("source conflict result = %+v, %v; want visible failure", stats, err)
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
