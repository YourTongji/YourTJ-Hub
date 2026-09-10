package datamigration

import (
	"encoding/json"
	"errors"
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"gorm.io/gorm"
)

// SecuritySettingsMigrationResult 汇报安全设置默认词库补齐（Blueprint R4）的结果。
type SecuritySettingsMigrationResult struct {
	Updated    bool
	Skipped    bool
	Failed     int
	LastFailed string
}

// securitySettingsArrayKeys 列出空数组时并入默认的数组键。bannedUsernames
// 刻意不在其中：默认 banned 为空，任何实例都不应被写入 banned 词——一旦
// 并入默认，会把存量账号静默推入冻结（防误冻结语义，见下注释）。
var securitySettingsArrayKeys = []string{"reservedUsernames", "sensitiveWords"}

// EnsureSecuritySettingsDefaults 为已保存过且数组为空的存量 securitySettings
// 配置并入新默认词库（Blueprint R4）：
//
//   - reservedUsernames / sensitiveWords 各自独立判空，仅当数组为空（len==0）
//     或键缺失时并入默认配置中对应数组；
//   - bannedUsernames 永不写入（默认 banned 为空，并入会把存量账号推入
//     静默冻结，防误冻结）；
//   - 其余字段与已非空数组一律原样保留，未知字段经 map 读改写原样透传。
//
// 幂等：补写后数组非空，重跑不再变更。无 securitySettings 行时 Skipped=true
// （读取侧 pageConfig.GetConfigByPageType 自然返回内置新默认，无需写行）。
func EnsureSecuritySettingsDefaults() SecuritySettingsMigrationResult {
	return EnsureSecuritySettingsDefaultsWithDB(db.Connect())
}

// EnsureSecuritySettingsDefaultsWithDB 是 EnsureSecuritySettingsDefaults 的可测试核心，
// 直接对 page_config 行做读改写。
func EnsureSecuritySettingsDefaultsWithDB(conn *gorm.DB) SecuritySettingsMigrationResult {
	result := SecuritySettingsMigrationResult{}
	if !conn.Migrator().HasTable("page_config") {
		return result
	}
	var entity struct {
		ID     uint64 `gorm:"column:id"`
		Config string `gorm:"column:config"`
	}
	if err := conn.Table("page_config").Where("page_type = ?", pageConfig.SecuritySettings).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Skipped = true
		} else {
			result.Failed = 1
			result.LastFailed = "read security settings config: " + err.Error()
			slog.Error("security settings defaults migration: read config failed", "err", err)
		}
		return result
	}

	config, changed, err := mergeEmptySecurityArrays(entity.Config)
	if err != nil {
		result.Failed = 1
		result.LastFailed = err.Error()
		slog.Error("security settings defaults migration: merge default arrays failed", "err", err)
		return result
	}
	if !changed {
		return result
	}
	if err := conn.Table("page_config").Where("page_type = ?", pageConfig.SecuritySettings).Update("config", config).Error; err != nil {
		result.Failed = 1
		result.LastFailed = "save security settings config: " + err.Error()
		slog.Error("security settings defaults migration: save config failed", "err", err)
		return result
	}
	result.Updated = true
	return result
}

// mergeEmptySecurityArrays 把已存 securitySettings 配置中为空（或缺失）的
// 数组键并入默认词库。返回 (更新后的配置, 是否发生变更, 错误)。
// 以 map[string]json.RawMessage 读改写：只替换 securitySettingsArrayKeys
// 中的数组键，其余键值保持 raw，未知字段原样透传。
func mergeEmptySecurityArrays(config string) (string, bool, error) {
	defaults := defaultconfig.GetDefaultSecuritySettingsConfig()
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(config), &root); err != nil {
		return "", false, err
	}

	changed := false
	for _, key := range securitySettingsArrayKeys {
		words := []string{}
		if raw, ok := root[key]; ok && len(raw) > 0 {
			if err := json.Unmarshal(raw, &words); err != nil {
				return "", false, err
			}
		}
		if len(words) > 0 {
			continue
		}
		var defaultWords []string
		switch key {
		case "reservedUsernames":
			defaultWords = defaults.ReservedUsernames
		case "sensitiveWords":
			defaultWords = defaults.SensitiveWords
		}
		merged, err := json.Marshal(defaultWords)
		if err != nil {
			return "", false, err
		}
		root[key] = merged
		changed = true
	}
	if !changed {
		return config, false, nil
	}
	updated, err := json.Marshal(root)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}
