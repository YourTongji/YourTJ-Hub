package datamigration

import (
	"slices"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func openSecuritySettingsTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	// cache=shared 的进程内 DB 在包内测试间持久存在，连接名必须唯一，
	// 否则后续测试建表会撞已有表（admin_secret_plaintext_test.go 同约定）。
	conn, err := gorm.Open(sqlite.Open("file:migration-security-"+name+"?mode=memory&cache=shared"), &gorm.Config{})
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
	return conn
}

func securityStrings(t *testing.T, config map[string]any, key string) []string {
	t.Helper()
	raw, ok := config[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("%s = %#v, want array", key, raw)
	}
	out := make([]string, len(items))
	for index, item := range items {
		out[index], _ = item.(string)
	}
	return out
}

func TestEnsureSecuritySettingsDefaultsNoOpWhenNoTable(t *testing.T) {
	// 连上一个没有 page_config 表的库：迁移必须零副作用 no-op。
	conn, err := gorm.Open(sqlite.Open("file:migration-security-no-table-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed > 0 || result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want no-op", result.Updated, result.Skipped, result.Failed)
	}
}

func TestEnsureSecuritySettingsDefaultsSkipsWhenNoRow(t *testing.T) {
	conn := openSecuritySettingsTestDB(t, t.Name())
	// 库里有别的 page_config 行但没有 securitySettings 行：不写行，
	// 读取侧自然吃新默认（GetConfigByPageType 无行返回默认值）。
	seedLegacyPageConfig(t, conn, pageConfig.PostingSettings, `{"textControl":{}}`)

	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed > 0 || result.Updated || !result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want skipped", result.Updated, result.Skipped, result.Failed)
	}
	var count int64
	if err := conn.Table("page_config").Where("page_type = ?", pageConfig.SecuritySettings).Count(&count).Error; err != nil {
		t.Fatalf("count security settings rows: %v", err)
	}
	if count != 0 {
		t.Fatalf("security settings row count = %d after skip, want 0", count)
	}
}

func TestEnsureSecuritySettingsDefaultsFillsEmptyArrays(t *testing.T) {
	conn := openSecuritySettingsTestDB(t, t.Name())
	seedLegacyPageConfig(t, conn, pageConfig.SecuritySettings,
		`{"enableSignup":false,"reservedUsernames":[],"bannedUsernames":[],"sensitiveWords":[],"captchaRequired":false,"customFlag":"keep-me"}`)

	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed > 0 || !result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want updated", result.Updated, result.Skipped, result.Failed)
	}

	config := readPageConfig(t, conn, pageConfig.SecuritySettings)
	defaults := defaultconfig.GetDefaultSecuritySettingsConfig()
	if got := securityStrings(t, config, "reservedUsernames"); !slices.Equal(got, defaults.ReservedUsernames) {
		t.Fatalf("reservedUsernames = %v, want default %v", got, defaults.ReservedUsernames)
	}
	if got := securityStrings(t, config, "sensitiveWords"); !slices.Equal(got, defaults.SensitiveWords) {
		t.Fatalf("sensitiveWords = %v, want default %v", got, defaults.SensitiveWords)
	}
	// banned 默认为空：即使并入默认也应为空，且必须保持空（防误冻结）。
	if got := securityStrings(t, config, "bannedUsernames"); len(got) != 0 {
		t.Fatalf("bannedUsernames = %v, want empty (never written)", got)
	}
	// 非数组字段与未知字段原样保留。
	if enableSignup, _ := config["enableSignup"].(bool); enableSignup {
		t.Fatal("enableSignup flipped to true")
	}
	if captchaRequired, _ := config["captchaRequired"].(bool); captchaRequired {
		t.Fatal("captchaRequired flipped to true")
	}
	if flag, _ := config["customFlag"].(string); flag != "keep-me" {
		t.Fatalf("customFlag = %q, want keep-me (unknown fields preserved)", flag)
	}
}

