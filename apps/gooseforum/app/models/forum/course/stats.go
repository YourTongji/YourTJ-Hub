package course

import (
	"time"

	"gorm.io/gorm"
)

const courseStatsTableName = "course_review_stats"

// Entity 课程级评价统计投影：保存 rating_sum/rating_count/review_count，
// 避免频繁扫描与浮点累计误差；可用 rebuild-course-stats 全量重建。
type CourseStatsEntity struct {
	CourseId    uint64         `gorm:"primaryKey;column:course_id;not null;" json:"courseId"`
	RatingCount int            `gorm:"column:rating_count;not null;default:0;" json:"ratingCount"`
	RatingSum   int            `gorm:"column:rating_sum;not null;default:0;" json:"ratingSum"`
	ReviewCount int            `gorm:"column:review_count;not null;default:0;" json:"reviewCount"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

func (itself *CourseStatsEntity) TableName() string {
	return courseStatsTableName
}
