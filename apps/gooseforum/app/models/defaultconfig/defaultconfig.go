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
	// 列表字段拷贝并保证非 nil（issue #643）：append([]string(nil),...) 在
	// 默认空列表时退化为 nil，JSON 序列化产出 null，违反 SecurityAndRegistration
	// 的 required string[] 契约并使登录页前端 setup 解引用白屏。
	config.AllowedDomains = append([]string{}, config.AllowedDomains...)
	config.ReservedUsernames = append([]string{}, config.ReservedUsernames...)
	config.BannedUsernames = append([]string{}, config.BannedUsernames...)
	config.SensitiveWords = append([]string{}, config.SensitiveWords...)
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

// GetDefaultScheduleSettingsConfig 排课器节次作息表默认值（现行 11 节制：
// 2025-2026 学年起白天 1-8 节 + 晚间 9/10/11 节 18:30 起），
// 与前端内置 11 节默认作息表保持一致；未保存配置时 SSR/管理端回显该默认。
// 历史 12 节制学期（calendarId<120）课表由前端内置历史表渲染，不受此配置影响。
func GetDefaultScheduleSettingsConfig() pageConfig.ScheduleSettingsConfig {
	return pageConfig.ScheduleSettingsConfig{
		Numbering: pageConfig.ScheduleNumberingCurrent,
		SectionTimes: []pageConfig.ScheduleSectionTime{
			{Section: 1, Start: "08:00", End: "08:45"},
			{Section: 2, Start: "08:50", End: "09:35"},
			{Section: 3, Start: "10:00", End: "10:45"},
			{Section: 4, Start: "10:50", End: "11:35"},
			{Section: 5, Start: "13:30", End: "14:15"},
			{Section: 6, Start: "14:20", End: "15:05"},
			{Section: 7, Start: "15:30", End: "16:15"},
			{Section: 8, Start: "16:20", End: "17:05"},
			{Section: 9, Start: "18:30", End: "19:15"},
			{Section: 10, Start: "19:20", End: "20:05"},
			{Section: 11, Start: "20:10", End: "20:55"},
		},
	}
}

// NormalizeStoredScheduleSettings 读取侧归一节次作息配置（review P1）：
// PR #496 之前保存的配置没有 numbering 标记且为旧 12 节编号（第 9 节 17:10、
// 晚间 10/11/12 节），直接按节次号合并进现行 11 节视图会把晚间整体错位一格，
// 且管理端回显旧值后保存会把错值再次持久化。归一规则：
//
//   - Numbering == 现行（"11"）：完全透传——现行语义下第 9 节允许被管理员合法
//     设为任意时间（含 17:10-17:55），不得基于时间值猜测编号体系；
//   - 其余（缺省 = PR #496 之前写入的存量行，或显式旧标记）：按旧 12 节编号
//     解释——白天 1-8 节照搬，新 9/10/11 节取旧 10/11/12 节（晚间物理时段未变，
//     仅编号前移，旧第 9 节时段已取消故丢弃），即使第 9 节被管理员自定义过。
//
// 归一结果恒以现行编号盖章返回，SSR 与管理端 GET 均得到现行语义；管理端
// 保存（服务端盖章现行编号）后存储自愈，无需数据迁移或人工修复。
func NormalizeStoredScheduleSettings(cfg pageConfig.ScheduleSettingsConfig) pageConfig.ScheduleSettingsConfig {
	if cfg.Numbering == pageConfig.ScheduleNumberingCurrent {
		return cfg
	}
	if len(cfg.SectionTimes) == 0 {
		return cfg
	}
	bySection := make(map[int]pageConfig.ScheduleSectionTime, len(cfg.SectionTimes))
	for _, item := range cfg.SectionTimes {
		if item.Section >= 1 && item.Section <= 12 {
			bySection[item.Section] = item
		}
	}
	if len(bySection) == 0 {
		return cfg
	}

	// 未版本化（存量）或显式旧标记：按旧编号重映射，新 9/10/11 节 ← 旧 10/11/12 节。
	normalized := make([]pageConfig.ScheduleSectionTime, 0, 11)
	for section := 1; section <= 8; section++ {
		if item, ok := bySection[section]; ok {
			normalized = append(normalized, item)
		}
	}
	for section := 10; section <= 12; section++ {
		if item, ok := bySection[section]; ok {
			item.Section = section - 1
			normalized = append(normalized, item)
		}
	}
	return pageConfig.ScheduleSettingsConfig{
		Numbering:    pageConfig.ScheduleNumberingCurrent,
		SectionTimes: normalized,
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
