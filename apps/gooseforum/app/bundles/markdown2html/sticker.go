package markdown2html

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// stickerTokenRe 匹配表情包 token [:sticker:name:]：name 为贴纸安全字符集
// （非空白、非冒号、非方括号），长度与 stickers.name 一致（<=64）。
var stickerTokenRe = regexp.MustCompile(`\[:sticker:([^:\[\]\s]{1,64}):\]`)

type stickerRange struct {
	start, end int
	name       string
}

// stickerExclusions 收集表情包 token 不展开的源文本区间：行内代码/代码块/
// 链接/图片/autolink/原始 HTML 的内容范围。goldmark 会在链接语法边界处
// 把 [:sticker: 拆进不同文本节点，因此 token 定位必须走整段源文本正则，
// 代码等排除语义则由 AST 区间承接——两层各取所长。
func stickerExclusions(source []byte) [][2]int {
	doc := GetParser().Parser().Parse(text.NewReader(source))
	ranges := make([][2]int, 0, 8)
	add := func(start, stop int) {
		if stop > start {
			ranges = append(ranges, [2]int{start, stop})
		}
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.CodeSpan, *ast.Link, *ast.Image, *ast.AutoLink:
			// 行内构造：取子文本节点首尾覆盖的跨度（链接文本可能被
			// goldmark 拆成多个 Text 节点，首尾跨度一并覆盖间隙）。
			start, stop := -1, -1
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if tn, ok := child.(*ast.Text); ok {
					if start < 0 {
						start = tn.Segment.Start
					}
					stop = tn.Segment.Stop
				}
			}
			add(start, stop)
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock:
			lines := node.Lines()
			if lines.Len() > 0 {
				add(lines.At(0).Start, lines.At(lines.Len()-1).Stop)
			}
			return ast.WalkSkipChildren, nil
		case *ast.RawHTML:
			if node.Segments.Len() > 0 {
				add(node.Segments.At(0).Start, node.Segments.At(node.Segments.Len()-1).Stop)
			}
		}
		return ast.WalkContinue, nil
	})
	return ranges
}

// inMathSegment 报告 [start,end) 是否与任一 math 段重叠。
func inMathSegment(segments []mathSegment, start, end int) bool {
	for _, segment := range segments {
		if start < segment.end && segment.start < end {
			return true
		}
	}
	return false
}

// ExpandStickerTokens 把源 Markdown 中的 [:sticker:name:] 重写为标准图片
// 语法 ![sticker:name](url)，使其进入既有渲染/净化管线；resolver 返回
// false（未知或停用表情）时 token 原样保留。代码/链接/图片/autolink/
// 原始 HTML/数学公式内的 token 不展开。
func ExpandStickerTokens(markdown string, resolve func(name string) (url string, ok bool)) string {
	if !strings.Contains(markdown, "[:sticker:") {
		return markdown
	}
	source := []byte(markdown)
	tokens := make([]stickerRange, 0, 4)
	for _, match := range stickerTokenRe.FindAllSubmatchIndex(source, -1) {
		tokens = append(tokens, stickerRange{
			start: match[0],
			end:   match[1],
			name:  string(source[match[2]:match[3]]),
		})
	}
	if len(tokens) == 0 {
		return markdown
	}
	sort.Slice(tokens, func(i, j int) bool { return tokens[i].start < tokens[j].start })
	exclusions := stickerExclusions(source)
	mathSegments := extractMathSegments(markdown)
	excluded := func(start, end int) bool {
		for _, r := range exclusions {
			if start < r[1] && r[0] < end {
				return true
			}
		}
		return inMathSegment(mathSegments, start, end)
	}
	var rewritten strings.Builder
	cursor := 0
	for _, token := range tokens {
		if token.start < cursor || excluded(token.start, token.end) {
			continue
		}
		url, ok := resolve(token.name)
		if !ok {
			continue
		}
		rewritten.Write(source[cursor:token.start])
		rewritten.WriteString("![sticker:" + token.name + "](" + url + ")")
		cursor = token.end
	}
	rewritten.Write(source[cursor:])
	return rewritten.String()
}