func TestEnsureSecuritySettingsDefaultsFillsOnlyEmptyArrays(t *testing.T) {
	conn := openSecuritySettingsTestDB(t, t.Name())
	seedLegacyPageConfig(t, conn, pageConfig.SecuritySettings,
		`{"enableSignup":true,"reservedUsernames":["custom-admin"],"bannedUsernames":[],"sensitiveWords":[],"futureField":{"nested":true}}`)

	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed > 0 || !result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want updated", result.Updated, result.Skipped, result.Failed)
	}

	config := readPageConfig(t, conn, pageConfig.SecuritySettings)
	defaults := defaultconfig.GetDefaultSecuritySettingsConfig()
	// reserved 非空视为管理员已维护：原样保留，绝不覆盖。
	if got := securityStrings(t, config, "reservedUsernames"); !slices.Equal(got, []string{"custom-admin"}) {
		t.Fatalf("reservedUsernames = %v, want [custom-admin] preserved", got)
	}
	if got := securityStrings(t, config, "sensitiveWords"); !slices.Equal(got, defaults.SensitiveWords) {
		t.Fatalf("sensitiveWords = %v, want default %v", got, defaults.SensitiveWords)
	}
	future, ok := config["futureField"].(map[string]any)
	if !ok || future["nested"] != true {
		t.Fatalf("futureField = %#v, want preserved", config["futureField"])
	}
}

func TestEnsureSecuritySettingsDefaultsNeverWritesBanned(t *testing.T) {
	// banned 非空：迁移绝不触碰（并入默认 banned=[] 会把存量账号静默推入冻结）。
	conn := openSecuritySettingsTestDB(t, t.Name()+"-populated")
	seedLegacyPageConfig(t, conn, pageConfig.SecuritySettings,
		`{"reservedUsernames":[],"bannedUsernames":["x"],"sensitiveWords":[]}`)

	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed > 0 || !result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want updated", result.Updated, result.Skipped, result.Failed)
	}
	config := readPageConfig(t, conn, pageConfig.SecuritySettings)
	if got := securityStrings(t, config, "bannedUsernames"); !slices.Equal(got, []string{"x"}) {
		t.Fatalf("bannedUsernames = %v, want [x] untouched", got)
	}

	// banned 键缺失：迁移绝不创建该键（写入空 banned 与不写同形，直接验证键不出现）。
	connNoBanned := openSecuritySettingsTestDB(t, t.Name()+"-missing-key")
	seedLegacyPageConfig(t, connNoBanned, pageConfig.SecuritySettings,
		`{"reservedUsernames":[],"sensitiveWords":[]}`)
	result = EnsureSecuritySettingsDefaultsWithDB(connNoBanned)
	if result.Failed > 0 || !result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want updated", result.Updated, result.Skipped, result.Failed)
	}
	config = readPageConfig(t, connNoBanned, pageConfig.SecuritySettings)
	if _, exists := config["bannedUsernames"]; exists {
		t.Fatalf("bannedUsernames key created by migration: %#v", config["bannedUsernames"])
	}
}

func TestEnsureSecuritySettingsDefaultsIsIdempotent(t *testing.T) {
	conn := openSecuritySettingsTestDB(t, t.Name())
	seedLegacyPageConfig(t, conn, pageConfig.SecuritySettings,
		`{"reservedUsernames":[],"bannedUsernames":[],"sensitiveWords":[]}`)

	first := EnsureSecuritySettingsDefaultsWithDB(conn)
	if first.Failed > 0 || !first.Updated || first.Skipped {
		t.Fatalf("first migration = updated %v skipped %v failed %d, want updated", first.Updated, first.Skipped, first.Failed)
	}

	// 重跑：数组已非空，不再写入（Updated=false）。
	second := EnsureSecuritySettingsDefaultsWithDB(conn)
	if second.Failed > 0 || second.Updated || second.Skipped {
		t.Fatalf("second migration = updated %v skipped %v failed %d, want no-op", second.Updated, second.Skipped, second.Failed)
	}
	config := readPageConfig(t, conn, pageConfig.SecuritySettings)
	defaults := defaultconfig.GetDefaultSecuritySettingsConfig()
	if got := securityStrings(t, config, "reservedUsernames"); !slices.Equal(got, defaults.ReservedUsernames) {
		t.Fatalf("reservedUsernames = %v, want default %v", got, defaults.ReservedUsernames)
	}
	if got := securityStrings(t, config, "sensitiveWords"); !slices.Equal(got, defaults.SensitiveWords) {
		t.Fatalf("sensitiveWords = %v, want default %v", got, defaults.SensitiveWords)
	}
}

func TestEnsureSecuritySettingsDefaultsFailsOnMalformedConfig(t *testing.T) {
	conn := openSecuritySettingsTestDB(t, t.Name())
	seedLegacyPageConfig(t, conn, pageConfig.SecuritySettings, `{not-json`)

	result := EnsureSecuritySettingsDefaultsWithDB(conn)
	if result.Failed == 0 || result.Updated || result.Skipped {
		t.Fatalf("migration = updated %v skipped %v failed %d, want failed", result.Updated, result.Skipped, result.Failed)
	}
	if result.LastFailed == "" {
		t.Fatal("LastFailed empty on failure")
	}
}
