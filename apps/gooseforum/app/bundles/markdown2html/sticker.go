package markdown2html

import (
	"bytes"
	"crypto/rand"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	nethtml "golang.org/x/net/html"
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

// StickerTokenSegment 是一段纯文本：Name 非空表示一个贴纸 token，否则 Text 是原样文字。
type StickerTokenSegment struct {
	Name string
	Text string
}

// StripStickerTokens 与 SplitStickerTokens 服务于已经是纯文本的摘要/预览：
// 没有 Markdown 结构可判定位置，一律按 token 处理；Markdown 源用
// stickerRanges（代码、链接目标里的 token 保持字面量）。

// stripNonProseStickerTokens 从 Markdown 源里删掉“非正文位置”的贴纸 token
// （行内代码/代码块/链接/图片/autolink/原始 HTML/数学公式，以及被反斜杠转义
// 的写法）。这些位置按渲染规则保持字面量，摘要若保留它们，预览端会把 token
// 当图片展开，出现「帖内是原文、列表是图片」的矛盾。判定与
// RenderWithStickerTokens 共用 stickerRanges。
func stripNonProseStickerTokens(markdown string) string {
	if !strings.Contains(markdown, "[:sticker:") {
		return markdown
	}
	prose := make(map[int]bool, 4)
	for _, token := range stickerRanges(markdown) {
		prose[token.start] = true
	}
	matches := stickerTokenRe.FindAllStringIndex(markdown, -1)
	if len(matches) == 0 {
		return markdown
	}
	var builder strings.Builder
	cursor := 0
	for _, match := range matches {
		if prose[match[0]] {
			continue
		}
		start := match[0]
		// 转义的 token 连同前面的反斜杠一起删掉，避免摘要留下孤立的 `\`。
		if start > 0 && markdown[start-1] == '\\' {
			slashes := 0
			for i := start - 1; i >= 0 && markdown[i] == '\\'; i-- {
				slashes++
			}
			if slashes%2 != 0 {
				start--
			}
		}
		if start < cursor {
			continue
		}
		builder.WriteString(markdown[cursor:start])
		// 用空格占位而不是整体删除：行内代码里 `[:sticker:x:]` 被删空后反引号
		// 会退化成字面量 ``，留一个空格能让代码段继续成立。
		builder.WriteByte(' ')
		cursor = match[1]
	}
	if cursor == 0 {
		return markdown
	}
	builder.WriteString(markdown[cursor:])
	return builder.String()
}

// StripStickerTokens removes sticker tokens from plain text.
func StripStickerTokens(text string) string {
	if !strings.Contains(text, "[:sticker:") {
		return text
	}
	return strings.Join(strings.Fields(stickerTokenRe.ReplaceAllString(text, " ")), " ")
}

// SplitStickerTokens splits plain text into literal text and sticker token names,
// so an image-capable surface can escape text and render resolvable tokens.
func SplitStickerTokens(text string) []StickerTokenSegment {
	matches := stickerTokenRe.FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return []StickerTokenSegment{{Text: text}}
	}
	segments := make([]StickerTokenSegment, 0, len(matches)*2+1)
	cursor := 0
	for _, match := range matches {
		if match[0] > cursor {
			segments = append(segments, StickerTokenSegment{Text: text[cursor:match[0]]})
		}
		segments = append(segments, StickerTokenSegment{Name: text[match[2]:match[3]]})
		cursor = match[1]
	}
	if cursor < len(text) {
		segments = append(segments, StickerTokenSegment{Text: text[cursor:]})
	}
	return segments
}

func ExpandStickerTokens(markdown string, resolve func(string) (string, bool)) string {
	return expandStickerTokens(markdown, resolve, func(name, url string) string {
		return "![sticker:" + name + "](" + EscapeMarkdownImageURL(url) + ")"
	})
}

// RenderWithStickerTokens marks only images generated from resolved prose
// tokens. A private per-render alt carries provenance through the normal image
// URL sanitizer; user-authored alt, titles, URLs and raw HTML cannot grant it.
func RenderWithStickerTokens(markdown string, resolve func(string) (string, bool), render func(string) string) string {
	if !strings.Contains(markdown, "[:sticker:") {
		return render(markdown)
	}
	markers := map[string]string{}
	prefix := "GFSTICKER" + rand.Text()
	expanded := expandStickerTokens(markdown, resolve, func(name, url string) string {
		marker := prefix + strconv.Itoa(len(markers))
		markers[marker] = name
		return "![" + marker + "](" + EscapeMarkdownImageURL(url) + ")"
	})
	rendered := render(expanded)
	if len(markers) == 0 {
		return rendered
	}
	root, err := nethtml.Parse(strings.NewReader("<div>" + rendered + "</div>"))
	if err != nil {
		return rendered
	}
	var walk func(*nethtml.Node)
	walk = func(node *nethtml.Node) {
		if node.Type == nethtml.ElementNode && node.Data == "img" {
			if name, ok := markers[getHTMLAttr(node, "alt")]; ok {
				setHTMLAttr(node, "alt", "sticker:"+name)
				setHTMLAttr(node, "data-gf-sticker", name)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	container := findFirstElement(root, "div")
	if container == nil {
		return rendered
	}
	var buf bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err := nethtml.Render(&buf, child); err != nil {
			return rendered
		}
	}
	return buf.String()
}

func expandStickerTokens(markdown string, resolve func(string) (string, bool), imageMarkdown func(string, string) string) string {
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
		rewritten.WriteString(imageMarkdown(token.name, url))
		cursor = token.end
	}
	if cursor == 0 {
		return markdown
	}
	rewritten.WriteString(markdown[cursor:])
	return rewritten.String()
}
