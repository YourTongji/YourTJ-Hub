package courseservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 课程沿革合并错误 sentinel（控制器映射语义 HTTP 状态）。
var (
	// ErrRelationNotFound 沿革候选不存在。
	ErrRelationNotFound = errors.New("course relation not found")
	// ErrRelationNotMergeable 该关系类型不允许合并（仅 EQUIVALENT/RENAMED_FROM 可合并）。
	ErrRelationNotMergeable = errors.New("course relation not mergeable")
	// ErrRelationAlreadyMerged 该候选已合并，重复合并被拒绝。
	ErrRelationAlreadyMerged = errors.New("course relation already merged")
	// ErrMergeConflict 目标课程存在未处理的等价候选冲突（from 卡仍有其他 pending 合并候选）。
	ErrMergeConflict = errors.New("course merge conflict")
	// ErrMergeTargetHidden 目标课程已隐藏，不可作为合并目标。
	ErrMergeTargetHidden = errors.New("course merge target hidden")
	// ErrReviewScopeInvalid review_scope 取值非法（仅 teacher/team/course）。
	ErrReviewScopeInvalid = errors.New("course review scope invalid")
	// ErrRelationConfidenceInvalid 沿革候选置信度超出 [0,1]（契约 minimum/maximum 约束的服务端落地）。
	ErrRelationConfidenceInvalid = errors.New("course relation confidence invalid")
)

// RelationReviewScope 三档课评范围。
const (
	ReviewScopeTeacher string = "teacher" // 默认：评价挂 (code, teacher) 卡
	ReviewScopeTeam    string = "team"    // 教学团队：team_key 非空，详情页读时聚合团队全部卡
	ReviewScopeCourse  string = "course"  // 课程级：课程卡本身为评价目标
)

// MergeResult 合并/撤销结果（供控制器写审计日志）。
type MergeResult struct {
	RelationId      uint64 `json:"relationId"`
	FromCourseId    uint64 `json:"fromCourseId"`
	ToCourseId      uint64 `json:"toCourseId"`
	FromName        string `json:"fromName"`
	ToName          string `json:"toName"`
	MovedOfferings  int    `json:"movedOfferings"`
	MigratedAliases int    `json:"migratedAliases"`
	SkippedAliases  int    `json:"skippedAliases"`
}

// mergeSnapshot 合并快照（存入 relations.evidence_json，撤销合并的唯一依据）。
// 撤销时按快照反向迁移：offering 迁回 from 卡、alias 迁回 from 卡、收藏迁回、
// from 卡恢复合并前状态（快照记录原 status，非无条件恢复可见）。
type mergeSnapshot struct {
	MovedOfferingIds []uint64 `json:"movedOfferingIds,omitempty"`
	MigratedAliasIds []uint64 `json:"migratedAliasIds,omitempty"`
	SkippedAliases   []string `json:"skippedAliases,omitempty"`
	FromName         string   `json:"fromName"`
	FromCode         string   `json:"fromCode"`
	ToName           string   `json:"toName"`
	ToCode           string   `json:"toCode"`
	MergedAt         string   `json:"mergedAt"`
	// FromStatus 合并时 from 卡原状态：撤销时还原该状态，而非无条件恢复可见
	// （from 卡可能在合并前已被其它原因隐藏，review Should）。
	FromStatus int8 `json:"fromStatus"`
	// CoalescedBookmarks 收藏迁移快照：from 卡每个收藏用户的去向（含合并时 to 行
	// 收藏状态），撤销时恢复 to 原状态并重建 from 收藏行。
	CoalescedBookmarks []bookmarkSnapshot `json:"coalescedBookmarks,omitempty"`
}

