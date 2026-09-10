package markdown2html

import (
	"strings"
	"testing"

	nethtml "golang.org/x/net/html"
)

func TestMathProtectionKeepsTypographerMathLiteral(t *testing.T) {
	html := MarkdownToHTML("Inline $f'(x)$ and block $$a--b$$")

	for _, want := range []string{"$f&#39;(x)$", "$$a--b$$"} {
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

func TestMathProtectionEscapesRawHTMLInsideMath(t *testing.T) {
	cases := []string{
		`$<script>alert(1)</script>$`,
		`$$<img src=x onerror=alert(1)>$$`,
		"`$<img src=x onerror=alert(1)>$`",
		`$<b onclick="x()">bold</b>$`,
		`$<svg onload="alert(1)"></svg>$`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			assertNoHTMLInjection(t, MarkdownToHTML(in))
		})
	}
}

func TestPostMarkdownToHTMLEscapesRawHTMLInsideMath(t *testing.T) {
	got := PostMarkdownToHTML(`$<img src=x onerror=alert(1)>$`)
	assertNoHTMLInjection(t, got)
}

func assertNoHTMLInjection(t *testing.T, got string) {
	t.Helper()
	root, err := nethtml.Parse(strings.NewReader(got))
	if err != nil {
		t.Fatalf("parse output %q: %v", got, err)
	}
	forbidden := map[string]bool{"script": true, "img": true, "svg": true, "iframe": true, "style": true, "link": true, "meta": true, "b": true}
	var walk func(*nethtml.Node)
	walk = func(n *nethtml.Node) {
		if n.Type == nethtml.ElementNode {
			if forbidden[n.Data] {
				t.Errorf("forbidden element <%s> in output: %s", n.Data, got)
			}
			for _, attr := range n.Attr {
				key := strings.ToLower(attr.Key)
				if strings.HasPrefix(key, "on") || strings.HasPrefix(strings.ToLower(attr.Val), "javascript:") {
					t.Errorf("dangerous attribute %q in output: %s", attr.Key, got)
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
}

func TestMathProtectionSupportsLatexDelimiters(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: `\(x+y\)`, want: `\(x+y\)`},
		{in: `\[x+y\]`, want: `\[x+y\]`},
		{in: `\begin{equation}E=mc^2\end{equation}`, want: `\begin{equation}E=mc^2\end{equation}`},
		{in: `\begin{align}a b\end{align}`, want: `\begin{align}a b\end{align}`},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := MarkdownToHTML(tc.in)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("rendered HTML missing %q: %s", tc.want, got)
			}
			if strings.Contains(got, "@@YOURTJ_MATH_") {
				t.Fatalf("math placeholder leaked into rendered HTML: %s", got)
			}
		})
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

func TestMathProtectionEscapesAttributeBreakoutInsideLinkText(t *testing.T) {
	got := MarkdownToHTML(`[$x" onmouseover="alert(1)$](https://example.com)`)
	assertNoHTMLInjection(t, got)
	if !strings.Contains(got, `<a href="https://example.com">`) {
		t.Fatalf("link itself was not preserved: %s", got)
	}
}
