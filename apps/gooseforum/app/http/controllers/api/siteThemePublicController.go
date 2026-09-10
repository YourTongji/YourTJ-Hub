package api

import (
	"encoding/json"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/themeservice"
)

// PublicSiteThemeTokensItem 单个模式的主题 token 集（mode: "light"|"dark"）。
type PublicSiteThemeTokensItem struct {
	Mode   string            `json:"mode"`
	Tokens map[string]string `json:"tokens"`
}

// PublicSiteThemeTokensResult GET /api/site-theme/tokens 响应 data
// （mobile Route A 主题同步：公开下发发布态主题 token）。
type PublicSiteThemeTokensResult struct {
	Enabled     bool                        `json:"enabled"`
	Version     int                         `json:"version"`
	PublishedAt *string                     `json:"publishedAt"`
	Themes      []PublicSiteThemeTokensItem `json:"themes"`
}

// GetPublicSiteThemeTokens 返回已发布站点主题的 token（公开只读，无鉴权）。
// 数据源与 /site-theme.css（themeservice.LoadConfig → page_config SiteTheme）
// 一致：仅发布态生效（Prepublish 不对外）；未启用/未发布时返回
// {enabled:false,version:0,publishedAt:null,themes:[]}，客户端回退内置主题。
// tokens 键为设计 token 名（如 color-primary），值已按发布归一化。
func GetPublicSiteThemeTokens() component.Response {
	config := themeservice.LoadConfig()
	if !config.Enabled {
		return component.SuccessResponse(PublicSiteThemeTokensResult{
			Enabled: false,
			Version: 0,
			Themes:  []PublicSiteThemeTokensItem{},
		})
	}

	result := PublicSiteThemeTokensResult{
		Enabled: true,
		Version: config.Version,
		Themes:  make([]PublicSiteThemeTokensItem, 0, len(config.Themes)),
	}
	if config.PublishedAt != "" {
		publishedAt := config.PublishedAt
		result.PublishedAt = &publishedAt
	}
	for _, theme := range config.Themes {
		// 结构体 JSON tag 即 token 键（kebab-case）；经 struct 序列化避免
		// 手维护键表漂移。归一化保证合法 token 值原样保留。
		tokens := map[string]string{}
		raw, err := json.Marshal(theme.Tokens)
		if err == nil {
			_ = json.Unmarshal(raw, &tokens)
		}
		result.Themes = append(result.Themes, PublicSiteThemeTokensItem{
			Mode:   theme.ColorScheme,
			Tokens: tokens,
		})
	}
	return component.SuccessResponse(result)
}
