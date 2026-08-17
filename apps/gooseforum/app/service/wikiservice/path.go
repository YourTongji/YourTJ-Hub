package wikiservice

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// 命名空间/路径段约束（GitHub 唯一真实源：顶层目录名即命名空间，须与文件系统
// 目录名兼容，支持中文等 Unicode 字符；不再做小写归一、不再限制字母数字连字符）。
//
// 约束规则（与文件系统目录名兼容，D2 决策）：
//   - TrimSpace 后按字符（rune）计数，长度 1..64（命名空间）/ 1..64（路径段）；
//   - 拒绝 "." / ".." 与以 "." 开头（隐藏目录/文件）；
//   - 拒绝中间或首尾空白（含全角空格）与全部控制字符；
//   - 拒绝文件系统保留字符 / \ : * ? " < > |；
//   - 不做 ToLower 归一（中文无大小写；保留原始大小写，URL 按编码处理）。
const (
	maxNamespaceLen = 64
	maxSegmentLen   = 64
	maxPathLen      = 255
)

// reservedPathChars 文件系统保留字符（Windows 保留，跨平台目录名安全）。
// 额外拒绝 % 与 #（review M2）：Markdown URL 语法无法可靠表示——`%` 是
// 转义前缀（`100%done.md` 无法被 url.Parse 解析），`#` 开启 fragment
// （`a#b.md` 被截断为 `a`）；GitHub 外链拼接同样受影响。
const reservedPathChars = `/ \ : * ? " < > | % #`

// validSegment 校验单个路径段（命名空间或子路径段）是否合法。
func validSegment(seg string) bool {
	if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, ".") {
		return false
	}
	if utf8.RuneCountInString(seg) > maxSegmentLen {
		return false
	}
	for _, r := range seg {
		switch {
		case unicode.IsControl(r) || unicode.IsSpace(r):
			return false
		case strings.ContainsRune(reservedPathChars, r):
			return false
		}
	}
	return true
}

// ValidateNamespace 校验 namespace 名称（GitHub 仓库顶层目录名）。
// 只做 TrimSpace 与约束校验，不做小写归一（支持中文等 Unicode）。
func ValidateNamespace(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxNamespaceLen {
		return false
	}
	return validSegment(name)
}

// ValidatePath 校验完整 wiki 路径："namespace/path[/path...]"（首段即仓库顶层目录名）。
// 返回规范化后的 path（仅 TrimSpace，保留大小写与 Unicode）；非法返回 false。
// 总长按码点（rune）计数 ≤255：与 DB varchar(255) 的字符语义及前端
// 码点计数对齐（此前按字节 len() 计数，100 个中文字符 300 字节会被误拒）。
func ValidatePath(path string) (string, bool) {
	norm, err := ValidatePathError(path)
	return norm, err == nil
}

// ValidatePathError 与 ValidatePath 同规则，但返回具体拒绝原因
// （同步器等需要向运维暴露精确路径与原因的场景使用）。
func ValidatePathError(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if utf8.RuneCountInString(path) > maxPathLen {
		return "", fmt.Errorf("path longer than %d characters", maxPathLen)
	}
	segments := strings.Split(path, "/")
	if len(segments) < 2 {
		return "", fmt.Errorf("path must be at least \"namespace/path\", got %q", path)
	}
	for _, seg := range segments {
		if !validSegment(seg) {
			return "", fmt.Errorf("invalid segment %q (must be 1..%d characters, no \".\"/\"..\"/leading dot, no whitespace/control characters, no reserved chars %s)", seg, maxSegmentLen, reservedPathChars)
		}
	}
	return path, nil
}

// NamespaceOf 返回 path 的 namespace 段。
func NamespaceOf(path string) string {
	if idx := strings.IndexByte(path, '/'); idx > 0 {
		return path[:idx]
	}
	return ""
}
