package wikiPageRevisions

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "wiki_page_revisions"

// 修订状态
const (
	StatusPending    int8 = 0 // 待审
	StatusApproved   int8 = 1 // 已通过（公开可见）
	StatusRejected   int8 = 2 // 已拒绝
	StatusSuperseded int8 = 3 // 被更新的编辑取代（旧 pending）
)

type Entity struct {
	Id           uint64         `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	PageId       uint64         `gorm:"column:page_id;not null;default:0;index:idx_wiki_rev_page,priority:1;index:idx_wiki_rev_status,priority:2;uniqueIndex:uniq_wiki_rev_page_no,priority:1;" json:"pageId"`
	RevisionNo   int            `gorm:"column:revision_no;type:int;not null;default:0;uniqueIndex:uniq_wiki_rev_page_no,priority:2;" json:"revisionNo"`
	Title        string         `gorm:"column:title;type:varchar(512);not null;default:'';" json:"title"`
	Content      string         `gorm:"column:content;type:text;" json:"content"`
	RenderedHTML string         `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	Toc          string         `gorm:"column:toc;type:text;" json:"toc"`
	Status       int8           `gorm:"column:status;not null;default:0;index:idx_wiki_rev_status,priority:1;" json:"status"`
	EditorId     uint64         `gorm:"column:editor_id;not null;default:0;" json:"editorId"`
	ReviewedBy   uint64         `gorm:"column:reviewed_by;not null;default:0;" json:"reviewedBy"`
	ReviewedAt   *time.Time     `gorm:"column:reviewed_at;" json:"reviewedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;" json:"-"`
	CreatedAt    time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
