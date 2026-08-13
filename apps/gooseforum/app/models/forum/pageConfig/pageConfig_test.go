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
