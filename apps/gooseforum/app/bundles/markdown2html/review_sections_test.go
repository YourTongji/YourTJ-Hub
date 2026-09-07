package markdown2html

import (
	"strings"
	"testing"
)

func TestNormalizeCourseReviewSections(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "standalone heading with full-width colon",
			in:   "课程内容：\n测试",
			want: "## 课程内容\n测试",
		},
		{
			name: "standalone heading with ascii colon",
			in:   "考核标准:",
			want: "## 考核标准",
		},
		{
			name: "inline heading splits then keeps rest",
			in:   "授课质量：好",
			want: "## 授课质量\n好",
		},
		{
			name: "legacy full text",
			in:   "课程内容：\n\n上课自由度：\n自由\n\n考核标准：\n严格",
			want: "## 课程内容\n\n## 上课自由度\n自由\n\n## 考核标准\n严格",
		},
		{
			name: "already markdown heading untouched",
			in:   "## 课程内容\n测试",
			want: "## 课程内容\n测试",
		},
		{
			name: "fenced code untouched",
			in:   "```\n课程内容：x\n```",
			want: "```\n课程内容：x\n```",
		},
		{
			name: "long heading wins over short prefix",
			in:   "授课质量与给分：A",
			want: "## 授课质量与给分\nA",
		},
		{
			name: "blank input",
			in:   "",
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeCourseReviewSections(tc.in)
			if got != tc.want {
				t.Fatalf("NormalizeCourseReviewSections(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestPostMarkdownToHTMLRendersLegacyReviewHeadings(t *testing.T) {
	html := PostMarkdownToHTML(NormalizeCourseReviewSections("课程内容：\n测试\n\n授课质量：\n好"))
	for _, want := range []string{
		`<h2 id="课程内容">课程内容</h2>`,
		`<h2 id="授课质量">授课质量</h2>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered HTML to contain %q, got %s", want, html)
		}
	}
}
