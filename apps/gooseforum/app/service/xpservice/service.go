package xpservice

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	xpmodel "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/xp"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DailyBaseRuleKey             = "daily_base"
	ShanghaiTimeZone             = "Asia/Shanghai"
	maxSQLiteRulePublishAttempts = 8
)

var (
	ErrIneligible            = errors.New("account is not eligible for XP")
	ErrRuleUnavailable       = errors.New("XP reward rule unavailable")
	ErrRewardLimit           = errors.New("XP reward limit exceeded")
	ErrInvalidRule           = errors.New("invalid XP reward rule")
	ErrInvalidAdjustment     = errors.New("invalid XP adjustment")
	ErrSourceKeyConflict     = errors.New("XP source key already used for a different operation")
	ErrNegativeBalance       = errors.New("XP balance cannot become negative")
	ErrReversalLimit         = errors.New("XP reversal exceeds original award")
	ErrOriginalAwardNotFound = errors.New("original XP award not found")
	ErrAuditActorRequired    = errors.New("authenticated XP audit actor is required")
)

type auditActorContextKey struct{}

type Service struct {
	db     *gorm.DB
	policy func() pageConfig.SecurityAndRegistration
}

func New(db *gorm.DB) *Service {
	return &Service{db: db, policy: hotdataserve.GetSecuritySettingsConfigCache}
}

func newWithPolicy(db *gorm.DB, policy pageConfig.SecurityAndRegistration) *Service {
	return &Service{db: db, policy: func() pageConfig.SecurityAndRegistration { return policy }}
}

// WithAuthenticatedActor marks a server-side context with the actor resolved
// from authenticated session state after the caller has completed its own
// authorization checks. Never derive userID from request body or query input.
func WithAuthenticatedActor(ctx context.Context, userID uint64) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, auditActorContextKey{}, userID)
}

func authenticatedActorID(ctx context.Context) (uint64, bool) {
	if ctx == nil {
		return 0, false
	}
	userID, ok := ctx.Value(auditActorContextKey{}).(uint64)
	return userID, ok && userID != 0
}

// IsTerminal reports domain rejections that should be acknowledged by a future
// reward outbox instead of retried. Unclassified errors (including database
// and transport errors) remain retryable.
func IsTerminal(err error) bool {
	for _, terminal := range []error{
		ErrIneligible,
		ErrRuleUnavailable,
		ErrRewardLimit,
		ErrInvalidRule,
		ErrInvalidAdjustment,
		ErrSourceKeyConflict,
		ErrNegativeBalance,
		ErrReversalLimit,
		ErrOriginalAwardNotFound,
		ErrAuditActorRequired,
	} {
		if errors.Is(err, terminal) {
			return true
		}
	}
	return false
}

func EligibleForXP(user users.EntityComplete, security pageConfig.SecurityAndRegistration) bool {
	if user.Id == 0 || user.ActorType != users.ActorTypeHuman || user.IsFrozen != users.StatusNormal || user.DeletedAt.Valid {
		return false
	}
	return !security.EnableEmailVerification || user.IsActivated == 1
}

type AwardRequest struct {
	// UserID must be the real numeric forum account ID. Business-event adapters
	// resolve anonymous personas to their owning account before calling; this
	// service intentionally does not create persona-scoped XP wallets.
	UserID     uint64
	RuleKey    string
	SourceKey  string
	OccurredAt time.Time
}

type AwardResult struct {
	Entry          xpmodel.Entry
	AlreadyApplied bool
}

