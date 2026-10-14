package stickerservice

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

var (
	enabledListCache = &localcache.Cache[[]StickerItem]{MaxEntries: 1}
	stickerURLCache  = &localcache.Cache[string]{MaxEntries: 1024}
)

const stickerCacheTTL = 5 * time.Minute

func init() {
	sticker.SetOnMutation(InvalidateCache)
}

// InvalidateCache clears the in-memory sticker caches on mutations.
func InvalidateCache() {
	enabledListCache.Clear()
	stickerURLCache.Clear()
}

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

// StickerItem is a shared asset view; private-library results can also contain
// disabled assets so the owner can still remove or reorder their membership.
type StickerItem struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	DisplayName  string `json:"displayName"`
	Pack         string `json:"pack"`
	IsOfficial   bool   `json:"isOfficial"`
	IsEnabled    bool   `json:"isEnabled"`
	ReviewStatus string `json:"reviewStatus"`
}

// EnabledList returns enabled stickers with public access paths, ordered by
// sort_order for stable picker layout.
func EnabledList() ([]StickerItem, error) {
	return enabledListCache.GetOrLoadE("all_enabled", func() ([]StickerItem, error) {
		entities, err := sticker.AllEnabled()
		if err != nil {
			return nil, err
		}
		items := make([]StickerItem, 0, len(entities))
		for _, entity := range entities {
			if entity.FileName == "" {
				continue
			}
			items = append(items, itemFor(entity, ""))
		}
		return items, nil
	}, stickerCacheTTL)
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
	if len(names) == 0 {
		return map[string]string{}, nil
	}
	urls := make(map[string]string, len(names))
	missing := make([]string, 0, len(names))
	for _, name := range names {
		if cachedURL, ok := stickerURLCache.Get(name); ok {
			if cachedURL != "" {
				urls[name] = cachedURL
			}
		} else {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 {
		return urls, nil
	}
	entities, err := sticker.EnabledByNames(missing)
	if err != nil {
		return urls, err
	}
	foundNames := make(map[string]struct{}, len(entities))
	for _, entity := range entities {
		foundNames[entity.Name] = struct{}{}
		if entity.FileName != "" {
			u := ResolveURLFor(entity)
			urls[entity.Name] = u
			stickerURLCache.Set(entity.Name, u, stickerCacheTTL)
		} else {
			stickerURLCache.Set(entity.Name, "", stickerCacheTTL)
		}
	}
	for _, name := range missing {
		if _, ok := foundNames[name]; !ok {
			stickerURLCache.Set(name, "", stickerCacheTTL)
		}
	}
	return urls, nil
}

// itemFor exposes no creator identity or another user's private label/order.
func itemFor(entity sticker.Entity, label string) StickerItem {
	if label == "" {
		label = entity.DisplayName
	}
	if label == "" {
		label = entity.Name
	}
	return StickerItem{ID: entity.Id, Name: entity.Name, URL: ResolveURLFor(entity), DisplayName: label, Pack: entity.Pack, IsOfficial: entity.IsOfficial, IsEnabled: entity.IsEnabled, ReviewStatus: entity.ReviewStatus}
}
