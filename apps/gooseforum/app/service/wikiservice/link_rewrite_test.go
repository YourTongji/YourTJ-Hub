package wikiservice

import (
	"strings"
	"testing"
)

// wikiRewriteTestCtx 相对链接重写测试上下文：仓库文件清单 + 页面对照
// （含 slug 命名空间：中文目录 同济新手教程 → URL key tongji-freshman-guide）。
func wikiRewriteTestCtx() linkRewriteContext {
	return linkRewriteContext{
		cfg: GitConfig{Repo: "https://github.com/YourTongji/YourTJ-Wiki.git", Branch: "main"},
		pageBySource: map[string]string{
			"guide/start":     "guide/start",
			"guide/other":     "guide/other",
			"guide/index":     "guide/index",
			"guide/sub/start": "guide/sub/start",
			"guide/sub/other": "guide/sub/other",
			"guide/sub/index": "guide/sub/index",
			"同济新手教程/start":    "tongji-freshman-guide/start",
			"同济新手教程/index":    "tongji-freshman-guide/index",
		},
		repoFiles: map[string]struct{}{
			"guide/start.md":     {},
			"guide/other.md":     {},
			"guide/index.md":     {},
			"guide/sub/start.md": {},
			"guide/sub/other.md": {},
			"guide/sub/index.md": {},
			"assets/a.png":       {},
			"assets/my img.png":  {},
			"files/x.zip":        {},
			"README.md":          {},
		},
	}
}

// TestRewriteWikiRef 相对链接/资源重写规则（渲染后 HTML 上的 <a href>/<img src>）：
// 页内 .md 链接 → /wiki/<path>；图片/附件 → GitHub raw URL；外部/锚点/查询/
// 协议相对/站内 /wiki 路由原样保留；目录链接 → index.md；缺失/越界 → 错误。
func TestRewriteWikiRef(t *testing.T) {
	ctx := wikiRewriteTestCtx()
	cases := []struct {
		name       string
		html       string
		sourcePath string
		want       string
		wantErr    bool
		errSub     string
	}{
		{"relative page link", `<a href="other.md">next</a>`, "guide/start", `href="/wiki/guide/other"`, false, ""},
		{"nested page link", `<a href="sub/other.md">x</a>`, "guide/start", `href="/wiki/guide/sub/other"`, false, ""},
		{"parent page link", `<a href="../other.md">x</a>`, "guide/sub/start", `href="/wiki/guide/other"`, false, ""},
		{"anchor preserved", `<a href="#sec">x</a>`, "guide/start", `href="#sec"`, false, ""},
		{"query preserved", `<a href="?v=1">x</a>`, "guide/start", `href="?v=1"`, false, ""},
		{"query and anchor on page link", `<a href="other.md?v=2#s">x</a>`, "guide/start", `href="/wiki/guide/other?v=2#s"`, false, ""},
		{"external http", `<a href="https://example.com/x">x</a>`, "guide/start", `href="https://example.com/x"`, false, ""},
		{"protocol-relative", `<a href="//example.com/x">x</a>`, "guide/start", `href="//example.com/x"`, false, ""},
		{"mailto", `<a href="mailto:a@b.c">x</a>`, "guide/start", `href="mailto:a@b.c"`, false, ""},
		{"data image", `<img src="data:image/png;base64,AA==">`, "guide/start", `src="data:image/png;base64,AA=="`, false, ""},
		{"image relative", `<img src="../assets/a.png">`, "guide/start", `src="https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/assets/a.png"`, false, ""},
		{"attachment relative", `<a href="../files/x.zip">x</a>`, "guide/start", `href="https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/files/x.zip"`, false, ""},
		{"root-relative asset", `<img src="/assets/a.png">`, "guide/start", `src="https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/assets/a.png"`, false, ""},
		{"root-relative page", `<a href="/guide/other.md">x</a>`, "guide/start", `href="/wiki/guide/other"`, false, ""},
		{"wiki route kept", `<a href="/wiki/guide/other">x</a>`, "guide/start", `href="/wiki/guide/other"`, false, ""},
		{"site root kept", `<a href="/">x</a>`, "guide/start", `href="/"`, false, ""},
		{"dir link to index", `<a href="./">x</a>`, "guide/start", `href="/wiki/guide/index"`, false, ""},
		{"dir dot link", `<a href=".">x</a>`, "guide/sub/start", `href="/wiki/guide/sub/index"`, false, ""},
		{"percent-encoded asset", `<img src="../assets/my%20img.png">`, "guide/start", `src="https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/assets/my%20img.png"`, false, ""},
		{"unicode namespace page link", `<a href="start.md">x</a>`, "同济新手教程/index", `href="/wiki/tongji-freshman-guide/start"`, false, ""},
		{"broken page link", `<a href="missing.md">x</a>`, "guide/start", "", true, "no such wiki page"},
		{"non-page md", `<a href="/README.md">x</a>`, "guide/start", "", true, "not a projected wiki page"},
		{"broken asset", `<img src="../assets/nope.png">`, "guide/start", "", true, "no such file"},
		{"escape repo root", `<a href="../../x.png">x</a>`, "guide/start", "", true, "outside the wiki repository root"},
		{"bad percent encoding", `<a href="bad%zz.md">x</a>`, "guide/start", "", true, "invalid percent-encoding"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := rewriteRenderedWikiHTML(ctx, tc.html, tc.sourcePath)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				if !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("error %q missing substring %q", err, tc.errSub)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("output %q missing expected %q", got, tc.want)
			}
		})
	}
}

// TestRenderWikiPageHTMLMarkdownEscaping 转义链接与代码块中的链接在渲染后不是
// <a>/<img> 元素，不得被重写；正常链接照常重写（escaping 处理是确定性的）。
func TestRenderWikiPageHTMLMarkdownEscaping(t *testing.T) {
	ctx := wikiRewriteTestCtx()
	body := "[下一页](other.md)\n\n\\[转义](other.md)\n\n```\n[代码](other.md)\n```"
	got, err := renderWikiPageHTML(ctx, wantedPage{body: body, sourcePath: "guide/start"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Count(got, `href="/wiki/guide/other"`) != 1 {
		t.Fatalf("exactly one rewritten link expected, escaped/code links must stay literal: %s", got)
	}
	if !strings.Contains(got, "[转义](other.md)") || !strings.Contains(got, "[代码](other.md)") {
		t.Fatalf("escaped/code link text should remain literal: %s", got)
	}
}

// TestGitConfigRawURL GitHub raw 外链：逐段转义；repo 未配置时返回空串。
func TestGitConfigRawURL(t *testing.T) {
	cfg := GitConfig{Repo: "https://github.com/YourTongji/YourTJ-Wiki.git", Branch: "main"}
	got := cfg.RawURL("assets/C# a.png")
	want := "https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/assets/C%23%20a.png"
	if got != want {
		t.Fatalf("RawURL = %q, want %q", got, want)
	}
	got = cfg.RawURL("guide/start")
	want = "https://raw.githubusercontent.com/YourTongji/YourTJ-Wiki/main/guide/start"
	if got != want {
		t.Fatalf("RawURL(plain) = %q, want %q", got, want)
	}
	if got := (GitConfig{Repo: "", Branch: "main"}).RawURL("assets/a.png"); got != "" {
		t.Fatalf("RawURL without repo = %q, want empty", got)
	}
}
