package hotdataserve

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

const (
	configFastCacheTTL = 5 * time.Second
	configSlowCacheTTL = time.Minute
	configRareCacheTTL = time.Hour
)

var sponsorsConfigCache = &localcache.Cache[pageConfig.SponsorsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func SponsorsConfigCache() pageConfig.SponsorsConfig {
	return sponsorsConfigCache.GetOrLoad("", func() (pageConfig.SponsorsConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.SponsorsPage, defaultconfig.GetDefaultSponsorsConfig()), nil
	}, configSlowCacheTTL)
}

var siteSettingsConfigCache = &localcache.Cache[pageConfig.SiteSettingsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetSiteSettingsConfigCache() pageConfig.SiteSettingsConfig {
	return siteSettingsConfigCache.GetOrLoad("", func() (pageConfig.SiteSettingsConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.SiteSettings, defaultconfig.GetDefaultSiteSettingsConfig()), nil
	}, configFastCacheTTL)
}

var siteThemeConfigCache = &localcache.Cache[pageConfig.SiteThemeConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetSiteThemeConfigCache() pageConfig.SiteThemeConfig {
	return siteThemeConfigCache.GetOrLoad("", func() (pageConfig.SiteThemeConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.SiteTheme, defaultconfig.GetDefaultSiteThemeConfig()), nil
	}, configFastCacheTTL)
}

var siteChromeConfigCache = &localcache.Cache[pageConfig.SiteChromeConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetSiteChromeConfigCache() pageConfig.SiteChromeConfig {
	return siteChromeConfigCache.GetOrLoad("", func() (pageConfig.SiteChromeConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.SiteChrome, defaultconfig.GetDefaultSiteChromeConfig()), nil
	}, configFastCacheTTL)
}

var mailSettingsConfigCache = &localcache.Cache[pageConfig.MailSettingsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetMailSettingsConfigCache() pageConfig.MailSettingsConfig {
	return mailSettingsConfigCache.GetOrLoad("", func() (pageConfig.MailSettingsConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.EmailSettings, defaultconfig.GetDefaultEmailSettingsConfig()), nil
	}, configFastCacheTTL)
}

