package course

import (
	"time"

	"gorm.io/gorm"
)

const reviewTableName = "course_review"

// Entity 课程评价：挂在 offering 上；原生评价 author_user_id 必填，
// 导入的历史匿名评价允许为 NULL；rating 为 1..5，历史 0 转 NULL 不计平均。
type ReviewEntity struct {
	Id                 uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	OfferingId         uint64         `gorm:"column:offering_id;not null;default:0;index:idx_course_review_offering;" json:"offeringId"`
	AuthorUserId       uint64         `gorm:"column:author_user_id;not null;default:0;index:idx_course_review_author;" json:"authorUserId"`
	Rating             *int           `gorm:"column:rating;type:int;" json:"rating"`
	Content            string         `gorm:"column:content;type:text;not null;default:'';" json:"content"`
	IsAnonymous        bool           `gorm:"column:is_anonymous;not null;default:false;" json:"isAnonymous"`
	Status             int8           `gorm:"column:status;not null;default:0;index:idx_course_review_status;" json:"status"`
	LegacyHelpfulCount int            `gorm:"column:legacy_helpful_count;not null;default:0;" json:"legacyHelpfulCount"`
	Source             string         `gorm:"column:source;type:varchar(64);not null;default:'';" json:"source"`
	CreatedAt          time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt          time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt          gorm.DeletedAt `json:"-"`
}

// 评价状态
const (
	ReviewStatusVisible int8 = 0 // 可见
	ReviewStatusHidden  int8 = 1 // 隐藏（对普通用户 404）
	ReviewStatusDeleted int8 = 2 // 删除（隔离窗口后清理正文与作者关联）
)

// 评价来源（source 列）：原生评价留空，历史导入固定为 legacy-import。
const ReviewSourceLegacyImport string = "legacy-import"

func (itself *ReviewEntity) TableName() string {
	return reviewTableName
}
