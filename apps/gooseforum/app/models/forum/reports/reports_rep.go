package reports

import (
	"errors"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func CreateOpen(entity Entity) (Entity, bool, error) {
	var existing Entity
	err := builder().
		Where(queryopt.Eq(fieldReporterId, entity.ReporterId)).
		Where(queryopt.Eq(fieldTargetType, entity.TargetType)).
		Where(queryopt.Eq(fieldTargetId, entity.TargetId)).
		Where(queryopt.Eq(fieldStatus, StatusOpen)).
		First(&existing).Error
	if err == nil && existing.Id > 0 {
		return existing, false, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return Entity{}, false, err
	}
	entity.Status = StatusOpen
	if err := builder().Create(&entity).Error; err != nil {
		return Entity{}, false, err
	}
	return entity, true, nil
}

func Get(id uint64) (entity Entity) {
	if id == 0 {
		return entity
	}
	builder().First(&entity, id)
	return
}

// HasOpenForTopic reports whether retention must preserve an active moderation case.
func HasOpenForTopic(topicID uint64) bool {
	if topicID == 0 {
		return false
	}
	var count int64
	builder().Where(queryopt.Eq("topic_id", topicID)).Where(queryopt.Eq(fieldStatus, StatusOpen)).Count(&count)
	return count > 0
}

type CursorPageQuery struct {
	TargetType       string
	Status           string
	Statuses         []string
	ScopeCategoryIDs []uint64
	Cursor, PageSize uint64
}

func CursorPage(q CursorPageQuery) []Entity {
	var list []Entity
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	b := builder()
	if q.Status != "" {
		b = b.Where(queryopt.Eq(fieldStatus, q.Status))
	} else if len(q.Statuses) > 0 {
		b = b.Where(queryopt.In(fieldStatus, q.Statuses))
	}
	if q.TargetType != "" {
		b = b.Where(queryopt.Eq(fieldTargetType, q.TargetType))
	}
	if len(q.ScopeCategoryIDs) > 0 {
		b = b.Where(`EXISTS (
			SELECT 1 FROM topic_category_index idx
			WHERE idx.topic_id = reports.topic_id
				AND idx.category_id IN ?
				AND idx.effective = ?
		)`, q.ScopeCategoryIDs, 1)
	}
	if q.Cursor > 0 {
		b = b.Where(queryopt.Lt("id", q.Cursor))
	}
	b.Limit(int(q.PageSize)).Order(queryopt.Desc("id")).Find(&list)
	return list
}

func UpdateStatus(id uint64, status string, resolution string, handlerId uint64) error {
	now := time.Now()
	return builder().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"status":     status,
		"resolution": resolution,
		"handler_id": handlerId,
		"handled_at": &now,
	}).Error
}

// ClearExpiredEvidenceSnapshots clears evidence_snapshot on closed reports older than before.
// Skips rows whose topic has LEGAL_HOLD or EVIDENCE_HOLD retention (hold overrides TTL).
// Open reports are never cleared. Returns number of rows updated.
//
// 跨库注意：evidence_snapshot 是 json 列。不要在 SQL 里直接与字符串比较
// （PostgreSQL 对 json 列的 `!= ''` 会报 `42883 json <> unknown`，MySQL 行为也不同），
// 因此"空快照"过滤放在 Go 层用 evidenceSnapshotIsEmpty 完成（NULL/`{}` 反序列化后
// 均为零值快照，会被跳过）。
func ClearExpiredEvidenceSnapshots(before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}

	var candidates []Entity
	err := builder().
		Where("status IN ?", []string{StatusResolved, StatusRejected}).
		Where("handled_at IS NOT NULL").
		Where("handled_at < ?", before).
		Where(`NOT EXISTS (
			SELECT 1 FROM topics
			WHERE topics.id = reports.topic_id
			  AND topics.retention_status IN (?, ?)
		)`, "LEGAL_HOLD", "EVIDENCE_HOLD").
		Order("handled_at ASC").
		Limit(limit).
		Find(&candidates).Error
	if err != nil {
		return 0, err
	}
	if len(candidates) == 0 {
		return 0, nil
	}

	ids := make([]uint64, 0, len(candidates))
	for _, candidate := range candidates {
		if evidenceSnapshotIsEmpty(candidate.EvidenceSnapshot) {
			continue
		}
		ids = append(ids, candidate.Id)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	// Clear to empty JSON object. Raw "{}" matches the zero-value snapshot shape
	// without relying on GORM's struct serializer in map-style updates.
	result := builder().Where("id IN ?", ids).Update("evidence_snapshot", "{}")
	return int(result.RowsAffected), result.Error
}

func evidenceSnapshotIsEmpty(snapshot EvidenceSnapshotData) bool {
	return snapshot.TargetType == "" &&
		snapshot.TargetID == 0 &&
		snapshot.TopicID == 0 &&
		snapshot.Title == "" &&
		snapshot.Excerpt == "" &&
		snapshot.AuthorID == 0 &&
		snapshot.AuthorName == "" &&
		len(snapshot.CategoryIDs) == 0 &&
		snapshot.TargetURL == ""
}

// CountByTargetIds 统计每个 target_id 的举报总数（跨全部状态），
// 供审核队列展示"该对象累计被举报次数"（不限于当前分页/状态）。
func CountByTargetIds(targetType string, targetIds []uint64) map[uint64]int {
	result := make(map[uint64]int, len(targetIds))
	if len(targetIds) == 0 {
		return result
	}
	type row struct {
		TargetId uint64
		Cnt      int
	}
	var rows []row
	builder().
		Select("target_id, COUNT(*) AS cnt").
		Where(queryopt.Eq(fieldTargetType, targetType)).
		Where(queryopt.In(fieldTargetId, targetIds)).
		Group("target_id").
		Scan(&rows)
	for _, r := range rows {
		result[r.TargetId] = r.Cnt
	}
	return result
}
