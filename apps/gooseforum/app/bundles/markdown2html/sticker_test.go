package markdown2html

import (
	"strings"
	"testing"
)

func TestExpandStickerTokensBasic(t *testing.T) {
	resolve := func(name string) (string, bool) {
		if name == "滑稽" {
			return "/file/img/stickers/huaji.png", true
		}
		return "", false
	}
	got := ExpandStickerTokens("你好 [:sticker:滑稽:] 世界", resolve)
	want := "你好 ![sticker:滑稽](/file/img/stickers/huaji.png) 世界"
	if got != want {
		t.Fatalf("ExpandStickerTokens = %q, want %q", got, want)
	}
}

func TestExpandStickerTokensUnknownKept(t *testing.T) {
	resolve := func(string) (string, bool) { return "", false }
	src := "未知 [:sticker:不存在的:] 保持原样"
	if got := ExpandStickerTokens(src, resolve); got != src {
		t.Fatalf("unknown token rewritten: %q", got)
	}
}

func TestExpandStickerTokensDisabledKept(t *testing.T) {
	resolve := func(name string) (string, bool) {
		if name == "disabled" {
			return "", false
		}
		return "/file/img/x.png", true
	}
	got := ExpandStickerTokens("[:sticker:disabled:] and [:sticker:ok:]", resolve)
	if !strings.Contains(got, "[:sticker:disabled:]") {
		t.Fatalf("disabled token expanded: %q", got)
	}
	if !strings.Contains(got, "![sticker:ok]") {
		t.Fatalf("enabled token not expanded: %q", got)
	}
}

func TestExpandStickerTokensSkipsCodeSpanAndFence(t *testing.T) {
	resolve := func(string) (string, bool) { return "/file/img/x.png", true }
	src := "代码 `[:sticker:a:]` 与\n\n```\n[:sticker:b:]\n```\n"
	got := ExpandStickerTokens(src, resolve)
	if strings.Contains(got, "![sticker:a]") || strings.Contains(got, "![sticker:b]") {
		t.Fatalf("token inside code expanded: %q", got)
	}
}

func TestExpandStickerTokensSkipsLinkAndImage(t *testing.T) {
	resolve := func(string) (string, bool) { return "/file/img/x.png", true }
	src := "[文本 [:sticker:a:]](/target) 与 ![alt [:sticker:b:]](/img.png)"
	got := ExpandStickerTokens(src, resolve)
	if strings.Contains(got, "![sticker:a]") || strings.Contains(got, "![sticker:b]") {
		t.Fatalf("token inside link/image expanded: %q", got)
	}
}

func TestExpandStickerTokensSkipsMath(t *testing.T) {
	resolve := func(string) (string, bool) { return "/file/img/x.png", true }
	src := "$x + [:sticker:a:]$ 公式内不展开，$y$ 后 [:sticker:b:] 展开"
	got := ExpandStickerTokens(src, resolve)
	if strings.Count(got, "![sticker:") != 1 {
		t.Fatalf("math-adjacent expansion count = %d, want 1: %q", strings.Count(got, "![sticker:"), got)
	}
	if strings.Contains(got, "x + ![sticker:") {
		t.Fatalf("token inside math expanded: %q", got)
	}
}

func TestExpandStickerTokensNoTokensFastPath(t *testing.T) {
	src := "普通文本不含 token"
	resolve := func(string) (string, bool) { return "/x", true }
	if got := ExpandStickerTokens(src, resolve); got != src {
		t.Fatalf("plain text modified: %q", got)
	}
}

func TestStickerTokenReBoundaries(t *testing.T) {
	cases := map[string]bool{
		"[:sticker:a:]":                               true,
		"[:sticker:长_名-字2:]":                          true,
		"[:sticker:]":                                 false,
		"[:sticker:a:b:]":                             false,
		"[:sticker:带 空格:]":                            false,
		"[:sticker:带]括号:]":                            false,
		"[:sticker:" + strings.Repeat("长", 65) + ":]": false,
	}
	for input, want := range cases {
		if got := stickerTokenRe.MatchString(input); got != want {
			t.Fatalf("stickerTokenRe(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestStickerReviewExclusionContexts(t *testing.T) {
	for _, source := range []string{
		"[docs](https://host/[:sticker:smile:])",
		"![alt](https://host/[:sticker:smile:])",
		"<https://host/[:sticker:smile:]>",
		"[**[:sticker:smile:]**](/target)",
		"[ref]: https://host/[:sticker:smile:]\n\n[text][ref]",
		"\\[:sticker:smile:]",
	} {
		got := ExpandStickerTokens(source, func(string) (string, bool) { return "/x.png", true })
		if got != source {
			t.Errorf("excluded source changed: %q -> %q", source, got)
		}
	}
}
func TestStickerReviewRepeatedNamesResolveOnce(t *testing.T) {
	calls := 0
	ExpandStickerTokens(strings.Repeat("[:sticker:smile:] ", 1000), func(string) (string, bool) { calls++; return "/x.png", true })
	if calls != 1 {
		t.Fatalf("resolver called %d times, want 1", calls)
	}
}
func TestStickerReviewEscapesURL(t *testing.T) {
	rendered := PostMarkdownToHTML(ExpandStickerTokens("[:sticker:smile:]", func(string) (string, bool) { return "https://cdn.example/a)b.png", true }))
	if !strings.Contains(rendered, `src="https://cdn.example/a%29b.png"`) {
		t.Fatalf("URL was corrupted: %s", rendered)
	}
}

func TestRenderedStickerMarkerPreservesSanitizerAndExclusionBoundaries(t *testing.T) {
	source := "@member [:sticker:smile:] [:sticker:unsafe:] [:sticker:unknown:]\n\n" +
		"`[:sticker:smile:]` [[:sticker:smile:]](/target)\n\n" +
		"![sticker:smile](/photo.png \"data-gf-sticker=smile\")\n\n" +
		"<img src=\"/forged.png\" alt=\"sticker:smile\" data-gf-sticker=\"smile\">"
	got := RenderWithStickerTokens(source, func(name string) (string, bool) {
		switch name {
		case "smile":
			return "https://cdn.example/a)b.png?x=1&y=2", true
		case "unsafe":
			return "javascript:alert(1)", true
		default:
			return "", false
		}
	}, func(expanded string) string {
		return PostMarkdownToHTMLWithMentions(expanded, map[string]uint64{"member": 42})
	})
	if strings.Count(got, `data-gf-sticker="smile"`) != 1 {
		t.Fatalf("ordinary or excluded content gained sticker provenance: %s", got)
	}
	for _, unexpected := range []string{"javascript:", "/forged.png", "GFSTICKER"} {
		if strings.Contains(got, unexpected) {
			t.Fatalf("unsafe URL, forged HTML or internal marker leaked: %s", got)
		}
	}
	for _, expected := range []string{`src="https://cdn.example/a%29b.png?x=1&amp;y=2"`, `src="/photo.png"`, `loading="lazy"`, `[:sticker:unknown:]`, `<code>[:sticker:smile:]</code>`, `<a href="/u/42">@member</a>`} {
		if !strings.Contains(got, expected) {
			t.Fatalf("safe image or fallback content changed (missing %s): %s", expected, got)
		}
	}
}
