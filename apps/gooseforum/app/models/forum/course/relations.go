package course

import (
	"time"

	"gorm.io/gorm"
)

const relationsTableName = "course_relations"

// RelationType 课程沿革关系类型。
// 只有 EQUIVALENT / RENAMED_FROM（经人工确认）允许合并历史课评；
// SPLIT_FROM / MERGED_FROM / RELATED 仅作为「改革前相关评价」展示，不进入主评分。
type RelationType string

const (
	RelationEquivalent RelationType = "EQUIVALENT"
	RelationRenamed    RelationType = "RENAMED_FROM"
	RelationSplit      RelationType = "SPLIT_FROM"
	RelationMerged     RelationType = "MERGED_FROM"
	RelationRelated    RelationType = "RELATED"
)

// RelationStatus 沿革候选状态。
type RelationStatus string

const (
	RelationStatusPending  RelationStatus = "pending"  // 规则产出候选，待人工审核
	RelationStatusApproved RelationStatus = "approved" // 人工确认（SPLIT/RELATED 等非合并关系终态）
	RelationStatusIgnored  RelationStatus = "ignored"  // 人工忽略
	RelationStatusMerged   RelationStatus = "merged"   // 人工确认等价并已物理合并
)

// RelationSource 候选来源。
const (
	RelationSourceRule   string = "rule"   // 确定性规则
	RelationSourceManual string = "manual" // 人工创建
)

// RelationEntity 课程沿革关系：from（历史/旧卡）→ to（当前/新卡）。
// 只表达语义，不参与课程身份；合并由 MergeCourses 显式执行。
type RelationEntity struct {
	Id           uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	FromCourseId uint64         `gorm:"column:from_course_id;not null;default:0;index:idx_course_relations_from;uniqueIndex:uniq_course_relations_from_to_type,priority:1;" json:"fromCourseId"`
	ToCourseId   uint64         `gorm:"column:to_course_id;not null;default:0;index:idx_course_relations_to;uniqueIndex:uniq_course_relations_from_to_type,priority:2;" json:"toCourseId"`
	RelationType string         `gorm:"column:relation_type;type:varchar(16);not null;default:'';uniqueIndex:uniq_course_relations_from_to_type,priority:3;" json:"relationType"`
	Source       string         `gorm:"column:source;type:varchar(16);not null;default:'rule';" json:"source"`
	Confidence   float64        `gorm:"column:confidence;not null;default:0;" json:"confidence"`
	EvidenceJson string         `gorm:"column:evidence_json;type:text;not null;default:'';" json:"evidenceJson"`
	Manual       bool           `gorm:"column:manual;not null;default:false;" json:"manual"`
	Status       string         `gorm:"column:status;type:varchar(16);not null;default:'pending';index:idx_course_relations_status;" json:"status"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `json:"-"`
}

func (itself *RelationEntity) TableName() string {
	return relationsTableName
}
