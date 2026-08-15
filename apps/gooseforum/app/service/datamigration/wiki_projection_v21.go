package datamigration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"sort"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiProjectionV21Result 迁移 v21（issue #259）执行结果。
type WikiProjectionV21Result struct {
	PagesBackfilled int
	PagesSkipped    int
	EditorsMigrated int
	EditorsSkipped  int
	TablesDropped   []string
	Failed          int
	LastFailed      string
}

// contributorSeed 迁移 v21 写入 contributors_json 的种子条目（与 wikiservice.Contributor
// 同构；同步引擎上线后用 git 历史重写该列）。
type contributorSeed struct {
	UserId       uint64    `json:"userId"`
	Username     string    `json:"username"`
	AvatarUrl    string    `json:"avatarUrl"`
	Count        int       `json:"count"`
	LastEditedAt time.Time `json:"lastEditedAt"`
}

// MigrateWikiProjectionV21 迁移 v21：wiki_pages 加渲染/溯源列 + 退役 revision/editors 表。
//
// 在 DROP 前回填（数据保全）：
//  1. rendered_html/toc ← 各页最新 approved 修订（revision 表仍是唯一正文源时读它）
//  2. content_hash ← sha256(最新 approved 修订 markdown)；last_commit_at ← 修订创建时间
//  3. published_revision_no 仍为 0 的页面补回填（覆盖 v19 未跑过的 <19 升级实例）
//  4. wiki_namespace_editors 数据种子迁移到该 namespace 下各页 contributors_json
//  5. 显式 DROP wiki_page_revisions / wiki_namespace_editors（AutoMigrate 不删表）
//
// 幂等：列已回填的页面跳过；表已删除则整体 no-op。回填失败不 DROP（数据保全优先），
// 版本不推进，下次启动重试。
func MigrateWikiProjectionV21() WikiProjectionV21Result {
	return MigrateWikiProjectionV21WithDB(dbconnect.Connect())
}

func MigrateWikiProjectionV21WithDB(conn *gorm.DB) WikiProjectionV21Result {
	result := WikiProjectionV21Result{}
	// 回填仅在数据源表存在时执行（revision 是投影回填的数据源，editors 是种子数据源；
	// 全新库/已退役库两表缺失，无数据可回填，直接跳到 DROP 清理）。
	if conn.Migrator().HasTable(&wikiPages.Entity{}) && conn.Migrator().HasTable("wiki_page_revisions") {
		if !backfillPageProjections(conn, &result) {
			return result
		}
		if conn.Migrator().HasTable("wiki_namespace_editors") {
			migrateNamespaceEditors(conn, &result)
		}
		if result.Failed > 0 {
			return result
		}
	}
	// DROP 独立执行（review MEDIUM：若用 revision 表存在性门控整个迁移，partial-drop
	// 状态——revision 已删、editors 残留——会让迁移整体 no-op、版本照常推进到 21，
	// editors 表永久孤儿。dropWikiLegacyTables 对每张旧表独立 HasTable+DROP，重启后
	// 仍会清理遗留表）。
	dropWikiLegacyTables(conn, &result)
	return result
}

// backfillPageProjections 逐页回填投影列；失败返回 false（回填未完成不 DROP）。
func backfillPageProjections(conn *gorm.DB, result *WikiProjectionV21Result) bool {
	const batchSize = 200
	var cursor uint64
	for {
		var batch []wikiPages.Entity
		err := conn.Model(&wikiPages.Entity{}).
			Select("id", "topic_id", "published_revision_no").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error
		if err != nil {
			result.Failed++
			result.LastFailed = "pages_scan:" + err.Error()
			return false
		}
		for i := range batch {
			page := &batch[i]
			cursor = page.Id
			if err := backfillOnePage(conn, page, result); err != nil {
				result.Failed++
				result.LastFailed = "page:" + err.Error()
				return false
			}
		}
		if len(batch) < batchSize {
			return true
		}
	}
}

