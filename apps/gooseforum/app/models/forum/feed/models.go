package feed

import "time"

// Derived work is coalesced by topic; Version and Generation prevent stale
// workers from confirming newer dirtiness, including delete/recreate ABA.
type Schedule struct {
	ProjectionDirty bool      `gorm:"not null;default:false"`
	TopicID         uint64    `gorm:"primaryKey;autoIncrement:false"`
	Version         uint64    `gorm:"not null"`
	Generation      string    `gorm:"size:32;not null"`
	DueAt           time.Time `gorm:"not null;index"`
	Dirty           bool      `gorm:"not null;default:true"`
	Failures        int       `gorm:"not null;default:0"`
	LastError       string    `gorm:"size:256"`
}

func (Schedule) TableName() string { return "topic_rank_schedule" }

type ViewFact struct {
	TopicID   uint64    `gorm:"primaryKey;autoIncrement:false"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;index"`
	ViewedAt  time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (ViewFact) TableName() string { return "topic_view_fact" }

// CreditAt survives toggles during the cooldown. Active is the authoritative
// state of this credit; cancellation never refreshes its age.
type ActionCredit struct {
	TopicID   uint64    `gorm:"primaryKey;autoIncrement:false"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;index"`
	Kind      string    `gorm:"primaryKey;size:16"`
	CreditAt  time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	Active    bool      `gorm:"not null"`
}

func (ActionCredit) TableName() string { return "topic_action_credit" }

type Owner struct {
	ExpiresAt time.Time `gorm:"not null;index"`
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false"`
	Version   uint64    `gorm:"not null;default:0"`
	Purged    bool      `gorm:"not null;default:false"`
	Closed    bool      `gorm:"not null;default:false;index"`
}

func (Owner) TableName() string { return "feed_owner" }

type Serve struct {
	ID            string    `gorm:"primaryKey;size:32"`
	UserID        uint64    `gorm:"not null;index"`
	Feed          string    `gorm:"size:16;not null"`
	Hash          string    `gorm:"size:32;not null"`
	RankHash      string    `gorm:"size:32;not null"`
	Experiment    string    `gorm:"size:64;not null"`
	Variant       string    `gorm:"size:16;not null"`
	WeightVariant string    `gorm:"size:16;not null;default:default"`
	Capability    string    `gorm:"size:16;not null"`
	Items         string    `gorm:"type:text;not null"`
	CreatedAt     time.Time `gorm:"not null;index"`
	ExpiresAt     time.Time `gorm:"not null;index"`
	Applied       bool      `gorm:"not null;default:false;index"`
}

func (Serve) TableName() string { return "feed_serve_log" }

type Observation struct {
	ID             string    `gorm:"primaryKey;size:32"`
	UserID         uint64    `gorm:"not null;index"`
	Visible        uint32    `gorm:"not null;default:0"`
	Opened         uint32    `gorm:"not null;default:0"`
	Dwell          string    `gorm:"type:text;not null;default:'{}'"`
	AppliedVisible uint32    `gorm:"not null;default:0"`
	AppliedOpened  uint32    `gorm:"not null;default:0"`
	AppliedDwell   string    `gorm:"type:text;not null;default:'{}'"`
	CreatedAt      time.Time `gorm:"not null"`
	ExpiresAt      time.Time `gorm:"not null;index"`
}

func (Observation) TableName() string { return "feed_page_observation" }

// Event is reliable transaction-bound work, not another outbox. One row per
// committed native state transition; no post body or session credential.
type Event struct {
	ID            string    `gorm:"primaryKey;size:32"`
	UserID        uint64    `gorm:"not null;index"`
	TopicID       uint64    `gorm:"not null"`
	ObjectID      uint64    `gorm:"not null"`
	Kind          string    `gorm:"size:24;not null"`
	SourceVersion uint64    `gorm:"not null"`
	Source        string    `gorm:"size:16;not null;default:unattributed"`
	Active        bool      `gorm:"not null"`
	ServeID       string    `gorm:"size:32"`
	Position      int       `gorm:"not null;default:-1"`
	CreatedAt     time.Time `gorm:"not null;index"`
	ExpiresAt     time.Time `gorm:"not null;index"`
	Applied       bool      `gorm:"not null;default:false;index"`
}

func (Event) TableName() string { return "feed_event_log" }

type Assignment struct {
	Experiment string    `gorm:"primaryKey;size:64"`
	UserID     uint64    `gorm:"primaryKey;autoIncrement:false;index"`
	Variant    string    `gorm:"size:16;not null"`
	Hash       string    `gorm:"size:32;not null"`
	CreatedAt  time.Time `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null;index"`
}

func (Assignment) TableName() string { return "feed_experiment_assignment" }

type UserDaily struct {
	UserID        uint64    `gorm:"primaryKey;autoIncrement:false;index"`
	Day           string    `gorm:"primaryKey;size:10"`
	Active        bool      `gorm:"not null;default:false"`
	Likes         int       `gorm:"not null;default:0"`
	Bookmarks     int       `gorm:"not null;default:0"`
	PublicTopics  int       `gorm:"not null;default:0"`
	PublicReplies int       `gorm:"not null;default:0"`
	ExpiresAt     time.Time `gorm:"not null;index"`
}

func (UserDaily) TableName() string { return "feed_user_daily" }

type CandidateSample struct {
	ID         string    `gorm:"primaryKey;size:32"`
	UserID     uint64    `gorm:"not null;index"`
	Hash       string    `gorm:"size:32;not null"`
	Candidates string    `gorm:"type:text;not null"`
	Seed       uint64    `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
	ExpiresAt  time.Time `gorm:"not null;index"`
}

func (CandidateSample) TableName() string { return "feed_candidate_sample" }

type RankSnapshot struct {
	ID        string `gorm:"primaryKey;size:32"`
	Hash      string `gorm:"size:32"`
	Items     string `gorm:"type:text"`
	CreatedAt time.Time
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (RankSnapshot) TableName() string { return "feed_rank_snapshot" }

// Only bounded aggregate dimensions survive raw expiry. No viewer or topic ID.
type MetricsDaily struct {
	WeightVariant string `gorm:"primaryKey;size:16;not null;default:default"`
	Capability    string `gorm:"primaryKey;size:16"`
	RankHash      string `gorm:"primaryKey;size:32"`
	Day           string `gorm:"primaryKey;size:10"`
	Feed          string `gorm:"primaryKey;size:16"`
	Hash          string `gorm:"primaryKey;size:32"`
	Experiment    string `gorm:"primaryKey;size:64"`
	Variant       string `gorm:"primaryKey;size:16"`
	Metric        string `gorm:"primaryKey;size:32"`
	Count         int64  `gorm:"not null;default:0"`
}

func (MetricsDaily) TableName() string { return "feed_metrics_daily" }

type ParamsVersion struct {
	Hash      string `gorm:"primaryKey;size:32"`
	Params    string `gorm:"type:text;not null"`
	CreatedAt time.Time
}

func (ParamsVersion) TableName() string { return "feed_params_version" }

type State struct {
	Key   string `gorm:"primaryKey;size:64"`
	Value string `gorm:"type:text;not null"`
}

func (State) TableName() string { return "feed_state" }

func Models() []any {
	return []any{&SeenState{}, &NewTopicOutcome{}, &ActorWork{}, &Schedule{}, &ViewFact{}, &ActionCredit{}, &Owner{}, &Serve{}, &Observation{}, &Event{}, &Assignment{}, &UserDaily{}, &CandidateSample{}, &RankSnapshot{}, &MetricsDaily{}, &ParamsVersion{}, &State{}, &ActionResult{}, &ExperimentPeriod{}, &PeriodProgress{}}
}

// RawTables is shared by cleanup/closure/export. Deployment backups use the
// same list (deploy/scripts/feed-raw-tables.json); its freshness is tested.
var RawTables = []string{"feed_seen_state", "feed_new_topic_outcome", "feed_actor_work", "feed_period_progress", "feed_action_result", "feed_owner", "topic_view_fact", "topic_action_credit", "feed_serve_log", "feed_page_observation", "feed_event_log", "feed_experiment_assignment", "feed_user_daily", "feed_candidate_sample", "feed_rank_snapshot"}

// ActionResult holds compensation state only while the original event can be
// retained. The first activation date is preserved through repeat toggles.
type ActionResult struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false;index"`
	ObjectID  uint64 `gorm:"primaryKey;autoIncrement:false"`
	Kind      string `gorm:"primaryKey;size:24"`
	Version   uint64
	ServeID   string `gorm:"size:32"`
	Position  int
	Source    string `gorm:"size:16"`
	Active    bool
	Day       string    `gorm:"size:10"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (ActionResult) TableName() string { return "feed_action_result" }

// Period and its final result contain anonymous sufficient statistics. Cursor
// state lives separately and expires with the raw experiment records.
type ExperimentPeriod struct {
	ID                string `gorm:"primaryKey;size:64"`
	Hash              string `gorm:"size:32"`
	StartedAt         time.Time
	EnrollUntil       time.Time
	AnalyzeAt         time.Time
	Aborted           string `gorm:"size:128"`
	AssignedControl   int64
	AssignedTreatment int64
	ClosedControl     int64
	ClosedTreatment   int64
	Result            string `gorm:"type:text"`
}

func (ExperimentPeriod) TableName() string { return "feed_experiment_period" }

type PeriodProgress struct {
	ID        string `gorm:"primaryKey;size:64"`
	Cursor    uint64
	Stats     string    `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (PeriodProgress) TableName() string { return "feed_period_progress" }

// ActorWork invalidates contributor projections in bounded owner queries.
// Version resets a concurrent walk when identity eligibility changes again.
type ActorWork struct {
	UserID    uint64 `gorm:"primaryKey;autoIncrement:false"`
	Version   uint64
	Phase     int
	Cursor    uint64
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (ActorWork) TableName() string { return "feed_actor_work" }

// NewTopicOutcome is a 30-day publication cohort watermark, without reader IDs.
// UserID is the author and allows account closure to purge the topic watermark.
type NewTopicOutcome struct {
	TopicID   uint64 `gorm:"primaryKey;autoIncrement:false"`
	UserID    uint64 `gorm:"not null;index"`
	Hash      string `gorm:"size:32"`
	RankHash  string `gorm:"size:32"`
	PublicAt  time.Time
	Visible   bool
	Replied   bool
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (NewTopicOutcome) TableName() string { return "feed_new_topic_outcome" }
