package markdown2html

import (
	"strings"
	"testing"
)

func TestMathProtectionKeepsTypographerMathLiteral(t *testing.T) {
	html := MarkdownToHTML("Inline $f'(x)$ and block $$a--b$$")

	for _, want := range []string{"$f'(x)$", "$$a--b$$"} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered HTML missing %q: %s", want, html)
		}
	}
	for _, unwanted := range []string{"&rsquo;", "&ndash;", "&mdash;"} {
		if strings.Contains(html, unwanted) {
			t.Fatalf("math content was changed by typographer (%q): %s", unwanted, html)
		}
	}
}

func TestMathProtectionKeepsInlineMathFromEmphasis(t *testing.T) {
	html := MarkdownToHTML("$a*b*c$")

	if !strings.Contains(html, "$a*b*c$") {
		t.Fatalf("rendered HTML missing inline math: %s", html)
	}
	if strings.Contains(html, "<em>") {
		t.Fatalf("inline math was split by emphasis: %s", html)
	}
}

func TestMathProtectionKeepsBlockMathWithBlankLines(t *testing.T) {
	html := MarkdownToHTML("$$\n\nE = mc^2\n\n$$")

	if !strings.Contains(html, "<p>$$") || !strings.Contains(html, "$$</p>") {
		t.Fatalf("block math was split into separate paragraphs: %s", html)
	}
	if strings.Contains(html, "<p>$$</p>") {
		t.Fatalf("block math delimiters became empty paragraphs: %s", html)
	}
}

func TestMathProtectionRestoresMathInsideInlineCode(t *testing.T) {
	html := MarkdownToHTML("`$x$`")

	if !strings.Contains(html, "<code>$x$</code>") {
		t.Fatalf("inline code content was not restored: %s", html)
	}
	if strings.Contains(html, "@@YOURTJ_MATH_") {
		t.Fatalf("math placeholder leaked into rendered HTML: %s", html)
	}
}