func backfillOnePage(conn *gorm.DB, page *wikiPages.Entity, result *WikiProjectionV21Result) error {
	var latest wikiPageRevisions.Entity
	err := conn.Model(&wikiPageRevisions.Entity{}).
		Where("page_id = ?", page.Id).
		Where("status = ?", wikiPageRevisions.StatusApproved).
		Order("revision_no DESC").
		Order("id DESC").
		First(&latest).Error
	if err == gorm.ErrRecordNotFound {
		// 无 approved 修订（异常态页面）：跳过，保留原列值。
		result.PagesSkipped++
		return nil
	}
	if err != nil {
		return err
	}
	// content_hash 契约：sha256(剥离 frontmatter 后的 markdown body)。revision 存的
	// Content 即裸正文（frontmatter 从不进 revision），与 #258 同步引擎「先剥离 frontmatter
	// 再 hash」的口径一致；wiki-export 生成的种子文件是 frontmatter + body，未来同步引擎
	// 必须剥离 frontmatter 后 hash 才能与回填值匹配（review MEDIUM：口径需钉死）。
	updates := map[string]any{
		"rendered_html":  latest.RenderedHTML,
		"toc":            latest.Toc,
		"content_hash":   sha256Hex(latest.Content),
		"last_commit_at": latest.CreatedAt,
	}
	if page.PublishedRevisionNo == 0 && latest.RevisionNo > 0 {
		updates["published_revision_no"] = latest.RevisionNo
	}
	if err := conn.Model(&wikiPages.Entity{}).
		Where("id = ?", page.Id).
		Updates(updates).Error; err != nil {
		return err
	}
	result.PagesBackfilled++
	return nil
}

// sha256Hex 计算字符串的 sha256 十六进制（content_hash 语义：页面 markdown）。
func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// migrateNamespaceEditors 把 wiki_namespace_editors 数据种子迁移到 contributors_json。
// 「如需保留」语义：表为空则整体跳过；每个有编辑者的 namespace 把其编辑者列表写入该
// namespace 下全部页面的 contributors_json（粒度从 namespace 级落到页面级，由同步引擎
// 后续用 git 历史精确重写）。
func migrateNamespaceEditors(conn *gorm.DB, result *WikiProjectionV21Result) {
	if !conn.Migrator().HasTable("wiki_namespace_editors") {
		return
	}
	var editors []wikiNamespaceEditors.Entity
	if err := conn.Model(&wikiNamespaceEditors.Entity{}).Find(&editors).Error; err != nil {
		result.Failed++
		result.LastFailed = "editors_scan:" + err.Error()
		return
	}
	if len(editors) == 0 {
		result.EditorsSkipped++
		return
	}
	byNS := make(map[string][]*wikiNamespaceEditors.Entity)
	for i := range editors {
		e := &editors[i]
		if e.Namespace == "" {
			continue
		}
		byNS[e.Namespace] = append(byNS[e.Namespace], e)
	}
	for ns, list := range byNS {
		if err := writeNamespaceContributors(conn, ns, list); err != nil {
			result.Failed++
			result.LastFailed = "contributors:" + err.Error()
			return
		}
		result.EditorsMigrated++
	}
}

func writeNamespaceContributors(conn *gorm.DB, namespace string, editors []*wikiNamespaceEditors.Entity) error {
	userIDs := make([]uint64, 0, len(editors))
	for _, e := range editors {
		userIDs = append(userIDs, e.UserId)
	}
	userMap := users.GetMapByIds(userIDs)
	contributors := make([]contributorSeed, 0, len(editors))
	for _, e := range editors {
		entry := contributorSeed{
			UserId:       e.UserId,
			Count:        1,
			LastEditedAt: e.CreatedAt,
		}
		if u, ok := userMap[e.UserId]; ok && u != nil {
			entry.Username = u.Username
			entry.AvatarUrl = u.GetWebAvatarUrl()
		}
		contributors = append(contributors, entry)
	}
	// review LOW：按 userId 稳定排序（byNS map 迭代顺序不确定），保证迁移幂等输出稳定。
	sort.Slice(contributors, func(i, j int) bool { return contributors[i].UserId < contributors[j].UserId })
	data, err := json.Marshal(contributors)
	if err != nil {
		return err
	}
	return conn.Model(&wikiPages.Entity{}).
		Where("namespace = ?", namespace).
		Update("contributors_json", string(data)).Error
}

// dropWikiLegacyTables 显式物理删除退役的 revision/editors 表。
func dropWikiLegacyTables(conn *gorm.DB, result *WikiProjectionV21Result) {
	for _, table := range []string{"wiki_page_revisions", "wiki_namespace_editors"} {
		if !conn.Migrator().HasTable(table) {
			continue
		}
		if err := conn.Migrator().DropTable(table); err != nil {
			result.Failed++
			result.LastFailed = "drop:" + table + ":" + err.Error()
			return
		}
		result.TablesDropped = append(result.TablesDropped, table)
		slog.Info("migration v21: dropped legacy wiki table", "table", table)
	}
}
