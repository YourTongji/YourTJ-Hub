package postRevisions

import "time"

const tableName = "post_revisions"

// Entity 帖子内容版本快照（append-only）。
// 每次内容编辑（首楼与回复均包含）在同一事务内追加一条新版本，
// 同时创建时播种 v1（editor = 作者）。版本对用户只读（历史查看），
// 不提供回滚/写入接口；内容被永久删除或隐私擦除时一并清空，
// 不允许绕过现有删除生命周期留存正文。
type Entity struct {
	Id            uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PostId        uint64    `gorm:"column:post_id;not null;default:0;index;" json:"postId"`
	Version       uint64    `gorm:"column:version;not null;default:0;index;" json:"version"`
	EditorId      uint64    `gorm:"column:editor_id;not null;default:0;" json:"editorId"`
	Content       string    `gorm:"column:content;type:text;" json:"content"`
	RenderedHTML  string    `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	ProcessStatus int8      `gorm:"column:process_status;not null;default:0;" json:"processStatus"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
