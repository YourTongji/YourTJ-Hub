package markdown2html

import (
	"strings"
	"testing"
)

func TestSplitFrontmatterStripsBlock(t *testing.T) {
	content := "---\ntitle: 入门指南\ndescription: 快速开始\n---\n\n# 正文\n\n内容"
	meta, body := SplitFrontmatter(content)
	if meta == nil {
		t.Fatal("SplitFrontmatter returned nil meta for valid frontmatter")
	}
	if got, _ := meta["title"].(string); got != "入门指南" {
		t.Fatalf("meta title=%v, want 入门指南", meta["title"])
	}
	if got, _ := meta["description"].(string); got != "快速开始" {
		t.Fatalf("meta description=%v, want 快速开始", meta["description"])
	}
	if body != "\n# 正文\n\n内容" {
		t.Fatalf("body=%q, want %q", body, "\n# 正文\n\n内容")
	}
	if strings.Contains(body, "---") || strings.Contains(body, "title:") {
		t.Fatalf("body must not contain frontmatter markers, got %q", body)
	}
}

func TestSplitFrontmatterNoFrontmatter(t *testing.T) {
	content := "# 直接正文\n\n没有元数据"
	meta, body := SplitFrontmatter(content)
	if meta != nil {
		t.Fatalf("meta=%v, want nil for content without frontmatter", meta)
	}
	if body != content {
		t.Fatalf("body=%q, want original content %q", body, content)
	}
}

func TestSplitFrontmatterUnclosedDelimiterFallsBack(t *testing.T) {
	// 首行是 --- 但没有闭合行：按普通正文处理（body 原样，不剥离）。
	content := "---\ntitle: 未闭合\n\n# 正文"
	meta, body := SplitFrontmatter(content)
	if meta != nil {
		t.Fatalf("meta=%v, want nil for unclosed frontmatter", meta)
	}
	if body != content {
		t.Fatalf("body=%q, want original %q", body, content)
	}
}

func TestSplitFrontmatterInvalidYAMLFallsBack(t *testing.T) {
	content := "---\ntitle: [未闭合\n---\n\n# 正文"
	meta, body := SplitFrontmatter(content)
	if meta != nil {
		t.Fatalf("meta=%v, want nil for invalid yaml", meta)
	}
	// 宽松层仍剥离块：body 不含元数据行。
	if strings.Contains(body, "title:") || strings.Contains(body, "---") {
		t.Fatalf("body must not contain frontmatter, got %q", body)
	}
}

func TestSplitFrontmatterEmptyBlock(t *testing.T) {
	content := "---\n---\n\n# 正文"
	meta, body := SplitFrontmatter(content)
	if meta == nil {
		t.Fatal("empty frontmatter block should parse to empty map, got nil")
	}
	if body != "\n# 正文" {
		t.Fatalf("body=%q, want %q", body, "\n# 正文")
	}
}

func TestSplitFrontmatterCRLF(t *testing.T) {
	content := "---\r\ntitle: CRLF\r\n---\r\n\r\n# 正文"
	meta, body := SplitFrontmatter(content)
	if meta == nil {
		t.Fatal("CRLF frontmatter should parse")
	}
	if got, _ := meta["title"].(string); got != "CRLF" {
		t.Fatalf("meta title=%v, want CRLF", meta["title"])
	}
	if strings.Contains(body, "title:") {
		t.Fatalf("body must not contain frontmatter, got %q", body)
	}
}

func TestParseFrontmatterTyped(t *testing.T) {
	content := "---\ntitle: 指南\norder: 2\ndescription: 简短描述\ntags:\n  - 入门\n  - 教程\ndraft: true\n---\n\n# 正文"
	fm, body, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm.Title != "指南" {
		t.Fatalf("title=%q, want 指南", fm.Title)
	}
	if fm.Order == nil || *fm.Order != 2 {
		t.Fatalf("order=%v, want 2", fm.Order)
	}
	if fm.Description != "简短描述" {
		t.Fatalf("description=%q, want 简短描述", fm.Description)
	}
	if len(fm.Tags) != 2 || fm.Tags[0] != "入门" || fm.Tags[1] != "教程" {
		t.Fatalf("tags=%v, want [入门 教程]", fm.Tags)
	}
	if !fm.Draft {
		t.Fatal("draft=true not parsed")
	}
	if strings.Contains(body, "title:") || strings.Contains(body, "---") {
		t.Fatalf("body must not contain frontmatter, got %q", body)
	}
}

func TestParseFrontmatterDefaults(t *testing.T) {
	content := "---\ntitle: 最小页面\n---\n\n正文"
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm.Order != nil {
		t.Fatalf("order=%v, want nil default", fm.Order)
	}
	if fm.Draft {
		t.Fatal("draft default should be false")
	}
	if len(fm.Tags) != 0 {
		t.Fatalf("tags=%v, want empty", fm.Tags)
	}
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	fm, body, err := ParseFrontmatter("# 纯正文")
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm.Title != "" {
		t.Fatalf("title=%q, want empty", fm.Title)
	}
	if body != "# 纯正文" {
		t.Fatalf("body=%q, want original", body)
	}
}

func TestParseFrontmatterRejectsInvalid(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"yaml syntax error", "---\ntitle: [未闭合\n---\n\n正文"},
		{"missing title", "---\ndescription: 无标题\n---\n\n正文"},
		{"empty title", "---\ntitle: \"\"\n---\n\n正文"},
		{"title not string", "---\ntitle: 123\n---\n\n正文"},
		{"order not int", "---\ntitle: 标题\norder: abc\n---\n\n正文"},
		{"description not string", "---\ntitle: 标题\ndescription: [1,2]\n---\n\n正文"},
		{"tags not list", "---\ntitle: 标题\ntags: 标签\n---\n\n正文"},
		{"tags item not string", "---\ntitle: 标题\ntags: [1]\n---\n\n正文"},
		{"draft not bool", "---\ntitle: 标题\ndraft: maybe\n---\n\n正文"},
		{"title too long", "---\ntitle: " + strings.Repeat("长", FrontmatterTitleMaxLen+1) + "\n---\n\n正文"},
		{"description too long", "---\ntitle: 标题\ndescription: " + strings.Repeat("长", FrontmatterDescriptionMaxLen+1) + "\n---\n\n正文"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := ParseFrontmatter(tc.content); err == nil {
				t.Fatalf("ParseFrontmatter(%q) should fail", tc.name)
			}
		})
	}
}

func TestParseFrontmatterAcceptsZeroValues(t *testing.T) {
	// order:0 / draft:false / 空 tags 是合法边界值，不得误报。
	content := "---\ntitle: 边界\ndescription: \"\"\norder: 0\ntags: []\ndraft: false\n---\n\n正文"
	fm, _, err := ParseFrontmatter(content)
	if err != nil {
		t.Fatalf("ParseFrontmatter: %v", err)
	}
	if fm.Order == nil || *fm.Order != 0 {
		t.Fatalf("order=%v, want 0", fm.Order)
	}
	if fm.Draft {
		t.Fatal("draft should be false")
	}
}
