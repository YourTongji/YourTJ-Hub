package markdown2html

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var stickerTokenRe = regexp.MustCompile(`\[:sticker:([^:\[\]\s]{1,64}):\]`)

type stickerRange struct {
	start, end int
	name       string
}

// Accept token starts only from real prose nodes. Link destinations, reference
// definitions, raw HTML and code have no eligible Text node; unlike exclusion
// ranges derived from link children this also protects the entire destination.
func stickerRanges(markdown string) []stickerRange {
	if !strings.Contains(markdown, "[:sticker:") {
		return nil
	}
	source := []byte(markdown)
	doc := GetParser().Parser().Parse(text.NewReader(source))
	var prose [][2]int
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Text:
			if !node.IsRaw() {
				prose = append(prose, [2]int{node.Segment.Start, node.Segment.Stop})
			}
		case *ast.CodeSpan, *ast.Link, *ast.Image, *ast.AutoLink, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	math := extractMathSegments(markdown)
	var tokens []stickerRange
	segment := 0
	for _, m := range stickerTokenRe.FindAllSubmatchIndex(source, -1) {
		for segment < len(prose) && prose[segment][1] <= m[0] {
			segment++
		}
		if segment == len(prose) || m[0] < prose[segment][0] {
			continue
		}
		slashes := 0
		for i := m[0] - 1; i >= 0 && source[i] == '\\'; i-- {
			slashes++
		}
		if slashes%2 != 0 {
			continue
		}
		excluded := false
		for _, r := range math {
			if m[0] < r.end && r.start < m[1] {
				excluded = true
				break
			}
		}
		if !excluded {
			tokens = append(tokens, stickerRange{m[0], m[1], string(source[m[2]:m[3]])})
		}
	}
	return tokens
}

// ExtractStickerNames returns distinct prose tokens for one batched lookup.
func ExtractStickerNames(markdown string) []string {
	var names []string
	seen := map[string]bool{}
	for _, token := range stickerRanges(markdown) {
		if !seen[token.name] {
			seen[token.name] = true
			names = append(names, token.name)
		}
	}
	return names
}

// EscapeMarkdownImageURL preserves URL bytes that could terminate a Markdown
// destination. Percent escapes survive the existing sanitizer and image viewer.
func EscapeMarkdownImageURL(url string) string {
	return strings.NewReplacer("(", "%28", ")", "%29", "<", "%3C", ">", "%3E", " ", "%20", "\\", "%5C", "\n", "%0A", "\r", "%0D", "\t", "%09").Replace(url)
}

func ExpandStickerTokens(markdown string, resolve func(string) (string, bool)) string {
	var rewritten strings.Builder
	cursor := 0
	resolved := map[string]string{}
	for _, token := range stickerRanges(markdown) {
		url, cached := resolved[token.name]
		if !cached {
			value, ok := resolve(token.name)
			if ok {
				url = value
			}
			resolved[token.name] = url
		}
		if url == "" {
			continue
		}
		rewritten.WriteString(markdown[cursor:token.start])
		rewritten.WriteString("![sticker:" + token.name + "](" + EscapeMarkdownImageURL(url) + ")")
		cursor = token.end
	}
	if cursor == 0 {
		return markdown
	}
	rewritten.WriteString(markdown[cursor:])
	return rewritten.String()
}
