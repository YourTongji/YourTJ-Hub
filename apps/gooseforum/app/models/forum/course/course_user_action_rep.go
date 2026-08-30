package course

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- 课程收藏（course_user_action） ----

// GetCourseUserAction 读取用户对某课程的收藏状态（未收藏返回零值 Id==0）。
func GetCourseUserAction(userId, courseId uint64) (entity CourseUserActionEntity) {
	dbconnect.Connect().Table(courseUserActionTableName).
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("course_id", courseId)).
		First(&entity)
	return
}

// SetCourseBookmarked 设置课程收藏状态，返回是否发生了状态迁移。
// 语义与 topicUserAction.SetBookmarked 一致：
//   - 取消：UPDATE 命中已收藏行才算迁移；
//   - 收藏：先 UPDATE 未收藏行（命中即迁移），未命中再 INSERT（冲突静默）。
func SetCourseBookmarked(userId, courseId uint64, bookmarked bool) bool {
	return setCourseBookmarkAt(userId, courseId, timeForCourseBookmark(bookmarked))
}

func timeForCourseBookmark(active bool) *time.Time {
	if !active {
		return nil
	}
	now := time.Now()
	return &now
}

// setCourseBookmarkAt 原子地设置收藏时间戳（幂等 upsert，仅覆盖 bookmarked_at）。
func setCourseBookmarkAt(userId, courseId uint64, value *time.Time) bool {
	if userId == 0 || courseId == 0 {
		return false
	}
	if value == nil {
		result := dbconnect.Connect().Table(courseUserActionTableName).
			Where(queryopt.Eq("user_id", userId)).
			Where(queryopt.Eq("course_id", courseId)).
			Where("bookmarked_at IS NOT NULL").
			Updates(map[string]any{"bookmarked_at": nil, "updated_at": time.Now()})
		return result.Error == nil && result.RowsAffected > 0
	}
	// 1) 更新当前未收藏的行：命中即 "未收藏 → 已收藏" 迁移
	result := dbconnect.Connect().Table(courseUserActionTableName).
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("course_id", courseId)).
		Where("bookmarked_at IS NULL").
		Updates(map[string]any{"bookmarked_at": value, "updated_at": time.Now()})
	if result.Error == nil && result.RowsAffected > 0 {
		return true
	}
	// 2) 行不存在时插入；已存在（并发已设置）则冲突静默
	insert := dbconnect.Connect().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "course_id"}},
		DoNothing: true,
	}).Create(&CourseUserActionEntity{
		UserId:       userId,
		CourseId:     courseId,
		BookmarkedAt: value,
	})
	return insert.Error == nil && insert.RowsAffected > 0
}

// ListBookmarkedCourseIDs 返回用户已收藏的课程 id 列表（按收藏时间倒序），
// 供目录 SSR props 判定表格收藏状态（登录用户）。
func ListBookmarkedCourseIDs(userId uint64) ([]uint64, error) {
	if userId == 0 {
		return []uint64{}, nil
	}
	var ids []uint64
	err := dbconnect.Connect().Table(courseUserActionTableName).
		Select("course_id").
		Where(queryopt.Eq("user_id", userId)).
		Where("bookmarked_at IS NOT NULL").
		Order("bookmarked_at DESC, id DESC").
		Pluck("course_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// courseUserActionBuilder 便捷访问器（同包内其他 rep 文件可能需要事务封装）。
func courseUserActionBuilder() *gorm.DB {
	return dbconnect.Connect().Table(courseUserActionTableName)
}
