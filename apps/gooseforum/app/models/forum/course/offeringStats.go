package course

import (
	"time"

	"gorm.io/gorm"
)

const offeringStatsTableName = "offering_review_stats"

// Entity 开课实例级评价统计投影：结构与课程级一致，按 offering 聚合。
type OfferingStatsEntity struct {
	OfferingId  uint64         `gorm:"primaryKey;column:offering_id;not null;" json:"offeringId"`
	RatingCount int            `gorm:"column:rating_count;not null;default:0;" json:"ratingCount"`
	RatingSum   int            `gorm:"column:rating_sum;not null;default:0;" json:"ratingSum"`
	ReviewCount int            `gorm:"column:review_count;not null;default:0;" json:"reviewCount"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

func (itself *OfferingStatsEntity) TableName() string {
	return offeringStatsTableName
}
