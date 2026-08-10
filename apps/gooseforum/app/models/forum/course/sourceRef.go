package course

import (
	"time"

	"gorm.io/gorm"
)

const sourceRefTableName = "course_source_ref"

// Entity 导入来源映射：唯一 (source, entity_type, external_id)，
// 记录每行 checksum 用于幂等增量；旧外部 ID 只用于 redirect/排障，不作为新 ID。
type SourceRefEntity struct {
	Id          uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	ImportRunId uint64         `gorm:"column:import_run_id;not null;default:0;index:idx_course_source_ref_run;" json:"importRunId"`
	Source      string         `gorm:"column:source;type:varchar(64);not null;default:'';uniqueIndex:uniq_course_source_ref_key,priority:1;" json:"source"`
	EntityType  string         `gorm:"column:entity_type;type:varchar(32);not null;default:'';uniqueIndex:uniq_course_source_ref_key,priority:2;" json:"entityType"`
	ExternalId  string         `gorm:"column:external_id;type:varchar(128);not null;default:'';uniqueIndex:uniq_course_source_ref_key,priority:3;" json:"externalId"`
	LocalId     uint64         `gorm:"column:local_id;not null;default:0;" json:"localId"`
	Checksum    string         `gorm:"column:checksum;type:varchar(64);not null;default:'';" json:"checksum"`
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	DeletedAt   gorm.DeletedAt `json:"-"`
}

func (itself *SourceRefEntity) TableName() string {
	return sourceRefTableName
}

// 实体类型
const (
	EntityTypeCourse     string = "course"
	EntityTypeInstructor string = "instructor"
	EntityTypeOffering   string = "offering"
	EntityTypeTerm       string = "term"
)
