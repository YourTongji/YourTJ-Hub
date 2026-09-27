package appleauthservice

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAppleBindingsRequireExplicitAccountAndProtectRefreshToken(t *testing.T) {
	preferences.Set("app.signingKey", "apple-auth-test-signing-key-with-32-random-characters")
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userOAuth.Entity{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{81001, 81002} {
		if err := conn.Create(&users.EntityComplete{Id: id, Username: "apple-user-" + strconv.FormatUint(id, 10), Email: ""}).Error; err != nil {
			t.Fatal(err)
		}
	}
	proof := identity{Subject: "private-apple-sub", RefreshToken: "secret-refresh-token"}
	if _, err := storeIdentity(context.Background(), conn, proof, 0); !errors.Is(err, ErrNoBinding) {
		t.Fatalf("unbound login: %v", err)
	}
	user, err := storeIdentity(context.Background(), conn, proof, 81001)
	if err != nil || user.Id != 81001 {
		t.Fatalf("bind: %v", err)
	}
	var binding userOAuth.Entity
	if err := conn.Where("provider = ? AND user_id = ?", Provider, 81001).First(&binding).Error; err != nil {
		t.Fatal(err)
	}
	if binding.AppleRefreshToken == "" || strings.Contains(binding.AppleRefreshToken, proof.RefreshToken) {
		t.Fatal("refresh token not encrypted")
	}
	decrypted, err := securestore.DecryptPurpose(binding.AppleRefreshToken, refreshPurpose(81001))
	if err != nil || decrypted != proof.RefreshToken {
		t.Fatal("refresh token did not roundtrip")
	}
	encoded, err := json.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), binding.AppleRefreshToken) {
		t.Fatal("JSON disclosed private token")
	}
	if _, err := securestore.DecryptPurpose(binding.AppleRefreshToken, refreshPurpose(81002)); err == nil {
		t.Fatal("ciphertext can cross account boundary")
	}
	if _, err := storeIdentity(context.Background(), conn, proof, 81002); !errors.Is(err, ErrAlreadyBound) {
		t.Fatalf("cross-account rebinding: %v", err)
	}
	if _, err := storeIdentity(context.Background(), conn, identity{Subject: "other-apple", RefreshToken: "new-secret"}, 81001); !errors.Is(err, ErrAlreadyBound) {
		t.Fatalf("replace existing binding: %v", err)
	}
	user, err = storeIdentity(context.Background(), conn, proof, 0)
	if err != nil || user.Id != 81001 {
		t.Fatalf("sub-based login: %v", err)
	}
	if err := conn.Model(&users.EntityComplete{}).Where("id = ?", 81001).Update("is_frozen", users.StatusFrozen).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := storeIdentity(context.Background(), conn, proof, 0); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("frozen login: %v", err)
	}
	if err := conn.Delete(&users.EntityComplete{}, 81001).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := storeIdentity(context.Background(), conn, proof, 0); !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("closed account login: %v", err)
	}
}

func TestAppleRevocationAndAccountClosureAreRetryable(t *testing.T) {
	preferences.Set("app.signingKey", "apple-auth-test-signing-key-with-32-random-characters")
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&users.EntityComplete{}, &users.PrivateNoteEntity{}, &users.BlockEntity{}, &userOAuth.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: 82001, Username: "apple-revoke-owner"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := storeIdentity(t.Context(), conn, identity{Subject: "revocable-apple", RefreshToken: "encrypted-grant"}, 82001); err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var fail atomic.Bool
	fail.Store(true)
	var revoked atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.URL.Path != "/auth/revoke" || r.Form.Get("token") != "encrypted-grant" || r.Form.Get("token_type_hint") != "refresh_token" {
			t.Error("invalid revoke request")
		}
		if fail.Load() {
			w.WriteHeader(503)
			return
		}
		revoked.Add(1)
		w.WriteHeader(200)
	}))
	defer server.Close()
	client := newClient(Config{ClientID: "tj.yourtj.forumApp", TeamID: "TEAM", KeyID: "KEY", PrivateKey: key}, server.Client())
	client.baseURL = server.URL
	previous := currentClient.Swap(client)
	t.Cleanup(func() { currentClient.Store(previous) })
	closeAccount := func() error {
		return conn.Transaction(func(tx *gorm.DB) error {
			if err := RevokeAndDeleteTx(t.Context(), tx, 82001); err != nil {
				return err
			}
			return users.CloseAccountTx(tx, 82001)
		})
	}
	if err := closeAccount(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unavailable revoke: %v", err)
	}
	var count int64
	conn.Model(&users.EntityComplete{}).Where("id = ?", 82001).Count(&count)
	if count != 1 {
		t.Fatal("failed revoke closed the account")
	}
	conn.Model(&userOAuth.Entity{}).Where("user_id = ?", 82001).Count(&count)
	if count != 1 {
		t.Fatal("failed revoke discarded the retry credential")
	}
	fail.Store(false)
	if err := closeAccount(); err != nil {
		t.Fatal(err)
	}
	conn.Model(&users.EntityComplete{}).Where("id = ?", 82001).Count(&count)
	if count != 0 {
		t.Fatal("account still active after closure")
	}
	conn.Unscoped().Model(&userOAuth.Entity{}).Where("user_id = ?", 82001).Count(&count)
	if count != 0 || revoked.Load() != 1 {
		t.Fatalf("revoked grant retained: rows=%d calls=%d", count, revoked.Load())
	}
}
