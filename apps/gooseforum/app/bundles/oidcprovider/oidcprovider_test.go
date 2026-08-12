package oidcprovider

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	jose "github.com/go-jose/go-jose/v4"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

func TestLoadGeneratesAndPersistsKey(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "sub", "signing_key.pem")
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", keyFile)

	first, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if first.KeyID() == "" {
		t.Fatal("KeyID() = empty")
	}

	// 文件必须存在且权限为 0600
	info, err := os.Stat(keyFile)
	if err != nil {
		t.Fatalf("key file not persisted: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key file perm = %o, want 600", perm)
	}

	// 再次加载必须得到同一把密钥（重启后 kid 不变）
	second, err := Load()
	if err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	if second.KeyID() != first.KeyID() {
		t.Fatalf("kid changed across loads: %s vs %s", first.KeyID(), second.KeyID())
	}
	if !second.PublicKey().Equal(first.PublicKey()) {
		t.Fatal("public keys differ across loads")
	}
}

func TestLoadPrefersInlinePEM(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "signing_key.pem")
	generated, err := Load()
	if err != nil {
		t.Fatalf("seed Load() error = %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(generated.PrivateKey())
	inline := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}))

	preferences.Set("oidc.signing_key", inline)
	preferences.Set("oidc.signing_key_file", keyFile)

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() with inline PEM error = %v", err)
	}
	if !loaded.PublicKey().Equal(generated.PublicKey()) {
		t.Fatal("inline key does not match generated key")
	}
	if _, err := os.Stat(keyFile); !os.IsNotExist(err) {
		t.Fatalf("key file should not be created when inline key set, stat err = %v", err)
	}
}

func TestLoadRejectsGarbageInlinePEM(t *testing.T) {
	preferences.Set("oidc.signing_key", "not-a-pem")
	preferences.Set("oidc.signing_key_file", filepath.Join(t.TempDir(), "k.pem"))
	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want parse failure")
	}
}

func TestLoadReturnsNonMissingKeyFileReadError(t *testing.T) {
	keyPath := t.TempDir()
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", keyPath)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want key file read failure")
	}
	if !strings.Contains(err.Error(), "read key file") {
		t.Fatalf("Load() error = %q, want original read failure", err)
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("stat key path: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("key path was replaced instead of preserving the read error")
	}
}

func TestJWKSContainsPublicKeyOnly(t *testing.T) {
	dir := t.TempDir()
	preferences.Set("oidc.signing_key", "")
	preferences.Set("oidc.signing_key_file", filepath.Join(dir, "k.pem"))

	km, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	data, err := km.JWKS()
	if err != nil {
		t.Fatalf("JWKS() error = %v", err)
	}
	var set jose.JSONWebKeySet
	if err := json.Unmarshal(data, &set); err != nil {
		t.Fatalf("JWKS() invalid json: %v", err)
	}
	if len(set.Keys) != 1 {
		t.Fatalf("JWKS keys = %d, want 1", len(set.Keys))
	}
	key := set.Keys[0]
	if key.KeyID != km.KeyID() {
		t.Fatalf("kid = %q, want %q", key.KeyID, km.KeyID())
	}
	if key.Algorithm != string(jose.RS256) {
		t.Fatalf("alg = %q, want RS256", key.Algorithm)
	}
	if key.Use != "sig" {
		t.Fatalf("use = %q, want sig", key.Use)
	}
	if _, ok := key.Key.(*rsa.PublicKey); !ok {
		t.Fatalf("JWKS exposes non-public key type %T", key.Key)
	}
	if _, ok := key.Key.(*rsa.PrivateKey); ok {
		t.Fatal("JWKS leaks the private key")
	}
}
