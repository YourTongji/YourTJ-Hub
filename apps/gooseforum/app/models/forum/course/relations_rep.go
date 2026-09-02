package course

import (
	"errors"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- 课程沿革关系（course_relations）----

// RelationQuery 管理端沿革候选检索条件。
type RelationQuery struct {
	Status string // 空 = 全部；pending/approved/ignored/merged
	Page   int
	Size   int
}

// RelationPage 管理端沿革候选分页结果。
type RelationPage struct {
	List    []RelationEntity `json:"list"`
	Page    int              `json:"page"`
	Size    int              `json:"size"`
	Total   int64            `json:"total"`
	HasNext bool             `json:"hasNext"`
}

func relationBuilder() *gorm.DB {
	return db.Connect().Table(relationsTableName)
}

// GetRelation 按主键读取沿革候选（含软删除过滤）。
func GetRelation(id uint64) (entity RelationEntity, err error) {
	err = relationBuilder().Where("id = ?", id).First(&entity).Error
	return
}

// GetRelationByIdTx 事务内按主键读取沿革候选（含软删除过滤）。
func GetRelationByIdTx(tx *gorm.DB, id uint64) (entity RelationEntity, err error) {
	err = tx.Table(relationsTableName).Where("id = ?", id).First(&entity).Error
	return
}

// ListPendingMergeByFromCourseTx 返回某课程出发的未处理合并候选（status=pending 且类型可合并）。
// 合并冲突守卫用：from 卡存在其他 pending 等价候选时拒绝合并（避免同卡并入多个目标）。
func ListPendingMergeByFromCourseTx(tx *gorm.DB, fromCourseId uint64) ([]RelationEntity, error) {
	var entities []RelationEntity
	err := tx.Table(relationsTableName).
		Where("from_course_id = ?", fromCourseId).
		Where("status = ?", RelationStatusPending).
		Where("relation_type IN ?", []string{string(RelationEquivalent), string(RelationRenamed)}).
		Find(&entities).Error
	return entities, err
}

// GetMergedTargetByFromCourseTx 事务内返回从某课程出发、已确认合并（status=merged 且
// 类型 EQUIVALENT/RENAMED_FROM）的合并目标课程 id；无则返回 ErrRecordNotFound。
// 物化链重定向用：from 卡被合并隐藏后，offering 写入应指向合并目标卡而非旧卡。
func GetMergedTargetByFromCourseTx(tx *gorm.DB, fromCourseId uint64) (uint64, error) {
	var entity RelationEntity
	err := tx.Table(relationsTableName).
		Where("from_course_id = ?", fromCourseId).
		Where("status = ?", RelationStatusMerged).
		Where("relation_type IN ?", []string{string(RelationEquivalent), string(RelationRenamed)}).
		Order("id ASC").
		First(&entity).Error
	if err != nil {
		return 0, err
	}
	return entity.ToCourseId, nil
}

// GetRelationByFromToTypeTx 事务内按 (from_course_id, to_course_id, relation_type) 精确查找
// （幂等/冲突检测：同一对课程同一类型只允许一行，唯一索引保证）。
func GetRelationByFromToTypeTx(tx *gorm.DB, fromCourseId, toCourseId uint64, relationType string) (entity RelationEntity, err error) {
	err = tx.
		Where(queryopt.Eq("from_course_id", fromCourseId)).
		Where(queryopt.Eq("to_course_id", toCourseId)).
		Where(queryopt.Eq("relation_type", relationType)).
		First(&entity).Error
	return
}

// CreateRelationTx 事务内创建沿革候选（幂等：同 (from,to,type) 已存在则返回已存在行）。
// 并发安全：INSERT 走 ON CONFLICT DO NOTHING 兜底唯一索引——纯 SELECT-then-INSERT
// 在并发下会撞 uniq_course_relations_from_to_type 报错终止调用方，与幂等语义不符
// （review Should）。
func CreateRelationTx(tx *gorm.DB, entity *RelationEntity) (RelationEntity, error) {
	// 快速路径：已存在（含软删过滤）直接返回。
	if existing, err := GetRelationByFromToTypeTx(tx, entity.FromCourseId, entity.ToCourseId, entity.RelationType); err == nil {
		return existing, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return RelationEntity{}, err
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(entity).Error; err != nil {
		return RelationEntity{}, err
	}
	if entity.Id == 0 {
		// 并发下唯一索引兜底命中：重查并返回已存在行（幂等契约）。
		return GetRelationByFromToTypeTx(tx, entity.FromCourseId, entity.ToCourseId, entity.RelationType)
	}
	return *entity, nil
}

// UpdateRelationStatusTx 事务内更新沿革候选状态（pending → approved/ignored/merged）。
func UpdateRelationStatusTx(tx *gorm.DB, relationId uint64, status string) (RelationEntity, error) {
	var entity RelationEntity
	if err := tx.Model(&RelationEntity{}).Where("id = ?", relationId).First(&entity).Error; err != nil {
		return entity, err
	}
	if err := tx.Model(&RelationEntity{}).Where("id = ?", relationId).
		Update("status", status).Error; err != nil {
		return entity, err
	}
	entity.Status = status
	return entity, nil
}

// ListRelationsByToCourse 返回指向某课程的沿革关系（详情页原名标注/沿革区块用）。
// statuses 空 = 全部状态；非空 = 仅匹配指定状态（如 approved+merged 合并历史）。
func ListRelationsByToCourse(toCourseId uint64, statuses []string) ([]RelationEntity, error) {
	b := relationBuilder().Where(queryopt.Eq("to_course_id", toCourseId)).Order("id ASC")
	if len(statuses) > 0 {
		b = b.Where(queryopt.In("status", statuses))
	}
	var entities []RelationEntity
	err := b.Find(&entities).Error
	return entities, err
}

// ListRelationsByFromCourse 返回从某课程出发的沿革关系（撤销合并/管理端详情用）。
func ListRelationsByFromCourse(fromCourseId uint64) ([]RelationEntity, error) {
	var entities []RelationEntity
	err := relationBuilder().Where(queryopt.Eq("from_course_id", fromCourseId)).Order("id ASC").Find(&entities).Error
	return entities, err
}

// ListRelations 管理端沿革候选分页（status 过滤 + 时间倒序，最新候选在前）。
func ListRelations(q RelationQuery) (RelationPage, error) {
	page := q.Page
	if page <= 0 {
		page = 1
	}
	size := q.Size
	if size <= 0 {
		size = 20
	}
	if size > 50 {
		size = 50
	}
	b := relationBuilder().Where("deleted_at IS NULL")
	if q.Status != "" {
		b = b.Where(queryopt.Eq("status", q.Status))
	}
	var total int64
	if err := b.Count(&total).Error; err != nil {
		return RelationPage{}, err
	}
	var entities []RelationEntity
	if err := b.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&entities).Error; err != nil {
		return RelationPage{}, err
	}
	if entities == nil {
		entities = []RelationEntity{}
	}
	return RelationPage{List: entities, Page: page, Size: size, Total: total, HasNext: int64(page)*int64(size) < total}, nil
}
