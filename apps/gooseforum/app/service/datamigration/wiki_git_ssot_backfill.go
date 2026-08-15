package datamigration

import (
	"crypto/sha256"
	"encoding/hex"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"gorm.io/gorm"
)

// WikiGitSSOTBackfillResult v21 回填结果。
type WikiGitSSOTBackfillResult struct {
	PagesBackfilled int
	PagesSkipped    int
	Failed          int
	LastFailed      string
}

// BackfillWikiGitSSOT GitHub SSOT 架构 v21 回填：
// 把存量 wiki 页面的最新 approved 修订内容快照复制到 wiki_pages 投影列
// （title/content/rendered_html/toc/content_hash）。
//
// 背景：wiki 改造后公开读路径直接读 wiki_pages 投影列（不再查修订表）。
// 部署前已存在的 wiki 页面没有投影列数据，必须先回填，否则升级后这些页面
// 在 GitHub 首次同步前会显示空白。回填后 content_hash = 修订正文 sha256，
// 与 GitHub 仓库文件的 hash 不一致时首轮同步自然触发更新（幂等衔接）。
//
// 幂等：content_hash 非空的页面跳过；无 approved 修订的页面跳过（异常态）。
func BackfillWikiGitSSOT() WikiGitSSOTBackfillResult {
	return BackfillWikiGitSSOTWithDB(dbconnect.Connect())
}

func BackfillWikiGitSSOTWithDB(conn *gorm.DB) WikiGitSSOTBackfillResult {
	result := WikiGitSSOTBackfillResult{}
	if !conn.Migrator().HasTable(&wikiPages.Entity{}) || !conn.Migrator().HasTable(&wikiPageRevisions.Entity{}) {
		return result
	}

	const batchSize = 200
	var cursor uint64
	for {
		var batch []wikiPages.Entity
		err := conn.Model(&wikiPages.Entity{}).
			Select("id", "content_hash", "sort_order").
			Where("id > ?", cursor).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error
		if err != nil {
			result.Failed++
			result.LastFailed = "pages_scan:" + err.Error()
			slog.Error("wiki git ssot backfill: scan pages failed", "err", err)
			return result
		}
		for i := range batch {
			page := &batch[i]
			cursor = page.Id
			if page.ContentHash != "" {
				result.PagesSkipped++
				continue
			}
			// 最新 approved 修订（无则跳过，异常态）。
			var latest wikiPageRevisions.Entity
			err := conn.Model(&wikiPageRevisions.Entity{}).
				Where("page_id = ?", page.Id).
				Where("status = ?", wikiPageRevisions.StatusApproved).
				Order("revision_no DESC").
				Order("id DESC").
				First(&latest).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					result.PagesSkipped++
					continue
				}
				result.Failed++
				result.LastFailed = "latest_rev:" + err.Error()
				slog.Error("wiki git ssot backfill: latest revision failed", "pageId", page.Id, "err", err)
				continue
			}
			if latest.Content == "" {
				result.PagesSkipped++
				continue
			}
			sum := sha256.Sum256([]byte(latest.Content))
			hash := hex.EncodeToString(sum[:])
			rendered := markdown2html.PostMarkdownToHTML(latest.Content)
			updates := map[string]any{
				"title":         latest.Title,
				"content":       latest.Content,
				"rendered_html": rendered,
				"content_hash":  hash,
				"sort_order":    page.SortOrder,
			}
			if latest.Toc != "" {
				updates["toc"] = latest.Toc
			}
			if err := conn.Model(&wikiPages.Entity{}).
				Where("id = ?", page.Id).
				Updates(updates).Error; err != nil {
				result.Failed++
				result.LastFailed = "page_update:" + err.Error()
				slog.Error("wiki git ssot backfill: page update failed", "pageId", page.Id, "err", err)
				continue
			}
			result.PagesBackfilled++
		}
		if len(batch) < batchSize {
			break
		}
	}
	return result
}
