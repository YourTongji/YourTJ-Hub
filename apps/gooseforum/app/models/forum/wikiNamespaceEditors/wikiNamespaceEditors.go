package wikiNamespaceEditors

import "time"

const tableName = "wiki_namespace_editors"

type Entity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Namespace string    `gorm:"column:namespace;type:varchar(64);not null;default:'';uniqueIndex:uniq_wiki_ns_editor,priority:1;" json:"namespace"`
	UserId    uint64    `gorm:"column:user_id;not null;default:0;uniqueIndex:uniq_wiki_ns_editor,priority:2;" json:"userId"`
	AddedBy   uint64    `gorm:"column:added_by;not null;default:0;" json:"addedBy"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}