func (s *Service) ApplyReward(ctx context.Context, request AwardRequest) (AwardResult, error) {
	if s == nil || s.db == nil || s.policy == nil {
		return AwardResult{}, errors.New("XP service is not configured")
	}
	if request.UserID == 0 || strings.TrimSpace(request.SourceKey) == "" || len(request.SourceKey) > 255 || strings.TrimSpace(request.RuleKey) == "" || len(request.RuleKey) > 64 || request.OccurredAt.IsZero() {
		return AwardResult{}, ErrInvalidRule
	}
	request.SourceKey = strings.TrimSpace(request.SourceKey)
	request.RuleKey = strings.TrimSpace(request.RuleKey)
	location, err := shanghaiLocation()
	if err != nil {
		return AwardResult{}, fmt.Errorf("load XP calendar location %q: %w", ShanghaiTimeZone, err)
	}
	var result AwardResult
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		balance, err := lockBalance(tx, request.UserID)
		if err != nil {
			return err
		}

		var existing xpmodel.Entry
		err = tx.Where("source_key = ?", request.SourceKey).Take(&existing).Error
		if err == nil {
			if existing.UserID != request.UserID || existing.RuleKey != request.RuleKey || existing.Delta <= 0 {
				return ErrSourceKeyConflict
			}
			result = AwardResult{Entry: existing, AlreadyApplied: true}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if _, err := lockEligibleAccount(tx, request.UserID, s.policy()); err != nil {
			return err
		}

		localTime := request.OccurredAt.In(location)
		var ruleSet xpmodel.RuleSet
		err = tx.Where("effective_at <= ?", request.OccurredAt).Order("effective_at DESC").Take(&ruleSet).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRuleUnavailable
		}
		if err != nil {
			return err
		}
		if !ruleSet.Enabled || ruleSet.TimeZone != ShanghaiTimeZone {
			return ErrRuleUnavailable
		}
		var rule xpmodel.RewardRule
		if err := tx.Where("rule_set_version = ? AND rule_key = ? AND enabled = ?", ruleSet.Version, request.RuleKey, true).Take(&rule).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRuleUnavailable
		} else if err != nil {
			return err
		}
		if rule.RewardXP <= 0 {
			return ErrInvalidRule
		}
		amount := rule.RewardXP
		if rule.MaxPerEventXP != nil && amount > *rule.MaxPerEventXP {
			amount = *rule.MaxPerEventXP
		}
		if amount <= 0 || balance.CurrentXP > math.MaxInt64-amount {
			return ErrInvalidRule
		}

		entry := xpmodel.Entry{
			UserID: request.UserID, SourceKey: request.SourceKey,
			RuleSetVersion: ruleSet.Version, RuleKey: request.RuleKey,
			Delta: amount, OccurredAt: request.OccurredAt.UTC(),
		}
		insert := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_key"}}, DoNothing: true}).Create(&entry)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			if err := tx.Where("source_key = ?", request.SourceKey).Take(&existing).Error; err != nil {
				return err
			}
			if existing.UserID != request.UserID || existing.RuleKey != request.RuleKey || existing.Delta <= 0 {
				return ErrSourceKeyConflict
			}
			result = AwardResult{Entry: existing, AlreadyApplied: true}
			return nil
		}

		usageDate := localTime.Format("2006-01-02")
		ruleUsage, err := getDailyRuleUsage(tx, request.UserID, request.RuleKey, usageDate)
		if err != nil {
			return err
		}
		if rule.DailyCountLimit != nil && ruleUsage.UsedCount >= *rule.DailyCountLimit {
			return ErrRewardLimit
		}
		if ruleUsage.UsedCount == math.MaxInt64 {
			return ErrInvalidRule
		}
		ruleUsage.UsedCount++
		if err := saveUsage(tx, &ruleUsage); err != nil {
			return err
		}

		dailyUsage, err := getDailyUsage(tx, request.UserID, usageDate)
		if err != nil {
			return err
		}
		if amount > math.MaxInt64-dailyUsage.PositiveXP {
			return ErrInvalidRule
		}
		if ruleSet.DailyPositiveXPLimit != nil && amount > *ruleSet.DailyPositiveXPLimit-dailyUsage.PositiveXP {
			return ErrRewardLimit
		}
		dailyUsage.PositiveXP += amount
		if err := saveUsage(tx, &dailyUsage); err != nil {
			return err
		}

		balance.CurrentXP += amount
		if err := tx.Model(&xpmodel.Balance{}).Where("user_id = ?", request.UserID).UpdateColumn("current_xp", balance.CurrentXP).Error; err != nil {
			return err
		}
		result.Entry = entry
		return nil
	})
	return result, err
}

type AdjustmentRequest struct {
	UserID            uint64
	SourceKey         string
	Delta             int64
	Reason            string
	OccurredAt        time.Time
	ReversalOfEntryID uint64 // Required: the original positive award being reversed.
}

