package postservice

import (
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
)

// RenderPostHTML 渲染帖子正文：先把 [:sticker:name:] 表情包 token 展开为
// 标准图片语法（未知/停用表情保持原样），再把解析到有效用户的 @mention
// 渲染为指向 /u/{userId} 的链接；未知/失效用户名保持普通文本。
func RenderPostHTML(content string) string {
	content = markdown2html.ExpandStickerTokens(content, stickerservice.ResolveURL)
	usernames := markdown2html.ExtractUsernames(content)
	if len(usernames) == 0 {
		return markdown2html.PostMarkdownToHTML(content)
	}
	targets := users.GetMentionTargetIds(usernames)
	if len(targets) == 0 {
		return markdown2html.PostMarkdownToHTML(content)
	}
	return markdown2html.PostMarkdownToHTMLWithMentions(content, targets)
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
