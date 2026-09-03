package hotdataserve

import (
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
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

// GetMailSettingsConfigCache 读取邮件设置（smtpPassword 为运行时解密明文，
// 仅存于服务内存，绝不随 JSON 序列化导出；落库为 securestore 密文，issue #324 S2）。
// 兼容 v25 迁移前的存量明文密码（SmtpPassword 字段原样使用）。
func GetMailSettingsConfigCache() pageConfig.MailSettingsConfig {
	return mailSettingsConfigCache.GetOrLoad("", func() (pageConfig.MailSettingsConfig, error) {
		entity := pageConfig.GetByPageType(pageConfig.EmailSettings)
		if entity.Id == 0 {
			return defaultconfig.GetDefaultEmailSettingsConfig(), nil
		}
		storage := jsonopt.Decode[pageConfig.MailSettingsStorage](entity.Config)
		cfg := storage.ToConfig()
		if storage.SmtpPasswordEncrypted != "" {
			plain, err := securestore.DecryptPurpose(storage.SmtpPasswordEncrypted, securestore.MailSmtpPasswordPurpose)
			if err != nil {
				slog.Warn("mail smtp password decrypt failed (signing key rotated?)", "err", err)
				cfg.SmtpPassword = ""
			} else {
				cfg.SmtpPassword = plain
			}
		}
		return cfg, nil
	}, configFastCacheTTL)
}

// GetMailSettingsView 读取邮件设置的管理端回显视图（不含密码明文/密文，仅回显
// 是否已配置；issue #324 S2）。
func GetMailSettingsView() pageConfig.MailSettingsView {
	entity := pageConfig.GetByPageType(pageConfig.EmailSettings)
	if entity.Id == 0 {
		return defaultconfig.GetDefaultEmailSettingsConfig().ToView()
	}
	return jsonopt.Decode[pageConfig.MailSettingsStorage](entity.Config).ToView()
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

// GetStorageSettingsConfigCache 读取存储设置（accessKey/secretKey 为运行时解密
// 明文，仅存于服务内存，绝不随 JSON 序列化导出；落库为 securestore 密文，issue #324 S3）。
// 兼容 v25 迁移前的存量明文凭据（AccessKey/SecretKey 字段原样使用）。
func GetStorageSettingsConfigCache() pageConfig.StorageSettings {
	return storageSettingsConfigCache.GetOrLoad("", func() (pageConfig.StorageSettings, error) {
		entity := pageConfig.GetByPageType(pageConfig.StorageSettingsPage)
		if entity.Id == 0 {
			return defaultconfig.GetDefaultStorageSettingsConfig(), nil
		}
		storage := jsonopt.Decode[pageConfig.StorageSettingsStorage](entity.Config)
		cfg := storage.ToConfig()
		decrypt := func(encrypted, purpose string) string {
			if encrypted == "" {
				return ""
			}
			plain, err := securestore.DecryptPurpose(encrypted, purpose)
			if err != nil {
				slog.Warn("storage credential decrypt failed (signing key rotated?)", "err", err)
				return ""
			}
			return plain
		}
		if storage.AccessKeyEncrypted != "" {
			cfg.AccessKey = decrypt(storage.AccessKeyEncrypted, securestore.StorageAccessKeyPurpose)
		}
		if storage.SecretKeyEncrypted != "" {
			cfg.SecretKey = decrypt(storage.SecretKeyEncrypted, securestore.StorageSecretKeyPurpose)
		}
		return cfg, nil
	}, configFastCacheTTL)
}

// GetStorageSettingsView 读取存储设置的管理端回显视图（不含凭据明文/密文，仅
// 回显是否已配置；issue #324 S3）。
func GetStorageSettingsView() pageConfig.StorageSettingsView {
	entity := pageConfig.GetByPageType(pageConfig.StorageSettingsPage)
	if entity.Id == 0 {
		return defaultconfig.GetDefaultStorageSettingsConfig().ToView()
	}
	return jsonopt.Decode[pageConfig.StorageSettingsStorage](entity.Config).ToView()
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
		return pageConfig.GetPostingSettingsConfig(defaultconfig.GetDefaultPostingSettingsConfig()), nil
	}, configFastCacheTTL)
}

var rateLimitConfigCache = &localcache.Cache[pageConfig.RateLimitConfig]{MaxEntries: cacheconfig.Current().PageConfig}