// PostAdjustment writes a human-authored negative reversal. The caller must
// pass a context marked by trusted authenticated and authorized server code;
// actor identity is never read from AdjustmentRequest. Reversals do not consume
// or refund positive-award quota and are capped by the original award amount.
func (s *Service) PostAdjustment(ctx context.Context, request AdjustmentRequest) (AwardResult, error) {
	actorUserID, ok := authenticatedActorID(ctx)
	if !ok {
		return AwardResult{}, ErrAuditActorRequired
	}
	return s.postAdjustment(ctx, request, actorUserID)
}

// PostSystemAdjustment is reserved for trusted server-side correction jobs.
// Its actor_user_id is zero to distinguish automated reversals from human
// actions; HTTP handlers must use PostAdjustment with authenticated context.
func (s *Service) PostSystemAdjustment(ctx context.Context, request AdjustmentRequest) (AwardResult, error) {
	return s.postAdjustment(ctx, request, 0)
}

func (s *Service) postAdjustment(ctx context.Context, request AdjustmentRequest, actorUserID uint64) (AwardResult, error) {
	if s == nil || s.db == nil || request.UserID == 0 || strings.TrimSpace(request.SourceKey) == "" || len(request.SourceKey) > 255 || request.Delta >= 0 || request.Delta == math.MinInt64 || strings.TrimSpace(request.Reason) == "" || request.ReversalOfEntryID == 0 {
		return AwardResult{}, ErrInvalidAdjustment
	}
	if request.OccurredAt.IsZero() {
		request.OccurredAt = time.Now().UTC()
	}
	request.SourceKey = strings.TrimSpace(request.SourceKey)
	request.Reason = strings.TrimSpace(request.Reason)
	var result AwardResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		balance, err := lockBalance(tx, request.UserID)
		if err != nil {
			return err
		}
		var existing xpmodel.Entry
		if err := tx.Where("source_key = ?", request.SourceKey).Take(&existing).Error; err == nil {
			if existing.UserID != request.UserID || existing.Delta != request.Delta || existing.ReversalOfEntry == nil || *existing.ReversalOfEntry != request.ReversalOfEntryID {
				return ErrSourceKeyConflict
			}
			result = AwardResult{Entry: existing, AlreadyApplied: true}
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := accountExists(tx, request.UserID); err != nil {
			return err
		}
		var original xpmodel.Entry
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", request.ReversalOfEntryID).Take(&original).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrOriginalAwardNotFound
		} else if err != nil {
			return err
		}
		if original.UserID != request.UserID || original.Delta <= 0 {
			return ErrReversalLimit
		}
		var alreadyReversed int64
		if err := tx.Model(&xpmodel.Entry{}).Select("COALESCE(SUM(-delta), 0)").Where("reversal_of_entry_id = ? AND delta < 0", original.ID).Scan(&alreadyReversed).Error; err != nil {
			return err
		}
		if alreadyReversed > original.Delta || -request.Delta > original.Delta-alreadyReversed {
			return ErrReversalLimit
		}
		if balance.CurrentXP < -request.Delta {
			return ErrNegativeBalance
		}
		entry := xpmodel.Entry{
			UserID: request.UserID, ActorUserID: actorUserID, SourceKey: request.SourceKey,
			Delta: request.Delta, OccurredAt: request.OccurredAt.UTC(), Reason: request.Reason,
			ReversalOfEntry: &request.ReversalOfEntryID,
		}
		entry.RuleKey = original.RuleKey
		entry.RuleSetVersion = original.RuleSetVersion
		insert := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_key"}}, DoNothing: true}).Create(&entry)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			if err := tx.Where("source_key = ?", request.SourceKey).Take(&existing).Error; err != nil {
				return err
			}
			if existing.UserID != request.UserID || existing.Delta != request.Delta || existing.ReversalOfEntry == nil || *existing.ReversalOfEntry != request.ReversalOfEntryID {
				return ErrSourceKeyConflict
			}
			result = AwardResult{Entry: existing, AlreadyApplied: true}
			return nil
		}
		balance.CurrentXP += request.Delta
		if err := tx.Model(&xpmodel.Balance{}).Where("user_id = ?", request.UserID).UpdateColumn("current_xp", balance.CurrentXP).Error; err != nil {
			return err
		}
		result.Entry = entry
		return nil
	})
	return result, err
}

