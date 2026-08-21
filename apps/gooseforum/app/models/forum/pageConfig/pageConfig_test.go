package pageConfig

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestOneSystemSettingsConfigJSONOmitsCiphertext 验证领域结构不随 JSON 序列化泄漏密文，
// 落库形状保留密文（review MEDIUM）。
func TestOneSystemSettingsConfigJSONOmitsCiphertext(t *testing.T) {
	const secret = "cipher-secret-abc"

	// 领域结构 json:"-"：任何 JSON 序列化都不得带出密文。
	cfg := OneSystemSettingsConfig{CookieEncrypted: secret}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if strings.Contains(string(b), secret) {
		t.Errorf("OneSystemSettingsConfig JSON leaked ciphertext: %s", b)
	}

	// 落库形状保留密文，持久化（jsonopt.Encode）不受影响。
	storage := OneSystemSettingsStorage{CookieEncrypted: secret}
	sb, err := json.Marshal(storage)
	if err != nil {
		t.Fatalf("marshal storage: %v", err)
	}
	if !strings.Contains(string(sb), secret) {
		t.Errorf("OneSystemSettingsStorage JSON should persist ciphertext: %s", sb)
	}

	// round-trip：storage -> config 保留密文。
	if got := storage.ToConfig().CookieEncrypted; got != secret {
		t.Errorf("ToConfig = %q, want %q", got, secret)
	}

	// 存量兼容：旧落库 JSON（{"cookieEncrypted":...}）仍能被 OneSystemSettingsStorage 反序列化，
	// 避免 DTO 分离后读取旧数据失败（json 标签保持不变）。
	var legacy OneSystemSettingsStorage
	if err := json.Unmarshal([]byte(`{"cookieEncrypted":"legacy-cipher"}`), &legacy); err != nil {
		t.Fatalf("unmarshal legacy storage: %v", err)
	}
	if legacy.CookieEncrypted != "legacy-cipher" {
		t.Errorf("legacy storage CookieEncrypted = %q, want legacy-cipher", legacy.CookieEncrypted)
	}
}

// TestSecretStorageShapesOmitCiphertextFromDomainJSON 验证 issue #324 S1-S3 的三个
// 落库形状：领域结构 json:"-" 不随 JSON 序列化泄漏明文/密文；落库形状保留密文；
// ToConfig 优先取 *Encrypted 字段、兼容迁移前明文字段。
func TestSecretStorageShapesOmitCiphertextFromDomainJSON(t *testing.T) {
	t.Run("mail", func(t *testing.T) {
		const secret = "smtp-secret-abc"
		cfg := MailSettingsConfig{SmtpPassword: secret}
		b, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal mail config: %v", err)
		}
		if strings.Contains(string(b), secret) {
			t.Errorf("MailSettingsConfig JSON leaked password: %s", b)
		}

		storage := MailSettingsStorage{SmtpPasswordEncrypted: secret}
		sb, _ := json.Marshal(storage)
		if !strings.Contains(string(sb), secret) {
			t.Errorf("MailSettingsStorage JSON should persist ciphertext: %s", sb)
		}
		if got := storage.ToConfig().SmtpPassword; got != secret {
			t.Errorf("ToConfig = %q, want %q", got, secret)
		}
		// 迁移前明文兼容：无密文时取明文。
		legacy := MailSettingsStorage{SmtpPassword: "legacy-plain"}
		if got := legacy.ToConfig().SmtpPassword; got != "legacy-plain" {
			t.Errorf("legacy ToConfig = %q, want legacy-plain", got)
		}
		// 回显视图：密文/明文任一存在即 configured。
		if !storage.ToView().SmtpPasswordConfigured || !legacy.ToView().SmtpPasswordConfigured {
			t.Errorf("ToView configured = %v/%v, want true/true", storage.ToView().SmtpPasswordConfigured, legacy.ToView().SmtpPasswordConfigured)
		}
	})

	t.Run("storage", func(t *testing.T) {
		const ak = "access-key-abc"
		const sk = "secret-key-abc"
		cfg := StorageSettings{AccessKey: ak, SecretKey: sk}
		b, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal storage config: %v", err)
		}
		if strings.Contains(string(b), ak) || strings.Contains(string(b), sk) {
			t.Errorf("StorageSettings JSON leaked credentials: %s", b)
		}

		storage := StorageSettingsStorage{AccessKeyEncrypted: ak, SecretKeyEncrypted: sk}
		sb, _ := json.Marshal(storage)
		if !strings.Contains(string(sb), ak) || !strings.Contains(string(sb), sk) {
			t.Errorf("StorageSettingsStorage JSON should persist ciphertext: %s", sb)
		}
		got := storage.ToConfig()
		if got.AccessKey != ak || got.SecretKey != sk {
			t.Errorf("ToConfig = %q/%q, want %q/%q", got.AccessKey, got.SecretKey, ak, sk)
		}
		legacy := StorageSettingsStorage{AccessKey: "legacy-ak", SecretKey: "legacy-sk"}
		gotLegacy := legacy.ToConfig()
		if gotLegacy.AccessKey != "legacy-ak" || gotLegacy.SecretKey != "legacy-sk" {
			t.Errorf("legacy ToConfig = %q/%q, want legacy-ak/legacy-sk", gotLegacy.AccessKey, gotLegacy.SecretKey)
		}
		if !storage.ToView().AccessKeyConfigured || !storage.ToView().SecretKeyConfigured {
			t.Errorf("ToView configured = %v/%v, want true/true", storage.ToView().AccessKeyConfigured, storage.ToView().SecretKeyConfigured)
		}
	})

	t.Run("http notify", func(t *testing.T) {
		const secret = "webhook-secret-abc"
		cfg := HttpNotifyConfig{Endpoints: []HttpNotifyEndpoint{{Id: "e1", Secret: secret}}}
		b, err := json.Marshal(cfg)
		if err != nil {
			t.Fatalf("marshal notify config: %v", err)
		}
		if strings.Contains(string(b), secret) {
			t.Errorf("HttpNotifyConfig JSON leaked secret: %s", b)
		}

		storage := HttpNotifyStorageConfig{Endpoints: []HttpNotifyStorageEndpoint{{Id: "e1", SecretEncrypted: secret}}}
		sb, _ := json.Marshal(storage)
		if !strings.Contains(string(sb), secret) {
			t.Errorf("HttpNotifyStorageConfig JSON should persist ciphertext: %s", sb)
		}
		if got := storage.ToConfig().Endpoints[0].Secret; got != secret {
			t.Errorf("ToConfig = %q, want %q", got, secret)
		}
		legacy := HttpNotifyStorageConfig{Endpoints: []HttpNotifyStorageEndpoint{{Id: "e1", Secret: "legacy-plain"}}}
		if got := legacy.ToConfig().Endpoints[0].Secret; got != "legacy-plain" {
			t.Errorf("legacy ToConfig = %q, want legacy-plain", got)
		}
		if !storage.ToView().Endpoints[0].SecretConfigured || !legacy.ToView().Endpoints[0].SecretConfigured {
			t.Errorf("ToView configured = %v/%v, want true/true", storage.ToView().Endpoints[0].SecretConfigured, legacy.ToView().Endpoints[0].SecretConfigured)
		}
	})
}
