package wikiPages

import "time"

const tableName = "wiki_pages"

type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TopicId   uint64    `gorm:"column:topic_id;not null;default:0;uniqueIndex:uniq_wiki_page_topic,priority:1;" json:"topicId"`
	Namespace string    `gorm:"column:namespace;type:varchar(64);not null;default:'';index:idx_wiki_page_namespace,priority:1;" json:"namespace"`
	Path      string    `gorm:"column:path;type:varchar(255);not null;default:'';uniqueIndex:uniq_wiki_page_path,priority:1;" json:"path"`
	ParentId  uint64    `gorm:"column:parent_id;not null;default:0;" json:"parentId"`
	SortOrder int       `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
