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
	}
	return System
}

func (receiver OptEnum) Name() string {
	switch receiver {
	case EditUser:
		return "操作用户"
	case EditTopic:
		return "编辑主题"
	case EditCategory:
		return "编辑分类"
	case RevealCourseReviewAuthor:
		return "揭示课评匿名作者"
	}
	return ""
}

func (receiver OptEnum) toInt() int {
	return cast.ToInt(receiver)
}

const (
	EditUser OptEnum = iota
	EditTopic
	EditCategory
	RevealCourseReviewAuthor
)

type TargetTypeEnum int

func (receiver TargetTypeEnum) Name() string {
	switch receiver {
	case System:
		return "系统"
	case User:
		return "用户"
	case Topic:
		return "主题"
	case DocProject:
		return "文档项目"
	case DocVersion:
		return "文档版本"
	case DocContent:
		return "文档内容"
	case Category:
		return "分类"
	case CourseReview:
		return "课评"
	default:
		return ""
	}
}

func (receiver TargetTypeEnum) toInt() int {
	return cast.ToInt(receiver)
}

const (
	System       TargetTypeEnum = iota
	User                        = iota
	Topic                       = iota
	DocProject                  = iota
	DocVersion                  = iota
	DocContent                  = iota
	Category                    = iota
	CourseReview                = iota
)
