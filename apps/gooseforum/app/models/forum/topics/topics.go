package topics

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "topics"

type Entity struct {
	Id            uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_topics_list_default,priority:5,sort:desc;index:idx_topics_list_hot,priority:4,sort:desc;index:idx_topics_list_popular,priority:4,sort:desc;index:idx_topics_list_new,priority:4,sort:desc;index:idx_topics_admin_list,priority:3,sort:desc;index:idx_topics_admin_user_list,priority:4,sort:desc;" json:"id"`
	Title         string         `gorm:"column:title;type:varchar(512);not null;default:'';" json:"title"`
	CategoryIds   []uint64       `gorm:"column:category_id;type:varchar(255);not null;default:'[]';serializer:json" json:"categoryIds"`
	UserId        uint64         `gorm:"column:user_id;not null;default:0;index:idx_topics_user_status,priority:1;index:idx_topics_admin_user_list,priority:1;" json:"userId"`
	Status        int8           `gorm:"column:status;not null;default:0;index:idx_topics_user_status,priority:2;index:idx_topics_list_default,priority:1;index:idx_topics_list_hot,priority:1;index:idx_topics_list_popular,priority:1;index:idx_topics_list_new,priority:1;" json:"status"`
	ProcessStatus int8           `gorm:"column:process_status;not null;default:0;index:idx_topics_user_status,priority:3;index:idx_topics_list_default,priority:2;index:idx_topics_list_hot,priority:2;index:idx_topics_list_popular,priority:2;index:idx_topics_list_new,priority:2;" json:"processStatus"`
	TopicType     int8           `gorm:"column:topic_type;not null;default:0;index:idx_topics_type_status,priority:1;" json:"topicType"`
	LikeCount     uint64         `gorm:"column:like_count;not null;default:0;" json:"likeCount"`
	ViewCount     uint64         `gorm:"column:view_count;not null;default:0;index:idx_topics_list_popular,priority:3,sort:desc;" json:"viewCount"`
	PostCount     uint64         `gorm:"column:post_count;not null;default:0;" json:"postCount"`
	ReplyCount    uint64         `gorm:"column:reply_count;not null;default:0;index:idx_topics_list_hot,priority:3,sort:desc;" json:"replyCount"`
	PostSeq       uint64         `gorm:"column:post_seq;not null;default:0;" json:"postSeq"`
	FirstPostId   uint64         `gorm:"column:first_post_id;not null;default:0;" json:"firstPostId"`
	LastPostId    uint64         `gorm:"column:last_post_id;not null;default:0;" json:"lastPostId"`
	LastPostedAt  *time.Time     `gorm:"column:last_posted_at;index:idx_topics_last_posted,sort:desc;" json:"lastPostedAt"`
	PinWeight     int            `gorm:"column:pin_weight;type:int;not null;default:0;index:idx_topics_list_default,priority:3,sort:desc;index:idx_topics_admin_list,priority:1,sort:desc;index:idx_topics_admin_user_list,priority:2,sort:desc;" json:"pinWeight"`
	Excerpt       string         `gorm:"column:excerpt;type:varchar(255);not null;default:'';" json:"excerpt"`
	FirstImageURL string         `gorm:"column:first_image_url;type:varchar(512);not null;default:'';" json:"firstImageUrl"`
	ImageUrls     []string       `gorm:"column:image_urls;type:text;serializer:json" json:"imageUrls"`
	Posters       []Poster       `gorm:"column:posters;type:text;serializer:json" json:"posters"`
	CreatedAt     time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;index:idx_topics_list_new,priority:3,sort:desc;" json:"createdAt"`
	UpdatedAt     time.Time      `gorm:"column:updated_at;autoUpdateTime;index:idx_topics_list_default,priority:4,sort:desc;index:idx_topics_admin_list,priority:2,sort:desc;index:idx_topics_admin_user_list,priority:3,sort:desc;" json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-"`

	// 删除生命周期状态（visibility_status × retention_status）
	VisibilityStatus string `gorm:"column:visibility_status;type:varchar(32);not null;default:'ACTIVE';index:idx_topics_visibility_retention,priority:1;" json:"-"`
	RetentionStatus  string `gorm:"column:retention_status;type:varchar(32);not null;default:'NORMAL';index:idx_topics_visibility_retention,priority:2;" json:"-"`
	DeletedBy        uint64 `gorm:"column:deleted_by;not null;default:0;" json:"-"`
	DeleteReason     string `gorm:"column:delete_reason;type:varchar(512);not null;default:'';" json:"-"`
}

// 管理处理状态
const (
	ProcessStatusNormal  int8 = 0 // 正常
	ProcessStatusBlocked int8 = 1 // 封禁
	ProcessStatusPending int8 = 2 // 待审（敏感词转人工审核）
)

// 话题类型（topic_type）：与论坛 feed 隔离正交。
const (
	TopicTypeForum int8 = 0 // 论坛普通话题（默认）
	TopicTypeWiki  int8 = 1 // wiki 分站页面
)

// 可见性状态（visibility_status）：与 process_status（封禁/待审）正交。
// 封禁=内容仍在库但不可见（可逆）；删除=内容进入删除生命周期。
const (
	VisibilityActive            = "ACTIVE"
	VisibilityUserDeleted       = "USER_DELETED"       // 作者本人删除，进入 30 天恢复窗口
	VisibilityModeratorRemoved  = "MODERATOR_REMOVED"  // 管理员/版主治理删除
	VisibilityAccountAnonymized = "ACCOUNT_ANONYMIZED" // 账号注销联动匿名化
)

// 保留状态（retention_status）：决定数据在删除后保留多久、由谁访问。
const (
	RetentionNormal       = "NORMAL"        // 正常生命周期
	RetentionRecoverable  = "RECOVERABLE"   // 恢复窗口期（默认 30 天），仅作者本人可恢复
	RetentionEvidenceHold = "EVIDENCE_HOLD" // 存在举报/审核证据，保留证据副本
	RetentionLegalHold    = "LEGAL_HOLD"    // 法律保存要求，覆盖普通 TTL
	RetentionPurged       = "PURGED"        // 已永久删除，仅审计可查
)

type Poster struct {
	UserID uint64 `json:"user_id"`
}

func (itself *Entity) TableName() string {
	return tableName
}

func (itself *Entity) GetPosters() []Poster {
	if len(itself.Posters) == 0 {
		return []Poster{{UserID: itself.UserId}}
	}
	return itself.Posters
}
