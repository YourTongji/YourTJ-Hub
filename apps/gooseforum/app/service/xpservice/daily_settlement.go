package xpservice

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
	"gorm.io/gorm"
)

const (
	DailySettlementDateLayout       = "2006-01-02"
	DefaultDailySettlementBatchSize = 250
	MaxDailySettlementBatchSize     = 900 // Keep the source-key IN query below SQLite's traditional 999-parameter limit.
	maxSQLiteSettlementAttempts     = 10
)

var (
	ErrInvalidSettlementDate     = errors.New("invalid XP settlement date")
	ErrFutureSettlementDate      = errors.New("XP settlement date is in the future")
	ErrInvalidSettlementSize     = errors.New("invalid XP settlement batch size")
	ErrMissingSettlementBoundary = errors.New("XP apply requires an explicit settlement start date")
	ErrBeforeSettlementBoundary  = errors.New("XP settlement date is before the enabled start date")
)

const historicalEligibilityWarning = "Past-date settlement uses current account status because freeze-state and historical site-setting changes are not retained; verified accounts without an activation timestamp are skipped."

// DailySettlementStats contains aggregate counters only. It deliberately omits
// account identifiers so CLI output cannot expose or correlate anonymous users.
type DailySettlementStats struct {
	Date       string `json:"date"`
	DryRun     bool   `json:"dry_run"`
	Historical bool   `json:"historical"`
	// Scanned counts account rows read by the bounded ID cursor.
	Scanned int64 `json:"scanned"`
	// Eligible counts not-yet-settled accounts passing the static EligibleForXP
	// check. It may overlap Skipped when date-specific qualification or reward
	// limits reject.
	Eligible int64 `json:"eligible"`
	// Candidates counts pre-quota users that passed account and time-window
	// checks. ApplyReward may still reject them for quota; dry-run does not read
	// quota and therefore this value is not an award forecast.
	Candidates int64 `json:"candidates"`
	Awarded    int64 `json:"awarded"`
	// AlreadySettled counts existing source-key entries, including a concurrent
	// settlement that wins after this run's initial source-key lookup.
	AlreadySettled int64 `json:"already_settled"`
	// Skipped counts expected qualification, date-window, history, or quota
	// rejections. UnverifiableHistory is a subset of Skipped.
	Skipped             int64 `json:"skipped"`
	UnverifiableHistory int64 `json:"unverifiable_history"`
	// Failed is the number of fatal errors observed before the run returned;
	// previously committed accounts may still be counted in Awarded.
	Failed   int64    `json:"failed"`
	Warnings []string `json:"warnings,omitempty"`
}

// ParseDailySettlementDate accepts only an exact YYYY-MM-DD Asia/Shanghai
// calendar date and returns its local midnight.
func ParseDailySettlementDate(value string) (time.Time, error) {
	location, err := shanghaiLocation()
	if err != nil {
		return time.Time{}, fmt.Errorf("load XP calendar location %q: %w", ShanghaiTimeZone, err)
	}
	date, err := time.ParseInLocation(DailySettlementDateLayout, value, location)
	if err != nil || date.Format(DailySettlementDateLayout) != value {
		return time.Time{}, fmt.Errorf("%w: %q; expected YYYY-MM-DD", ErrInvalidSettlementDate, value)
	}
	return date, nil
}

// SettleDailyBase settles one explicit Asia/Shanghai day. Apply mode requires
// the operator-selected start date, which is enforced here for every caller.
// Dry-run mode only reads users, active rule versions, and existing source
// keys. Writes always go through ApplyReward so ledger, balance, and quota
// remain transactional.
func (s *Service) SettleDailyBase(ctx context.Context, date, startDate string, dryRun bool, batchSize int) (DailySettlementStats, error) {
	stats := DailySettlementStats{Date: date, DryRun: dryRun}
	settlementDate, err := ParseDailySettlementDate(date)
	if err != nil {
		stats.Failed++
		return stats, err
	}
	if !dryRun && strings.TrimSpace(startDate) == "" {
		stats.Failed++
		return stats, ErrMissingSettlementBoundary
	}
	if strings.TrimSpace(startDate) != "" {
		boundary, err := ParseDailySettlementDate(startDate)
		if err != nil {
			stats.Failed++
			return stats, fmt.Errorf("invalid settlement start date: %w", err)
		}
		if settlementDate.Before(boundary) {
			stats.Failed++
			return stats, fmt.Errorf("%w: %s < %s", ErrBeforeSettlementBoundary, date, startDate)
		}
	}
	return s.settleDailyBase(ctx, date, dryRun, batchSize, time.Now())
}

