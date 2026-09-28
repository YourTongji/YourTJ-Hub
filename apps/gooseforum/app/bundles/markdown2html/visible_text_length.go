package markdown2html

import (
	"strings"
	"unicode"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// VisibleTextLength 统计 Markdown 渲染后实际可见文字的 Unicode 码点数，
// 是正文/回复长度校验的唯一口径（issue #890）：`**a**`、`[ab](url)` 这类
// 源文本很长但可见文字很少的内容不能绕过下限。
//
// 计入（与渲染结果一致）：
//   - 普通文字（反斜杠转义与 HTML 实体按渲染结果解码）；
//   - 行内代码与代码块中的文字；
//   - 链接的可点击标签文字；
//   - 软换行与硬换行各计 1，正文内部空白照常计入。
//
// 不计入：
//   - Markdown 标记（粗体/斜体/标题/引用/列表符号等）；
//   - 链接：`[标签](url)` 只计标签，目标 URL 与裸链接/自动链接（`https://…`、
//     `<https://…>`、邮箱）都不计——链接目标不构成正文，纯链接内容与纯图片
//     一样达不到下限；
//   - 图片语法：`![alt](url)` 的 alt 与 URL 都不计（渲染结果是图片）；
//   - 贴纸 token `[:sticker:name:]`：渲染期展开为图片，统一不计；未解析的
//     token 同样不计，避免用不存在的贴纸名伪造长度；
//   - 原始 HTML：渲染器未开启 Unsafe，raw HTML 按 CommonMark 省略，
//     `<div>hi</div>` 渲染后没有可见文字；`<b>hi</b>` 的内部文字仍计入；
//   - Unicode 格式字符（Cf，如零宽空格、零宽连接符、BOM、词连接符）：
//     渲染后不可见，不计入，避免用不可见字符伪造长度。
//
// 首尾空白不计。与渲染共用同一 goldmark 配置（GetParser），因此
// 代码块/链接/贴纸的边界判定与 RenderWithStickerTokens 保持一致。
//
// 已知差距（有意维持，非缺陷）：非 Cf 的空白渲染字符（U+2800 盲文空白、
// U+3164 Hangul Filler）与组合附加符号仍各计 1 个码点——本仓库的统一口径是
// 「Unicode 码点」而非字素簇，收紧到字素/空白字形需要单独的产品规则。
func VisibleTextLength(markdown string) int {
	source := []byte(stripStickerTokens(markdown))
	doc := GetParser().Parser().Parse(text.NewReader(source))
	var builder strings.Builder
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *ast.Text:
			value := node.Segment.Value(source)
			if !node.IsRaw() {
				value = util.UnescapePunctuations(value)
				value = util.ResolveNumericReferences(value)
				value = util.ResolveEntityNames(value)
			}
			builder.Write(value)
			if node.SoftLineBreak() || node.HardLineBreak() {
				builder.WriteByte('\n')
			}
		case *ast.String:
			// Typographer 等扩展把替换结果存为 HTML 实体（如 `&hellip;`），
			// 渲染后浏览器解码为单个字符，这里同样按实体解码后计数。
			value := util.ResolveNumericReferences(node.Value)
			builder.Write(util.ResolveEntityNames(value))
		case *ast.CodeBlock:
			builder.Write(node.Lines().Value(source))
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			builder.Write(node.Lines().Value(source))
			return ast.WalkSkipChildren, nil
		case *ast.Image, *ast.AutoLink, *ast.RawHTML, *ast.HTMLBlock:
			// 图片 alt/目标、自动链接 URL、HTML 标记都不是可见正文。
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})
	return countVisibleRunes(strings.TrimSpace(builder.String()))
}

// countVisibleRunes 统计可见码点数：Unicode 格式字符（Cf：零宽空格/连接符、
// BOM、词连接符等）渲染后不可见，不计入，避免用不可见字符伪造长度。
// 组合附加符号等仍按码点各计 1（与仓库既有「按 Unicode 码点计数」口径一致）。
func countVisibleRunes(text string) int {
	count := 0
	for _, r := range text {
		if unicode.Is(unicode.Cf, r) {
			continue
		}
		count++
	}
	return count
}

// stripStickerTokens 移除渲染期会展开为图片的贴纸 token，使其不参与计数。
// 位置判定与 RenderWithStickerTokens 共用 stickerRanges：代码、链接目标等
// 位置里的 token 渲染时保持字面量，这里也保持原样、照常计入。
func stripStickerTokens(markdown string) string {
	ranges := stickerRanges(markdown)
	if len(ranges) == 0 {
		return markdown
	}
	var builder strings.Builder
	cursor := 0
	for _, token := range ranges {
		builder.WriteString(markdown[cursor:token.start])
		cursor = token.end
	}
	builder.WriteString(markdown[cursor:])
	return builder.String()
}
