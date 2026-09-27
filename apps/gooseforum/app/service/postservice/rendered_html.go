package postservice

import (
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
)

// RenderPostHTML 渲染帖子正文：先把 [:sticker:name:] 表情包 token 展开为
// 标准图片语法（未知/停用表情保持原样），再把解析到有效用户的 @mention
// 渲染为指向 /u/{userId} 的链接；未知/失效用户名保持普通文本。
func RenderPostHTML(content string) string {
	return renderPostHTML(content, nil)
}

func renderPostHTML(content string, stickerURLs map[string]string) string {
	if names := markdown2html.ExtractStickerNames(content); len(names) > 0 {
		if stickerURLs == nil {
			var err error
			stickerURLs, err = stickerservice.ResolveURLs(names)
			if err != nil {
				slog.Warn("resolve sticker images failed", "error", err)
			}
		}
	}
	return markdown2html.RenderWithStickerTokens(content, func(name string) (string, bool) {
		value, ok := stickerURLs[name]
		return value, ok
	}, func(expanded string) string {
		usernames := markdown2html.ExtractUsernames(expanded)
		if len(usernames) == 0 {
			return markdown2html.PostMarkdownToHTML(expanded)
		}
		targets := users.GetMentionTargetIds(usernames)
		if len(targets) == 0 {
			return markdown2html.PostMarkdownToHTML(expanded)
		}
		return markdown2html.PostMarkdownToHTMLWithMentions(expanded, targets)
	})
}

func EnsureRenderedHTMLBatch(entities []*posts.Entity) {
	names := make([]string, 0)
	seen := make(map[string]struct{})
	for _, entity := range entities {
		if entity == nil {
			continue
		}
		for _, name := range markdown2html.ExtractStickerNames(entity.Content) {
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				names = append(names, name)
			}
		}
	}
	if len(names) == 0 {
		for _, entity := range entities {
			EnsureRenderedHTML(entity)
		}
		return
	}
	urls, err := stickerservice.ResolveURLs(names)
	if err != nil {
		slog.Warn("resolve sticker images failed", "error", err)
	}
	for _, entity := range entities {
		if entity == nil || entity.Id == 0 {
			continue
		}
		if strings.Contains(entity.Content, "[:sticker:") {
			refreshStickerHTMLInPlace(entity, urls)
		} else {
			EnsureRenderedHTML(entity)
		}
	}
}

// refreshStickerHTMLInPlace re-renders a token-bearing post at read time with the
// payload-scoped sticker URL map (nil resolves just for this entity), refreshing the
// in-place copy consumed by payload builders. Mutable sticker definitions are never
// persisted into the HTML cache.
func refreshStickerHTMLInPlace(entity *posts.Entity, stickerURLs map[string]string) {
	entity.RenderedHTML = renderPostHTML(entity.Content, stickerURLs)
	entity.RenderedVersion = markdown2html.GetPostVersion()
}

func EnsureRenderedHTML(entity *posts.Entity) string {
	html, err := ensureRenderedHTML(entity, posts.SaveNoUpdate)
	if err != nil {
		slog.Warn("save rebuilt post html failed", "postId", entity.Id, "error", err)
	}
	return html
}

func ensureRenderedHTML(entity *posts.Entity, save func(*posts.Entity) error) (string, error) {
	if entity == nil || entity.Id == 0 {
		return "", nil
	}
	// Sticker definitions are mutable. Resolve only token-bearing posts at read
	// time, so disable/delete/rename/import never leave stale persisted URLs.
	if strings.Contains(entity.Content, "[:sticker:") {
		// Payload builders also consume the entity in place. Refresh that request's
		// copy without persisting mutable sticker definitions into the HTML cache.
		refreshStickerHTMLInPlace(entity, nil)
		return entity.RenderedHTML, nil
	}
	if entity.RenderedVersion >= markdown2html.GetPostVersion() && entity.RenderedHTML != "" {
		return entity.RenderedHTML, nil
	}

	entity.RenderedHTML = RenderPostHTML(entity.Content)
	entity.RenderedVersion = markdown2html.GetPostVersion()
	if err := save(entity); err != nil {
		return entity.RenderedHTML, err
	}
	return entity.RenderedHTML, nil
}
