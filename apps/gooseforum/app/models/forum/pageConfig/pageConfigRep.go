package pageConfig

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/spf13/cast"
)

func create(entity *Entity) int64 {
	result := builder().Create(entity)
	return result.RowsAffected
}

func save(entity *Entity) int64 {
	result := builder().Save(entity)
	return result.RowsAffected
}

func CreateOrSave(entity *Entity) int64 {
	if entity.Id == 0 {
		return create(entity)
	}

	return save(entity)
}

func GetByPageType(pageType string) (entity Entity) {
	builder().Where(queryopt.Eq(filedPageType, pageType)).First(&entity)
	return
}

func GetConfigByPageType[T any](pageType string, defaultValue T) T {
	var entity Entity
	builder().Where(queryopt.Eq(filedPageType, pageType)).First(&entity)
	if entity.Id == 0 {
		return defaultValue
	}
	decoded, err := jsonopt.DecodeE[T](entity.Config)
	if err != nil {
		// 配置 JSON 损坏时回退默认值（review LOW）：损坏行不应让读方拿到空配置
		// （如节次作息整表消失、默认 12 节退化为 0 节），回退默认值保证页面可用；
		// 同时告警便于运维发现并修复该行。
		slog.Warn("pageConfig: stored config JSON corrupted, falling back to default",
			"page_type", pageType, "err", err)
		return defaultValue
	}
	return decoded
}

// GetPostingSettingsConfig 读取发布内容设置（issue #369，上游 c47cff94）。
// 与 GetConfigByPageType 的区别：升级前的存量配置缺少
// textControl.maxDailyTopicsPerUser 时用默认值补齐（避免旧配置在升级后被
// 当作 0=不限额）；非法负值归一为 0（不限额），与保存端归一化一致。
func GetPostingSettingsConfig(defaultValue PostingContent) PostingContent {
	entity := GetByPageType(PostingSettings)
	if entity.Id == 0 {
		return defaultValue
	}

	config := jsonopt.Decode[PostingContent](entity.Config)
	var raw struct {
		TextControl map[string]json.RawMessage `json:"textControl"`
	}
	if err := json.Unmarshal([]byte(entity.Config), &raw); err == nil {
		if _, exists := raw.TextControl["maxDailyTopicsPerUser"]; !exists {
			config.TextControl.MaxDailyTopicsPerUser = defaultValue.TextControl.MaxDailyTopicsPerUser
		}
	}
	if config.TextControl.MaxDailyTopicsPerUser < 0 {
		config.TextControl.MaxDailyTopicsPerUser = 0
	}
	return config
}

const AppMigrationVersion uint32 = 26

func GetMigrationVersion() uint32 {
	configEntity := GetByPageType(Migration)
	return cast.ToUint32(configEntity.Config)
}

func SyncMigrationVersion(version uint32) error {
	configEntity := GetByPageType(Migration)
	configEntity.PageType = Migration
	configEntity.Config = cast.ToString(version)
	if configEntity.Id == 0 {
		result := builder().Create(&configEntity)
		return result.Error
	}
	return builder().Save(&configEntity).Error
}

// wikiSyncSettingsMu 序列化 WikiSyncSettings 的读改写：webhook secret 与
// assetCDN 共用同一 Config blob，并发保存若各自读改写会互相覆盖（如清除
// 密钥的请求被并发 CDN 保存用旧读值覆盖，静默重新启用 webhook）。
// 进程内互斥即可——两个写入方都在本进程（HTTP handler）。
var wikiSyncSettingsMu sync.Mutex

// UpdateWikiSyncSettings 原子读改写 wiki 同步设置：mutate 在互斥区内拿到
// 当前落库形状，返回新形状后整体写回。调用方负责清 hotdataserve 缓存。
func UpdateWikiSyncSettings(mutate func(WikiSyncSettingsStorage) WikiSyncSettingsStorage) {
	wikiSyncSettingsMu.Lock()
	defer wikiSyncSettingsMu.Unlock()

	storage := GetConfigByPageType(WikiSyncSettings, WikiSyncSettingsStorage{})
	storage = mutate(storage)
	entity := GetByPageType(WikiSyncSettings)
	entity.PageType = WikiSyncSettings
	entity.Config = jsonopt.Encode(storage)
	CreateOrSave(&entity)
}
