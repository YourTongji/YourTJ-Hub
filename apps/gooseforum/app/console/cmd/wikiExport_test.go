package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

func TestRenderWikiFrontmatter(t *testing.T) {
	page := &wikiPages.Entity{TopicId: 0, SortOrder: 3}
	fm, err := renderWikiFrontmatter("学校简介", page, "学校始于1907年。\n\n详见正文。")
	if err != nil {
		t.Fatalf("renderWikiFrontmatter: %v", err)
	}
	for _, frag := range []string{
		"---\n",
		"title: 学校简介",
		"order: 3",
		"draft: false",
		"description: 学校始于1907年。",
	} {
		if !strings.Contains(fm, frag) {
			t.Errorf("frontmatter missing %q:\n%s", frag, fm)
		}
	}
}

// TestRenderWikiFrontmatterEscapesYAMLScalar 标题/摘要含 YAML 特殊字符必须正确转义
// （冒号、引号、列表标记），否则生成的种子 markdown 无法被后续 frontmatter 解析。
func TestRenderWikiFrontmatterEscapesYAMLScalar(t *testing.T) {
	page := &wikiPages.Entity{TopicId: 0, SortOrder: 0}
	fm, err := renderWikiFrontmatter(`标题: 带冒号 "and quotes"`, page, "正文")
	if err != nil {
		t.Fatalf("renderWikiFrontmatter: %v", err)
	}
	// yaml.v3 会把含特殊字符的标量加引号包裹。
	if !strings.Contains(fm, `"`) {
		t.Errorf("frontmatter should quote a scalar containing colon/quote:\n%s", fm)
	}
	if !strings.Contains(fm, "标题") {
		t.Errorf("frontmatter should preserve title text:\n%s", fm)
	}
}

func TestRenderWikiFrontmatterOmitsEmptyDescription(t *testing.T) {
	page := &wikiPages.Entity{TopicId: 0, SortOrder: 1}
	fm, err := renderWikiFrontmatter("空白页", page, "")
	if err != nil {
		t.Fatalf("renderWikiFrontmatter: %v", err)
	}
	if strings.Contains(fm, "description") {
		t.Errorf("frontmatter should omit empty description:\n%s", fm)
	}
}

func TestDeriveExcerpt(t *testing.T) {
	long := strings.Repeat("字", 300)
	got := deriveExcerpt(long)
	// 按 rune 截断到 200。
	if len([]rune(got)) != 200 {
		t.Errorf("deriveExcerpt long = %d runes, want 200", len([]rune(got)))
	}
	if got := deriveExcerpt(""); got != "" {
		t.Errorf("deriveExcerpt empty = %q, want empty", got)
	}
	if got := deriveExcerpt("短文本"); got != "短文本" {
		t.Errorf("deriveExcerpt short = %q, want short text", got)
	}
	medium := strings.Repeat("字", 100)
	if got := deriveExcerpt(medium); got != medium {
		t.Errorf("deriveExcerpt medium multibyte text changed: got %d runes", len([]rune(got)))
	}
}

func TestPrepareWikiExportDirRejectsStaleFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stale.md"), []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := prepareWikiExportDir(dir); err == nil {
		t.Fatal("prepareWikiExportDir accepted a non-empty output directory")
	}
}

func TestPrepareWikiExportDirCreatesMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "wiki-seed")
	if err := prepareWikiExportDir(dir); err != nil {
		t.Fatalf("prepareWikiExportDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("output directory was not created: info=%v err=%v", info, err)
	}
}

func TestRenderWikiExportContentNormalizesFrontmatter(t *testing.T) {
	page := &wikiPages.Entity{SortOrder: 2}
	revision := "---\ntitle: 旧标题\ndraft: true\n---\n\n# 正文"
	content, err := renderWikiExportContent("新标题", page, revision, true)
	if err != nil {
		t.Fatalf("renderWikiExportContent: %v", err)
	}
	if strings.Count(content, "---\n") != 2 {
		t.Fatalf("frontmatter delimiter count = %d, want 2:\n%s", strings.Count(content, "---\n"), content)
	}
	if strings.Contains(content, "旧标题") || strings.Contains(content, "draft: true") {
		t.Fatalf("legacy frontmatter leaked into export:\n%s", content)
	}
	if !strings.Contains(content, "title: 新标题") || !strings.HasSuffix(content, "# 正文") {
		t.Fatalf("normalized export missing title/body:\n%s", content)
	}
}
