package securestore

import (
	"encoding/base64"
	"os"
	"os/exec"
	"testing"

	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := "JBSWY3DPEHPK3PXP"
	encoded, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encoded == plaintext {
		t.Fatal("Encrypt() returned plaintext")
	}
	got, err := Decrypt(encoded)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	first, err := Encrypt("same-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	second, err := Encrypt("same-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if first == second {
		t.Fatal("Encrypt() produced identical ciphertext for same plaintext")
	}
}

func TestDecryptRejectsCorruptedCiphertext(t *testing.T) {
	encoded, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if _, err := Decrypt(encoded + "AAAA"); err == nil {
		t.Fatal("Decrypt() accepted corrupted ciphertext")
	}
}

func TestDecryptRejectsGarbageInput(t *testing.T) {
	if _, err := Decrypt("not-base64!!!"); err == nil {
		t.Fatal("Decrypt() accepted non-base64 input")
	}
	if _, err := Decrypt(""); err == nil {
		t.Fatal("Decrypt() accepted empty input")
	}
}

func TestDecryptRejectsTamperedPayload(t *testing.T) {
	encoded, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	raw[len(raw)-1] ^= 0x01
	tampered := base64.StdEncoding.EncodeToString(raw)
	if _, err := Decrypt(tampered); err == nil {
		t.Fatal("Decrypt() accepted tampered payload")
	}
}

// TestDeriveKeyFromRejectsWeakSigningKeys 验证 securestore 与 tokenservice 共享
// jwtopt 的 fail-closed 谓词（issue #106 point 1 的纵深防御）：空/空白密钥、
// 内置默认值与部署模板占位符一律拒绝派生 TOTP 加密密钥，防止以公开已知密钥
// 派生 AES-GCM key。
func TestDeriveKeyFromRejectsWeakSigningKeys(t *testing.T) {
	cases := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "whitespace", key: "   \t\n "},
		{name: "built-in default", key: jwtopt.DefaultSigningKey},
		{name: "deploy placeholder", key: "REPLACE_SIGNING_KEY"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := deriveKeyFrom(tt.key); err == nil {
				t.Fatalf("deriveKeyFrom(%q) succeeded, want fail-closed error", tt.key)
			}
		})
	}
}

func TestDeriveKeyFromStrongKey(t *testing.T) {
	key, err := deriveKeyFrom("a-strong-random-test-key")
	if err != nil {
		t.Fatalf("deriveKeyFrom(strong) error = %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("deriveKeyFrom key length = %d, want 32", len(key))
	}
	again, err := deriveKeyFrom("a-strong-random-test-key")
	if err != nil {
		t.Fatalf("deriveKeyFrom(strong) #2 error = %v", err)
	}
	if string(key) != string(again) {
		t.Fatal("deriveKeyFrom must be deterministic for a given key")
	}
}

// encryptWeakKeyHelperEnv gates the child-process branch below. It is only
// consulted inside TestEncryptWeakKeyHelperProcess, which runs only when the
// parent test re-execs the test binary with `-test.run` pointing at it — an
// unrelated GO_WANT_* variable in the environment cannot make the main test or
// any other test take the child branch.
const encryptWeakKeyHelperEnv = "YOURTJ_ISSUE106_ENCRYPT_WEAK_HELPER"

// TestEncryptFailsUnderWeakSigningKey 在子进程中验证：当 app.signingKey 是弱密钥
// 时 Encrypt 必须失败（TOTP 加密密钥无法派生）。子进程保证 keyOnce 是全新状态，
// 从而真正走到 deriveKey 的 fail-closed 分支，不受本进程已缓存有效密钥的干扰。
func TestEncryptFailsUnderWeakSigningKey(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestEncryptWeakKeyHelperProcess$")
	cmd.Env = append(os.Environ(), encryptWeakKeyHelperEnv+"=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("Encrypt under weak signing key must fail cleanly (child exit 0), err=%v out=%s", err, out)
	}
}

// TestEncryptWeakKeyHelperProcess is the child-process entry: with the helper
// env var set it configures a weak signing key and asserts Encrypt fails. When
// run as part of the normal test suite (no env var) it is a silent no-op.
func TestEncryptWeakKeyHelperProcess(t *testing.T) {
	if os.Getenv(encryptWeakKeyHelperEnv) != "1" {
		return
	}
	preferences.Set("app.signingKey", jwtopt.DefaultSigningKey)
	if _, err := Encrypt("secret"); err == nil {
		os.Exit(1) // 弱密钥下 Encrypt 成功 → 子进程非零退出 → 父进程断言失败
	}
	os.Exit(0)
}
