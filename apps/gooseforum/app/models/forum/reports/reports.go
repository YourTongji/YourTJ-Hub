package reports

import "time"

const tableName = "reports"

const (
	TargetTopic        = "topic"
	TargetPost         = "post"
	TargetCourseReview = "course_review"
)

const (
	ReasonSpam       = "spam"
	ReasonAbuse      = "abuse"
	ReasonIllegal    = "illegal"
	ReasonIrrelevant = "irrelevant"
	ReasonOther      = "other"
)

const (
	StatusOpen     = "open"
	StatusResolved = "resolved"
	StatusRejected = "rejected"
)

const (
	ResolutionBanned  = "banned"
	ResolutionIgnored = "ignored"
)

const (
	fieldTargetType = "target_type"
	fieldTargetId   = "target_id"
	fieldReporterId = "reporter_id"
	fieldStatus     = "status"
)

// EvidenceSnapshotData 举报时刻的目标内容快照（Issue #94 R6）。
// 举报一经创建就定格目标内容，即使作者随后删除内容，审核仍能基于快照进行，
// 封堵"删帖逃罚"。快照只保留最小必要字段，不含附件二进制。
type EvidenceSnapshotData struct {
	TargetType  string    `json:"targetType"`
	TargetID    uint64    `json:"targetId"`
	TopicID     uint64    `json:"topicId,omitempty"`
	Title       string    `json:"title,omitempty"`
	Excerpt     string    `json:"excerpt,omitempty"`
	AuthorID    uint64    `json:"authorId,omitempty"`
	AuthorName  string    `json:"authorName,omitempty"`
	CategoryIDs []uint64  `json:"categoryIds,omitempty"`
	TargetURL   string    `json:"targetUrl,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Entity struct {
	Id         uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_reports_status_id,priority:2,sort:desc;index:idx_reports_status_topic_id,priority:3,sort:desc;" json:"id"`
	TargetType string     `gorm:"column:target_type;type:varchar(32);not null;default:'';index:idx_reports_target,priority:1;index:idx_reports_reporter_target_status,priority:2;" json:"targetType"`
	TargetId   uint64     `gorm:"column:target_id;not null;default:0;index:idx_reports_target,priority:2;index:idx_reports_reporter_target_status,priority:3;" json:"targetId"`
	TopicId    uint64     `gorm:"column:topic_id;not null;default:0;index:idx_reports_status_topic_id,priority:2;" json:"topicId"`
	ReporterId uint64     `gorm:"column:reporter_id;not null;default:0;index:idx_reports_reporter_target_status,priority:1;" json:"reporterId"`
	Reason     string     `gorm:"column:reason;type:varchar(32);not null;default:'';" json:"reason"`
	Note       string     `gorm:"column:note;type:varchar(300);not null;default:'';" json:"note"`
	Status     string     `gorm:"column:status;type:varchar(32);not null;default:'open';index:idx_reports_status_id,priority:1;index:idx_reports_status_topic_id,priority:1;index:idx_reports_reporter_target_status,priority:4;" json:"status"`
	Resolution string     `gorm:"column:resolution;type:varchar(32);not null;default:'';" json:"resolution"`
	HandlerId  uint64     `gorm:"column:handler_id;not null;default:0;" json:"handlerId"`
	HandledAt  *time.Time `gorm:"column:handled_at;" json:"handledAt"`
	// EvidenceSnapshot 举报时刻的目标内容快照（JSON）。普通无违规内容删除时
	// 不额外创建独立快照记录，仅举报行内保留此定格数据。
	EvidenceSnapshot EvidenceSnapshotData `gorm:"column:evidence_snapshot;type:json;serializer:json" json:"evidenceSnapshot,omitempty"`
	CreatedAt        time.Time            `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt        time.Time            `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
