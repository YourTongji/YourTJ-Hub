package campusservice

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestProviderOIDCVerification(t *testing.T) {
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	if e != nil {
		t.Fatal(e)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/protocol/openid-connect/certs":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{"kid": "test", "kty": "RSA", "use": "sig", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/protocol/openid-connect/userinfo":
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "subject", "userid": "student-42"})
		case "/v1/dc/user/student_info":
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "A00000", "data": []any{map[string]any{"userId": "student-42"}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	p := NewProvider(Config{ClientID: "client"})
	p.Base = server.URL
	p.Issuer = server.URL
	for _, scenario := range []string{"valid", "nonce", "issuer", "audience", "azp", "expired", "subject", "algorithm", "unsigned"} {
		t.Run(scenario, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": server.URL, "aud": "client", "sub": "subject", "nonce": "nonce", "azp": "client", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix()}
			switch scenario {
			case "nonce":
				claims["nonce"] = "other"
			case "issuer":
				claims["iss"] = "https://evil.invalid"
			case "audience":
				claims["aud"] = "other"
			case "azp":
				claims["azp"] = "other"
			case "expired":
				claims["exp"] = time.Now().Add(-time.Hour).Unix()
			case "subject":
				claims["sub"] = "other"
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			var signing any = key
			if scenario == "algorithm" {
				token = jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				signing = []byte("known")
			}
			if scenario == "unsigned" {
				token = jwt.NewWithClaims(jwt.SigningMethodNone, claims)
				signing = jwt.UnsafeAllowNoneSignatureType
			}
			raw, e := token.SignedString(signing)
			if e != nil {
				t.Fatal(e)
			}
			c, e := p.Identity(context.Background(), Credentials{Access: "bearer", IDToken: raw}, "nonce")
			if scenario == "valid" {
				if e != nil || c.StudentID != "student-42" || c.IDToken != "" {
					t.Fatalf("valid identity failed: %v", e)
				}
			} else if e == nil {
				t.Fatal("invalid ID token accepted")
			}
		})
	}
}
func TestGatewayWrappedOAuthErrorAndPKCE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"error_error": map[string]any{"http_status_code": 400, "http_body": "{\"error\":\"invalid_grant\"}"}})
	}))
	defer server.Close()
	p := NewProvider(Config{ClientID: "client", RedirectURI: "https://forum.test/api/campus/tongji/callback"})
	p.Base = server.URL
	if _, e := p.Exchange(context.Background(), "code", "verifier"); !errors.Is(e, ErrAuthorization) {
		t.Fatal("HTTP 200 gateway error accepted")
	}
	u, e := url.Parse(p.Authorize("state", "nonce", "verifier", "replace"))
	if e != nil {
		t.Fatal(e)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "verifier" || q.Get("state") != "state" || q.Get("nonce") != "nonce" || q.Get("prompt") != "login" {
		t.Fatal("authorization parameters missing")
	}
}

func TestReauthorizationForcesSchoolLogin(t *testing.T) {
	p := NewProvider(Config{ClientID: "client"})
	for _, mode := range []string{"bind", "replace", "reauthorize", "login"} {
		u, err := url.Parse(p.Authorize("state", "nonce", "verifier", mode))
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if mode == "replace" || mode == "reauthorize" {
			want = "login"
		}
		if u.Query().Get("prompt") != want {
			t.Errorf("%s prompt = %q, want %q", mode, u.Query().Get("prompt"), want)
		}
	}
}
