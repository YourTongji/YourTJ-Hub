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
