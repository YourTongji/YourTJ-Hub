package course

import "time"

const courseUserActionTableName = "course_user_action"

// CourseUserActionEntity 用户对课程的收藏状态，与 topic_user_action 同构：
// 状态由时间戳是否为空表达，null 表示未收藏。
type CourseUserActionEntity struct {
	Id           uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;index:idx_cua_user_id,priority:2" json:"id"`
	UserId       uint64     `gorm:"column:user_id;not null;default:0;uniqueIndex:uniq_user_course_action,priority:1;index:idx_cua_user_id,priority:1;index:idx_cua_user_bookmark,priority:1" json:"userId"`
	CourseId     uint64     `gorm:"column:course_id;not null;default:0;uniqueIndex:uniq_user_course_action,priority:2" json:"courseId"`
	BookmarkedAt *time.Time `gorm:"column:bookmarked_at;index:idx_cua_user_bookmark,priority:2" json:"bookmarkedAt"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (CourseUserActionEntity) TableName() string {
	return courseUserActionTableName
}
