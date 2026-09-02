package course

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "course"

// Entity 课程目录中的 canonical course：一个实体代表一门课程的一个
// (primary_code, teacher) 身份（对齐上游 Serverless 的 code+teacher 分组）。
// teacher_id = 0 表示无教师课程；历史课号/简称等进入 course_alias。
type Entity struct {
	Id             uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PrimaryCode    string         `gorm:"column:primary_code;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_code_teacher,priority:1;" json:"primaryCode"`
	TeacherId      uint64         `gorm:"column:teacher_id;not null;default:0;uniqueIndex:uniq_course_code_teacher,priority:2;index:idx_course_teacher;" json:"teacherId"`
	Name           string         `gorm:"column:name;type:varchar(255);not null;default:'';" json:"name"`
	Department     string         `gorm:"column:department;type:varchar(255);not null;default:'';" json:"department"`
	CreditX10      int            `gorm:"column:credit_x10;not null;default:0;" json:"creditX10"`
	NormalizedName string         `gorm:"column:normalized_name;type:varchar(255);not null;default:'';index:idx_course_normalized_name;" json:"normalizedName"`
	NamePinyin     string         `gorm:"column:name_pinyin;type:varchar(255);not null;default:'';index:idx_course_name_pinyin;" json:"namePinyin"`
	NameInitials   string         `gorm:"column:name_initials;type:varchar(64);not null;default:'';index:idx_course_name_initials;" json:"nameInitials"`
	ReviewScope    string         `gorm:"column:review_scope;type:varchar(16);not null;default:'teacher';index:idx_course_review_scope;" json:"reviewScope"`
	TeamKey        string         `gorm:"column:team_key;type:varchar(64);not null;default:'';index:idx_course_team_key;" json:"teamKey"`
	Status         int8           `gorm:"column:status;not null;default:0;index:idx_course_status;" json:"status"`
	SearchVersion  uint64         `gorm:"column:search_version;not null;default:0;" json:"searchVersion"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt      gorm.DeletedAt `json:"-"`
}

// 课程状态
const (
	StatusVisible int8 = 0 // 可见
	StatusHidden  int8 = 1 // 隐藏（对普通用户不可见，保留数据与关联）
)

func (itself *Entity) TableName() string {
	return tableName
}