// bookmarkSnapshot 收藏合并快照项：用户 + 原收藏时间 + 合并时 to 行收藏状态
// （ToBookmarkedAt 为 nil 表示合并时 to 卡未收藏、撤销时无需恢复 to 行）。
type bookmarkSnapshot struct {
	UserId         uint64     `json:"userId"`
	BookmarkedAt   *time.Time `json:"bookmarkedAt"`
	ToBookmarkedAt *time.Time `json:"toBookmarkedAt,omitempty"`
}

// mergeEvidencePayload 合并后 evidence_json 载荷：保留规则原始证据 + 合并快照。
type mergeEvidencePayload struct {
	OriginalEvidence string        `json:"originalEvidence"`
	Merge            mergeSnapshot `json:"merge"`
}

// MergeCourses 人工确认等价后物理合并：from 卡（历史）并入 to 卡（当前）。
// 事务内：offering 迁移（评价/教师关联随 offering 走，零丢失）→ alias 迁移（冲突跳过）
// → from 卡隐藏 → relations 置 merged + manual → 入队搜索重建（from 删文档、to 重建）。
// 全量统计重建随事务入队（EnqueueCourseStatsRebuildTaskTx：transaction-bound outbox，
// 与业务同事务提交，避免"合并已提交而任务入队失败"导致统计永久陈旧）。
func MergeCourses(relationId uint64) (MergeResult, error) {
	var result MergeResult
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		relation, err := course.GetRelationByIdTx(tx, relationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRelationNotFound
			}
			return err
		}
		if relation.Status == string(course.RelationStatusMerged) {
			return ErrRelationAlreadyMerged
		}
		if relation.RelationType != string(course.RelationEquivalent) &&
			relation.RelationType != string(course.RelationRenamed) {
			return ErrRelationNotMergeable
		}
		fromEntity := course.GetCourseByIdTx(tx, relation.FromCourseId)
		toEntity := course.GetCourseByIdTx(tx, relation.ToCourseId)
		if fromEntity.Id == 0 || toEntity.Id == 0 {
			return ErrCourseNotFound
		}
		if toEntity.Status != course.StatusVisible {
			return ErrMergeTargetHidden
		}
		// 记录 from 卡合并前状态（撤销时还原，快照字段）。
		fromOriginalStatus := fromEntity.Status
		// 冲突守卫：from 卡不得有其他未处理的合并候选（避免同卡被两次并入不同目标）。
		pending, err := course.ListPendingMergeByFromCourseTx(tx, fromEntity.Id)
		if err != nil {
			return err
		}
		for _, p := range pending {
			if p.Id != relationId {
				return ErrMergeConflict
			}
		}
		// 1. offering 迁移：评价/offering_instructor/来源映射随 offering 走，零丢失。
		moved, err := course.ListOfferingIdsByCourseAllTx(tx, fromEntity.Id)
		if err != nil {
			return err
		}
		if len(moved) > 0 {
			if err := tx.Model(&course.OfferingEntity{}).
				Where("course_id = ?", fromEntity.Id).
				Update("course_id", toEntity.Id).Error; err != nil {
				return fmt.Errorf("merge: move offerings: %w", err)
			}
		}
		// 2. alias 迁移：from 卡别名转 to 卡。
		//    (kind, normalized_value) 全局唯一索引下，from 卡的 live 别名不可能被其它课程
		//    占用；该分支是 legacy 重复数据（索引建立前）的防御：显式查询「其它课程是否
		//    持有同 (kind, normalized) 的 live 行」，有则跳过并记录（避免制造新重复）。
		//    空闲别名改挂 to 卡（保留行 id 供撤销迁回）。
		var migratedAliasIds []uint64
		var skippedAliases []string
		fromAliases, err := course.ListAliasesByCourseTx(tx, fromEntity.Id)
		if err != nil {
			return err
		}
		for _, a := range fromAliases {
			var others []course.AliasEntity
			if err := tx.Table((&course.AliasEntity{}).TableName()).
				Where("kind = ? AND normalized_value = ?", a.Kind, a.NormalizedValue).
				Where("deleted_at IS NULL").
				Where("course_id <> ?", fromEntity.Id).
				Find(&others).Error; err != nil {
				return err
			}
			if len(others) > 0 {
				// 其它课程 live 占用该 (kind, normalized)：跳过并记录。
				skippedAliases = append(skippedAliases, a.Value)
				continue
			}
			// 空闲：改挂 to 卡（保留行 id 供撤销迁回）。
			if err := tx.Model(&course.AliasEntity{}).
				Where("id = ?", a.Id).Update("course_id", toEntity.Id).Error; err != nil {
				return err
			}
			migratedAliasIds = append(migratedAliasIds, a.Id)
		}
		// 2.5 收藏迁移：from 卡收藏并入 to 卡（用户可见收藏不因合并丢失，review P1）。
		//    to 卡已收藏 → 保留 to 行原时间、删除 from 行；to 卡未收藏/无行 →
		//    以 from 的收藏时间 upsert to 行、删除 from 行。快照记录每个用户的
		//    from 原收藏时间与 to 原状态，撤销时按快照恢复。
		var coalescedBookmarks []bookmarkSnapshot
		fromBookmarks, err := course.ListBookmarkedActionsByCourseTx(tx, fromEntity.Id)
		if err != nil {
			return err
		}
		for _, bm := range fromBookmarks {
			toAction := course.GetCourseUserActionTx(tx, bm.UserId, toEntity.Id)
			snap := bookmarkSnapshot{UserId: bm.UserId, BookmarkedAt: bm.BookmarkedAt}
			if toAction.Id != 0 && toAction.BookmarkedAt != nil {
				snap.ToBookmarkedAt = toAction.BookmarkedAt
			} else if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "course_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"bookmarked_at", "updated_at"}),
			}).Create(&course.CourseUserActionEntity{
				UserId:       bm.UserId,
				CourseId:     toEntity.Id,
				BookmarkedAt: bm.BookmarkedAt,
			}).Error; err != nil {
				return fmt.Errorf("merge: migrate bookmark user %d: %w", bm.UserId, err)
			}
			if err := course.DeleteCourseUserActionTx(tx, bm.UserId, fromEntity.Id); err != nil {
				return fmt.Errorf("merge: remove from bookmark user %d: %w", bm.UserId, err)
			}
			coalescedBookmarks = append(coalescedBookmarks, snap)
		}
		// 3. from 卡隐藏（保留数据，可撤销）。
		if err := tx.Model(&course.Entity{}).Where("id = ?", fromEntity.Id).
			Update("status", course.StatusHidden).Error; err != nil {
			return err
		}
		// 4. relations 置 merged + manual，写入合并快照（保留原始 evidence）。
		snapshot := mergeSnapshot{
			MovedOfferingIds:   moved,
			MigratedAliasIds:   migratedAliasIds,
			SkippedAliases:     skippedAliases,
			CoalescedBookmarks: coalescedBookmarks,
			FromName:           fromEntity.Name,
			FromCode:           fromEntity.PrimaryCode,
			ToName:             toEntity.Name,
			ToCode:             toEntity.PrimaryCode,
			MergedAt:           time.Now().Format(time.RFC3339),
			FromStatus:         fromOriginalStatus,
		}
		payload := mergeEvidencePayload{OriginalEvidence: relation.EvidenceJson, Merge: snapshot}
		evidenceJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("merge: marshal snapshot: %w", err)
		}
		if err := tx.Model(&course.RelationEntity{}).Where("id = ?", relationId).Updates(map[string]any{
			"status":        string(course.RelationStatusMerged),
			"manual":        true,
			"evidence_json": string(evidenceJSON),
		}).Error; err != nil {
			return err
		}
		// 5. 搜索重建：from 卡隐藏 → worker 删文档；to 卡别名已迁移 → worker 重建文档。
		if err := searchservice.EnqueueCourseSearchTask(tx, fromEntity.Id); err != nil {
			return err
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, toEntity.Id); err != nil {
			return err
		}
		// 6. 失效 AI 总结缓存：from/to 卡评价集合均已变化（review P2）。
		if err := course.DeleteCourseAiSummaryTx(tx, fromEntity.Id); err != nil {
			return fmt.Errorf("merge: invalidate ai summary from course: %w", err)
		}
		if err := course.DeleteCourseAiSummaryTx(tx, toEntity.Id); err != nil {
			return fmt.Errorf("merge: invalidate ai summary to course: %w", err)
		}
		// 7. 事务内入队全量统计重建（transaction-bound outbox：与业务同事务提交，
		//    避免"合并已提交而任务入队失败"导致统计永久陈旧，review P2）。
		if err := EnqueueCourseStatsRebuildTaskTx(tx); err != nil {
			return fmt.Errorf("merge: enqueue stats rebuild: %w", err)
		}
		result = MergeResult{
			RelationId:      relationId,
			FromCourseId:    fromEntity.Id,
			ToCourseId:      toEntity.Id,
			FromName:        fromEntity.Name,
			ToName:          toEntity.Name,
			MovedOfferings:  len(moved),
			MigratedAliases: len(migratedAliasIds),
			SkippedAliases:  len(skippedAliases),
		}
		return nil
	})
	if err != nil {
		return MergeResult{}, err
	}
	return result, nil
}

