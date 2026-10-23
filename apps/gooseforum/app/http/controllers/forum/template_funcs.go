package forum

import (
	"fmt"
	"html/template"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/gin-gonic/gin"
)

var templateFuncMap = template.FuncMap{
	"SafeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"Nl2br": func(text string) template.HTML {
		escaped := template.HTMLEscapeString(text)
		result := strings.ReplaceAll(escaped, "\n", "<br>")
		return template.HTML(result)
	},
	"json": func(v any) template.JS {
		return template.JS(jsonopt.Encode(v))
	},
	// t localizes a server-rendered string, e.g. {{ t .Lang "search" }}.
	// Extra args are alternating name/value pairs for {name} placeholders.
	"t": i18n.T,
	// serverMessage resolves a backend messageCode into a localized message,
	// mirroring the frontend resolveApiMessage. Usage:
	// {{ serverMessage .Lang .Payload.Props.MessageCode .Payload.Props.Params (t .Lang "common.loadFailed") }}.
	// The last argument is a translated fallback.
	"serverMessage": serverMessage,
	// topicListLabel returns the crawler list link label for a topic. Usage:
	// {{ topicListLabel .Title .Description }}.
	"topicListLabel": topicListLabel,
	// topicListDescription renders the crawler list excerpt; resolvable public
	// stickers become images. Usage: {{ topicListDescription .Description }}.
	"topicListDescription": topicListDescription,
}

// topicListDescription 渲染爬虫列表摘要：小字视图不展开贴纸图片，可解析的公开
// 贴纸显示为 [名字]，未知/停用/个人贴纸省略（#1304）。
func topicListDescription(description string) string {
	return stickerLabelText(description)
}

// stickerLabelText 把小字视图（爬虫列表、置顶条、列表行）里的贴纸 token 换成
// [名字]：可解析的公开贴纸保留名字，其余省略，绝不露出 [:sticker:…] 原文。
func stickerLabelText(description string) string {
	segments := markdown2html.SplitStickerTokens(description)
	urls := publicCrawlerStickerURLs(segments)
	var builder strings.Builder
	for _, segment := range segments {
		if segment.Name == "" {
			builder.WriteString(segment.Text)
			continue
		}
		if urls[segment.Name] != "" {
			builder.WriteString("[" + segment.Name + "]")
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

// publicCrawlerStickerURLs 只解析公开列表 API 同口径的贴纸（官方、启用、已审核）：
// ResolveURLs 还会按 token 解析个人 u_* 贴纸，爬虫可见 HTML 不展开这类名字。
func publicCrawlerStickerURLs(segments []markdown2html.StickerTokenSegment) map[string]string {
	names := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment.Name != "" {
			names = append(names, segment.Name)
		}
	}
	urls := make(map[string]string, len(names))
	if len(names) == 0 {
		return urls
	}
	resolved, err := stickerservice.ResolveURLs(names)
	if err != nil {
		slog.Warn("resolve crawler sticker images failed", "error", err)
	}
	items, err := stickerservice.EnabledList()
	if err != nil {
		slog.Warn("load public sticker list failed", "error", err)
		return urls
	}
	for _, item := range items {
		if url := resolved[item.Name]; url != "" {
			urls[item.Name] = url
		}
	}
	return urls
}

// topicListLabel 返回爬虫话题列表的链接文案：无标题瞬间回退到正文摘要（纯文本，
// 贴纸 token 已去掉），摘要也为空时使用与 Web TopicRow（runtime/topic-description.ts）
// 同族的颜文字，避免 SSR 列表出现空链接文案。
func topicListLabel(title, description string) string {
	if title != "" {
		return title
	}
	if description = strings.TrimSpace(stickerLabelText(description)); description != "" {
		return description
	}
	return "(｀・ω・´)"
}

// serverMessage adapts i18n.ServerMessage for templates: payload fields are
// typed component.MessageCode / component.MessageParams, which Go templates
// pass as interface{}, so the adapter unwraps them before delegating.
func serverMessage(lang string, code any, params any, fallback string) string {
	var codeStr string
	switch v := code.(type) {
	case nil:
	case string:
		codeStr = v
	case component.MessageCode:
		codeStr = string(v)
	default:
		codeStr = fmt.Sprint(v)
	}
	var paramsMap map[string]any
	switch v := params.(type) {
	case nil:
	case map[string]any:
		paramsMap = v
	case component.MessageParams:
		paramsMap = map[string]any(v)
	default:
		paramsMap = nil
	}
	return i18n.ServerMessage(lang, codeStr, paramsMap, fallback)
}

// requestLang resolves and normalizes the request locale. It delegates to
// component.RequestLang so the templates and the activation page share one
// source of truth. The normalized value is also safe as an <html lang> value.
func requestLang(c *gin.Context) string {
	return component.RequestLang(c)
}
