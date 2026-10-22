// Package xp owns the forum-local XP projection, immutable ledger, and
// versioned reward policy tables. XP is independent from legacy Credit/points.
package xp

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrImmutableRecord = errors.New("xp record is append-only")

type Balance struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:user_id"`
	CurrentXP int64     `gorm:"type:bigint;not null;default:0;column:current_xp;check:chk_xp_balances_nonnegative,current_xp >= 0"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime;<-:create"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (Balance) TableName() string { return "xp_balances" }

// Entry is an immutable XP ledger row. Corrections and reversals are new rows;
// the source key is global so one business fact can settle only once.
type Entry struct {
	ID              uint64    `gorm:"primaryKey;autoIncrement;not null;column:id"`
	UserID          uint64    `gorm:"not null;index:idx_xp_entries_user_time,priority:1;column:user_id"`
	ActorUserID     uint64    `gorm:"not null;default:0;index:idx_xp_entries_actor,priority:1;column:actor_user_id"`
	SourceKey       string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_xp_entries_source_key;column:source_key"`
	RuleSetVersion  uint64    `gorm:"not null;default:0;column:rule_set_version"`
	RuleKey         string    `gorm:"type:varchar(64);not null;default:'';column:rule_key"`
	Delta           int64     `gorm:"type:bigint;not null;column:delta"`
	OccurredAt      time.Time `gorm:"not null;index:idx_xp_entries_user_time,priority:2;column:occurred_at"`
	ReversalOfEntry *uint64   `gorm:"index;column:reversal_of_entry_id"`
	Reason          string    `gorm:"type:text;not null;default:'';column:reason"`
	CreatedAt       time.Time `gorm:"not null;autoCreateTime;<-:create"`
}

func (Entry) TableName() string { return "xp_entries" }

func (*Entry) BeforeUpdate(*gorm.DB) error { return ErrImmutableRecord }
func (*Entry) BeforeDelete(*gorm.DB) error { return ErrImmutableRecord }

// RuleSet is an immutable, effective-dated reward policy version. A disabled
// version pauses positive rewards from EffectiveAt until the next version.
type RuleSet struct {
	Version              uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:version"`
	PublishedBy          uint64    `gorm:"not null;default:0;index:idx_xp_rule_sets_publisher;column:published_by"`
	Enabled              bool      `gorm:"not null;column:enabled"`
	TimeZone             string    `gorm:"type:varchar(64);not null;default:'Asia/Shanghai';column:time_zone"`
	DailyPositiveXPLimit *int64    `gorm:"type:bigint;column:daily_positive_xp_limit"`
	EffectiveAt          time.Time `gorm:"not null;uniqueIndex:idx_xp_rule_sets_effective_at;column:effective_at"`
	CreatedAt            time.Time `gorm:"not null;autoCreateTime;<-:create"`
}

func (RuleSet) TableName() string { return "xp_rule_sets" }

func (*RuleSet) BeforeUpdate(*gorm.DB) error { return ErrImmutableRecord }
func (*RuleSet) BeforeDelete(*gorm.DB) error { return ErrImmutableRecord }

// RulePublicationLock is mutable coordination state, not policy history. A
// single row is written before rule publication reads the current version so
// PostgreSQL and SQLite serialize publishers consistently.
type RulePublicationLock struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement:false;not null;column:id"`
	LockVersion uint64 `gorm:"not null;default:0;column:lock_version"`
}

func (RulePublicationLock) TableName() string { return "xp_rule_publication_locks" }

// RewardRule is one immutable event rule inside a rule-set version. Nil daily
// limits are unbounded; zero is a deliberate deny-all limit.
type RewardRule struct {
	RuleSetVersion  uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:rule_set_version"`
	RuleKey         string    `gorm:"primaryKey;type:varchar(64);not null;column:rule_key"`
	Enabled         bool      `gorm:"not null;column:enabled"`
	RewardXP        int64     `gorm:"type:bigint;not null;column:reward_xp"`
	MaxPerEventXP   *int64    `gorm:"type:bigint;column:max_per_event_xp"`
	DailyCountLimit *int64    `gorm:"type:bigint;column:daily_count_limit"`
	CreatedAt       time.Time `gorm:"not null;autoCreateTime;<-:create"`
}

func (RewardRule) TableName() string { return "xp_reward_rules" }

func (*RewardRule) BeforeUpdate(*gorm.DB) error { return ErrImmutableRecord }
func (*RewardRule) BeforeDelete(*gorm.DB) error { return ErrImmutableRecord }

type DailyUsage struct {
	UserID     uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:user_id"`
	UsageDate  string    `gorm:"primaryKey;type:varchar(10);not null;column:usage_date"`
	PositiveXP int64     `gorm:"type:bigint;not null;default:0;column:positive_xp;check:chk_xp_daily_usage_nonnegative,positive_xp >= 0"`
	CreatedAt  time.Time `gorm:"not null;autoCreateTime;<-:create"`
	UpdatedAt  time.Time `gorm:"not null;autoUpdateTime"`
}

func (DailyUsage) TableName() string { return "xp_daily_usage" }

type DailyRuleUsage struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;not null;column:user_id"`
	RuleKey   string    `gorm:"primaryKey;type:varchar(64);not null;column:rule_key"`
	UsageDate string    `gorm:"primaryKey;type:varchar(10);not null;column:usage_date"`
	UsedCount int64     `gorm:"type:bigint;not null;default:0;column:used_count;check:chk_xp_daily_rule_usage_nonnegative,used_count >= 0"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime;<-:create"`
	UpdatedAt time.Time `gorm:"not null;autoUpdateTime"`
}

func (DailyRuleUsage) TableName() string { return "xp_daily_rule_usage" }

func Models() []any {
	return []any{
		&Balance{},
		&Entry{},
		&RuleSet{},
		&RulePublicationLock{},
		&RewardRule{},
		&DailyUsage{},
		&DailyRuleUsage{},
	}
}
