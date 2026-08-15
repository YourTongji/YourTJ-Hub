package wikiPages

import (
	"time"

	"gorm.io/gorm"
)

const tableName = "wiki_pages"

type Entity struct {
	Id        uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TopicId   uint64 `gorm:"column:topic_id;not null;default:0;uniqueIndex:uniq_wiki_page_topic,priority:1;" json:"topicId"`
	Namespace string `gorm:"column:namespace;type:varchar(64);not null;default:'';index:idx_wiki_page_namespace,priority:1;" json:"namespace"`
	Path      string `gorm:"column:path;type:varchar(255);not null;default:'';uniqueIndex:uniq_wiki_page_path,priority:1;" json:"path"`
	ParentId  uint64 `gorm:"column:parent_id;not null;default:0;" json:"parentId"`
	SortOrder int    `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	// 版本指针：当前已发布修订的 revision_no（单一事件源的水印基准）。
	// 每次编辑 = 追加一条 approved 修订 + 指针前移（同事务 CAS）；回滚 = 指针回退 + 硬删后续修订。
	// 物化视图（posts/topics/搜索）以此列判断是否过期：synced < published 即需重物化。
	PublishedRevisionNo int            `gorm:"column:published_revision_no;type:int;not null;default:0;" json:"publishedRevisionNo"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;" json:"-"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
	// Git 溯源 + 渲染投影列（迁移 v21，GitHub 仓库为唯一真源后由同步引擎写入）：
	//   content_hash     sha256(剥离 frontmatter 后的 markdown body) 十六进制——同步引擎
	//                    必须先剥 frontmatter 再 hash（#258），口径与 v21 回填一致
	//                    （多页可同内容，不建唯一索引）
	//   last_commit_sha 投影对应的 git commit SHA-1 hex；last_commit_at 对应提交时间
	//   rendered_html    剥离 frontmatter 后的 goldmark 渲染快照
	//   toc              目录 JSON 字符串（与 wikiPageRevisions.Toc 同构，读侧 json.Unmarshal）
	//   contributors_json 贡献者列表 JSON（同步引擎从 git 历史聚合；迁移 v21 由 wiki_namespace_editors 种子迁移）
	ContentHash      string     `gorm:"column:content_hash;type:varchar(64);not null;default:'';" json:"contentHash"`
	LastCommitSha    string     `gorm:"column:last_commit_sha;type:varchar(40);not null;default:'';" json:"lastCommitSha"`
	LastCommitAt     *time.Time `gorm:"column:last_commit_at;" json:"lastCommitAt"`
	RenderedHTML     string     `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	Toc              string     `gorm:"column:toc;type:text;" json:"toc"`
	ContributorsJson string     `gorm:"column:contributors_json;type:text;" json:"contributorsJson"`
}

func (itself *Entity) TableName() string {
	return tableName
}
