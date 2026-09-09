package markdown2html

import (
	"bytes"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	headingid "github.com/jkboxomine/goldmark-headingid"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	nethtml "golang.org/x/net/html"
)

// maxMentionUsernameLen 与 users.username 字段上限一致（varchar(64)）。
// 注册规则为 [a-zA-Z0-9_-]{6,32}，但存量/导入账号最长可达 64。
const maxMentionUsernameLen = 64

// mentionUsernameRe 匹配候选 mention 的起始：@ 前必须是文本开头或非用户名、
// 非 @/URL/转义相关字符（空白/常见标点）。排除字符避免把邮箱（foo@bar.com）、
// URL 路径（x.com/@alice）、数学占位符（@@YOURTJ_MATH_0@@，其 @ 前是 @）
// 与转义（\@）误识别。group1 是前置分隔字符（可为空=文本开头），
// 其后紧跟 @。用户名部分由 ExtractUsernames/render 在源文本上向前扫描
// [a-zA-Z0-9_-] 取得——goldmark 会把 @alice_smith 在下划线处拆成两个
// Text 节点，按节点值正则会截断用户名。
var mentionUsernameRe = regexp.MustCompile("(^|[^\\w@/\\\\\\[\\]<>!.`-])@")

// scanMentionUsername 从 at 之后的源文本位置向前扫描用户名
// [a-zA-Z0-9_-]（跨 goldmark 文本节点边界），上限与 users.username
// 字段一致（64）。扫描结果为空或超长（>64，非合法用户名）时返回空。
func scanMentionUsername(source []byte, at int) string {
	end := at + 1
	for end < len(source) && end-at-1 < maxMentionUsernameLen && isMentionUsernameChar(source[end]) {
		end++
	}
	if end-at-1 == 0 || end-at-1 > maxMentionUsernameLen {
		return ""
	}
	return string(source[at+1 : end])
}

// isMentionUsernameChar 判断字符是否属于用户名字符集 [a-zA-Z0-9_-]。
func isMentionUsernameChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

// ExtractUsernames 从 Markdown 中提取去重后的 @mention 候选用户名（按出现顺序）。
// 基于 Goldmark AST：只识别真实文本节点；inline/fenced code、链接文本与目标、
// autolink（URL/邮箱）、图片 alt、原始 HTML 与数学内容不识别；
// @ 前须为文本开头或空白/常见标点。结果可直接批量解析用户。
func ExtractUsernames(markdown string) []string {
	if markdown == "" || !strings.Contains(markdown, "@") {
		return nil
	}
	protected, _ := protectMathSegments(markdown)
	reader := text.NewReader([]byte(protected))
	doc := GetParser().Parser().Parse(reader)
	source := reader.Source()

	seen := make(map[string]struct{}, 8)
	usernames := make([]string, 0, 8)
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Text:
			if node.IsRaw() {
				return ast.WalkContinue, nil
			}
			segment := node.Segment
			value := string(segment.Value(source))
			for _, match := range mentionUsernameRe.FindAllStringSubmatchIndex(value, -1) {
				// match: [fullStart fullEnd group1Start group1End]；@ 位于 match[3]。
				at := segment.Start + match[3]
				username := scanMentionUsername(source, at)
				if username == "" {
					continue
				}
				if _, ok := seen[username]; ok {
					continue
				}
				seen[username] = struct{}{}
				usernames = append(usernames, username)
			}
			return ast.WalkContinue, nil
		case *ast.CodeSpan, *ast.Link, *ast.AutoLink, *ast.Image,
			*ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return usernames
}

