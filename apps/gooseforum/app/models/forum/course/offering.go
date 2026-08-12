package course

import (
	"time"

	"gorm.io/gorm"
)

const offeringTableName = "course_offering"

// Entity 开课实例：一个学期中的真实开课，评价必须挂在 offering 而非 course。
type OfferingEntity struct {
	Id        uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	CourseId  uint64         `gorm:"column:course_id;not null;default:0;index:idx_course_offering_course;" json:"courseId"`
	TermId    uint64         `gorm:"column:term_id;not null;default:0;index:idx_course_offering_term;" json:"termId"`
	Campus    string         `gorm:"column:campus;type:varchar(64);not null;default:'';" json:"campus"`
	Faculty   string         `gorm:"column:faculty;type:varchar(255);not null;default:'';" json:"faculty"`
	Status    int8           `gorm:"column:status;not null;default:0;" json:"status"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

// 开课状态
const (
	OfferingStatusVisible int8 = 0 // 正常开课
	OfferingStatusHidden  int8 = 1 // 隐藏（对普通用户不可见）
)

func (itself *OfferingEntity) TableName() string {
	return offeringTableName
}