// UndoMergeCourse 撤销已合并的沿革关系：offering 迁回 from 卡、alias 迁回、
// from 卡恢复合并前状态（review Should），relations 标记回 approved（manual 保留）。
// 入队搜索重建与全量统计重建（事务内 outbox）。
func UndoMergeCourse(relationId uint64) (MergeResult, error) {
	var result MergeResult
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		relation, err := course.GetRelationByIdTx(tx, relationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRelationNotFound
			}
			return err
		}
		if relation.Status != string(course.RelationStatusMerged) {
			return ErrRelationNotMergeable
		}
		var payload mergeEvidencePayload
		if err := json.Unmarshal([]byte(relation.EvidenceJson), &payload); err != nil {
			return fmt.Errorf("undo: parse merge snapshot: %w", err)
		}
		fromEntity := course.GetCourseByIdTx(tx, relation.FromCourseId)
		toEntity := course.GetCourseByIdTx(tx, relation.ToCourseId)
		if fromEntity.Id == 0 || toEntity.Id == 0 {
			return ErrCourseNotFound
		}
		// 1. offering 迁回 from 卡（按快照 id，避免误迁 to 卡原生 offering）。
		if len(payload.Merge.MovedOfferingIds) > 0 {
			if err := tx.Model(&course.OfferingEntity{}).
				Where("id IN ?", payload.Merge.MovedOfferingIds).
				Update("course_id", fromEntity.Id).Error; err != nil {
				return fmt.Errorf("undo: move offerings back: %w", err)
			}
		}
		// 2. alias 迁回 from 卡（按快照 id）。
		if len(payload.Merge.MigratedAliasIds) > 0 {
			if err := tx.Model(&course.AliasEntity{}).
				Where("id IN ?", payload.Merge.MigratedAliasIds).
				Update("course_id", fromEntity.Id).Error; err != nil {
				return fmt.Errorf("undo: move aliases back: %w", err)
			}
		}
		// 2.5 收藏迁回（按快照）：to 行恢复合并时状态、from 行按原时间重建（review P1）。
		for _, snap := range payload.Merge.CoalescedBookmarks {
			if snap.ToBookmarkedAt != nil {
				if err := tx.Model(&course.CourseUserActionEntity{}).
					Where("user_id = ? AND course_id = ?", snap.UserId, toEntity.Id).
					Update("bookmarked_at", snap.ToBookmarkedAt).Error; err != nil {
					return fmt.Errorf("undo: restore to bookmark user %d: %w", snap.UserId, err)
				}
			} else {
				if err := course.SetCourseBookmarkedTx(tx, snap.UserId, toEntity.Id, false); err != nil {
					return fmt.Errorf("undo: clear to bookmark user %d: %w", snap.UserId, err)
				}
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "course_id"}},
				DoUpdates: clause.AssignmentColumns([]string{"bookmarked_at", "updated_at"}),
			}).Create(&course.CourseUserActionEntity{
				UserId:       snap.UserId,
				CourseId:     fromEntity.Id,
				BookmarkedAt: snap.BookmarkedAt,
			}).Error; err != nil {
				return fmt.Errorf("undo: restore from bookmark user %d: %w", snap.UserId, err)
			}
		}
		// 3. from 卡恢复合并前状态（review Should：合并前已被其它原因隐藏的卡，
		//    撤销时还原原 status，而非无条件置可见——旧快照无 fromStatus 字段时
		//    解码为 0 = StatusVisible，保持既有行为）。
		if err := tx.Model(&course.Entity{}).Where("id = ?", fromEntity.Id).
			Update("status", payload.Merge.FromStatus).Error; err != nil {
			return err
		}
		// 4. relations 标记回 approved（manual 保留）。
		if err := tx.Model(&course.RelationEntity{}).Where("id = ?", relationId).
			Updates(map[string]any{
				"status":        string(course.RelationStatusApproved),
				"evidence_json": payload.OriginalEvidence,
			}).Error; err != nil {
			return err
		}
		// 5. 搜索重建（from 卡恢复 → 重建文档；to 卡别名减少 → 重建文档）。
		if err := searchservice.EnqueueCourseSearchTask(tx, fromEntity.Id); err != nil {
			return err
		}
		if err := searchservice.EnqueueCourseSearchTask(tx, toEntity.Id); err != nil {
			return err
		}
		// 6. 失效 AI 总结缓存（from/to 评价集合恢复原状后重新生成，review P2）。
		if err := course.DeleteCourseAiSummaryTx(tx, fromEntity.Id); err != nil {
			return fmt.Errorf("undo: invalidate ai summary from course: %w", err)
		}
		if err := course.DeleteCourseAiSummaryTx(tx, toEntity.Id); err != nil {
			return fmt.Errorf("undo: invalidate ai summary to course: %w", err)
		}
		// 7. 事务内入队全量统计重建。
		if err := EnqueueCourseStatsRebuildTaskTx(tx); err != nil {
			return fmt.Errorf("undo: enqueue stats rebuild: %w", err)
		}
		result = MergeResult{
			RelationId:      relationId,
			FromCourseId:    fromEntity.Id,
			ToCourseId:      toEntity.Id,
			FromName:        fromEntity.Name,
			ToName:          toEntity.Name,
			MovedOfferings:  len(payload.Merge.MovedOfferingIds),
			MigratedAliases: len(payload.Merge.MigratedAliasIds),
			SkippedAliases:  len(payload.Merge.SkippedAliases),
		}
		return nil
	})
	if err != nil {
		return MergeResult{}, err
	}
	return result, nil
}

