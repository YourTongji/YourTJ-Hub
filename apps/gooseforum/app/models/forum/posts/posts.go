package posts

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "posts"

// 管理处理状态
const (
	ProcessStatusNormal  int8 = 0 // 正常
	ProcessStatusBlocked int8 = 1 // 封禁
	ProcessStatusPending int8 = 2 // 待审（敏感词转人工审核）
)

// 可见性/保留状态常量，与 topics 包保持一致语义。
const (
	VisibilityActive            = "ACTIVE"
	VisibilityUserDeleted       = "USER_DELETED"
	VisibilityModeratorRemoved  = "MODERATOR_REMOVED"
	VisibilityAccountAnonymized = "ACCOUNT_ANONYMIZED"
)

const (
	RetentionNormal       = "NORMAL"
	RetentionRecoverable  = "RECOVERABLE"
	RetentionEvidenceHold = "EVIDENCE_HOLD"
	RetentionLegalHold    = "LEGAL_HOLD"
	RetentionPurged       = "PURGED"
)

type Entity struct {
	Id              uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_posts_topic_id,priority:2;" json:"id"`
	TopicId         uint64         `gorm:"column:topic_id;not null;default:0;index:idx_posts_topic_created,priority:1;uniqueIndex:idx_posts_topic_no,priority:1;index:idx_posts_topic_id,priority:1;index:idx_posts_topic_process,priority:1;" json:"topicId"`
	PostNo          uint64         `gorm:"column:post_no;not null;default:0;uniqueIndex:idx_posts_topic_no,priority:2;" json:"postNo"`
	UserId          uint64         `gorm:"column:user_id;not null;default:0;index;" json:"userId"`
	ReplyToPostId   uint64         `gorm:"column:reply_to_post_id;not null;default:0;" json:"replyToPostId"`
	Content         string         `gorm:"column:content;type:text;" json:"content"`
	RenderedHTML    string         `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	RenderedVersion uint32         `gorm:"column:rendered_version;not null;default:0;" json:"renderedVersion"`
	ProcessStatus   int8           `gorm:"column:process_status;not null;default:0;index:idx_posts_topic_process,priority:2;" json:"processStatus"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;index:idx_posts_topic_created,priority:2;" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-"`

	// 删除生命周期状态（visibility_status × retention_status）
	VisibilityStatus string `gorm:"column:visibility_status;type:varchar(32);not null;default:'ACTIVE';index:idx_posts_visibility_retention,priority:1;" json:"-"`
	RetentionStatus  string `gorm:"column:retention_status;type:varchar(32);not null;default:'NORMAL';index:idx_posts_visibility_retention,priority:2;" json:"-"`
	DeletedBy        uint64 `gorm:"column:deleted_by;not null;default:0;" json:"-"`
	DeleteReason     string `gorm:"column:delete_reason;type:varchar(512);not null;default:'';" json:"-"`
}

func (itself *Entity) TableName() string {
	return tableName
}
