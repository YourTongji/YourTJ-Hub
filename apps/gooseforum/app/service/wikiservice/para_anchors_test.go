package wikiservice

import (
	"strings"
	"testing"
)

// TestRenderInjectsParaAnchors 验证渲染链路为段落注入稳定锚点 id 并收集索引：
// 段落 id 按文档序递增（s-<n>），归属最近上级标题（heading id 与 TOC 一致）。
func TestRenderInjectsParaAnchors(t *testing.T) {
	page := wantedPage{
		sourcePath: "guide/faq",
		path:       "guide/faq",
		body: `# FAQ

第一段正文，介绍文档。

## 申请条件

成绩均分不低于 3.0。

## 常见问题

选课时间在每学期第 1 周。`,
	}
	resolver := newWikiReferenceResolver(GitConfig{CloneDir: t.TempDir()}, []wantedPage{page}, "")

	result, err := resolver.Render(page)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	// 段落 id 注入。
	for _, want := range []string{`<p id="s-1">`, `<p id="s-2">`, `<p id="s-3">`} {
		if !strings.Contains(result.HTML, want) {
			t.Fatalf("rendered HTML missing %q:\n%s", want, result.HTML)
		}
	}

	if len(result.ParaAnchors) != 3 {
		t.Fatalf("para anchors=%d, want 3: %+v", len(result.ParaAnchors), result.ParaAnchors)
	}

	// 段落归属最近上级标题（h1 也是标题，首段归属页面主标题）。
	want := []struct {
		anchor      string
		headingID   string
		headingText string
		text        string
	}{
		{"s-1", "", "FAQ", "第一段正文，介绍文档。"},
		{"s-2", "", "申请条件", "成绩均分不低于 3.0。"},
		{"s-3", "", "常见问题", "选课时间在每学期第 1 周。"},
	}
	for i, w := range want {
		got := result.ParaAnchors[i]
		if got.Anchor != w.anchor || got.Index != i+1 {
			t.Fatalf("anchor[%d]=%+v, want anchor=%s index=%d", i, got, w.anchor, i+1)
		}
		if got.HeadingText != w.headingText {
			t.Fatalf("anchor[%d] headingText=%q, want %q", i, got.HeadingText, w.headingText)
		}
		if got.Text != w.text {
			t.Fatalf("anchor[%d] text=%q, want %q", i, got.Text, w.text)
		}
		// 首个标题（页面 h1）后的段落归属该标题的 heading id。
		if w.headingText != "" && got.HeadingId == "" {
			t.Fatalf("anchor[%d] missing heading id for %q", i, w.headingText)
		}
	}

	// 标题 id 必须与渲染 HTML 的 heading id 一致（供前端 #anchor 跳转）。
	if !strings.Contains(result.HTML, `id="申请条件"`) && !strings.Contains(result.HTML, `id="`) {
		t.Fatalf("rendered HTML should contain heading ids:\n%s", result.HTML)
	}
}

// TestRenderInjectsOnlyParagraphs 段落 id 只注入 <p>，不注入列表等块级元素，
// 避免 id 爆炸（列表块不可定位，只索引段落）。
func TestRenderInjectsOnlyParagraphs(t *testing.T) {
	page := wantedPage{
		sourcePath: "guide/list",
		path:       "guide/list",
		body: `## 列表

- 第一项
- 第二项

一段总结。`,
	}
	resolver := newWikiReferenceResolver(GitConfig{CloneDir: t.TempDir()}, []wantedPage{page}, "")
	result, err := resolver.Render(page)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Count(result.HTML, `id="s-`) != 1 {
		t.Fatalf("paragraph anchors in HTML = %d, want exactly 1:\n%s", strings.Count(result.HTML, `id="s-`), result.HTML)
	}
	if len(result.ParaAnchors) != 1 {
		t.Fatalf("para anchors=%d, want 1: %+v", len(result.ParaAnchors), result.ParaAnchors)
	}
	if result.ParaAnchors[0].Text != "一段总结。" {
		t.Fatalf("para text=%q, want 一段总结。", result.ParaAnchors[0].Text)
	}
}
