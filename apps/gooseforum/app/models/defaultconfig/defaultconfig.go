package defaultconfig

import (
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

//go:embed pageconfig/*.json
var defaultConfigFS embed.FS

type pageConfigDefaults struct {
	Announcement pageConfig.AnnouncementConfig
	Email        pageConfig.MailSettingsConfig
	FriendLinks  []pageConfig.FriendLinksGroup
	Posting      pageConfig.PostingContent
	Security     pageConfig.SecurityAndRegistration
	Site         pageConfig.SiteSettingsConfig
	SiteTheme    pageConfig.SiteThemeConfig
	Sponsors     pageConfig.SponsorsConfig
	Storage      pageConfig.StorageSettings
	Terms        pageConfig.TermsOfServiceConfig
	Privacy      pageConfig.PrivacyPolicyConfig
	RateLimit    pageConfig.RateLimitConfig
	MCP          pageConfig.MCPSettingsConfig
	AiSummary    pageConfig.AiSummaryConfig
}

var loadPageConfigDefaults = sync.OnceValues(func() (pageConfigDefaults, error) {
	var defaults pageConfigDefaults
	if err := loadJSON("announcement.json", &defaults.Announcement); err != nil {
		return defaults, err
	}
	if err := loadJSON("email.json", &defaults.Email); err != nil {
		return defaults, err
	}
	if err := loadJSON("friend_links.json", &defaults.FriendLinks); err != nil {
		return defaults, err
	}
	if err := loadJSON("posting.json", &defaults.Posting); err != nil {
		return defaults, err
	}
	if err := loadJSON("security.json", &defaults.Security); err != nil {
		return defaults, err
	}
	if err := loadJSON("site.json", &defaults.Site); err != nil {
		return defaults, err
	}
	if err := loadJSON("site_theme.json", &defaults.SiteTheme); err != nil {
		return defaults, err
	}
	if err := loadJSON("sponsors.json", &defaults.Sponsors); err != nil {
		return defaults, err
	}
	if err := loadJSON("terms.json", &defaults.Terms); err != nil {
		return defaults, err
	}
	if err := loadJSON("privacy.json", &defaults.Privacy); err != nil {
		return defaults, err
	}
	if err := loadJSON("storage.json", &defaults.Storage); err != nil {
		return defaults, err
	}
	if err := loadJSON("ratelimit.json", &defaults.RateLimit); err != nil {
		return defaults, err
	}
	if err := loadJSON("mcp.json", &defaults.MCP); err != nil {
		return defaults, err
	}
	if err := loadJSON("ai_summary.json", &defaults.AiSummary); err != nil {
		return defaults, err
	}
	return defaults, nil
})

func mustPageConfigDefaults() pageConfigDefaults {
	defaults, err := loadPageConfigDefaults()
	if err != nil {
		panic(err)
	}
	return defaults
}

func loadJSON(name string, out any) error {
	data, err := defaultConfigFS.ReadFile("pageconfig/" + name)
	if err != nil {
		return fmt.Errorf("read default page config %s: %w", name, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode default page config %s: %w", name, err)
	}
	return nil
}

func GetDefaultAnnouncementConfig() pageConfig.AnnouncementConfig {
	return mustPageConfigDefaults().Announcement
}

func GetDefaultEmailSettingsConfig() pageConfig.MailSettingsConfig {
	return mustPageConfigDefaults().Email
}

func GetDefaultFriendLinksConfig() []pageConfig.FriendLinksGroup {
	return cloneFriendLinks(mustPageConfigDefaults().FriendLinks)
}

func GetDefaultPostingSettingsConfig() pageConfig.PostingContent {
	config := mustPageConfigDefaults().Posting
	config.UploadControl.AuthorizedExtensions = append([]string(nil), config.UploadControl.AuthorizedExtensions...)
	return config
}

func GetDefaultHttpNotifyConfig() pageConfig.HttpNotifyConfig {
	return pageConfig.HttpNotifyConfig{Endpoints: []pageConfig.HttpNotifyEndpoint{}}
}

func GetDefaultSecuritySettingsConfig() pageConfig.SecurityAndRegistration {
	config := mustPageConfigDefaults().Security
	config.AllowedDomains = append([]string(nil), config.AllowedDomains...)
	config.ReservedUsernames = append([]string(nil), config.ReservedUsernames...)
	config.BannedUsernames = append([]string(nil), config.BannedUsernames...)
	config.SensitiveWords = append([]string(nil), config.SensitiveWords...)
	return config
}

func GetDefaultStorageSettingsConfig() pageConfig.StorageSettings {
	return mustPageConfigDefaults().Storage
}

func GetDefaultTermsOfServiceConfig() pageConfig.TermsOfServiceConfig {
	return mustPageConfigDefaults().Terms
}

func GetDefaultPrivacyPolicyConfig() pageConfig.PrivacyPolicyConfig {
	return mustPageConfigDefaults().Privacy
}

func GetDefaultRateLimitConfig() pageConfig.RateLimitConfig {
	config := mustPageConfigDefaults().RateLimit
	config.Actions = append([]pageConfig.RateLimitRule(nil), config.Actions...)
	return config
}

func GetDefaultMCPSettingsConfig() pageConfig.MCPSettingsConfig {
	return mustPageConfigDefaults().MCP
}

// GetDefaultScheduleSettingsConfig 排课器节次作息表默认值（12 节），
// 与前端内置默认作息表保持一致；未保存配置时 SSR/管理端回显该默认。
func GetDefaultScheduleSettingsConfig() pageConfig.ScheduleSettingsConfig {
	return pageConfig.ScheduleSettingsConfig{
		SectionTimes: []pageConfig.ScheduleSectionTime{
			{Section: 1, Start: "08:00", End: "08:45"},
			{Section: 2, Start: "08:50", End: "09:35"},
			{Section: 3, Start: "10:00", End: "10:45"},
			{Section: 4, Start: "10:50", End: "11:35"},
			{Section: 5, Start: "13:30", End: "14:15"},
			{Section: 6, Start: "14:20", End: "15:05"},
			{Section: 7, Start: "15:30", End: "16:15"},
			{Section: 8, Start: "16:20", End: "17:05"},
			{Section: 9, Start: "17:10", End: "17:55"},
			{Section: 10, Start: "18:30", End: "19:15"},
			{Section: 11, Start: "19:20", End: "20:05"},
			{Section: 12, Start: "20:10", End: "20:55"},
		},
	}
}

func GetDefaultAiSummaryConfig() pageConfig.AiSummaryConfig {
	return mustPageConfigDefaults().AiSummary
}

func GetDefaultSiteSettingsConfig() pageConfig.SiteSettingsConfig {
	return mustPageConfigDefaults().Site
}

func GetDefaultSiteChromeConfig() pageConfig.SiteChromeConfig {
	return pageConfig.SiteChromeConfig{
		Header:        GetDefaultSiteChromeHeader(),
		MainMenu:      []pageConfig.ChromeItem{},
		Resources:     []pageConfig.ChromeItem{},
		SidebarGroups: []pageConfig.ChromeGroup{},
		FooterInfo: pageConfig.FooterInfo{
			Primary: []pageConfig.PItem{{Content: "Providing reliable tech since 2025"}},
			List: []pageConfig.FooterItem{
				{Name: "Github", Url: "https://github.com/YourTongji/YourTJ-Hub/apps/gooseforum"},
				{Name: "License", Url: "https://github.com/YourTongji/YourTJ-Hub/apps/gooseforum/blob/main/LICENSE"},
				{Name: "LeanCodeBox", Url: "https://github.com/leancodebox"},
			},
		},
		BrandType: "default",
	}
}

func GetDefaultSiteChromeHeader() []pageConfig.ChromeItem {
	return []pageConfig.ChromeItem{
		{ID: "sponsors", Enabled: true, Type: "link", Label: "Sponsors", I18nLabel: "shell.nav.sponsors", URL: "/sponsors"},
		{ID: "links", Enabled: true, Type: "link", Label: "Links", I18nLabel: "shell.nav.links", URL: "/links"},
	}
}

func GetDefaultSiteThemeConfig() pageConfig.SiteThemeConfig {
	config := mustPageConfigDefaults().SiteTheme
	config.Themes = cloneSiteThemeDefinitions(config.Themes)
	config.Prepublish = cloneSiteThemePrepublish(config.Prepublish)
	return config
}

func GetDefaultSponsorsConfig() pageConfig.SponsorsConfig {
	return cloneSponsorsConfig(mustPageConfigDefaults().Sponsors)
}

func cloneFriendLinks(groups []pageConfig.FriendLinksGroup) []pageConfig.FriendLinksGroup {
	if groups == nil {
		return nil
	}
	cloned := make([]pageConfig.FriendLinksGroup, len(groups))
	for i, group := range groups {
		cloned[i] = group
		cloned[i].Links = append([]pageConfig.LinkItem(nil), group.Links...)
	}
	return cloned
}

func cloneSponsorsConfig(config pageConfig.SponsorsConfig) pageConfig.SponsorsConfig {
	config.Sponsors.Level0 = append([]pageConfig.SponsorItem(nil), config.Sponsors.Level0...)
	config.Sponsors.Level1 = append([]pageConfig.SponsorItem(nil), config.Sponsors.Level1...)
	config.Sponsors.Level2 = append([]pageConfig.SponsorItem(nil), config.Sponsors.Level2...)
	config.Sponsors.Level3 = append([]pageConfig.SponsorItem(nil), config.Sponsors.Level3...)
	config.Rules = append([]pageConfig.SponsorsRule(nil), config.Rules...)
	return config
}

func cloneSiteThemeDefinitions(items []pageConfig.SiteThemeDefinition) []pageConfig.SiteThemeDefinition {
	if items == nil {
		return nil
	}
	cloned := make([]pageConfig.SiteThemeDefinition, len(items))
	copy(cloned, items)
	return cloned
}

func cloneSiteThemePrepublish(item *pageConfig.SiteThemePrepublish) *pageConfig.SiteThemePrepublish {
	if item == nil {
		return nil
	}
	cloned := *item
	cloned.Themes = cloneSiteThemeDefinitions(item.Themes)
	return &cloned
}
