package datamigration

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiSlugRevertResult v24 slug 机制移除的数据迁移结果。
type WikiSlugRevertResult struct {
	Migrated          int
	Failed            int
	LastFailed        string
	SlugIndexDropped  bool
	SlugColumnDropped bool
}

// RevertWikiNamespaceSlugs v24 数据迁移：把已分配 slug 的命名空间页面
// path 首段与 namespace 列迁回仓库顶层目录名（显示名）。
//
// 背景：slug 机制整体移除，URL 语义回归"仓库顶层目录名即 path 首段"。
// 存量部署中 slug 已生效（wiki_pages.path 首段 / namespace 列 = slug，
// 与仓库目录名分离）的行需要迁回，否则新语义下直查 path 会 404
// （ResolvePageByURLPath 不再做 name→slug 重建）。
//
// 权威依据：wiki_pages.source_path（v23 起由同步器按仓库真实相对路径回填/
// 刷新，即新语义下 path 的目标值）。source_path 为空的 slug 行（v23 时已
// 软删、未被 v23 回填覆盖，slug 时代同步器仍迁移了其 path/namespace）在
// drop slug 列前用遗留 slug→name 映射推导真实路径（review Blocker）；
// 两者皆不可用时留给同步器收养修复。
//
// 唯一键安全（review P1）：两个命名空间可能互为 slug（目录 a 的 slug=b、
// 目录 b 的 slug=a），逐行更新会因目标 path 仍被对方占用而违反
// uniq_wiki_page_path。因此两阶段更新：先全部迁到临时唯一路径释放旧路径，
// 再写入最终路径，全程单事务。临时路径做碰撞检查（review Should fix）：
// "__wiki_slug_revert__/" 前缀未被路径规则保留，仓库可合法同名，阶段 1
// 不能撞上真实页面路径。
//
// 重渲染衔接（review P2）：迁移只改 path/namespace，不重写
// rendered_html/first-post HTML（仓库 clone 与引用解析不属于迁移职责）。
// 受影响页面清空 content_hash，下次同步必然走更新路径重渲染投影
// （同步器幂等判断以 content_hash 为准），期间旧渲染保留（可展示，旧 slug
// 链接在同步前 404——这正是 slug 移除的预期语义）。
//
// 幂等：path 已等于目标路径的行跳过；slug 列已删（迁移已执行）的库
// 零操作；无法推导的行留给下次同步自动收养修复。
func RevertWikiNamespaceSlugs() WikiSlugRevertResult {
	return RevertWikiNamespaceSlugsWithDB(dbconnect.Connect())
}

func RevertWikiNamespaceSlugsWithDB(conn *gorm.DB) WikiSlugRevertResult {
	result := WikiSlugRevertResult{}
	if conn.Migrator().HasTable(&wikiPages.Entity{}) {
		var pages []wikiPages.Entity
		if err := conn.Unscoped().Model(&wikiPages.Entity{}).Find(&pages).Error; err != nil {
			result.Failed++
			result.LastFailed = "scan_pages:" + err.Error()
			slog.Error("wiki slug revert: scan pages failed", "err", err)
			return result
		}

		pending := make([]pendingPage, 0, len(pages))
		// review Blocker：source_path 为空的 slug 行（v23 前已软删、未被
		// source_path 回填覆盖）不能跳过——drop slug 列前先读取遗留
		// slug→显示名映射推导真实仓库路径（保留 slug 后的相对后缀）。
		slugToName := loadLegacySlugToName(conn)
		existingPaths := make(map[string]struct{}, len(pages))
		for i := range pages {
			existingPaths[pages[i].Path] = struct{}{}
		}
		for i := range pages {
			p := &pages[i]
			if p.Path == "" {
				continue // 无路径可推导（从未同步），留给同步器收养修复
			}
			newPath := p.SourcePath
			if newPath == "" {
				ns := namespaceOfPath(p.Path)
				if name, ok := slugToName[ns]; ok && name != "" && len(p.Path) > len(ns) {
					newPath = name + p.Path[len(ns):]
				}
			}
			if newPath == "" {
				continue // 无法推导，留给同步器收养修复
			}
			dirName, _, _ := strings.Cut(newPath, "/")
			if p.Path == newPath && p.Namespace == dirName {
				continue
			}
			pending = append(pending, pendingPage{id: p.Id, newPath: newPath, dirName: dirName})
		}
		if len(pending) > 0 {
			// 两阶段 + 单事务（review P1）。临时前缀做碰撞检查（review
			// Should fix）："__wiki_slug_revert__" 未被路径规则保留，仓库
			// 可能真实存在同名页面路径，阶段 1 必须避开以免撞 uniq_wiki_page_path。
			tempPrefix := pickTempPrefix(existingPaths, pending)
			if tempPrefix == "" {
				result.Failed++
				result.LastFailed = "temp_prefix_collision"
				slog.Error("wiki slug revert: no collision-free temp prefix")
				return result
			}
			if err := conn.Transaction(func(tx *gorm.DB) error {
				for _, pp := range pending {
					tmpPath := fmt.Sprintf("%s/%d", tempPrefix, pp.id)
					if err := tx.Unscoped().Model(&wikiPages.Entity{}).Where("id = ?", pp.id).
						Update("path", tmpPath).Error; err != nil {
						return fmt.Errorf("stage1 temp path id %d: %w", pp.id, err)
					}
				}
				for _, pp := range pending {
					if err := tx.Unscoped().Model(&wikiPages.Entity{}).Where("id = ?", pp.id).
						Updates(map[string]any{
							"path":         pp.newPath,
							"namespace":    pp.dirName,
							"source_path":  pp.newPath, // 收养匹配的权威键（review Blocker）
							"content_hash": "",         // 强制下次同步重渲染投影（review P2）
							"updated_at":   gorm.Expr("updated_at"),
						}).Error; err != nil {
						return fmt.Errorf("stage2 final path id %d: %w", pp.id, err)
					}
				}
				return nil
			}); err != nil {
				result.Failed++
				result.LastFailed = "update_page:" + err.Error()
				slog.Error("wiki slug revert: update page failed", "err", err)
				return result
			}
			result.Migrated = len(pending)
		}
	}

	// AutoMigrate 不删除从模型消失的字段：显式清理遗留 slug 列与唯一索引
	// （review P2）。仅当数据迁移成功（或无需迁移）时执行。
	dropLegacySlugSchema(conn, &result)
	return result
}

