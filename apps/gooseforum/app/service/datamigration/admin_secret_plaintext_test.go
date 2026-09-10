package datamigration

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// seedLegacyPageConfig 插入 v25 之前的明文配置行（page_config 表）。
func seedLegacyPageConfig(t *testing.T, conn *gorm.DB, pageType, config string) {
	t.Helper()
	if err := conn.Exec(
		`INSERT INTO page_config (page_type, config, created_at, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		pageType, config,
	).Error; err != nil {
		t.Fatalf("seed %s config: %v", pageType, err)
	}
}

func readPageConfig(t *testing.T, conn *gorm.DB, pageType string) map[string]any {
	t.Helper()
	var config string
	if err := conn.Raw(`SELECT config FROM page_config WHERE page_type = ?`, pageType).Scan(&config).Error; err != nil {
		t.Fatalf("read %s config: %v", pageType, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(config), &decoded); err != nil {
		t.Fatalf("decode %s config: %v", pageType, err)
	}
	return decoded
}

func TestMigrateAdminSecretPlaintextEncryptsAtRest(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-admin-secret?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE page_config (
		id integer primary key autoincrement,
		page_type varchar(128) not null default '',
		config text,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create page_config: %v", err)
	}

	seedLegacyPageConfig(t, conn, "emailSetting", `{"enableMail":true,"smtpHost":"smtp.example.com","smtpPort":465,"useSSL":true,"smtpUsername":"mailer","smtpPassword":"legacy-smtp-secret","fromName":"站务","fromEmail":"noreply@example.com"}`)
	seedLegacyPageConfig(t, conn, "storageSettings", `{"provider":"s3","endpoint":"https://s3.example.com","bucket":"b","region":"r","bucketLookup":"auto","secure":true,"accessKey":"legacy-ak","secretKey":"legacy-sk","publicUrlPrefix":""}`)
	seedLegacyPageConfig(t, conn, "httpNotify", `{"enabled":true,"endpoints":[{"id":"ep1","name":"webhook","enabled":true,"url":"https://hook.example.com","secret":"legacy-webhook-secret","events":["topic.created"],"timeoutSeconds":5,"failureCount":0,"lastError":"","abnormalTerminated":false}]}`)
	// 已加密的配置行应跳过（幂等）。
	seedLegacyPageConfig(t, conn, "onesystemSettings", `{"cookieEncrypted":"already-sealed"}`)

	result := MigrateAdminSecretPlaintextWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("migration failed: %+v", result)
	}
	if result.MailEncrypted != 1 || result.StorageKeys != 2 || result.NotifySecrets != 1 {
		t.Fatalf("migration counts = mail %d storage %d notify %d, want 1/2/1",
			result.MailEncrypted, result.StorageKeys, result.NotifySecrets)
	}

	// 邮件：明文清空，密文可解密回原文。
	mail := readPageConfig(t, conn, "emailSetting")
	if plain, _ := mail["smtpPassword"].(string); plain != "" {
		t.Fatalf("mail smtpPassword still present: %q", plain)
	}
	sealed, _ := mail["smtpPasswordEncrypted"].(string)
	if sealed == "" {
		t.Fatal("mail smtpPasswordEncrypted missing after migration")
	}
	if plain, err := securestore.DecryptPurpose(sealed, securestore.MailSmtpPasswordPurpose); err != nil || plain != "legacy-smtp-secret" {
		t.Fatalf("mail decrypt = %q, err %v", plain, err)
	}

	// 存储：两个凭据均加密。
	storage := readPageConfig(t, conn, "storageSettings")
	if ak, _ := storage["accessKey"].(string); ak != "" {
		t.Fatalf("storage accessKey still present: %q", ak)
	}
	akSealed, _ := storage["accessKeyEncrypted"].(string)
	skSealed, _ := storage["secretKeyEncrypted"].(string)
	if akSealed == "" || skSealed == "" {
		t.Fatal("storage encrypted fields missing after migration")
	}
	if plain, err := securestore.DecryptPurpose(akSealed, securestore.StorageAccessKeyPurpose); err != nil || plain != "legacy-ak" {
		t.Fatalf("accessKey decrypt = %q, err %v", plain, err)
	}
	if plain, err := securestore.DecryptPurpose(skSealed, securestore.StorageSecretKeyPurpose); err != nil || plain != "legacy-sk" {
		t.Fatalf("secretKey decrypt = %q, err %v", plain, err)
	}

	// 通知：端点明文清空、密文可解密。
	notify := readPageConfig(t, conn, "httpNotify")
	endpoints, _ := notify["endpoints"].([]any)
	ep := endpoints[0].(map[string]any)
	if secret, _ := ep["secret"].(string); secret != "" {
		t.Fatalf("notify secret still present: %q", secret)
	}
	epSealed, _ := ep["secretEncrypted"].(string)
	if epSealed == "" {
		t.Fatal("notify secretEncrypted missing after migration")
	}
	if plain, err := securestore.DecryptPurpose(epSealed, securestore.HttpNotifySecretPurpose); err != nil || plain != "legacy-webhook-secret" {
		t.Fatalf("notify decrypt = %q, err %v", plain, err)
	}

	// 幂等：再次运行不再改动（计数为 0）。
	second := MigrateAdminSecretPlaintextWithDB(conn)
	if second.Failed > 0 || second.MailEncrypted != 0 || second.StorageKeys != 0 || second.NotifySecrets != 0 {
		t.Fatalf("second migration not idempotent: %+v", second)
	}
}

func TestMigrateAdminSecretPlaintextSkipsAlreadyEncrypted(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open("file:migration-admin-secret-enc?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := conn.Exec(`CREATE TABLE page_config (
		id integer primary key autoincrement,
		page_type varchar(128) not null default '',
		config text,
		created_at datetime,
		updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create page_config: %v", err)
	}
	sealed, err := securestore.EncryptPurpose("modern-secret", securestore.MailSmtpPasswordPurpose)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	seedLegacyPageConfig(t, conn, "emailSetting",
		`{"enableMail":true,"smtpHost":"smtp.example.com","smtpPort":465,"useSSL":false,"smtpUsername":"u","smtpPasswordEncrypted":"`+sealed+`","fromName":"站务","fromEmail":"noreply@example.com"}`)

	result := MigrateAdminSecretPlaintextWithDB(conn)
	if result.Failed > 0 || result.MailEncrypted != 0 {
		t.Fatalf("already-encrypted config was re-migrated: %+v", result)
	}
	mail := readPageConfig(t, conn, "emailSetting")
	if got, _ := mail["smtpPasswordEncrypted"].(string); got != sealed {
		t.Fatalf("smtpPasswordEncrypted changed: %q", got)
	}
	if strings.Contains(readRawConfig(t, conn, "emailSetting"), "modern-secret") {
		t.Fatal("plaintext secret leaked into stored config")
	}
}

func readRawConfig(t *testing.T, conn *gorm.DB, pageType string) string {
	t.Helper()
	var config string
	if err := conn.Raw(`SELECT config FROM page_config WHERE page_type = ?`, pageType).Scan(&config).Error; err != nil {
		t.Fatalf("read %s config: %v", pageType, err)
	}
	return config
}
