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

// PostingSettingsMigrationResult 汇报发帖设置每日主题上限默认值补齐（issue #369）的结果。
type PostingSettingsMigrationResult struct {
	Updated    bool
	Skipped    bool
	Failed     int
	LastFailed string
}

// EnsurePostingSettingsTopicLimit 为存量 postingSettings 配置补齐
// textControl.maxDailyTopicsPerUser（默认 10，0 = 不限额；issue #369，上游 c47cff94）。
//
// 幂等：已存在该键的配置跳过；无 postingSettings 行时 no-op。读取侧
// （pageConfig.GetPostingSettingsConfig）在迁移前兼容缺键，迁移只做落库归一。
func EnsurePostingSettingsTopicLimit() PostingSettingsMigrationResult {
	return EnsurePostingSettingsTopicLimitWithDB(db.Connect())
}

// EnsurePostingSettingsTopicLimitWithDB 是 EnsurePostingSettingsTopicLimit 的可测试核心，
// 直接对 page_config 行做读改写。
func EnsurePostingSettingsTopicLimitWithDB(conn *gorm.DB) PostingSettingsMigrationResult {
	result := PostingSettingsMigrationResult{}
	if !conn.Migrator().HasTable("page_config") {
		return result
	}
	var entity struct {
		ID     uint64 `gorm:"column:id"`
		Config string `gorm:"column:config"`
	}
	if err := conn.Table("page_config").Where("page_type = ?", pageConfig.PostingSettings).First(&entity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Skipped = true
		} else {
			result.Failed = 1
			result.LastFailed = "read posting settings config: " + err.Error()
			slog.Error("posting settings topic limit migration: read config failed", "err", err)
		}
		return result
	}

	defaultLimit := defaultconfig.GetDefaultPostingSettingsConfig().TextControl.MaxDailyTopicsPerUser
	config, changed, err := addDefaultTopicLimit(entity.Config, defaultLimit)
	if err != nil {
		result.Failed = 1
		result.LastFailed = err.Error()
		slog.Error("posting settings topic limit migration: add default topic limit failed", "err", err)
		return result
	}
	if !changed {
		return result
	}
	if err := conn.Table("page_config").Where("page_type = ?", pageConfig.PostingSettings).Update("config", config).Error; err != nil {
		result.Failed = 1
		result.LastFailed = "save posting settings config: " + err.Error()
		slog.Error("posting settings topic limit migration: save config failed", "err", err)
		return result
	}
	result.Updated = true
	return result
}

// addDefaultTopicLimit 在 textControl 中写入 maxDailyTopicsPerUser（缺键时）。
// 返回 (更新后的配置, 是否发生变更, 错误)。已存在该键时原样返回不写库。
func addDefaultTopicLimit(config string, defaultLimit int) (string, bool, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal([]byte(config), &root); err != nil {
		return "", false, err
	}

	textControl := map[string]json.RawMessage{}
	if raw, ok := root["textControl"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &textControl); err != nil {
			return "", false, err
		}
	}
	if _, exists := textControl["maxDailyTopicsPerUser"]; exists {
		return config, false, nil
	}

	limit, err := json.Marshal(defaultLimit)
	if err != nil {
		return "", false, err
	}
	textControl["maxDailyTopicsPerUser"] = limit
	textControlJSON, err := json.Marshal(textControl)
	if err != nil {
		return "", false, err
	}
	root["textControl"] = textControlJSON
	updated, err := json.Marshal(root)
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}
