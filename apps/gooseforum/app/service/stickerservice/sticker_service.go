package stickerservice

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// stickerNameRe 限制表情包名：Unicode 字母/数字/下划线/连字符，1-64 字符。
// 名字直接出现在 [:sticker:name:] token 与渲染出的图片 alt 中，空白、
// 冒号、方括号、圆括号等字符会破坏 token 或 Markdown 解析，一律禁止。
var stickerNameRe = regexp.MustCompile(`^[\p{L}\p{N}_\-]{1,64}$`)

// ValidateName reports whether name is a sticker-safe identifier.
func ValidateName(name string) bool {
	return stickerNameRe.MatchString(name)
}

// ResolveURLFor returns the public access path for one sticker entity,
// including disabled rows (admin console still shows the image).
func ResolveURLFor(entity sticker.Entity) string {
	if entity.FileName == "" {
		return ""
	}
	return storageservice.PublicAccessPath(entity.FileName)
}

// StemName returns a file name's stem (without directory and extension) for
// sticker name derivation during pack import.
func StemName(fileName string) string {
	base := fileName
	if idx := strings.LastIndexByte(base, '/'); idx >= 0 {
		base = base[idx+1:]
	}
	if idx := strings.LastIndexByte(base, '.'); idx > 0 {
		base = base[:idx]
	}
	return base
}

// StickerItem is the public shape of an enabled sticker (editor picker and
// client-side token replacement map).
type StickerItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// EnabledList returns enabled stickers with public access paths, ordered by
// sort_order for stable picker layout.
func EnabledList() ([]StickerItem, error) {
	entities, err := sticker.AllEnabled()
	if err != nil {
		return nil, err
	}
	items := make([]StickerItem, 0, len(entities))
	for _, entity := range entities {
		if entity.FileName == "" {
			continue
		}
		items = append(items, StickerItem{Name: entity.Name, URL: storageservice.PublicAccessPath(entity.FileName)})
	}
	return items, nil
}

// SanitizeName normalizes raw file-name-derived input into a sticker-safe
// name: token-breaking characters become '-', results are trimmed and
// rune-truncated to MaxNameLen. Empty result means unusable input.
func SanitizeName(raw string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		switch {
		case r == '-' || r == '_':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	name := strings.Trim(b.String(), "-")
	runes := []rune(name)
	if len(runes) > sticker.MaxNameLen {
		runes = runes[:sticker.MaxNameLen]
	}
	name = strings.Trim(string(runes), "-")
	if !ValidateName(name) {
		return ""
	}
	return name
}

func ResolveURLs(names []string) (map[string]string, error) {
	entities, err := sticker.EnabledByNames(names)
	urls := make(map[string]string, len(entities))
	for _, entity := range entities {
		if entity.FileName != "" {
			urls[entity.Name] = ResolveURLFor(entity)
		}
	}
	return urls, err
}
