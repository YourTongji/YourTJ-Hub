package hotdataserve

import (
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/cacheconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

var aiModerationConfigCache = &localcache.Cache[pageConfig.AiModerationConfig]{MaxEntries: cacheconfig.Current().PageConfig}

// GetAiModerationConfigCache 读取 AI 图文审查运行时配置（issue #975，5s TTL 热缓存）。
// 选项先归一化；两个 API key 在内存中解密为明文（json:"-"，绝不随 JSON 导出）。
// 解密失败（如 signing key 轮换）时该 key 置空并告警——审查链路据此视为未配置，
// 按 fail-closed 把内容送人工审核，而不是用错误凭据调用 provider。
func GetAiModerationConfigCache() pageConfig.AiModerationConfig {
	return aiModerationConfigCache.GetOrLoad("", func() (pageConfig.AiModerationConfig, error) {
		storage := GetAiModerationSettingsStorage()
		return pageConfig.AiModerationConfig{
			AiModerationOptions: storage.AiModerationOptions.Normalize(),
			JevAPIKey:           decryptAiModerationKey(storage.JevAPIKeyEncrypted, securestore.ModerationJevAPIKeyPurpose),
			VisionAPIKey:        decryptAiModerationKey(storage.VisionAPIKeyEncrypted, securestore.ModerationVisionAPIKeyPurpose),
		}, nil
	}, configFastCacheTTL)
}

func decryptAiModerationKey(encrypted, purpose string) string {
	encrypted = strings.TrimSpace(encrypted)
	if encrypted == "" {
		return ""
	}
	plain, err := securestore.DecryptPurpose(encrypted, purpose)
	if err != nil {
		slog.Warn("ai moderation api key decrypt failed (signing key rotated?)", "purpose", purpose, "err", err)
		return ""
	}
	return plain
}

// GetAiModerationSettingsStorage 读取 AI 审查配置的落库形状（含密文，仅供保存
// 路径合并旧密文使用，绝不直接回给客户端）。
func GetAiModerationSettingsStorage() pageConfig.AiModerationSettingsStorage {
	entity := pageConfig.GetByPageType(pageConfig.AiModerationPage)
	if entity.Id == 0 {
		return pageConfig.AiModerationSettingsStorage{}
	}
	return jsonopt.Decode[pageConfig.AiModerationSettingsStorage](entity.Config)
}

// GetAiModerationSettingsView 管理端回显：密钥只回显是否已配置。
func GetAiModerationSettingsView() pageConfig.AiModerationSettingsView {
	return GetAiModerationSettingsStorage().ToView()
}

func ClearAiModerationConfigCache() {
	aiModerationConfigCache.Clear()
}
