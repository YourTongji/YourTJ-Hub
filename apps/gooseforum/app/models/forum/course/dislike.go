package course

import (
	"time"

	"gorm.io/gorm"
)

const dislikeTableName = "course_review_dislike"

// Entity 评价的 dislike 标记：唯一约束 (review_id, user_id)，不使用 IP hash。
// 与 helpful 表同构（登录用户唯一索引，软删 + 物理删除恢复生命周期）。
type DislikeEntity struct {
	ReviewId  uint64         `gorm:"primaryKey;column:review_id;not null;index:idx_course_review_dislike_user;" json:"reviewId"`
	UserId    uint64         `gorm:"primaryKey;column:user_id;not null;" json:"userId"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (itself *DislikeEntity) TableName() string {
	return dislikeTableName
}
