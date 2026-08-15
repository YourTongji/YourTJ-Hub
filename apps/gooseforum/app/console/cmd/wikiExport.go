package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func init() {
	cmd := &cobra.Command{
		Use:   "wiki-export <output-dir>",
		Short: "Export wiki pages to a hierarchical markdown tree (seed for the git repo, reverse of wiki-import)",
		Args:  cobra.ExactArgs(1),
		RunE:  runWikiExport,
		// Export reads the compatibility revision table directly and must not run
		// unrelated startup migrations before validating its data source.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {},
	}
	cmd.Flags().Bool("dry-run", false, "validate and print what would be exported, do not write")
	cmd.Flags().Bool("with-frontmatter", true, "write YAML frontmatter (title/order/description/draft) per wiki-markdown-format.md")
	appendCommand(cmd)
}

// ---------------------------------------------------------------------------
// wiki-export：把兼容期 wiki_page_revisions 导出为 git 仓库种子。
// 方向与 wiki-import 相反：从 DB 读 wiki_pages + 最新 approved 修订，
// 导出为 git 仓库规范的 markdown 文件树 <namespace>/<path>.md，每页带
// frontmatter（docs/product/wiki-markdown-format.md）。生成的内容即为
// 新仓库 YourTongji/YourTJ-Wiki 的种子。
// 表已缺失时 hard fail，绝不静默空导出。
// ---------------------------------------------------------------------------

func runWikiExport(cmd *cobra.Command, args []string) error {
	outDir := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	withFrontmatter, _ := cmd.Flags().GetBool("with-frontmatter")

	// 数据源表缺失时必须失败，避免 0 页导出掩盖异常 schema。
	if !dbconnect.Connect().Migrator().HasTable("wiki_page_revisions") {
		return fmt.Errorf("wiki_page_revisions table does not exist; cannot export wiki source data")
	}

	pages := wikiPages.ListAll()
	if len(pages) == 0 {
		return fmt.Errorf("wiki-export found no wiki pages; refusing to produce an empty seed")
	}
	type exportItem struct {
		page *wikiPages.Entity
		rev  wikiPageRevisions.Entity
	}
	items := make([]exportItem, 0, len(pages))
	missing := make([]string, 0)
	for _, page := range pages {
		rev := wikiPageRevisions.GetLatestApproved(page.Id)
		if rev.Id == 0 {
			missing = append(missing, page.Path+".md")
			continue
		}
		items = append(items, exportItem{page: page, rev: rev})
	}
	if len(missing) > 0 {
		return fmt.Errorf("wiki-export cannot produce a complete seed: %d page(s) have no approved revision: %s", len(missing), strings.Join(missing, ", "))
	}
	if !dryRun {
		if err := prepareWikiExportDir(outDir); err != nil {
			return err
		}
	}

	var written int
	var errs []string
	for _, item := range items {
		page, rev := item.page, item.rev
		relPath := page.Path + ".md"
		if dryRun {
			fmt.Printf("[dry-run] would write %-48s title=%q content=%dB\n", relPath, rev.Title, len(rev.Content))
			written++
			continue
		}
		content, err := renderWikiExportContent(rev.Title, page, rev.Content, withFrontmatter)
		if err != nil {
			errs = append(errs, fmt.Sprintf("frontmatter %s: %v", relPath, err))
			continue
		}
		// review LOW：防御路径穿越——path 段经 ValidatePath 校验（^[a-z0-9]+(-[a-z0-9]+)*$）
		// 不含 ".."，仅历史脏数据/手工改库有风险；Clean 后断言仍在 outDir 内，越界即失败。
		target := filepath.Join(outDir, filepath.FromSlash(relPath))
		cleaned := filepath.Clean(target)
		if !strings.HasPrefix(cleaned, filepath.Clean(outDir)+string(os.PathSeparator)) {
			errs = append(errs, fmt.Sprintf("path escape blocked: %s", relPath))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(cleaned), 0o755); err != nil {
			errs = append(errs, fmt.Sprintf("mkdir %s: %v", filepath.Dir(cleaned), err))
			continue
		}
		if err := os.WriteFile(cleaned, []byte(content), 0o644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", relPath, err))
			continue
		}
		written++
		fmt.Printf("  wrote %-48s title=%q\n", relPath, rev.Title)
	}

	fmt.Printf("wiki-export: output=%s dryRun=%v frontmatter=%v\n", outDir, dryRun, withFrontmatter)
	fmt.Printf("  total=%d written=%d errors=%d\n", len(items), written, len(errs))
	for _, e := range errs {
		fmt.Printf("  [error] %s\n", e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("wiki-export finished with %d error(s)", len(errs))
	}
	return nil
}

func prepareWikiExportDir(outDir string) error {
	entries, err := os.ReadDir(outDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read output dir: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("wiki-export output dir must be empty: %s", outDir)
	}
	return nil
}

func renderWikiExportContent(title string, page *wikiPages.Entity, revisionContent string, withFrontmatter bool) (string, error) {
	_, body := markdown2html.SplitFrontmatter(revisionContent)
	if !withFrontmatter {
		return body, nil
	}
	fm, err := renderWikiFrontmatter(title, page, body)
	if err != nil {
		return "", err
	}
	return fm + "\n" + body, nil
}

// renderWikiFrontmatter 按 wiki-markdown-format.md §2 生成 YAML frontmatter。
// title 必填；description 取 topics.Excerpt（为空则省略，由 ExtractDescription 兜底）；
// order 映射 sort_order；draft 恒为 false（导出即发布）。yaml.v3 对 map 键排序，
// 输出稳定。
func renderWikiFrontmatter(title string, page *wikiPages.Entity, content string) (string, error) {
	fm := map[string]any{
		"title": title,
		"order": page.SortOrder,
		"draft": false,
	}
	desc := excerptOf(page.TopicId)
	if strings.TrimSpace(desc) == "" {
		desc = deriveExcerpt(content)
	}
	if desc != "" {
		fm["description"] = desc
	}
	data, err := yaml.Marshal(fm)
	if err != nil {
		return "", err
	}
	return "---\n" + string(data) + "---", nil
}

// excerptOf 返回页面 topic 的物化摘要（写路径已用 ExtractDescription 同步）。
func excerptOf(topicID uint64) string {
	if topicID == 0 {
		return ""
	}
	return topics.Get(topicID).Excerpt
}

// deriveExcerpt 当 topic 无摘要时从修订正文提取（与写路径 ExtractDescription 对齐的
// 轻量回退；不引 markdown2html，避免导出命令拉入渲染依赖）。
// description 为单行摘要语义：先把连续空白/换行压缩为单空格，再按 rune 截断到 200。
func deriveExcerpt(content string) string {
	clean := strings.Join(strings.Fields(content), " ")
	r := []rune(clean)
	if len(r) <= 200 {
		return clean
	}
	return string(r[:200])
}
