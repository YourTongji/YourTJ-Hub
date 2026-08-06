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

	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

// derivationLabel is bound into the HMAC so a key derived for TOTP secrets
// can never be reused for another purpose even if the signing key leaks.
const derivationLabel = "yourtj-totp-secret"

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
// HMAC-SHA256(signingKey, "yourtj-totp-secret").
func deriveKey() ([]byte, error) {
	signingKey := preferences.GetString("app.signingKey", "mq+ZeGafL+b1xdC0u9vSVg==")
	if signingKey == "" {
		return nil, errors.New("securestore: empty signing key")
	}
	mac := hmac.New(sha256.New, []byte(signingKey))
	if _, err := mac.Write([]byte(derivationLabel)); err != nil {
		return nil, fmt.Errorf("securestore: derive key: %w", err)
	}
	return mac.Sum(nil), nil
}
