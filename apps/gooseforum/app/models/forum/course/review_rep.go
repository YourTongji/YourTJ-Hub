package course

import (
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- Review ----

func CreateReview(entity *ReviewEntity) error {
	return reviewBuilder().Create(entity).Error
}

// CreateReviewTx 事务内创建评价。
func CreateReviewTx(tx *gorm.DB, entity *ReviewEntity) error {
	return tx.Table(reviewTableName).Create(entity).Error
}

// GetReview 按 ID 读取评价（含 soft-delete 过滤）。
func GetReview(id uint64) (entity ReviewEntity, err error) {
	err = reviewBuilder().Where("id = ?", id).First(&entity).Error
	return
}

// GetReviewMapByIds 批量读取评价（审核/举报队列用）。
func GetReviewMapByIds(ids []uint64) map[uint64]ReviewEntity {
	result := make(map[uint64]ReviewEntity, len(ids))
	if len(ids) == 0 {
		return result
	}
	var entities []ReviewEntity
	reviewBuilder().
		Where(queryopt.In("id", ids)).
		Find(&entities)
	for _, e := range entities {
		result[e.Id] = e
	}
	return result
}

// GetReviewTx 事务内按 ID 读取评价。
func GetReviewTx(tx *gorm.DB, id uint64) (entity ReviewEntity, err error) {
	err = tx.Table(reviewTableName).Where("id = ?", id).First(&entity).Error
	return
}

// GetOfferingTx 事务内按 ID 读取开课实例。
func GetOfferingTx(tx *gorm.DB, id uint64) (entity OfferingEntity, err error) {
	err = tx.Table(offeringTableName).Where("id = ?", id).First(&entity).Error
	return
}

// FindReviewByOfferingAndUserTx 事务内查同一用户对同一 offering 的评价（全部状态）。
// 唯一索引 (offering_id, author_user_id) 不含 status，删除仅软删（status=deleted），
// 因此 deleted 行仍占用唯一键：调用方据此决定返回 duplicate（visible/hidden）
// 还是恢复重写（deleted，见 ReactivateReviewTx）。
func FindReviewByOfferingAndUserTx(tx *gorm.DB, offeringId, userId uint64) (entity ReviewEntity, err error) {
	err = tx.Table(reviewTableName).
		Where(queryopt.Eq("offering_id", offeringId)).
		Where(queryopt.Eq("author_user_id", userId)).
		Where("status IN ?", []int8{ReviewStatusVisible, ReviewStatusHidden, ReviewStatusDeleted}).
		First(&entity).Error
	return
}

// ReactivateReviewTx 事务内恢复并重写一条已删除（隔离窗口）的评价。
// 唯一索引 (offering_id, author_user_id) 被软删行占用，重新评价必须复用该行；
// 带 status 条件（CAS）防止并发事务重复恢复，RowsAffected=0 表示已被并发恢复。
// 恢复时一并回写作者关联并清除 deleted_at（隔离窗口标记）：行在窗口内
// author_user_id 仍为用户本人，回写为幂等；若清理任务已断开作者（置 NULL，
// 仅可能发生在行仍被占用前的边界），恢复后该用户重新占用唯一键，
// 与"同 offering+用户至多一条"约束保持一致。
func ReactivateReviewTx(tx *gorm.DB, id, authorUserId uint64, rating *int, content string, isAnonymous bool) (bool, error) {
	res := tx.Table(reviewTableName).
		Where("id = ? AND status = ?", id, ReviewStatusDeleted).
		Updates(map[string]any{
			"author_user_id": authorUserId,
			"rating":         rating,
			"content":        content,
			"is_anonymous":   isAnonymous,
			"status":         ReviewStatusVisible,
			"deleted_at":     nil,
			"updated_at":     time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ListHelpfulReviewIDsByUser 批量返回该用户已标记 helpful 的 review ID 集合。
func ListHelpfulReviewIDsByUser(userId uint64, reviewIds []uint64) (map[uint64]bool, error) {
	result := make(map[uint64]bool, len(reviewIds))
	if len(reviewIds) == 0 {
		return result, nil
	}
	var ids []uint64
	if err := helpfulBuilder().
		Select("review_id").
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.In("review_id", reviewIds)).
		Where("deleted_at IS NULL").
		Scan(&ids).Error; err != nil {
		return nil, err
	}
	for _, id := range ids {
		result[id] = true
	}
	return result, nil
}

// FindLegacyReviewByOfferingTx 事务内查某 offering 的历史导入评价
// （author_user_id=0 且 source=legacy-import，每个 offering 至多一条）。
func FindLegacyReviewByOfferingTx(tx *gorm.DB, offeringId uint64) (entity ReviewEntity, err error) {
	err = tx.Table(reviewTableName).
		Where(queryopt.Eq("offering_id", offeringId)).
		Where(queryopt.Eq("author_user_id", 0)).
		Where(queryopt.Eq("source", ReviewSourceLegacyImport)).
		First(&entity).Error
	return
}

// SaveReviewTx 事务内更新评价。
func SaveReviewTx(tx *gorm.DB, entity *ReviewEntity) error {
	return tx.Table(reviewTableName).Save(entity).Error
}

// UpdateReviewStatusTx 事务内更新评价状态。
func UpdateReviewStatusTx(tx *gorm.DB, id uint64, status int8) error {
	return tx.Table(reviewTableName).Where("id = ?", id).Update("status", status).Error
}

// UpdateReviewStatusFromTx 事务内带旧状态条件的 CAS 状态转换：
// 仅当当前 status 仍为 from 时更新为 to，返回是否成功转换。
// 并发 hide/delete 双写时只有一个事务能拿到转换权，另一个 RowsAffected=0，
// 调用方据此决定是否调整 stats，避免按陈旧旧值重复扣减/累加。
func UpdateReviewStatusFromTx(tx *gorm.DB, id uint64, from, to int8) (bool, error) {
	res := tx.Table(reviewTableName).
		Where("id = ? AND status = ?", id, from).
		Update("status", to)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkReviewDeletedFromTx 事务内带旧状态条件的 CAS 删除：置 status=deleted，
// 并写入 deleted_at 作为隔离窗口起点（30 天，见 courseservice.ReviewCleanupWindow）。
// Table 更新不会自动维护时间戳（GORM 仅对带 Schema 的更新自动处理），
// 因此 deleted_at/updated_at 必须显式写入，否则窗口起点缺失、
// 删除后立即进入可清理状态。
func MarkReviewDeletedFromTx(tx *gorm.DB, id uint64, from int8) (bool, error) {
	now := time.Now()
	res := tx.Table(reviewTableName).
		Where("id = ? AND status = ?", id, from).
		Updates(map[string]any{
			"status":     ReviewStatusDeleted,
			"deleted_at": now,
			"updated_at": now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// UpdateReviewRatingFromTx 事务内带旧评分条件的 CAS 评分更新：
// 仅当当前 rating 仍为 oldRating 时更新，返回是否成功。
// 并发 PATCH 评分时避免两笔事务都基于同一旧值累加 stats delta。
func UpdateReviewRatingFromTx(tx *gorm.DB, id uint64, oldRating int, newRating int) (bool, error) {
	var res *gorm.DB
	if oldRating > 0 {
		res = tx.Table(reviewTableName).
			Where("id = ? AND rating = ?", id, oldRating).
			Update("rating", newRating)
	} else {
		res = tx.Table(reviewTableName).
			Where("id = ? AND rating IS NULL", id).
			Update("rating", newRating)
	}
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// DeleteHelpfulTx 事务内取消 helpful（物理删除）。
// helpful 表主键为 (review_id, user_id) 且带软删；软删后主键仍占位，
// 再次标记会撞唯一约束。物理删除恢复 false→true 生命周期。
func DeleteHelpfulTx(tx *gorm.DB, reviewId, userId uint64) error {
	return tx.Unscoped().Table(helpfulTableName).
		Where(queryopt.Eq("review_id", reviewId)).
		Where(queryopt.Eq("user_id", userId)).
		Delete(&HelpfulEntity{}).Error
}

// DeleteHelpful 取消 helpful（物理删除，理由同 DeleteHelpfulTx）。
func DeleteHelpful(reviewId, userId uint64) error {
	return helpfulBuilder().Unscoped().
		Where(queryopt.Eq("review_id", reviewId)).
		Where(queryopt.Eq("user_id", userId)).
		Delete(&HelpfulEntity{}).Error
}

// ListCourseIDsByInstructorTx 返回引用该教师的所有可见 offering 所属课程 ID
// （教师变更后 fan-out 入队课程搜索同步用）。
func ListCourseIDsByInstructorTx(tx *gorm.DB, instructorId uint64) ([]uint64, error) {
	var ids []uint64
	err := tx.Table(offeringTableName+" o").
		Select("DISTINCT o.course_id").
		Joins("JOIN "+offeringInstructorTableName+" oi ON oi.offering_id = o.id").
		Where("oi.instructor_id = ?", instructorId).
		Where("o.deleted_at IS NULL").
		Scan(&ids).Error
	return ids, err
}

// CreateHelpfulTx 事务内标记 helpful（唯一约束 (review_id, user_id) 防重）。
func CreateHelpfulTx(tx *gorm.DB, entity *HelpfulEntity) error {
	return tx.Table(helpfulTableName).Create(entity).Error
}

// FindReviewByOfferingAndUser 同一用户对同一 offering 的评价（唯一约束 (offering_id, author_user_id)）。
func FindReviewByOfferingAndUser(offeringId, userId uint64) (entity ReviewEntity, err error) {
	err = reviewBuilder().
		Where(queryopt.Eq("offering_id", offeringId)).
		Where(queryopt.Eq("author_user_id", userId)).
		First(&entity).Error
	return
}

// SaveReview 更新评价（Save 会写全部字段）。
func SaveReview(entity *ReviewEntity) error {
	return reviewBuilder().Save(entity).Error
}

// UpdateReviewStatus 更新评价状态（隐藏/恢复/删除）。
func UpdateReviewStatus(id uint64, status int8) error {
	return reviewBuilder().Where("id = ?", id).Update("status", status).Error
}

// ListReviewsByOffering 按 offering 列出可见评价（时间倒序）。
func ListReviewsByOffering(offeringId uint64) (entities []ReviewEntity, err error) {
	err = reviewBuilder().
		Where(queryopt.Eq("offering_id", offeringId)).
		Where(queryopt.Eq("status", ReviewStatusVisible)).
		Order("id DESC").
		Find(&entities).Error
	return
}

// ListReviewsByOfferings 批量按 offering 列出可见评价（列表页 N+1 防护）。
func ListReviewsByOfferings(offeringIds []uint64) (entities []ReviewEntity, err error) {
	if len(offeringIds) == 0 {
		return []ReviewEntity{}, nil
	}
	err = reviewBuilder().
		Where(queryopt.In("offering_id", offeringIds)).
		Where(queryopt.Eq("status", ReviewStatusVisible)).
		Order("offering_id ASC, id DESC").
		Find(&entities).Error
	return
}

// ---- Helpful ----

// CreateHelpful 标记 helpful（唯一约束 (review_id, user_id) 防重）。
func CreateHelpful(entity *HelpfulEntity) error {
	return helpfulBuilder().Create(entity).Error
}

// GetHelpful 查询某用户的 helpful 状态。
func GetHelpful(reviewId, userId uint64) (entity HelpfulEntity, err error) {
	err = helpfulBuilder().
		Where(queryopt.Eq("review_id", reviewId)).
		Where(queryopt.Eq("user_id", userId)).
		First(&entity).Error
	return
}

// CountHelpfulByReviewIds 批量统计各评价的 helpful 数。
func CountHelpfulByReviewIds(reviewIds []uint64) map[uint64]int64 {
	result := make(map[uint64]int64, len(reviewIds))
	if len(reviewIds) == 0 {
		return result
	}
	type row struct {
		ReviewId uint64
		Cnt      int64
	}
	var rows []row
	helpfulBuilder().
		Select("review_id, COUNT(*) AS cnt").
		Where(queryopt.In("review_id", reviewIds)).
		Where("deleted_at IS NULL").
		Group("review_id").
		Scan(&rows)
	for _, r := range rows {
		result[r.ReviewId] = r.Cnt
	}
	return result
}

// ---- Stats ----

// GetCourseStats 课程级统计。
func GetCourseStats(courseId uint64) (entity CourseStatsEntity, err error) {
	err = courseStatsBuilder().Where("course_id = ?", courseId).First(&entity).Error
	return
}

// GetOfferingStats offering 级统计。
func GetOfferingStats(offeringId uint64) (entity OfferingStatsEntity, err error) {
	err = offeringStatsBuilder().Where("offering_id = ?", offeringId).First(&entity).Error
	return
}

// UpsertCourseStatsTx 事务内课程级统计原子累加（INSERT ... ON CONFLICT DO UPDATE + delta）。
// 用 SQL 表达式做原子增量，而非事务内 read-modify-write：两笔并发事务同时读到同一行再
// 各自 Save 会互相覆盖、丢一次更新，使 review 计数/评分合计低于实际。重建命令可对账，
// 但写路径必须本身不丢更新。跨方言（SQLite/PostgreSQL 均支持 ON CONFLICT）。
func UpsertCourseStatsTx(tx *gorm.DB, courseId uint64, deltaRatingCount, deltaRatingSum, deltaReviewCount int) error {
	return tx.Table(courseStatsTableName).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "course_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"rating_count": gorm.Expr(courseStatsTableName+".rating_count + ?", deltaRatingCount),
			"rating_sum":   gorm.Expr(courseStatsTableName+".rating_sum + ?", deltaRatingSum),
			"review_count": gorm.Expr(courseStatsTableName+".review_count + ?", deltaReviewCount),
			"updated_at":   time.Now(),
		}),
	}).Create(&CourseStatsEntity{
		CourseId:    courseId,
		RatingCount: deltaRatingCount,
		RatingSum:   deltaRatingSum,
		ReviewCount: deltaReviewCount,
	}).Error
}

// UpsertOfferingStatsTx 事务内 offering 级统计原子累加（INSERT ... ON CONFLICT DO UPDATE + delta）。
// 与 UpsertCourseStatsTx 同理：原子增量避免并发事务互相覆盖导致丢更新。
func UpsertOfferingStatsTx(tx *gorm.DB, offeringId uint64, deltaRatingCount, deltaRatingSum, deltaReviewCount int) error {
	return tx.Table(offeringStatsTableName).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "offering_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"rating_count": gorm.Expr(offeringStatsTableName+".rating_count + ?", deltaRatingCount),
			"rating_sum":   gorm.Expr(offeringStatsTableName+".rating_sum + ?", deltaRatingSum),
			"review_count": gorm.Expr(offeringStatsTableName+".review_count + ?", deltaReviewCount),
			"updated_at":   time.Now(),
		}),
	}).Create(&OfferingStatsEntity{
		OfferingId:  offeringId,
		RatingCount: deltaRatingCount,
		RatingSum:   deltaRatingSum,
		ReviewCount: deltaReviewCount,
	}).Error
}

