package wikiNamespaces

import "time"

const tableName = "wiki_namespaces"

type Entity struct {
	Id          uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Name        string    `gorm:"column:name;type:varchar(64);not null;default:'';uniqueIndex:uniq_wiki_namespace_name,priority:1;" json:"name"`
	// Slug URL 友好标识（^[a-z0-9]+(-[a-z0-9]+)*$ ≤64），与显示名 name（可为中文目录名）分离。
	// 可空：中文命名空间在仓库 index.md 未提供 slug 前为 NULL（NULL 不参与唯一约束，
	// 多个未分配 slug 的命名空间不冲突）；分配后由 uniqueIndex 保证全局唯一。
	Slug        *string   `gorm:"column:slug;type:varchar(64);uniqueIndex:uniq_wiki_namespace_slug,priority:1;" json:"slug"`
	Description string    `gorm:"column:description;type:varchar(255);not null;default:'';" json:"description"`
	SortOrder   int       `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (itself *Entity) TableName() string {
	return tableName
}

// SlugOrEmpty 返回 slug 字符串（未分配时返回空串，供 API 输出/前端展示）。
func (itself *Entity) SlugOrEmpty() string {
	if itself.Slug == nil {
		return ""
	}
	return *itself.Slug
}