var announcementConfigCache = &localcache.Cache[pageConfig.AnnouncementConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetAnnouncementConfigCache() pageConfig.AnnouncementConfig {
	return announcementConfigCache.GetOrLoad("", func() (pageConfig.AnnouncementConfig, error) {
		config := pageConfig.GetConfigByPageType(pageConfig.Announcement, defaultconfig.GetDefaultAnnouncementConfig())
		config.PrepareHTML()
		return config, nil
	}, configFastCacheTTL)
}

var securitySettingsConfigCache = &localcache.Cache[pageConfig.SecurityAndRegistration]{MaxEntries: cacheconfig.Current().PageConfig}

func GetSecuritySettingsConfigCache() pageConfig.SecurityAndRegistration {
	return securitySettingsConfigCache.GetOrLoad("", func() (pageConfig.SecurityAndRegistration, error) {
		return pageConfig.GetConfigByPageType(pageConfig.SecuritySettings, defaultconfig.GetDefaultSecuritySettingsConfig()), nil
	}, configFastCacheTTL)
}

var storageSettingsConfigCache = &localcache.Cache[pageConfig.StorageSettings]{MaxEntries: cacheconfig.Current().PageConfig}

func GetStorageSettingsConfigCache() pageConfig.StorageSettings {
	return storageSettingsConfigCache.GetOrLoad("", func() (pageConfig.StorageSettings, error) {
		return pageConfig.GetConfigByPageType(pageConfig.StorageSettingsPage, defaultconfig.GetDefaultStorageSettingsConfig()), nil
	}, configFastCacheTTL)
}

var termsOfServiceConfigCache = &localcache.Cache[pageConfig.TermsOfServiceConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetTermsOfServiceConfigCache() pageConfig.TermsOfServiceConfig {
	return termsOfServiceConfigCache.GetOrLoad("", func() (pageConfig.TermsOfServiceConfig, error) {
		config := pageConfig.GetConfigByPageType(pageConfig.TermsOfService, defaultconfig.GetDefaultTermsOfServiceConfig())
		config.PrepareHTML()
		return config, nil
	}, configFastCacheTTL)
}

func ClearStorageSettingsConfigCache() {
	storageSettingsConfigCache.Clear()
}

func ClearTermsOfServiceConfigCache() {
	termsOfServiceConfigCache.Clear()
}

var privacyPolicyConfigCache = &localcache.Cache[pageConfig.PrivacyPolicyConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetPrivacyPolicyConfigCache() pageConfig.PrivacyPolicyConfig {
	return privacyPolicyConfigCache.GetOrLoad("", func() (pageConfig.PrivacyPolicyConfig, error) {
		config := pageConfig.GetConfigByPageType(pageConfig.PrivacyPolicy, defaultconfig.GetDefaultPrivacyPolicyConfig())
		config.PrepareHTML()
		return config, nil
	}, configFastCacheTTL)
}

func ClearPrivacyPolicyConfigCache() {
	privacyPolicyConfigCache.Clear()
}

var postingSettingsConfigCache = &localcache.Cache[pageConfig.PostingContent]{MaxEntries: cacheconfig.Current().PageConfig}

func GetPostingSettingsConfigCache() pageConfig.PostingContent {
	return postingSettingsConfigCache.GetOrLoad("", func() (pageConfig.PostingContent, error) {
		return pageConfig.GetConfigByPageType(pageConfig.PostingSettings, defaultconfig.GetDefaultPostingSettingsConfig()), nil
	}, configFastCacheTTL)
}

var rateLimitConfigCache = &localcache.Cache[pageConfig.RateLimitConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetRateLimitConfigCache() pageConfig.RateLimitConfig {
	return rateLimitConfigCache.GetOrLoad("", func() (pageConfig.RateLimitConfig, error) {
		cfg := pageConfig.GetConfigByPageType(pageConfig.RateLimitSettings, defaultconfig.GetDefaultRateLimitConfig())
		mergeDefaultRateLimitActions(&cfg)
		return cfg, nil
	}, configFastCacheTTL)
}

// mergeDefaultRateLimitActions 把默认配置中缺失的 action 补入已存配置，
// 保证升级后新增的 action（如 llms.index/full/topic）在存量部署自动生效。
// 管理员若想停用某 action，应把其 limit 清零（仍在列表内）；删除整条会被视为回到默认。
func mergeDefaultRateLimitActions(cfg *pageConfig.RateLimitConfig) {
	defaults := defaultconfig.GetDefaultRateLimitConfig()
	known := make(map[string]bool, len(cfg.Actions))
	for _, rule := range cfg.Actions {
		known[rule.Action] = true
	}
	for _, rule := range defaults.Actions {
		if !known[rule.Action] {
			cfg.Actions = append(cfg.Actions, rule)
			known[rule.Action] = true
		}
	}
}

func ClearRateLimitConfigCache() {
	rateLimitConfigCache.Clear()
}

var httpNotifyConfigCache = &localcache.Cache[pageConfig.HttpNotifyConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetHttpNotifyConfigCache() pageConfig.HttpNotifyConfig {
	return httpNotifyConfigCache.GetOrLoad("", func() (pageConfig.HttpNotifyConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.HttpNotify, defaultconfig.GetDefaultHttpNotifyConfig()), nil
	}, configRareCacheTTL)
}

var mcpSettingsConfigCache = &localcache.Cache[pageConfig.MCPSettingsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetMCPSettingsConfigCache() pageConfig.MCPSettingsConfig {
	return mcpSettingsConfigCache.GetOrLoad("", func() (pageConfig.MCPSettingsConfig, error) {
		return pageConfig.GetConfigByPageType(pageConfig.MCPSettings, defaultconfig.GetDefaultMCPSettingsConfig()), nil
	}, configFastCacheTTL)
}

func ClearMCPSettingsConfigCache() {
	mcpSettingsConfigCache.Clear()
}

func ClearSecuritySettingsConfigCache() {
	securitySettingsConfigCache.Clear()
}

func ClearPostingSettingsConfigCache() {
	postingSettingsConfigCache.Clear()
}

func ClearHttpNotifyConfigCache() {
	httpNotifyConfigCache.Clear()
}

func ClearSiteSettingsConfigCache() {
	siteSettingsConfigCache.Clear()
}

func ClearSiteThemeConfigCache() {
	siteThemeConfigCache.Clear()
}

func ClearSiteChromeConfigCache() {
	siteChromeConfigCache.Clear()
}

func ClearMailSettingsConfigCache() {
	mailSettingsConfigCache.Clear()
}

func ClearAnnouncementConfigCache() {
	announcementConfigCache.Clear()
}

func ClearSponsorsConfigCache() {
	sponsorsConfigCache.Clear()
}

var friendLinksConfigCache = &localcache.Cache[[]pageConfig.FriendLinksGroup]{MaxEntries: cacheconfig.Current().PageConfig}

func GetFriendLinksConfigCache() []pageConfig.FriendLinksGroup {
	return friendLinksConfigCache.GetOrLoad("", func() ([]pageConfig.FriendLinksGroup, error) {
		return pageConfig.GetConfigByPageType(pageConfig.FriendShipLinks, defaultconfig.GetDefaultFriendLinksConfig()), nil
	}, configSlowCacheTTL)
}

func ClearFriendLinksConfigCache() {
	friendLinksConfigCache.Clear()
}
