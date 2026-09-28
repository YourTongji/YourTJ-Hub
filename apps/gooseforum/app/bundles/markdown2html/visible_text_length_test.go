package markdown2html

import (
	"strings"
	"testing"
)

func TestVisibleTextLength(t *testing.T) {
	cases := []struct {
		name     string
		markdown string
		want     int
	}{
		{"ASCII 正文按码点计数", "hello", 5},
		{"CJK 正文按码点计数", "你好世界", 4},
		{"emoji 记 1 个码点", "😀😀", 2},
		{"零宽空格不计入", strings.Repeat("\u200B", 5), 0},
		{"实体编码零宽字符不计入", "&#8203;&#x200B;&#8203;", 0},
		{"emoji 序列中的零宽连接符不计入", "👨\u200D👩\u200D👧", 3},
		{"BOM 与词连接符不计入", "\uFEFF\u2060\uFEFF", 0},
		{"粗体标记不计入", "**a**", 1},
		{"粗体 CJK", "**你好**", 2},
		{"斜体标记不计入", "_ab_", 2},
		{"标题标记不计入", "# 标题", 2},
		{"引用标记不计入", "> 引用", 2},
		{"列表标记不计入", "- 一\n- 二", 2},
		{"任务列表勾选框不计入", "- [x] 完成", 2},
		{"链接只计标签", "[ab](https://example.com)", 2},
		{"链接标签为 CJK", "[你好](https://example.com)", 2},
		{"裸链接不计入", "https://example.com", 0},
		{"尖括号自动链接不计入", "<https://example.com>", 0},
		{"邮箱自动链接不计入", "foo@example.com", 0},
		{"空链接目标只计标签", "[x]()", 1},
		{"引用式链接只计标签", "[x][ref]\n\n[ref]: https://example.com", 1},
		{"引用式图片不计入", "![x][img]\n\n[img]: https://example.com/i.png", 0},
		{"表格单元格文字计入", "| 你好 | b |\n|---|---|", 3},
		{"图片语法不计入", "![x](https://example.com/i.png)", 0},
		{"图片 alt 为 CJK 也不计入", "![图片](https://example.com/i.png)", 0},
		{"贴纸 token 不计入", "[:sticker:smile:]", 0},
		{"贴纸 token 与正文混排", "你好[:sticker:smile:]世界", 4},
		{"多个贴纸 token 位于首尾", "[:sticker:a:]你好[:sticker:b:]", 2},
		{"代码块内的贴纸 token 保持字面量", "`[:sticker:smile:]`", 17},
		{"行内代码计入可见文字", "`code`", 4},
		{"围栏代码块计入可见文字", "```\ncode\n```", 4},
		{"缩进代码块计入可见文字", "    code", 4},
		{"HTML 标签剥离但内部文字计入", "<b>hi</b>", 2},
		{"HTML 块不计入", "<div>hi</div>", 0},
		{"HTML 实体按渲染结果解码", "&amp;&lt;", 2},
		{"弯引号省略号按渲染结果计数", "a...b", 3},
		{"转义标点按渲染结果计入", `\*a\*`, 3},
		{"软换行各计 1", "a\nb", 3},
		{"硬换行各计 1", "a  \nb", 3},
		{"段落分隔不计入", "a\n\nb", 2},
		{"首尾空白不计入", "  ab  ", 2},
		{"纯空白为 0", "   \n\n  ", 0},
		{"分隔线不计入", "---", 0},
		{"Setext 标题下划线不计入", "标题\n===", 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VisibleTextLength(tc.markdown); got != tc.want {
				t.Fatalf("VisibleTextLength(%q) = %d, want %d", tc.markdown, got, tc.want)
			}
		})
	}
}

// 纯文本（无 Markdown 语法）必须与旧的 TrimSpace + 码点计数口径一致，
// 避免规则切换改变普通评论的通过/拒绝结果。
func TestVisibleTextLengthMatchesPlainTextRuneCount(t *testing.T) {
	for _, plain := range []string{"hello", "你好世界", "a\nb", "  ab  ", "a b c"} {
		if got, want := VisibleTextLength(plain), len([]rune(strings.TrimSpace(plain))); got != want {
			t.Fatalf("VisibleTextLength(%q) = %d, want plain rune count %d", plain, got, want)
		}
	}
}
