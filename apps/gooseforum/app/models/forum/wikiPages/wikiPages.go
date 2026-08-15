package wikiPages

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"

	"gorm.io/gorm"
)

const tableName = "wiki_pages"

// Entity wiki 页面投影：内容由 GitHub wiki 仓库（唯一真实源）同步而来，
// 本站只读投影 + 互动层。内容/渲染快照/贡献者直接落本表，历史由 git 承担。
type Entity struct {
	Id         uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	TopicId    uint64 `gorm:"column:topic_id;not null;default:0;uniqueIndex:uniq_wiki_page_topic,priority:1;" json:"topicId"`
	Namespace  string `gorm:"column:namespace;type:varchar(64);not null;default:'';index:idx_wiki_page_namespace,priority:1;" json:"namespace"`
	Path       string `gorm:"column:path;type:varchar(255);not null;default:'';uniqueIndex:uniq_wiki_page_path,priority:1;" json:"path"`
	ParentId   uint64 `gorm:"column:parent_id;not null;default:0;" json:"parentId"`
	SortOrder  int    `gorm:"column:sort_order;type:int;not null;default:0;" json:"sortOrder"`
	SourcePath string `gorm:"column:source_path;type:varchar(255);not null;default:'';" json:"sourcePath"`
	// 内容快照（自 git 同步，frontmatter title/order 解析后落库）：
	Title        string `gorm:"column:title;type:varchar(512);not null;default:'';" json:"title"`
	Content      string `gorm:"column:content;type:text;" json:"content"`
	RenderedHTML string `gorm:"column:rendered_html;type:text;" json:"renderedHTML"`
	Toc          string `gorm:"column:toc;type:text;" json:"toc"`
	// git 溯源：内容哈希（幂等 diff 依据）、提交 SHA/时间、贡献者快照。
	ContentHash      string     `gorm:"column:content_hash;type:varchar(64);not null;default:'';index:idx_wiki_page_hash,priority:1;" json:"contentHash"`
	LastCommitSha    string     `gorm:"column:last_commit_sha;type:varchar(64);not null;default:'';" json:"lastCommitSha"`
	LastCommitAt     *time.Time `gorm:"column:last_commit_at;" json:"lastCommitAt"`
	ContributorsJSON string     `gorm:"column:contributors_json;type:text;" json:"-"`
	// 版本指针列保留（v19 单一事件源遗留，GitHub SSOT 后不再推进；
	// 保留避免动 topics/posts 水印物化函数面）。
	PublishedRevisionNo int            `gorm:"column:published_revision_no;type:int;not null;default:0;" json:"publishedRevisionNo"`
	DeletedAt           gorm.DeletedAt `gorm:"column:deleted_at;" json:"-"`
	CreatedAt           time.Time      `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt           time.Time      `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

// UpdateGitTrace 更新页面的 git 溯源列（贡献者快照/最后提交 SHA/时间）。
// 由同步器在每次 create/update 后调用；map 形式只更新给定列。
func UpdateGitTrace(id uint64, updates map[string]any) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(updates).Error
}

func (itself *Entity) TableName() string {
	return tableName
}
