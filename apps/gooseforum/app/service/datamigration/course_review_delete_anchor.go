package datamigration

import (
	"log/slog"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
	"gorm.io/gorm"
)

// CourseReviewDeleteAnchorResult 汇总存量课评删除锚点回填结果。
type CourseReviewDeleteAnchorResult struct {
	Backfilled int
	Failed     int
	LastFailed string
}

// BackfillCourseReviewDeleteAnchors 为上线前已删除的课评行回填 deleted_at
// （PR #194 B3 隔离窗口锚点）：
//   - status=deleted 且 deleted_at IS NULL 的行视为存量删除——旧删除路径
//     UpdateReviewStatusFromTx 只写 status、裸 Table().Update 不触发
//     autoUpdateTime，updated_at 停留在最后编辑时间，无法近似删除时刻；
//   - 回填为 now()：给存量 cohort 一个完整的新隔离窗口。若用 COALESCE
//     回退 updated_at，这些行会在部署后第一轮清理立即被清空，打破
//     "删除后 30 天可恢复"承诺（review 发现的窗口塌缩）；
//   - 幂等：只处理 deleted_at IS NULL 的行；course_review 表不存在时跳过
//     （全新库由 AutoMigrate 建表，无存量）。
func BackfillCourseReviewDeleteAnchors() CourseReviewDeleteAnchorResult {
	return BackfillCourseReviewDeleteAnchorsWithDB(db.Connect())
}

// BackfillCourseReviewDeleteAnchorsWithDB 使用指定连接执行回填，便于测试注入。
func BackfillCourseReviewDeleteAnchorsWithDB(conn *gorm.DB) CourseReviewDeleteAnchorResult {
	result := CourseReviewDeleteAnchorResult{}
	if !conn.Migrator().HasTable(&course.ReviewEntity{}) {
		return result
	}
	rows := conn.Table((&course.ReviewEntity{}).TableName()).
		Where("status = ?", course.ReviewStatusDeleted).
		Where("deleted_at IS NULL").
		Updates(map[string]any{"deleted_at": time.Now()})
	if rows.Error != nil {
		result.Failed++
		result.LastFailed = rows.Error.Error()
		slog.Error("backfill course review delete anchors failed", "err", rows.Error)
		return result
	}
	result.Backfilled = int(rows.RowsAffected)
	slog.Info("backfill course review delete anchors done", "backfilled", result.Backfilled)
	return result
}