func (s *Service) CurrentBalance(ctx context.Context, userID uint64) (int64, error) {
	var balance xpmodel.Balance
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Take(&balance).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	return balance.CurrentXP, err
}

func (s *Service) CurrentLevel(ctx context.Context, userID uint64) (Level, error) {
	var user users.EntityComplete
	if err := s.db.WithContext(ctx).Where("id = ?", userID).Take(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LevelSeedling, nil
		}
		return LevelSeedling, err
	}
	balance, err := s.CurrentBalance(ctx, userID)
	if err != nil {
		return LevelSeedling, err
	}
	return LevelForXP(balance, EligibleForXP(user, s.policy())), nil
}

type RuleSetSpec struct {
	Enabled              bool
	TimeZone             string
	DailyPositiveXPLimit *int64
	EffectiveAt          time.Time
	Rules                []xpmodel.RewardRule
}

// PublishRuleSet appends a new effective-dated ruleset. Published versions and
// their rules are immutable; disabling a policy requires publishing a disabled
// version instead of editing history. The publisher is resolved from trusted
// authenticated context, not from rule input.
func (s *Service) PublishRuleSet(ctx context.Context, spec RuleSetSpec) (uint64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("XP service is not configured")
	}
	publisherUserID, ok := authenticatedActorID(ctx)
	if !ok {
		return 0, ErrAuditActorRequired
	}
	if spec.TimeZone == "" {
		spec.TimeZone = ShanghaiTimeZone
	}
	if spec.TimeZone != ShanghaiTimeZone || spec.EffectiveAt.IsZero() || len(spec.Rules) == 0 || (spec.DailyPositiveXPLimit != nil && *spec.DailyPositiveXPLimit < 0) {
		return 0, ErrInvalidRule
	}
	seen := make(map[string]struct{}, len(spec.Rules))
	rules := append([]xpmodel.RewardRule(nil), spec.Rules...)
	for index := range rules {
		rule := &rules[index]
		rule.RuleKey = strings.TrimSpace(rule.RuleKey)
		if rule.RuleKey == "" || len(rule.RuleKey) > 64 || rule.RewardXP <= 0 || (rule.MaxPerEventXP != nil && *rule.MaxPerEventXP <= 0) || (rule.DailyCountLimit != nil && *rule.DailyCountLimit < 0) {
			return 0, ErrInvalidRule
		}
		if _, exists := seen[rule.RuleKey]; exists {
			return 0, ErrInvalidRule
		}
		seen[rule.RuleKey] = struct{}{}
	}
	for attempt := 0; ; attempt++ {
		version, err := s.publishRuleSetOnce(ctx, spec, rules, publisherUserID)
		if err == nil {
			return version, nil
		}
		if isEffectiveAtUniqueConflict(err) {
			return 0, ErrInvalidRule
		}
		if s.db.Name() != "sqlite" || !isSQLiteBusyOrLocked(err) || attempt+1 >= maxSQLiteRulePublishAttempts {
			return 0, err
		}
		delay := time.Duration(1<<attempt) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
}

func isEffectiveAtUniqueConflict(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_xp_rule_sets_effective_at") ||
		strings.Contains(message, "xp_rule_sets.effective_at")
}

func (s *Service) publishRuleSetOnce(ctx context.Context, spec RuleSetSpec, rules []xpmodel.RewardRule, publisherUserID uint64) (uint64, error) {
	var publishedVersion uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRulePublication(tx); err != nil {
			return err
		}
		var latest xpmodel.RuleSet
		err := tx.Order("version DESC").Take(&latest).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if latest.Version > 0 && !spec.EffectiveAt.After(latest.EffectiveAt) {
			return ErrInvalidRule
		}
		version := latest.Version + 1
		set := xpmodel.RuleSet{
			Version: version, PublishedBy: publisherUserID, Enabled: spec.Enabled,
			TimeZone: spec.TimeZone, DailyPositiveXPLimit: spec.DailyPositiveXPLimit,
			EffectiveAt: spec.EffectiveAt.UTC(),
		}
		if err := tx.Select("*").Create(&set).Error; err != nil {
			return err
		}
		for _, rule := range rules {
			rule.RuleSetVersion = version
			if err := tx.Select("*").Create(&rule).Error; err != nil {
				return err
			}
		}
		publishedVersion = version
		return nil
	})
	if err != nil {
		return 0, err
	}
	return publishedVersion, nil
}

