package course

import (
	"time"

	"gorm.io/gorm"
)

const helpfulTableName = "course_review_helpful"

// Entity 评价的 helpful 标记：唯一约束 (review_id, user_id)，不使用 IP hash。
type HelpfulEntity struct {
	ReviewId  uint64         `gorm:"primaryKey;column:review_id;not null;index:idx_course_review_helpful_user;" json:"reviewId"`
	UserId    uint64         `gorm:"primaryKey;column:user_id;not null;" json:"userId"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (itself *HelpfulEntity) TableName() string {
	return helpfulTableName
}
