package wikiservice

import (
	"errors"
	"regexp"
	"strings"
)

// 哨兵错误：控制器层据此映射稳定 messageCode。
var (
	ErrNamespaceNotFound    = errors.New("wiki: namespace not found")
	ErrNamespaceExists      = errors.New("wiki: namespace already exists")
	ErrNamespaceHasPages    = errors.New("wiki: namespace has pages")
	ErrPathInvalid          = errors.New("wiki: path invalid")
	ErrPathExists           = errors.New("wiki: path already exists")
	ErrPageNotFound         = errors.New("wiki: page not found")
	ErrForbidden            = errors.New("wiki: forbidden")
	ErrRevisionNotFound     = errors.New("wiki: revision not found")
	ErrPageHasChildren      = errors.New("wiki: page has children")
	ErrNamespaceNameInvalid = errors.New("wiki: namespace name invalid")
	// ErrConflict 版本 CAS 冲突：编辑基于的版本号已过期，需基于最新版本重编（409）。
	ErrConflict = errors.New("wiki: revision conflict")
	// ErrSensitiveBlocked 内容命中敏感词被拦截（写即发布无审核兜底，直接拒绝）。
	ErrSensitiveBlocked = errors.New("wiki: content sensitive blocked")
	// ErrTitleTooLong 标题超过 512 上限（独立哨兵，避免与 ErrPathInvalid 混淆）。
	ErrTitleTooLong = errors.New("wiki: title too long")
	// ErrContentEmpty 创建/编辑内容为空（契约 content minLength:1，写即发布无审核兜底）。
	ErrContentEmpty = errors.New("wiki: content empty")
	// ErrUserNotFound 贡献者设置引用了不存在的用户（避免幽灵贡献者行）。
	ErrUserNotFound = errors.New("wiki: user not found")
	// ErrBaseRevisionRequired 编辑未携带 baseRevisionNo（契约 required,min=1；
	// 0 = 客户端绕过 CAS 基线校验，静默覆盖他人已发布版本，必须拒绝）。
	ErrBaseRevisionRequired = errors.New("wiki: base revision required")
	// ErrFrontmatterInvalid 内容 frontmatter 非法：必填字段缺失、字段类型错误或
	// yaml 语法错误（issue #258：wiki 以 GitHub 仓库 Markdown 为唯一真源，
	// frontmatter 是页面元数据，渲染/摘要/搜索/TOC 前必须剥离）。
	ErrFrontmatterInvalid = errors.New("wiki: frontmatter invalid")
)

var (
	slugSegmentRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	namespaceRe   = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

const (
	maxNamespaceLen = 64
	maxSlugLen      = 64
	maxPathLen      = 255
)

// ValidateNamespace 校验 namespace 名称（小写 slug；大写输入按小写归一判定）。
func ValidateNamespace(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	return len(name) <= maxNamespaceLen && namespaceRe.MatchString(name)
}

// ValidatePath 校验完整 wiki 路径："namespace/slug[/slug...]"。
// 返回规范化后的 path（已小写）；非法返回 false。
func ValidatePath(path string) (string, bool) {
	path = strings.TrimSpace(path)
	path = strings.ToLower(path)
	if path == "" || len(path) > maxPathLen {
		return "", false
	}
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return "", false // 至少 namespace + 一个 slug 段
	}
	for _, seg := range segments {
		if seg == "" || seg == "." || seg == ".." || len(seg) > maxSlugLen {
			return "", false
		}
		if !slugSegmentRe.MatchString(seg) {
			return "", false
		}
	}
	return path, true
}

// NamespaceOf 返回 path 的 namespace 段。
func NamespaceOf(path string) string {
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return ""
}