func (s *Service) settleDailyBase(ctx context.Context, date string, dryRun bool, batchSize int, now time.Time) (DailySettlementStats, error) {
	stats := DailySettlementStats{Date: date, DryRun: dryRun}
	if s == nil || s.db == nil || s.policy == nil {
		stats.Failed++
		return stats, errors.New("XP service is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		stats.Failed++
		return stats, err
	}
	if batchSize == 0 {
		batchSize = DefaultDailySettlementBatchSize
	}
	if batchSize < 1 || batchSize > MaxDailySettlementBatchSize {
		stats.Failed++
		return stats, fmt.Errorf("%w: must be between 1 and %d", ErrInvalidSettlementSize, MaxDailySettlementBatchSize)
	}

	dayStart, err := ParseDailySettlementDate(date)
	if err != nil {
		stats.Failed++
		return stats, err
	}
	location := dayStart.Location()
	dayEnd := dayStart.AddDate(0, 0, 1)
	if now.IsZero() {
		now = time.Now()
	}
	nowLocal := now.In(location)
	todayStart := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, location)
	if dayStart.After(todayStart) {
		stats.Failed++
		return stats, fmt.Errorf("%w: %s", ErrFutureSettlementDate, date)
	}
	stats.Historical = dayStart.Before(todayStart)
	if stats.Historical {
		stats.Warnings = append(stats.Warnings, historicalEligibilityWarning)
	}

	windows, err := loadDailyBaseRuleWindows(ctx, s.db, dayStart, dayEnd, now, stats.Historical)
	if err != nil {
		stats.Failed++
		return stats, fmt.Errorf("load daily_base rule for %s: %w", date, err)
	}

	security := s.policy()
	var afterID uint64
	for {
		if err := ctx.Err(); err != nil {
			stats.Failed++
			return stats, err
		}

		var batch []users.EntityComplete
		err := s.db.WithContext(ctx).Unscoped().
			Select("id, actor_type, is_frozen, is_activated, activated_at, created_at, deleted_at").
			Where("id > ? AND created_at < ?", afterID, dayEnd.UTC()).
			Order("id ASC").Limit(batchSize).Find(&batch).Error
		if err != nil {
			stats.Failed++
			return stats, fmt.Errorf("scan XP settlement accounts: %w", err)
		}
		if len(batch) == 0 {
			return stats, nil
		}
		stats.Scanned += int64(len(batch))
		afterID = batch[len(batch)-1].Id

		keys := make([]string, 0, len(batch))
		for i := range batch {
			keys = append(keys, dailyBaseSourceKey(date, batch[i].Id))
		}
		already, err := loadDailyBaseEntries(ctx, s.db, keys)
		if err != nil {
			stats.Failed++
			return stats, fmt.Errorf("read daily_base settlement keys: %w", err)
		}

		for i := range batch {
			if err := ctx.Err(); err != nil {
				stats.Failed++
				return stats, err
			}
			user := batch[i]
			sourceKey := dailyBaseSourceKey(date, user.Id)
			if entry, exists := already[sourceKey]; exists {
				if entry.UserID != user.Id || entry.RuleKey != DailyBaseRuleKey || entry.Delta <= 0 {
					stats.Failed++
					return stats, fmt.Errorf("daily_base source key conflict for settlement date %s: %w", date, ErrSourceKeyConflict)
				}
				stats.AlreadySettled++
				continue
			}
			if !EligibleForXP(user, security) {
				stats.Skipped++
				continue
			}
			stats.Eligible++

			occurredAt, unknownHistory, skip := dailyBaseOccurrence(user, dayStart, dayEnd, now, security, stats.Historical, windows)
			if unknownHistory {
				stats.UnverifiableHistory++
				stats.Skipped++
				continue
			}
			if skip {
				stats.Skipped++
				continue
			}

			stats.Candidates++
			if dryRun {
				continue
			}

			result, err := s.applyDailyBaseWithRetry(ctx, AwardRequest{
				UserID: user.Id, RuleKey: DailyBaseRuleKey,
				SourceKey: sourceKey, OccurredAt: occurredAt.UTC(),
			})
			if errors.Is(err, ErrIneligible) || errors.Is(err, ErrRewardLimit) {
				stats.Skipped++
				continue
			}
			if err != nil {
				stats.Failed++
				return stats, fmt.Errorf("apply daily_base settlement for %s at %s: %w", date, occurredAt, err)
			}
			if result.AlreadyApplied {
				stats.Candidates--
				stats.AlreadySettled++
				continue
			}
			stats.Awarded++
		}
	}
}

type dailyBaseRuleWindow struct {
	effective time.Time
	end       time.Time
	available bool
}

