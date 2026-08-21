package datamigration

import (
	"encoding/json"
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"gorm.io/gorm"
)

// SecretAtRestMigrationResult 汇报管理端设置密钥明文迁移（issue #324 S1-S3）的结果。
type SecretAtRestMigrationResult struct {
	MailEncrypted int
	StorageKeys   int
	NotifySecrets int
	Failed        int
	LastFailed    string
}

// MigrateAdminSecretPlaintext 把 v25 之前落库的明文管理端密钥（邮件 SMTP 密码、
// 对象存储 accessKey/secretKey、HTTP 通知端点 secret）加密为 securestore 密文
// （AES-256-GCM），写入 *Encrypted 字段后清空明文。
//
// 幂等：只处理「明文非空且密文字段为空」的配置；已加密的配置跳过；无配置 no-op。
// 任一配置加密失败即返回失败信息，由上层决定不推进迁移版本，服务可正常启动并在
// 下次重启时重试。读取侧（hotdataserve）在迁移前兼容存量明文（原样使用）。
func MigrateAdminSecretPlaintext() SecretAtRestMigrationResult {
	return MigrateAdminSecretPlaintextWithDB(db.Connect())
}

// MigrateAdminSecretPlaintextWithDB 是 MigrateAdminSecretPlaintext 的可测试核心，
// 直接对 page_config 行做读改写。
func MigrateAdminSecretPlaintextWithDB(conn *gorm.DB) SecretAtRestMigrationResult {
	result := SecretAtRestMigrationResult{}
	if !conn.Migrator().HasTable("page_config") {
		return result
	}
	migrate := func(pageType string, mutate func(config map[string]any) error) {
		var entity struct {
			ID     uint64 `gorm:"column:id"`
			Config string `gorm:"column:config"`
		}
		if err := conn.Table("page_config").Where("page_type = ?", pageType).First(&entity).Error; err != nil {
			if err != gorm.ErrRecordNotFound {
				result.Failed++
				result.LastFailed = pageType + ": read config: " + err.Error()
				slog.Error("admin secret migration: read config failed", "pageType", pageType, "err", err)
			}
			return
		}
		var config map[string]any
		if err := json.Unmarshal([]byte(entity.Config), &config); err != nil {
			result.Failed++
			result.LastFailed = pageType + ": decode config: " + err.Error()
			slog.Error("admin secret migration: decode config failed", "pageType", pageType, "err", err)
			return
		}
		if err := mutate(config); err != nil {
			result.Failed++
			result.LastFailed = pageType + ": " + err.Error()
			slog.Error("admin secret migration: mutate failed", "pageType", pageType, "err", err)
			return
		}
		encoded, err := json.Marshal(config)
		if err != nil {
			result.Failed++
			result.LastFailed = pageType + ": encode config: " + err.Error()
			slog.Error("admin secret migration: encode config failed", "pageType", pageType, "err", err)
			return
		}
		if err := conn.Table("page_config").Where("page_type = ?", pageType).Update("config", string(encoded)).Error; err != nil {
			result.Failed++
			result.LastFailed = pageType + ": update config: " + err.Error()
			slog.Error("admin secret migration: update config failed", "pageType", pageType, "err", err)
		}
	}

	migrate(pageConfig.EmailSettings, func(config map[string]any) error {
		plain, _ := config["smtpPassword"].(string)
		encrypted, _ := config["smtpPasswordEncrypted"].(string)
		if plain == "" || encrypted != "" {
			return nil
		}
		sealed, err := securestore.EncryptPurpose(plain, securestore.MailSmtpPasswordPurpose)
		if err != nil {
			return err
		}
		config["smtpPasswordEncrypted"] = sealed
		delete(config, "smtpPassword")
		result.MailEncrypted++
		slog.Info("admin secret migration: encrypted mail smtp password")
		return nil
	})

	migrate(pageConfig.StorageSettingsPage, func(config map[string]any) error {
		for _, pair := range []struct {
			plainKey, encryptedKey, purpose string
		}{
			{"accessKey", "accessKeyEncrypted", securestore.StorageAccessKeyPurpose},
			{"secretKey", "secretKeyEncrypted", securestore.StorageSecretKeyPurpose},
		} {
			plain, _ := config[pair.plainKey].(string)
			encrypted, _ := config[pair.encryptedKey].(string)
			if plain == "" || encrypted != "" {
				continue
			}
			sealed, err := securestore.EncryptPurpose(plain, pair.purpose)
			if err != nil {
				return err
			}
			config[pair.encryptedKey] = sealed
			delete(config, pair.plainKey)
			result.StorageKeys++
		}
		if result.StorageKeys > 0 {
			slog.Info("admin secret migration: encrypted storage credentials", "count", result.StorageKeys)
		}
		return nil
	})

	migrate(pageConfig.HttpNotify, func(config map[string]any) error {
		endpoints, ok := config["endpoints"].([]any)
		if !ok {
			return nil
		}
		changed := false
		for _, raw := range endpoints {
			ep, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			plain, _ := ep["secret"].(string)
			encrypted, _ := ep["secretEncrypted"].(string)
			if plain == "" || encrypted != "" {
				continue
			}
			sealed, err := securestore.EncryptPurpose(plain, securestore.HttpNotifySecretPurpose)
			if err != nil {
				return err
			}
			ep["secretEncrypted"] = sealed
			delete(ep, "secret")
			changed = true
			result.NotifySecrets++
		}
		if changed {
			slog.Info("admin secret migration: encrypted http notify secrets", "count", result.NotifySecrets)
		}
		return nil
	})

	return result
}