func GetRateLimitConfigCache() pageConfig.RateLimitConfig {
	return rateLimitConfigCache.GetOrLoad("", func() (pageConfig.RateLimitConfig, error) {
		cfg := pageConfig.GetConfigByPageType(pageConfig.RateLimitSettings, defaultconfig.GetDefaultRateLimitConfig())
		mergeDefaultRateLimitActions(&cfg)
		cfg.BuildActionIndex()
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

// GetHttpNotifyConfigCache 读取 HTTP 通知配置（各端点 secret 为运行时解密明文，
// 仅存于服务内存，绝不随 JSON 序列化导出；落库为 securestore 密文，issue #324 S1）。
// 兼容 v25 迁移前的存量明文 secret（Secret 字段原样使用）。
func GetHttpNotifyConfigCache() pageConfig.HttpNotifyConfig {
	return httpNotifyConfigCache.GetOrLoad("", func() (pageConfig.HttpNotifyConfig, error) {
		entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
		if entity.Id == 0 {
			return defaultconfig.GetDefaultHttpNotifyConfig(), nil
		}
		storage := jsonopt.Decode[pageConfig.HttpNotifyStorageConfig](entity.Config)
		cfg := storage.ToConfig()
		for i := range cfg.Endpoints {
			secret := cfg.Endpoints[i].Secret
			if secret == "" {
				continue
			}
			// 仅当 SecretEncrypted 非空时才走解密路径（ToConfig 已优先取密文）；
			// 存量明文（Secret 字段）原样使用，等待 v25 迁移加密。
			if storage.Endpoints[i].SecretEncrypted != "" {
				if plain, err := securestore.DecryptPurpose(secret, securestore.HttpNotifySecretPurpose); err == nil {
					cfg.Endpoints[i].Secret = plain
				} else {
					slog.Warn("http notify secret decrypt failed (signing key rotated?)", "err", err)
					cfg.Endpoints[i].Secret = ""
				}
			}
		}
		return cfg, nil
	}, configRareCacheTTL)
}

// GetHttpNotifyView 读取 HTTP 通知设置的管理端回显视图（不含密钥明文/密文，仅
// 回显是否已配置；issue #324 S1）。
func GetHttpNotifyView() pageConfig.HttpNotifyView {
	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	if entity.Id == 0 {
		return defaultconfig.GetDefaultHttpNotifyConfig().ToView()
	}
	return jsonopt.Decode[pageConfig.HttpNotifyStorageConfig](entity.Config).ToView()
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

var aiSummarySettingsConfigCache = &localcache.Cache[pageConfig.AiSummaryConfig]{MaxEntries: cacheconfig.Current().PageConfig}

// GetAiSummarySettingsConfigCache 读取 AI 课程总结配置（5s TTL 热缓存）。
// 落库形状为 AiSummarySettingsStorage（apiKey 密文带 json 标签），读取后转
// 领域结构并解密 apiKey 为运行时明文（仅存服务内存，绝不随 JSON 导出）。
// 解密失败（如 signing key 轮换）时 apiKey 置空并告警，避免静默用错误密钥调用。
func GetAiSummarySettingsConfigCache() pageConfig.AiSummaryConfig {
	return aiSummarySettingsConfigCache.GetOrLoad("", func() (pageConfig.AiSummaryConfig, error) {
		storage := pageConfig.GetConfigByPageType(pageConfig.AiSummarySettings, pageConfig.AiSummarySettingsStorage{})
		cfg := storage.ToConfig()
		if encrypted := strings.TrimSpace(cfg.APIKey); encrypted != "" {
			plain, err := securestore.DecryptPurpose(encrypted, securestore.AiSummaryAPIKeyPurpose)
			if err != nil {
				slog.Warn("ai_summary api key decrypt failed (signing key rotated?)", "err", err)
				cfg.APIKey = ""
			} else {
				cfg.APIKey = plain
			}
		}
		return cfg, nil
	}, configFastCacheTTL)
}

// GetAiSummarySettingsView 读取 AI 课程总结配置的管理端回显视图：apiKey 仅回显
// 是否已配置（明文/密文均不出现在响应中，issue #324 安全模式）。
func GetAiSummarySettingsView() pageConfig.AiSummarySettingsView {
	entity := pageConfig.GetByPageType(pageConfig.AiSummarySettings)
	if entity.Id == 0 {
		return defaultconfig.GetDefaultAiSummaryConfig().ToView()
	}
	return jsonopt.Decode[pageConfig.AiSummarySettingsStorage](entity.Config).ToView()
}

func ClearAiSummarySettingsConfigCache() {
	aiSummarySettingsConfigCache.Clear()
}

var oneSystemSettingsConfigCache = &localcache.Cache[pageConfig.OneSystemSettingsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

// GetOnesystemSettingsConfigCache 读取一系统同步凭证配置（cookie 为密文）。
// 落库形状为 OneSystemSettingsStorage（密文带 json 标签），读取后转领域结构，
// 避免密文随领域结构被整包序列化导出（review MEDIUM）。
func GetOnesystemSettingsConfigCache() pageConfig.OneSystemSettingsConfig {
	return oneSystemSettingsConfigCache.GetOrLoad("", func() (pageConfig.OneSystemSettingsConfig, error) {
		storage := pageConfig.GetConfigByPageType(pageConfig.OneSystemSettings, pageConfig.OneSystemSettingsStorage{})
		return storage.ToConfig(), nil
	}, configFastCacheTTL)
}

func ClearOnesystemSettingsConfigCache() {
	oneSystemSettingsConfigCache.Clear()
}

var wikiSyncSettingsConfigCache = &localcache.Cache[pageConfig.WikiSyncSettingsConfig]{MaxEntries: cacheconfig.Current().PageConfig}

// GetWikiSyncSettingsConfigCache 读取 wiki GitHub webhook 验签密钥配置（密文）。
// 落库形状为 WikiSyncSettingsStorage，读取后转领域结构，避免密文随领域结构
// 被整包序列化导出。
func GetWikiSyncSettingsConfigCache() pageConfig.WikiSyncSettingsConfig {
	return wikiSyncSettingsConfigCache.GetOrLoad("", func() (pageConfig.WikiSyncSettingsConfig, error) {
		storage := pageConfig.GetConfigByPageType(pageConfig.WikiSyncSettings, pageConfig.WikiSyncSettingsStorage{})
		return storage.ToConfig(), nil
	}, configFastCacheTTL)
}

func ClearWikiSyncSettingsConfigCache() {
	wikiSyncSettingsConfigCache.Clear()
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