func lockRulePublication(tx *gorm.DB) error {
	lock := xpmodel.RulePublicationLock{ID: 1}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoNothing: true}).Create(&lock).Error; err != nil {
		return fmt.Errorf("ensure XP rule publication lock: %w", err)
	}
	// A row update is intentional: PostgreSQL serializes on this row, while
	// SQLite has no FOR UPDATE row lock and instead obtains its single-writer
	// reservation here before the version read.
	result := tx.Model(&xpmodel.RulePublicationLock{}).Where("id = ?", lock.ID).UpdateColumn("lock_version", gorm.Expr("lock_version + 1"))
	if result.Error != nil {
		return fmt.Errorf("lock XP rule publication: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return errors.New("XP rule publication lock row was not updated")
	}
	return nil
}

func isSQLiteBusyOrLocked(err error) bool {
	var sqliteErr interface{ Code() int }
	if errors.As(err, &sqliteErr) {
		code := sqliteErr.Code() & 0xff
		return code == 5 || code == 6 // SQLITE_BUSY / SQLITE_LOCKED, including extended codes.
	}
	return false
}

// EnsureInitialRules seeds only the confirmed daily +5 XP policy. Other reward
// values and daily caps remain unset until the team decides them.
func EnsureInitialRules(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockRulePublication(tx); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&xpmodel.RuleSet{}).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return nil
		}
		set := xpmodel.RuleSet{Version: 1, PublishedBy: 0, Enabled: true, TimeZone: ShanghaiTimeZone, EffectiveAt: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}
		if err := tx.Create(&set).Error; err != nil {
			return err
		}
		five := int64(5)
		one := int64(1)
		return tx.Create(&xpmodel.RewardRule{RuleSetVersion: 1, RuleKey: DailyBaseRuleKey, Enabled: true, RewardXP: five, MaxPerEventXP: &five, DailyCountLimit: &one}).Error
	})
}

func lockBalance(tx *gorm.DB, userID uint64) (xpmodel.Balance, error) {
	zero := xpmodel.Balance{UserID: userID}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true}).Create(&zero).Error; err != nil {
		return xpmodel.Balance{}, err
	}
	var balance xpmodel.Balance
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).Take(&balance).Error
	return balance, err
}

func lockEligibleAccount(tx *gorm.DB, userID uint64, security pageConfig.SecurityAndRegistration) (users.EntityComplete, error) {
	var user users.EntityComplete
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", userID).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user, ErrIneligible
	}
	if err != nil {
		return user, err
	}
	if !EligibleForXP(user, security) {
		return user, ErrIneligible
	}
	return user, nil
}

func accountExists(tx *gorm.DB, userID uint64) error {
	var id uint64
	err := tx.Unscoped().Model(&users.EntityComplete{}).Select("id").Where("id = ?", userID).Take(&id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrIneligible
	}
	return err
}

func getDailyUsage(tx *gorm.DB, userID uint64, date string) (xpmodel.DailyUsage, error) {
	row := xpmodel.DailyUsage{UserID: userID, UsageDate: date}
	err := tx.Where("user_id = ? AND usage_date = ?", userID, date).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, nil
	}
	return row, err
}

func getDailyRuleUsage(tx *gorm.DB, userID uint64, ruleKey, date string) (xpmodel.DailyRuleUsage, error) {
	row := xpmodel.DailyRuleUsage{UserID: userID, RuleKey: ruleKey, UsageDate: date}
	err := tx.Where("user_id = ? AND rule_key = ? AND usage_date = ?", userID, ruleKey, date).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return row, nil
	}
	return row, err
}

func saveUsage(tx *gorm.DB, row any) error {
	result := tx.Save(row)
	return result.Error
}

func shanghaiLocation() (*time.Location, error) {
	return time.LoadLocation(ShanghaiTimeZone)
}