// PostMarkdownToHTMLWithMentions 渲染帖子正文，并把 targets（username → userId）
// 中解析到有效用户的 @mention 包裹为指向 /u/{id} 的普通链接；未解析到用户的
// mention 保持普通文本。渲染走既有 goldmark 管线后仅对 HTML 文本节点后处理，
// 不重解析 Markdown，因此标题锚点、数学保护与转义等既有渲染结果不受影响；
// 包裹的 href 为服务端解析出的数值 userId，XSS 边界保持不变。
func PostMarkdownToHTMLWithMentions(markdown string, targets map[string]uint64) string {
	if len(targets) == 0 || !strings.Contains(markdown, "@") {
		return PostMarkdownToHTML(markdown)
	}
	protected, placeholders := protectMathSegments(markdown)
	var buf bytes.Buffer
	ctx := parser.NewContext(parser.WithIDs(headingid.NewIDs()))
	if err := md.Convert([]byte(protected), &buf, parser.WithContext(ctx)); err != nil {
		slog.Error("转化失败", "err", err)
	}
	// 在恢复数学片段之前包裹 mention：数学内容此时仍是占位符
	// （@@YOURTJ_MATH_0@@），不会命中 mention 正则，避免 $@alice$ 误识别。
	html := wrapMentionLinks(buf.String(), targets)
	html = restoreMathSegments(html, placeholders)
	return normalizePostHTML(html)
}

// wrapMentionLinks 遍历渲染后 HTML，把文本节点中的有效 mention 包裹为
// <a href="/u/{id}">@username</a>。code/pre/既有链接/脚本等元素内的
// 文本不处理（与 AST 提取的排除规则一致）。
func wrapMentionLinks(html string, targets map[string]uint64) string {
	root, err := nethtml.Parse(strings.NewReader("<div>" + html + "</div>"))
	if err != nil {
		return html
	}
	wrapMentionTextNodes(root, targets)

	container := findFirstElement(root, "div")
	if container == nil {
		return html
	}
	var buf bytes.Buffer
	for child := container.FirstChild; child != nil; child = child.NextSibling {
		if err := nethtml.Render(&buf, child); err != nil {
			return html
		}
	}
	return buf.String()
}

// wrapMentionTextNodes 深度优先处理文本节点；进入 code/pre/链接/脚本等
// 元素时整棵跳过，避免把代码或已有链接文本误包裹。
func wrapMentionTextNodes(node *nethtml.Node, targets map[string]uint64) {
	if node.Type == nethtml.ElementNode {
		switch node.Data {
		case "code", "pre", "a", "script", "style", "kbd", "samp", "textarea", "title":
			return
		}
	}
	if node.Type == nethtml.TextNode {
		wrapMentionsInTextNode(node, targets)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		wrapMentionTextNodes(child, targets)
	}
}

// wrapMentionsInTextNode 把单个文本节点中命中 targets 的 mention 拆分为
// 文本+链接+文本节点序列。mention 文本来自已转义 HTML 的原始字节
// （@username 不含 HTML 特殊字符），href 为数值 userId，直接输出安全。
func wrapMentionsInTextNode(node *nethtml.Node, targets map[string]uint64) {
	parent := node.Parent
	if parent == nil {
		return
	}
	text := node.Data
	var splits []struct{ start, end int }
	for _, match := range mentionUsernameRe.FindAllStringSubmatchIndex(text, -1) {
		username := scanMentionUsername([]byte(text), match[3])
		if username == "" {
			continue
		}
		if _, ok := targets[username]; !ok {
			continue
		}
		splits = append(splits, struct{ start, end int }{match[3], match[3] + 1 + len(username)})
	}
	if len(splits) == 0 {
		return
	}

	// 按正序逐个 InsertBefore(node) 插入：每个新节点都落在 node 之前、
	// 已插入节点之后，最终顺序与 splits 一致。
	cursor := 0
	for _, split := range splits {
		if cursor < split.start {
			parent.InsertBefore(
				&nethtml.Node{Type: nethtml.TextNode, Data: text[cursor:split.start]},
				node,
			)
		}
		link := &nethtml.Node{
			Type: nethtml.ElementNode,
			Data: "a",
			Attr: []nethtml.Attribute{{Key: "href", Val: "/u/" + strconv.FormatUint(targets[text[split.start+1:split.end]], 10)}},
		}
		link.AppendChild(&nethtml.Node{Type: nethtml.TextNode, Data: text[split.start:split.end]})
		parent.InsertBefore(link, node)
		cursor = split.end
	}
	if cursor < len(text) {
		parent.InsertBefore(
			&nethtml.Node{Type: nethtml.TextNode, Data: text[cursor:]},
			node,
		)
	}
	parent.RemoveChild(node)
}