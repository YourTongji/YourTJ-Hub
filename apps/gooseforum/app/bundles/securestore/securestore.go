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

	"github.com/leancodebox/GooseForum/app/bundles/preferences"
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
		// fail-closed: 不再回退到 jwtopt.DefaultSigningKey。serve 启动守卫已
		// 拒绝空值与已知坏值，进程进入到这里时 app.signingKey 必定有效；
		// 兜底再校验一次空值，防御性编程。
		signingKey := preferences.GetString("app.signingKey")
		if signingKey == "" {
			keyErr = errors.New("securestore: empty signing key")
			return
		}
		mac := hmac.New(sha256.New, []byte(signingKey))
		if _, err := mac.Write([]byte(derivationLabel)); err != nil {
			keyErr = fmt.Errorf("securestore: derive key: %w", err)
			return
		}
		key = mac.Sum(nil)
	})
	return key, keyErr
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
