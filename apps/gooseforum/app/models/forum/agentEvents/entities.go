// Package agentEvents owns immutable source occurrences, publication watermarks,
// and the per-Agent committed event stream. Content bodies stay in post revisions.
package agentEvents

import "time"

type Publication struct {
	InstanceID string `gorm:"primaryKey;size:128;not null"`
	PostID     uint64 `gorm:"primaryKey;autoIncrement:false;not null"`
	Version    uint64 `gorm:"not null"`
}

func (Publication) TableName() string { return "agent_publications" }

type Recipient struct {
	AgentID                uint64   `json:"agentId"`
	Reasons                []string `json:"reasons"`
	SubscriptionGeneration uint64   `json:"subscriptionGeneration"`
	EndpointGeneration     uint64   `json:"endpointGeneration"`
}

type Intent struct {
	ID              string      `gorm:"primaryKey;size:64;not null" json:"id"`
	InstanceID      string      `gorm:"size:128;not null;uniqueIndex:idx_agent_intent_version,priority:1,where:post_id > 0;index" json:"instanceId"`
	PostID          uint64      `gorm:"not null;uniqueIndex:idx_agent_intent_version,priority:2;index" json:"postId"`
	Version         uint64      `gorm:"not null;uniqueIndex:idx_agent_intent_version,priority:3" json:"version"`
	PreviousVersion uint64      `gorm:"not null" json:"previousVersion"`
	ActorID         uint64      `gorm:"not null" json:"-"`
	Recipients      []Recipient `gorm:"type:text;serializer:json;not null" json:"-"`
	TaskID          uint64      `gorm:"not null" json:"taskId"`
	Status          string      `gorm:"size:32;not null;default:'pending';index" json:"status"`
	LastError       string      `gorm:"size:128;not null;default:''" json:"lastError"`
	CreatedAt       time.Time   `gorm:"not null;autoCreateTime" json:"createdAt"`
	ExpiresAt       time.Time   `gorm:"not null;index" json:"expiresAt"`
}

func (Intent) TableName() string { return "agent_interaction_intents" }

type Entity struct {
	ID                     string     `gorm:"primaryKey;size:80;not null" json:"id"`
	InstanceID             string     `gorm:"size:128;not null;uniqueIndex:idx_agent_event_seq,priority:1;uniqueIndex:idx_agent_event_source,priority:1" json:"instanceId"`
	AgentID                uint64     `gorm:"not null;uniqueIndex:idx_agent_event_seq,priority:2;uniqueIndex:idx_agent_event_source,priority:3;index" json:"agentId"`
	Seq                    uint64     `gorm:"not null;uniqueIndex:idx_agent_event_seq,priority:3" json:"-"`
	SourceIntentID         string     `gorm:"size:64;not null;uniqueIndex:idx_agent_event_source,priority:2;index" json:"sourceIntentId"`
	SubscriptionGeneration uint64     `gorm:"not null" json:"-"`
	Type                   string     `gorm:"size:64;not null" json:"type"`
	TopicID                uint64     `gorm:"not null;index" json:"-"`
	PostID                 uint64     `gorm:"not null;index" json:"-"`
	PostNo                 uint64     `gorm:"not null" json:"-"`
	ReplyToPostID          uint64     `gorm:"not null" json:"-"`
	ActorID                uint64     `gorm:"not null" json:"-"`
	Reasons                []string   `gorm:"type:text;serializer:json;not null" json:"-"`
	OccurredAt             time.Time  `gorm:"not null" json:"occurredAt"`
	ExpiresAt              time.Time  `gorm:"not null;index" json:"expiresAt"`
	AckedAt                *time.Time `json:"ackedAt,omitempty"`
	WithdrawnAt            *time.Time `json:"withdrawnAt,omitempty"`
	ResultingTopicID       uint64     `gorm:"not null;default:0" json:"resultingTopicId,omitempty"`
	ResultingPostID        uint64     `gorm:"not null;default:0" json:"resultingPostId,omitempty"`
}

func (Entity) TableName() string { return "agent_events" }

// ReplayState retains one compact watermark per recipient after expired event
// diagnostics are purged. A missing page must not reset the replay floor to zero.
type ReplayState struct {
	InstanceID    string `gorm:"primaryKey;size:128;not null"`
	AgentID       uint64 `gorm:"primaryKey;autoIncrement:false;not null"`
	PurgedThrough uint64 `gorm:"not null"`
}

func (ReplayState) TableName() string { return "agent_event_replay_states" }
