// Package imagepolicy is the single source of truth for which image types the
// forum accepts for user uploads, and how extension tokens and the configured
// allowlist are canonicalized. The supported set mirrors the decoders blank
// imported by app/http/controllers/api/image_content.go (image/jpeg, image/png,
// image/gif, golang.org/x/image/webp and golang.org/x/image/bmp).
package imagepolicy

import (
	"path"
	"strings"
)

// supportedByExt 规范化扩展名（小写、含点）→ MIME。解码器不支持的扩展名
// （.svg/.html/.js/.xml/.pdf 等）永远不属于该集合，因此既不能保存进
// authorizedExtensions 配置，也无法通过任何上传路径的扩展名校验。
var supportedByExt = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
	".bmp":  "image/bmp",
}

// contentTypeByDecodedFormat 将 image.DecodeConfig 返回的格式名映射为 MIME。
var contentTypeByDecodedFormat = map[string]string{
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"bmp":  "image/bmp",
}

// orderedDefaults 保证 DefaultExtensions 的返回顺序稳定可读。
var orderedDefaults = []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}

// DefaultExtensions 返回内置支持集的副本；上传配置列表为空时它以生效 allowlist 身份出现。
func DefaultExtensions() []string {
	return append([]string(nil), orderedDefaults...)
}

// CanonicalizeToken 把用户输入（配置项或扩展名片段）归一化为小写、带点的形式：
// 容忍无点简写（"png"）、大写变体（".PNG"）与首尾空白。
func CanonicalizeToken(raw string) string {
	token := strings.ToLower(strings.TrimSpace(raw))
	if token != "" && !strings.HasPrefix(token, ".") {
		token = "." + token
	}
	return token
}

// ContentTypeForExt 返回扩展名（大小写/无点宽容）对应的规范化 MIME。
func ContentTypeForExt(ext string) (string, bool) {
	contentType, ok := supportedByExt[CanonicalizeToken(ext)]
	return contentType, ok
}

// ContentTypeForFilename 按文件名末段扩展名返回规范化 MIME，供上传校验与服务端
// 响应头使用：文件类型由存储对象名的扩展名权威决定，不采信客户端声明。
func ContentTypeForFilename(name string) (string, bool) {
	return ContentTypeForExt(path.Ext(name))
}

// DecodedFormatContentType 把 image.DecodeConfig 返回的格式名映射为规范化 MIME，
// 供解码内容校验与期望类型比对。
func DecodedFormatContentType(format string) (string, bool) {
	contentType, ok := contentTypeByDecodedFormat[strings.ToLower(format)]
	return contentType, ok
}

// IsAllowedExt 报告扩展名（大小写/无点宽容）是否属于给定规范化 allowlist。
func IsAllowedExt(ext string, allowed []string) bool {
	canonical := CanonicalizeToken(ext)
	for _, entry := range allowed {
		if CanonicalizeToken(entry) == canonical {
			return true
		}
	}
	return false
}

// CanonicalizeList 拆分配置列表：能映射到受支持图片类型的条目以规范化（小写带点、
// 首个出现顺序、去重）形式返回；集合外条目（危险扩展、双扩展、空串等）原样收集到
// dropped。它是保存端拒绝与读取端过滤共用的判据。
func CanonicalizeList(entries []string) (valid, dropped []string) {
	seen := make(map[string]bool, len(entries))
	for _, raw := range entries {
		canonical := CanonicalizeToken(raw)
		if _, ok := supportedByExt[canonical]; !ok {
			dropped = append(dropped, raw)
			continue
		}
		if seen[canonical] {
			continue
		}
		seen[canonical] = true
		valid = append(valid, canonical)
	}
	return valid, dropped
}

// FilterConfiguredList 是读取路径的归一化：保留能映射到受支持图片类型的原始条目，
// 其余（危险扩展、双扩展、无扩展等）过滤掉并返回。与 CanonicalizeList 的区别是
// 保留条目的原始拼写（历史无点/大写配置在读取回显时不改写，只负责不生效）。
func FilterConfiguredList(entries []string) (kept, dropped []string) {
	for _, raw := range entries {
		if _, ok := supportedByExt[CanonicalizeToken(raw)]; ok {
			kept = append(kept, raw)
			continue
		}
		dropped = append(dropped, raw)
	}
	return kept, dropped
}

// EffectiveAllowedExtensions 返回上传路径实际生效的扩展名列表：配置列表的规范化
// 子集为空时回退到内置全集。危险扩展在任何分支都不会进入结果——它们不属于
// supportedByExt，只会被丢弃。
func EffectiveAllowedExtensions(configured []string) []string {
	valid, _ := CanonicalizeList(configured)
	if len(valid) == 0 {
		return DefaultExtensions()
	}
	return valid
}
