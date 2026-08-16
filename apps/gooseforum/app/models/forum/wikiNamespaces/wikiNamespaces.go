package wikiNamespaces

import "time"

const tableName = "wiki_namespaces"

type Entity struct {
	Id          uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(64);not null;default:'';uniqueIndex:uniq_wiki_namespace_name,priority:1;" json:"name"`
	Description string    `gorm:"column:description;type:varchar(255);not null;default:'';" json:"description"`
	SortOrder   int       `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
