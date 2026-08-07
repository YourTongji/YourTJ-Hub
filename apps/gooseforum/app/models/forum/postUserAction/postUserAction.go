package postUserAction

import "time"

const tableName = "post_user_action"

// Entity 记录用户对楼层（Post）的点赞与收藏状态，与 topic_user_action 模式一致：
// 状态由时间戳是否为空表达，null 表示未触发该动作。
type Entity struct {
	Id           uint64     `gorm:"primaryKey;column:id;autoIncrement;not null" json:"id"`
	UserId       uint64     `gorm:"column:user_id;not null;default:0;uniqueIndex:uniq_user_post_action,priority:1" json:"userId"`
	PostId       uint64     `gorm:"column:post_id;not null;default:0;uniqueIndex:uniq_user_post_action,priority:2;index:idx_pua_post_id" json:"postId"`
	LikedAt      *time.Time `gorm:"column:liked_at" json:"likedAt"`
	BookmarkedAt *time.Time `gorm:"column:bookmarked_at" json:"bookmarkedAt"`
	CreatedAt    time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt    time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