// ---- 管理端沿革候选操作 ----

// AdminRelationList 返回管理端沿革候选分页（status 过滤）。
func AdminRelationList(q course.RelationQuery) (course.RelationPage, error) {
	return course.ListRelations(q)
}

// AdminRelationApprove 人工确认非合并关系（SPLIT_FROM/MERGED_FROM/RELATED → approved）。
// EQUIVALENT/RENAMED_FROM 需走 MergeCourses 落库合并；误传合并类型返回 ErrRelationNotMergeable。
func AdminRelationApprove(relationId uint64) (course.RelationEntity, error) {
	var result course.RelationEntity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		relation, err := course.GetRelationByIdTx(tx, relationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRelationNotFound
			}
			return err
		}
		if relation.RelationType == string(course.RelationEquivalent) ||
			relation.RelationType == string(course.RelationRenamed) {
			return ErrRelationNotMergeable
		}
		// 仅 pending 可批准（review P1：已忽略/已合并/已批准行拒绝，契约状态机
		// pending → approved）。
		if relation.Status != string(course.RelationStatusPending) {
			return ErrRelationNotMergeable
		}
		updated, err := course.UpdateRelationStatusTx(tx, relationId, string(course.RelationStatusApproved))
		if err != nil {
			return err
		}
		result = updated
		return nil
	})
	if err != nil {
		return course.RelationEntity{}, err
	}
	return result, nil
}

