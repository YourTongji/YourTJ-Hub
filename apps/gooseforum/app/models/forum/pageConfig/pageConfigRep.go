package pageConfig

import (
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
	if entity.Id > 0 {
		return jsonopt.Decode[T](entity.Config)
	}

	return defaultValue
}

const AppMigrationVersion uint32 = 23

func GetMigrationVersion() uint32 {
	configEntity := GetByPageType(Migration)
	return cast.ToUint32(configEntity.Config)
}

func SyncMigrationVersion(version uint32) {
	configEntity := GetByPageType(Migration)
	configEntity.PageType = Migration
	configEntity.Config = cast.ToString(version)
	CreateOrSave(&configEntity)
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
