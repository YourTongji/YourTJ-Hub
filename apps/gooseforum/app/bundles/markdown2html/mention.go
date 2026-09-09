package markdown2html

import (
	"bytes"
	"log/slog"
	"regexp"
	"sort"
	"strconv"
	"strings"

	headingid "github.com/jkboxomine/goldmark-headingid"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
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
	for end < len(source) && end-at-1 <= maxMentionUsernameLen && isMentionUsernameChar(source[end]) {
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
type mentionRange struct {
	start, end int
	username   string
}

// mentionRanges retains source offsets so escaped text and existing links can
// never turn into mentions merely because HTML rendering removed their syntax.
func mentionRanges(source []byte) []mentionRange {
	doc := GetParser().Parser().Parse(text.NewReader(source))
	var ranges []mentionRange
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
				at := segment.Start + match[3]
				slashes := 0
				for i := at - 1; i >= 0 && source[i] == '\\'; i-- {
					slashes++
				}
				if slashes%2 != 0 {
					continue
				}
				username := scanMentionUsername(source, at)
				if username != "" {
					ranges = append(ranges, mentionRange{at, at + 1 + len(username), username})
				}
			}
		case *ast.CodeSpan, *ast.Link, *ast.AutoLink, *ast.Image, *ast.CodeBlock, *ast.FencedCodeBlock, *ast.HTMLBlock, *ast.RawHTML:
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].start < ranges[j].start })
	return ranges
}

func ExtractUsernames(markdown string) []string {
	if !strings.Contains(markdown, "@") {
		return nil
	}
	protected, _ := protectMathSegments(markdown)
	seen := map[string]bool{}
	var names []string
	for _, mention := range mentionRanges([]byte(protected)) {
		if !seen[mention.username] {
			seen[mention.username] = true
			names = append(names, mention.username)
		}
	}
	return names
}

// PostMarkdownToHTMLWithMentions links only source tokens accepted by the same
// parser used for notification recipients. Ordinary Markdown links preserve the
// existing heading, sanitization and math pipeline; numeric targets are resolved
// by the server and underscore labels are escaped to remain literal usernames.
func PostMarkdownToHTMLWithMentions(markdown string, targets map[string]uint64) string {
	if len(targets) == 0 || !strings.Contains(markdown, "@") {
		return PostMarkdownToHTML(markdown)
	}
	protected, placeholders := protectMathSegments(markdown)
	var rewritten strings.Builder
	cursor := 0
	for _, mention := range mentionRanges([]byte(protected)) {
		id := targets[mention.username]
		if id == 0 || mention.start < cursor {
			continue
		}
		rewritten.WriteString(protected[cursor:mention.start])
		rewritten.WriteString("[@" + strings.ReplaceAll(mention.username, "_", "\\_") + "](/u/" + strconv.FormatUint(id, 10) + ")")
		cursor = mention.end
	}
	rewritten.WriteString(protected[cursor:])

	// Preserve heading IDs from the original source: the headingid extension
	// includes Markdown link destinations when deriving IDs from rewritten text.
	var headingIDs []any
	originalContext := parser.NewContext(parser.WithIDs(headingid.NewIDs()))
	original := md.Parser().Parse(text.NewReader([]byte(protected)), parser.WithContext(originalContext))
	_ = ast.Walk(original, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*ast.Heading); ok && entering {
			id, _ := n.AttributeString("id")
			headingIDs = append(headingIDs, id)
		}
		return ast.WalkContinue, nil
	})
	source := []byte(rewritten.String())
	ctx := parser.NewContext(parser.WithIDs(headingid.NewIDs()))
	doc := md.Parser().Parse(text.NewReader(source), parser.WithContext(ctx))
	headingIndex := 0
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if _, ok := n.(*ast.Heading); ok && entering {
			if headingIndex < len(headingIDs) && headingIDs[headingIndex] != nil {
				n.SetAttributeString("id", headingIDs[headingIndex])
			}
			headingIndex++
		}
		return ast.WalkContinue, nil
	})
	var buf bytes.Buffer
	if err := md.Renderer().Render(&buf, source, doc); err != nil {
		slog.Error("render mention markdown failed", "err", err)
	}
	return normalizePostHTML(restoreMathSegments(buf.String(), placeholders))
}