// AdminRelationIgnore 忽略沿革候选（pending → ignored）。
func AdminRelationIgnore(relationId uint64) (course.RelationEntity, error) {
	var result course.RelationEntity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		relation, err := course.GetRelationByIdTx(tx, relationId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRelationNotFound
			}
			return err
		}
		// 仅 pending 可忽略（review P1：approved/merged 行被置 ignored 会破坏
		// UndoMergeCourse 的撤销识别，契约状态机 pending → ignored）。
		if relation.Status != string(course.RelationStatusPending) {
			return ErrRelationNotMergeable
		}
		updated, err := course.UpdateRelationStatusTx(tx, relationId, string(course.RelationStatusIgnored))
		if err != nil {
			return err
		}
		result = updated
		return nil
	})
	if err != nil {
		return course.RelationEntity{}, err
	}
	return result, nil
}

// AdminRelationCreateInput 手动创建沿革关系的入参。
type AdminRelationCreateInput struct {
	FromCourseId uint64  `json:"fromCourseId" validate:"required"`
	ToCourseId   uint64  `json:"toCourseId" validate:"required"`
	RelationType string  `json:"relationType" validate:"required"`
	Evidence     string  `json:"evidence"`
	Confidence   float64 `json:"confidence"`
}

// AdminRelationCreate 手动创建沿革关系（source=manual；幂等：同 (from,to,type) 已存在则返回已存在行）。
func AdminRelationCreate(input AdminRelationCreateInput) (course.RelationEntity, error) {
	if input.FromCourseId == 0 || input.ToCourseId == 0 || input.FromCourseId == input.ToCourseId {
		return course.RelationEntity{}, fmt.Errorf("course relation from/to required and distinct")
	}
	if !validRelationType(input.RelationType) {
		return course.RelationEntity{}, fmt.Errorf("invalid relation type %q", input.RelationType)
	}
	// 置信度必须落在 [0,1]（OpenAPI schema minimum/maximum 的服务端落地，review P2），
	// 越界值会产出无意义的百分比与排序。
	if input.Confidence < 0 || input.Confidence > 1 {
		return course.RelationEntity{}, ErrRelationConfidenceInvalid
	}
	var result course.RelationEntity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		fromEntity := course.GetCourseByIdTx(tx, input.FromCourseId)
		toEntity := course.GetCourseByIdTx(tx, input.ToCourseId)
		if fromEntity.Id == 0 || toEntity.Id == 0 {
			return ErrCourseNotFound
		}
		entity := &course.RelationEntity{
			FromCourseId: input.FromCourseId,
			ToCourseId:   input.ToCourseId,
			RelationType: input.RelationType,
			Source:       course.RelationSourceManual,
			Confidence:   input.Confidence,
			EvidenceJson: input.Evidence,
			Manual:       true,
			Status:       string(course.RelationStatusPending),
		}
		created, err := course.CreateRelationTx(tx, entity)
		if err != nil {
			return err
		}
		result = created
		return nil
	})
	if err != nil {
		return course.RelationEntity{}, err
	}
	return result, nil
}

// validRelationType 校验关系类型取值（与 course.RelationType 常量集合一致）。
func validRelationType(t string) bool {
	switch course.RelationType(t) {
	case course.RelationEquivalent, course.RelationRenamed, course.RelationSplit,
		course.RelationMerged, course.RelationRelated:
		return true
	default:
		return false
	}
}
