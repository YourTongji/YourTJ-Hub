package optlogger

import "github.com/spf13/cast"

type OptEnum int

func (receiver OptEnum) TargetTypeEnum() TargetTypeEnum {
	switch receiver {
	case EditUser:
		return User
	case EditTopic:
		return Topic
	case EditCategory:
		return Category
	case RevealCourseReviewAuthor:
		return CourseReview
	case CreateCourse, UpdateCourse, DeleteCourse:
		return Course
	case UpdateReview, DeleteReview:
		return CourseReview
	}
	return System
}


func (receiver OptEnum) toInt() int {
	return cast.ToInt(receiver)
}

const (
	EditUser OptEnum = iota
	EditTopic
	EditCategory
	RevealCourseReviewAuthor
	CreateCourse
	UpdateCourse
	DeleteCourse
	UpdateReview
	DeleteReview
	// SyncPk 排课数据同步（issue #248）：归属 System，不新增 TargetTypeEnum 值，
	// 避免改动枚举序列影响已持久化的审计记录。
	SyncPk
)

type TargetTypeEnum int


func (receiver TargetTypeEnum) toInt() int {
	return cast.ToInt(receiver)
}

// TargetType 数值已持久化到 opt_records 审计表，前端按数字渲染历史记录
// （OptRecordsManagementPage targetTypeCodeMap）。删除 Doc* 类型后显式固定
// 剩余数值：3/4/5 空缺保留给已删除的 DocProject/DocVersion/DocContent，
// 不得用 iota 重排，否则历史审计记录语义漂移。
const (
	System       TargetTypeEnum = 0
	User                        = 1
	Topic                       = 2
	Category                    = 6
	CourseReview                = 7
	Course                      = 8
)
