package markdown2html

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type blankLineFixture struct {
	Marker      string `json:"marker"`
	MaxRendered int    `json:"maxRendered"`
	Cases       []struct {
		Name      string  `json:"name"`
		Markdown  string  `json:"markdown"`
		Marked    *string `json:"marked"`
		Unlimited bool    `json:"unlimited"`
	} `json:"cases"`
}

func TestMarkBlankLinesSharedFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate blank line test file")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "packages", "api-contract", "fixtures", "markdown-blank-lines.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture blankLineFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.MaxRendered != MaxRenderedBlankLines {
		t.Fatalf("fixture cap %d, renderer cap %d", fixture.MaxRendered, MaxRenderedBlankLines)
	}
	for _, item := range fixture.Cases {
		t.Run(item.Name, func(t *testing.T) {
			max := fixture.MaxRendered
			if item.Unlimited {
				max = -1
			}
			want := item.Markdown
			if item.Marked != nil {
				want = *item.Marked
			}
			if got := markBlankLines(item.Markdown, fixture.Marker, max); got != want {
				t.Fatalf("markBlankLines(%q) = %q, want %q", item.Markdown, got, want)
			}
		})
	}
}

func TestPostMarkdownToHTMLRendersBlankLineSpacers(t *testing.T) {
	count := func(html string) int { return strings.Count(html, BlankLineHTML) }
	if got := PostMarkdownToHTML("甲\n\n乙"); count(got) != 0 {
		t.Fatalf("paragraph break rendered spacers: %s", got)
	}
	if got := PostMarkdownToHTML("甲\n\n\n乙"); got != "<p>甲</p>\n"+BlankLineHTML+"\n<p>乙</p>\n" {
		t.Fatalf("one extra blank line = %q", got)
	}
	if got := PostMarkdownToHTML("甲" + strings.Repeat("\n", 9) + "乙"); count(got) != MaxRenderedBlankLines {
		t.Fatalf("capped run rendered %d spacers: %s", count(got), got)
	}
	if got := PostMarkdownToHTML("```\na\n\n\nb\n```"); count(got) != 0 || !strings.Contains(got, "a\n\n\nb") {
		t.Fatalf("code block changed: %s", got)
	}
	if got := PostMarkdownToHTML("GFBLANKLINE\n\n\n乙"); !strings.Contains(got, "<p>GFBLANKLINE</p>") || count(got) != 1 {
		t.Fatalf("marker-like text changed: %s", got)
	}
	// Heading IDs, mentions and spacers coexist on the mention path.
	got := PostMarkdownToHTMLWithMentions("# 标题\n\n\n\n@alice 你好", map[string]uint64{"alice": 7})
	if count(got) != 2 || !strings.Contains(got, `href="/u/7"`) || !strings.Contains(got, `<h1 id="标题">`) || strings.Contains(got, blankLineMarkerPrefix) {
		t.Fatalf("mention render = %s", got)
	}
}

func TestPostMarkdownToHTMLKeepsBlankLineMarkersOutOfMath(t *testing.T) {
	for _, source := range []string{
		"$$x\n\n\ny$$",
		"\\[x\n\n\ny\\]",
		"\\begin{aligned}a\n\n\nb\\end{aligned}",
		"甲\n\n\n$$x\n\n\ny$$\n\n\n乙",
	} {
		for _, rendered := range []string{
			PostMarkdownToHTML(source),
			PostMarkdownToHTMLWithMentions(source, map[string]uint64{"alice": 1}),
		} {
			if strings.Contains(rendered, blankLineMarkerPrefix) {
				t.Fatalf("marker leaked into math for %q: %s", source, rendered)
			}
		}
	}
}
