package course

import (
	"time"

	"gorm.io/gorm"
)

const aliasTableName = "course_alias"

// Entity 课程的别名/旧课号/简称：kind 为 code/name/abbreviation。
// 同一课程内 (kind, value) 唯一；跨课程冲突由 importer 检出并进入 quarantine。
type AliasEntity struct {
	Id              uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	CourseId        uint64         `gorm:"column:course_id;not null;default:0;index:idx_course_alias_course;" json:"courseId"`
	Kind            string         `gorm:"column:kind;type:varchar(32);not null;default:'';uniqueIndex:uniq_course_alias_kind_value,priority:1;" json:"kind"`
	Value           string         `gorm:"column:value;type:varchar(255);not null;default:'';" json:"value"`
	NormalizedValue string         `gorm:"column:normalized_value;type:varchar(255);not null;default:'';uniqueIndex:uniq_course_alias_kind_value,priority:2;index:idx_course_alias_normalized;" json:"normalizedValue"`
	Source          string         `gorm:"column:source;type:varchar(64);not null;default:'';" json:"source"`
	CreatedAt       time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt       time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `json:"-"`
}

// 别名类型
const (
	AliasKindCode         string = "code"         // 旧课号/其他系统课号
	AliasKindName         string = "name"         // 课程名称别名
	AliasKindAbbreviation string = "abbreviation" // 常用简称
)

func (itself *AliasEntity) TableName() string {
	return aliasTableName
}
