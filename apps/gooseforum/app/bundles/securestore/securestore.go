// Package securestore encrypts sensitive values at rest with AES-256-GCM.
// The encryption key is derived from the app signing key so no extra secret
// has to be provisioned; the nonce travels with the ciphertext
// (base64(nonce||ciphertext)).
package securestore

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

// derivationLabel is bound into the HMAC so a key derived for TOTP secrets
// can never be reused for another purpose even if the signing key leaks.
const derivationLabel = "yourtj-totp-secret"

var (
	keyOnce sync.Once
	key     []byte
	keyErr  error
)

// Encrypt encrypts plaintext and returns base64(nonce||ciphertext).
func Encrypt(plaintext string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("securestore: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("securestore: create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("securestore: random nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func Decrypt(encoded string) (string, error) {
	key, err := deriveKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("securestore: create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("securestore: create gcm: %w", err)
	}
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("securestore: decode payload: %w", err)
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("securestore: payload too short")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("securestore: decrypt: %w", err)
	}
	return string(plaintext), nil
}

// deriveKey derives a 32-byte AES key from the app signing key:
// HMAC-SHA256(signingKey, "yourtj-totp-secret"). The result is cached for the
// process lifetime (sync.Once) so its lifecycle aligns with jwtopt's JWT
// signing key: a runtime config reload of app.signingKey cannot silently
// invalidate TOTP secrets mid-flight.
func deriveKey() ([]byte, error) {
	keyOnce.Do(func() {
		// 与 tokenservice 共享 jwtopt 的 fail-closed 谓词（SigningKeyProblemFor）：
		// serve 启动守卫已拒绝空值/内置默认/占位符密钥，这里作为纵深防御再拦一道——
		// 即使绕过启动守卫（未来子命令、嵌入式调用、测试 harness），也不会以公开
		// 已知的密钥派生 TOTP 加密密钥（issue #106 根因之一）。
		key, keyErr = deriveKeyFrom(preferences.GetString("app.signingKey"))
	})
	return key, keyErr
}

// deriveKeyFrom derives the 32-byte AES key for a candidate signing key,
// applying the same fail-closed policy as the JWT signing path
// (jwtopt.SigningKeyProblemFor). It is extracted from deriveKey so the
// weak-key rejection is directly testable without disturbing the process-wide
// sync.Once cache.
func deriveKeyFrom(signingKey string) ([]byte, error) {
	if reason := jwtopt.SigningKeyProblemFor(signingKey); reason != "" {
		return nil, fmt.Errorf("securestore: %s", reason)
	}
	mac := hmac.New(sha256.New, []byte(signingKey))
	if _, err := mac.Write([]byte(derivationLabel)); err != nil {
		return nil, fmt.Errorf("securestore: derive key: %w", err)
	}
	return mac.Sum(nil), nil
}

// Pepper derives a 32-byte HMAC key bound to the app signing key and a
// purpose label, cached per label. It lets callers (e.g. recovery-code
// hashing) use a server-side pepper that never enters the database, so a DB
// leak alone does not allow offline brute force.
func Pepper(label string) ([]byte, error) {
	base, err := deriveKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, base)
	if _, err := mac.Write([]byte(label)); err != nil {
		return nil, fmt.Errorf("securestore: derive pepper: %w", err)
	}
	return mac.Sum(nil), nil
}
