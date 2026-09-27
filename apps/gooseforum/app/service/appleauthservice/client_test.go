package appleauthservice

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/golang-jwt/jwt/v5"
)

func TestAppleExchangeVerifiesIdentityAndSingleUseCode(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const nonce = "0123456789abcdefghijklmnopqrstuvwxyz-APPLE-NONCE"
	digest := sha256.Sum256([]byte(nonce))
	claims := jwt.MapClaims{"iss": appleIssuer, "aud": "tj.yourtj.forumApp", "sub": "apple-user", "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "nonce": hex.EncodeToString(digest[:])}
	sign := func(values jwt.MapClaims) string {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, values)
		token.Header["kid"] = "apple-test"
		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return signed
	}
	identity := sign(claims)
	var exchanges atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/keys":
			_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &key.PublicKey, KeyID: "apple-test", Algorithm: "RS256", Use: "sig"}}})
		case "/auth/token":
			if err := r.ParseForm(); err != nil {
				t.Error(err)
			}
			if r.Form.Get("client_id") != "tj.yourtj.forumApp" || r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "one-use-code" {
				t.Error("unexpected token request")
			}
			_, err := jwt.Parse(r.Form.Get("client_secret"), func(token *jwt.Token) (any, error) { return &privateKey.PublicKey, nil }, jwt.WithValidMethods([]string{"ES256"}), jwt.WithAudience(appleIssuer), jwt.WithIssuer("TEAM123"), jwt.WithSubject("tj.yourtj.forumApp"), jwt.WithExpirationRequired())
			if err != nil {
				t.Errorf("client secret: %v", err)
			}
			if exchanges.Add(1) > 1 {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"id_token": identity, "refresh_token": "private-refresh-token"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	client := newClient(Config{ClientID: "tj.yourtj.forumApp", TeamID: "TEAM123", KeyID: "key-test", PrivateKey: privateKey}, server.Client())
	client.baseURL = server.URL
	for _, field := range []string{"iss", "aud", "sub", "exp", "nonce"} {
		t.Run("reject_"+field, func(t *testing.T) {
			invalid := jwt.MapClaims{}
			for k, v := range claims {
				invalid[k] = v
			}
			invalid[field] = "invalid"
			if field == "sub" {
				invalid[field] = ""
			}
			if field == "exp" {
				invalid[field] = time.Now().Add(-time.Hour).Unix()
			}
			if _, err := client.Authenticate(context.Background(), "one-use-code", sign(invalid), nonce); err == nil {
				t.Fatal("accepted invalid identity")
			}
		})
	}
	if exchanges.Load() != 0 {
		t.Fatal("invalid identity consumed authorization code")
	}
	verified, err := client.Authenticate(context.Background(), "one-use-code", identity, nonce)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Subject != "apple-user" || verified.RefreshToken != "private-refresh-token" {
		t.Fatal("wrong identity or credentials")
	}
	if _, err := client.Authenticate(context.Background(), "one-use-code", identity, nonce); err == nil {
		t.Fatal("accepted replayed authorization code")
	}
}
