package postRevisions

import "time"

const tableName = "post_revisions"

// Entity retains immutable content snapshots; only review outcome fields change.
// Deletion guards hide retained audit bodies and attachments from normal reads.
type Entity struct {
	Title         string     `gorm:"column:title;type:text;" json:"title"`
	CategoryIds   []uint64   `gorm:"column:category_ids;type:text;serializer:json" json:"categoryIds"`
	ImageUrls     []string   `gorm:"column:image_urls;type:text;serializer:json" json:"imageUrls"`
	ContentType   int8       `gorm:"column:content_type;not null;default:0;" json:"contentType"`
	ReviewReason  string     `gorm:"column:review_reason;type:text;" json:"reviewReason"`
	ReviewActorId uint64     `gorm:"column:review_actor_id;not null;default:0;" json:"-"`
	ReviewedAt    *time.Time `gorm:"column:reviewed_at;" json:"reviewedAt,omitempty"`
	Id            uint64     `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PostId        uint64     `gorm:"column:post_id;not null;default:0;index;" json:"postId"`
	Version       uint64     `gorm:"column:version;not null;default:0;index;" json:"version"`
	EditorId      uint64     `gorm:"column:editor_id;not null;default:0;" json:"editorId"`
	Content       string     `gorm:"column:content;type:text;" json:"content"`
	RenderedHTML  string     `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	ProcessStatus int8       `gorm:"column:process_status;not null;default:0;" json:"processStatus"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