// RebuildAllCourseStats 全量重建课程/offering 统计投影（rebuild-course-stats 命令）。
// 以 review 事实表为准重新聚合，发现漂移时用于对账。
// 在单事务内先清空两张统计表再重插：中途失败整体回滚，避免留下空表/半成品；
// 事务快照保证并发课评写入不会在 DELETE 与重插之间插入同主键统计行。
func RebuildAllCourseStats() error {
	conn := db.Connect()
	return conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Table(courseStatsTableName).Where("1 = 1").Delete(&CourseStatsEntity{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Table(offeringStatsTableName).Where("1 = 1").Delete(&OfferingStatsEntity{}).Error; err != nil {
			return err
		}
		// offering 级聚合
		type offeringAgg struct {
			OfferingId  uint64
			RatingCount int
			RatingSum   int
			ReviewCount int
		}
		var offeringAggs []offeringAgg
		if err := tx.Table(reviewTableName).
			Select("offering_id, COUNT(CASE WHEN rating IS NOT NULL AND status = ? THEN 1 END) AS rating_count, COALESCE(SUM(CASE WHEN rating IS NOT NULL AND status = ? THEN rating END), 0) AS rating_sum, COUNT(CASE WHEN status = ? THEN 1 END) AS review_count",
				ReviewStatusVisible, ReviewStatusVisible, ReviewStatusVisible).
			Where("deleted_at IS NULL").
			Group("offering_id").
			Scan(&offeringAggs).Error; err != nil {
			return err
		}
		for _, agg := range offeringAggs {
			if err := tx.Table(offeringStatsTableName).Create(&OfferingStatsEntity{
				OfferingId:  agg.OfferingId,
				RatingCount: agg.RatingCount,
				RatingSum:   agg.RatingSum,
				ReviewCount: agg.ReviewCount,
				UpdatedAt:   time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		// course 级聚合（通过 offering 关联）
		type courseAgg struct {
			CourseId    uint64
			RatingCount int
			RatingSum   int
			ReviewCount int
		}
		var courseAggs []courseAgg
		if err := tx.Table(offeringTableName+" o").
			Select("o.course_id, COUNT(CASE WHEN r.rating IS NOT NULL AND r.status = ? THEN 1 END) AS rating_count, COALESCE(SUM(CASE WHEN r.rating IS NOT NULL AND r.status = ? THEN r.rating END), 0) AS rating_sum, COUNT(CASE WHEN r.status = ? THEN 1 END) AS review_count",
				ReviewStatusVisible, ReviewStatusVisible, ReviewStatusVisible).
			Joins("LEFT JOIN " + reviewTableName + " r ON r.offering_id = o.id AND r.deleted_at IS NULL").
			Where("o.deleted_at IS NULL").
			Group("o.course_id").
			Scan(&courseAggs).Error; err != nil {
			return err
		}
		for _, agg := range courseAggs {
			if err := tx.Table(courseStatsTableName).Create(&CourseStatsEntity{
				CourseId:    agg.CourseId,
				RatingCount: agg.RatingCount,
				RatingSum:   agg.RatingSum,
				ReviewCount: agg.ReviewCount,
				UpdatedAt:   time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
