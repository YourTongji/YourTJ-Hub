package courseservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/course"
)

// ---- 课程收藏（R5，issue #331） ----

// BookmarkCourseAction 收藏动作（1 收藏、2 取消），与主题/帖子收藏一致。
const (
	BookmarkCourseActionAdd    = 1
	BookmarkCourseActionRemove = 2
)

// BookmarkCourse 设置用户对某课程的收藏状态；课程不存在或已隐藏时返回
// ErrCourseNotFound（与详情页 404 语义一致）。返回是否发生了状态迁移。
func BookmarkCourse(userId, courseId uint64, action int) (bool, error) {
	entity := course.GetCourse(courseId)
	if entity.Id == 0 || entity.Status != course.StatusVisible {
		return false, ErrCourseNotFound
	}
	target := action == BookmarkCourseActionAdd
	return course.SetCourseBookmarked(userId, courseId, target), nil
}