func loadDailyBaseRuleWindows(ctx context.Context, conn *gorm.DB, dayStart, dayEnd, now time.Time, historical bool) ([]dailyBaseRuleWindow, error) {
	query := conn.WithContext(ctx)
	var ruleSets []xpmodel.RuleSet
	var previous xpmodel.RuleSet
	previousErr := query.Where("effective_at < ?", dayStart.UTC()).Order("effective_at DESC").Take(&previous).Error
	if previousErr != nil && !errors.Is(previousErr, gorm.ErrRecordNotFound) {
		return nil, previousErr
	}
	if previousErr == nil {
		ruleSets = append(ruleSets, previous)
	}
	var inDay []xpmodel.RuleSet
	if err := query.Where("effective_at >= ? AND effective_at < ?", dayStart.UTC(), dayEnd.UTC()).Order("effective_at ASC").Find(&inDay).Error; err != nil {
		return nil, err
	}
	ruleSets = append(ruleSets, inDay...)
	if len(ruleSets) == 0 {
		return nil, ErrRuleUnavailable
	}

	versions := make([]uint64, 0, len(ruleSets))
	for _, set := range ruleSets {
		versions = append(versions, set.Version)
	}
	var rules []xpmodel.RewardRule
	if err := query.Where("rule_set_version IN ? AND rule_key = ?", versions, DailyBaseRuleKey).Find(&rules).Error; err != nil {
		return nil, err
	}
	ruleByVersion := make(map[uint64]xpmodel.RewardRule, len(rules))
	for _, rule := range rules {
		ruleByVersion[rule.RuleSetVersion] = rule
	}

	sort.Slice(ruleSets, func(i, j int) bool { return ruleSets[i].EffectiveAt.Before(ruleSets[j].EffectiveAt) })
	windows := make([]dailyBaseRuleWindow, 0, len(ruleSets))
	for i, set := range ruleSets {
		rule, exists := ruleByVersion[set.Version]
		available := exists && set.Enabled && set.TimeZone == ShanghaiTimeZone && rule.Enabled && rule.RewardXP > 0
		if rule.MaxPerEventXP != nil && *rule.MaxPerEventXP <= 0 {
			available = false
		}
		end := dayEnd
		if i+1 < len(ruleSets) && ruleSets[i+1].EffectiveAt.Before(end) {
			end = ruleSets[i+1].EffectiveAt
		}
		windows = append(windows, dailyBaseRuleWindow{effective: set.EffectiveAt, end: end, available: available})
	}
	for _, window := range windows {
		start := laterTime(window.effective, dayStart)
		end := window.end
		if end.After(dayEnd) {
			end = dayEnd
		}
		if !window.available || !start.Before(end) {
			continue
		}
		// Today's settlement can replay any valid portion that has begun by
		// now, but it must not use a rule version whose effective time is still
		// in the future. Equality is inclusive for effective_at.
		if !historical && (window.effective.After(now) || start.After(now)) {
			continue
		}
		return windows, nil
	}
	return nil, ErrRuleUnavailable
}

func dailyBaseOccurrence(user users.EntityComplete, dayStart, dayEnd, now time.Time, security pageConfig.SecurityAndRegistration, historical bool, windows []dailyBaseRuleWindow) (occurredAt time.Time, unknownHistory bool, skip bool) {
	if user.CreatedAt.IsZero() {
		return time.Time{}, true, true
	}
	if user.CreatedAt.After(now) && !historical {
		return time.Time{}, false, true
	}
	eligibleAt := dayStart
	if user.CreatedAt.After(eligibleAt) {
		eligibleAt = user.CreatedAt
	}
	if security.EnableEmailVerification {
		if user.ActivatedAt == nil {
			if historical {
				return time.Time{}, true, true
			}
			eligibleAt = laterTime(eligibleAt, now)
		} else {
			if user.ActivatedAt.After(now) && !historical {
				return time.Time{}, false, true
			}
			eligibleAt = laterTime(eligibleAt, *user.ActivatedAt)
		}
	}
	if !eligibleAt.Before(dayEnd) {
		return time.Time{}, false, true
	}
	if !historical && eligibleAt.After(now) {
		return time.Time{}, false, true
	}
	for _, window := range windows {
		if !window.available {
			continue
		}
		windowStart := laterTime(window.effective, dayStart)
		windowEnd := window.end
		if windowEnd.After(dayEnd) {
			windowEnd = dayEnd
		}
		occurredAt := laterTime(eligibleAt, windowStart)
		if !occurredAt.Before(windowEnd) {
			continue
		}
		if !historical && (window.effective.After(now) || occurredAt.After(now)) {
			continue
		}
		return occurredAt, false, false
	}
	// The date-level preflight has already established that an award window
	// exists. A specific account can still become eligible only after every
	// such window, which is an expected skip rather than a run-wide rule error.
	return time.Time{}, false, true
}

func laterTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func dailyBaseSourceKey(date string, userID uint64) string {
	return fmt.Sprintf("xp:daily_base:%s:user:%d", date, userID)
}

func loadDailyBaseEntries(ctx context.Context, conn *gorm.DB, sourceKeys []string) (map[string]xpmodel.Entry, error) {
	entries := make(map[string]xpmodel.Entry, len(sourceKeys))
	if len(sourceKeys) == 0 {
		return entries, nil
	}
	var rows []xpmodel.Entry
	if err := conn.WithContext(ctx).Select("user_id, source_key, rule_key, delta").Where("source_key IN ?", sourceKeys).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		entries[row.SourceKey] = row
	}
	return entries, nil
}

func (s *Service) applyDailyBaseWithRetry(ctx context.Context, request AwardRequest) (AwardResult, error) {
	for attempt := 0; ; attempt++ {
		result, err := s.ApplyReward(ctx, request)
		if err == nil || s.db.Name() != "sqlite" || !isSQLiteBusyOrLocked(err) || attempt+1 >= maxSQLiteSettlementAttempts {
			return result, err
		}
		delay := time.Duration(1<<attempt) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return AwardResult{}, ctx.Err()
		case <-timer.C:
		}
	}
}