// pendingPage v24 待迁移页面（路径推导结果）。
type pendingPage struct {
	id      uint64
	newPath string
	dirName string
}

// namespaceOfPath 返回 path 首段（namespace/顶层目录名）。
func namespaceOfPath(path string) string {
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return path
}

// loadLegacySlugToName 读取遗留 wiki_namespaces.slug→name 映射（review
// Blocker）。v23 前已软删的页面未被 source_path 回填覆盖，slug 时代同步器
// 仍迁移了其 path/namespace；drop slug 列前必须用它推导真实仓库路径。
// slug 列不存在（新库/已迁移）时返回空映射。
func loadLegacySlugToName(conn *gorm.DB) map[string]string {
	m := make(map[string]string)
	if !conn.Migrator().HasColumn("wiki_namespaces", "slug") {
		return m
	}
	type nsRow struct {
		Name string
		Slug string
	}
	var rows []nsRow
	if err := conn.Table("wiki_namespaces").Select("name", "slug").Find(&rows).Error; err != nil {
		slog.Warn("wiki slug revert: read legacy slug mapping failed", "err", err)
		return m
	}
	for _, r := range rows {
		if r.Slug != "" {
			m[r.Slug] = r.Name
		}
	}
	return m
}

// pickTempPrefix 选择不与任何现有页面 path 冲突的临时前缀（review Should
// fix）："__wiki_slug_revert__" 未被路径规则保留，仓库可合法存在同名目录；
// 逐次尝试递增后缀直到所有候选临时路径都不与现有 path 冲突。返回 "" 表示
// 无可用的无碰撞前缀（调用方应失败退出）。
func pickTempPrefix(existingPaths map[string]struct{}, pending []pendingPage) string {
	for i := 0; i < 100; i++ {
		prefix := "__wiki_slug_revert__"
		if i > 0 {
			prefix = fmt.Sprintf("__wiki_slug_revert_%d__", i)
		}
		collides := false
		for _, pp := range pending {
			if _, ok := existingPaths[fmt.Sprintf("%s/%d", prefix, pp.id)]; ok {
				collides = true
				break
			}
		}
		if !collides {
			return prefix
		}
	}
	return ""
}

// dropLegacySlugSchema 删除 wiki_namespaces.slug 列与 uniq_wiki_namespace_slug
// 索引。GORM AutoMigrate 不会删除从模型消失的字段，存量库升级后列与索引会
// 残留并继续携带陈旧数据（review P2）。先删索引再删列。
func dropLegacySlugSchema(conn *gorm.DB, result *WikiSlugRevertResult) {
	if !conn.Migrator().HasTable("wiki_namespaces") {
		return
	}
	if conn.Migrator().HasIndex("wiki_namespaces", "uniq_wiki_namespace_slug") {
		// SQLite 与 PostgreSQL 均支持 DROP INDEX IF EXISTS（MySQL 语法不同，已不再支持 MySQL）。
		if err := conn.Exec("DROP INDEX IF EXISTS uniq_wiki_namespace_slug").Error; err != nil {
			result.Failed++
			result.LastFailed = "drop_slug_index:" + err.Error()
			slog.Error("wiki slug revert: drop slug index failed", "err", err)
			return
		}
		result.SlugIndexDropped = true
	}
	if conn.Migrator().HasColumn("wiki_namespaces", "slug") {
		if err := conn.Exec("ALTER TABLE wiki_namespaces DROP COLUMN slug").Error; err != nil {
			result.Failed++
			result.LastFailed = "drop_slug_column:" + err.Error()
			slog.Error("wiki slug revert: drop slug column failed", "err", err)
			return
		}
		result.SlugColumnDropped = true
	}
}
