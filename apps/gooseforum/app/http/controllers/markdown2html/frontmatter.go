package markdown2html

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// frontmatter 字段长度上限（与格式规范 docs/product/wiki-markdown-format.md 一致）。
const (
	FrontmatterTitleMaxLen       = 512
	FrontmatterDescriptionMaxLen = 255
)

// Frontmatter 是 wiki 页面 YAML frontmatter 的类型化元数据。
// 论坛写路径只做解析/校验并剥离；字段消费（order 侧栏排序、description 摘要、
// tags 搜索分类、draft 发布判定）由同步引擎等调用方按需使用。
type Frontmatter struct {
	Title       string   `yaml:"title"`
	Order       *int     `yaml:"order"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Draft       bool     `yaml:"draft"`
}

// SplitFrontmatter 剥离 wiki Markdown 开头的 YAML frontmatter 块。
// 返回 (meta, body)：无 frontmatter（首行不是 ---，或没有闭合的 --- 行）时
// meta 为 nil、body 为原内容。frontmatter 存在但 YAML 无法解析为 map 时采用
// 宽松 fallback：块仍被剥离、meta 为 nil（严格校验见 ParseFrontmatter）。
func SplitFrontmatter(content string) (map[string]any, string) {
	raw, body, ok := splitFrontmatterBlock(content)
	if !ok {
		return nil, content
	}
	meta := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, body
	}
	return meta, body
}

// ParseFrontmatter 严格解析 wiki Markdown 的 frontmatter：返回类型化元数据与
// 剥离后的正文。yaml 语法错误、非 map 根、字段类型不符、必填 title 缺失/为空/
// 超长、description 超长均返回错误。无 frontmatter 时返回零值 Frontmatter 与
// 原内容（不校验）。
func ParseFrontmatter(content string) (Frontmatter, string, error) {
	raw, body, ok := splitFrontmatterBlock(content)
	if !ok {
		return Frontmatter{}, content, nil
	}
	var meta map[string]any
	if err := yaml.Unmarshal([]byte(raw), &meta); err != nil {
		return Frontmatter{}, "", fmt.Errorf("frontmatter: %w", err)
	}
	fm, err := validateFrontmatter(meta)
	if err != nil {
		return Frontmatter{}, "", err
	}
	return fm, body, nil
}

// splitFrontmatterBlock 返回 frontmatter 的原始 YAML 与剥离后的正文。
// 仅当内容首行是 --- 且存在闭合的 --- 行时 ok=true。
func splitFrontmatterBlock(content string) (raw, body string, ok bool) {
	if content == "" || !strings.HasPrefix(content, "---") {
		return "", "", false
	}
	lines := strings.Split(content, "\n")
	if !isFrontmatterDelimiter(lines[0]) {
		return "", "", false
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if isFrontmatterDelimiter(lines[i]) {
			end = i
			break
		}
	}
	if end < 0 {
		return "", "", false
	}
	return strings.Join(lines[1:end], "\n"), strings.Join(lines[end+1:], "\n"), true
}

func isFrontmatterDelimiter(line string) bool {
	return strings.TrimRight(line, " \t\r") == "---"
}

func validateFrontmatter(meta map[string]any) (Frontmatter, error) {
	var fm Frontmatter

	if v, exists := meta["title"]; exists {
		s, ok := v.(string)
		if !ok {
			return fm, fmt.Errorf("frontmatter: title must be a string, got %T", v)
		}
		if strings.TrimSpace(s) == "" {
			return fm, fmt.Errorf("frontmatter: title must not be empty")
		}
		if utf8.RuneCountInString(s) > FrontmatterTitleMaxLen {
			return fm, fmt.Errorf("frontmatter: title longer than %d chars", FrontmatterTitleMaxLen)
		}
		fm.Title = s
	} else {
		return fm, fmt.Errorf("frontmatter: required field title missing")
	}

	if v, exists := meta["order"]; exists {
		n, ok := v.(int)
		if !ok {
			return fm, fmt.Errorf("frontmatter: order must be an integer, got %T", v)
		}
		fm.Order = &n
	}

	if v, exists := meta["description"]; exists {
		s, ok := v.(string)
		if !ok {
			return fm, fmt.Errorf("frontmatter: description must be a string, got %T", v)
		}
		if utf8.RuneCountInString(s) > FrontmatterDescriptionMaxLen {
			return fm, fmt.Errorf("frontmatter: description longer than %d chars", FrontmatterDescriptionMaxLen)
		}
		fm.Description = s
	}

	if v, exists := meta["tags"]; exists {
		list, ok := v.([]any)
		if !ok {
			return fm, fmt.Errorf("frontmatter: tags must be a list of strings, got %T", v)
		}
		tags := make([]string, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return fm, fmt.Errorf("frontmatter: tags must be a list of strings, got item %T", item)
			}
			tags = append(tags, s)
		}
		fm.Tags = tags
	}

	if v, exists := meta["draft"]; exists {
		b, ok := v.(bool)
		if !ok {
			return fm, fmt.Errorf("frontmatter: draft must be a boolean, got %T", v)
		}
		fm.Draft = b
	}

	return fm, nil
}
