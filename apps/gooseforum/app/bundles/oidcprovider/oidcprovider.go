// Package oidcprovider manages the persistent RSA signing key used by the
// built-in OIDC provider for RS256 ID tokens and the JWKS endpoint.
//
// Key load order: inline PEM (oidc.signing_key) -> key file
// (oidc.signing_key_file) -> generate RSA 2048 and atomically persist with
// 0600 permissions to the key file path. The key is intentionally independent
// from app.signingKey (HS256 forum JWT); never reuse it for other purposes.
package oidcprovider

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

const (
	keyBits = 2048
	// defaultKeyFile is relative to the working directory (storage root).
	defaultKeyFile = "./storage/oidc/signing_key.pem"
)

// KeyManager holds the loaded signing key and its stable key ID.
type KeyManager struct {
	key *rsa.PrivateKey
	kid string
}

// Load resolves the signing key following the documented load order.
func Load() (*KeyManager, error) {
	keyFile := preferences.GetString("oidc.signing_key_file", defaultKeyFile)

	if inline := preferences.GetString("oidc.signing_key", ""); inline != "" {
		key, err := parsePrivateKeyPEM(inline)
		if err != nil {
			return nil, fmt.Errorf("oidcprovider: parse oidc.signing_key: %w", err)
		}
		return newManager(key), nil
	}

	data, err := os.ReadFile(keyFile)
	if err == nil {
		key, perr := parsePrivateKeyPEM(string(data))
		if perr != nil {
			return nil, fmt.Errorf("oidcprovider: parse key file %s: %w", keyFile, perr)
		}
		return newManager(key), nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("oidcprovider: read key file %s: %w", keyFile, err)
	}

	key, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, fmt.Errorf("oidcprovider: generate rsa key: %w", err)
	}
	if err := persistKeyFile(keyFile, key); err != nil {
		return nil, fmt.Errorf("oidcprovider: persist key file: %w", err)
	}
	return newManager(key), nil
}

// PrivateKey returns the RSA private key (signing only; never expose publicly).
func (km *KeyManager) PrivateKey() *rsa.PrivateKey {
	return km.key
}

// PublicKey returns the RSA public key.
func (km *KeyManager) PublicKey() *rsa.PublicKey {
	return &km.key.PublicKey
}

// KeyID returns the stable key ID (SHA-256 fingerprint of the public key).
func (km *KeyManager) KeyID() string {
	return km.kid
}

// JWKS marshals the public key set as a JSON Web Key Set. The private key is
// never included.
func (km *KeyManager) JWKS() ([]byte, error) {
	set := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{
		Key:       km.PublicKey(),
		KeyID:     km.kid,
		Algorithm: string(jose.RS256),
		Use:       "sig",
	}}}
	return json.Marshal(set)
}

func newManager(key *rsa.PrivateKey) *KeyManager {
	der := x509.MarshalPKCS1PublicKey(&key.PublicKey)
	sum := sha256.Sum256(der)
	return &KeyManager{key: key, kid: hex.EncodeToString(sum[:16])}
}

func parsePrivateKeyPEM(data string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, errors.New("no PEM block found")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("unsupported private key format")
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return key, nil
}

// persistKeyFile atomically writes the PKCS1 PEM key with 0600 permissions.
func persistKeyFile(path string, key *rsa.PrivateKey) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	block := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, block, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
