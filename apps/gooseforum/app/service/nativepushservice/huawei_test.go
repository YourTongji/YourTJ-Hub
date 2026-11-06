package nativepushservice

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestHuaweiServiceAccountAndDelivery(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	configuration, err := json.Marshal(map[string]string{
		"project_id": "123456", "key_id": "test-key", "sub_account": "test-account",
		"private_key": string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, configuration, 0600); err != nil {
		t.Fatal(err)
	}
	auth, err := authorizeHuawei(HuaweiConfig{CredentialsPath: path})
	if err != nil {
		t.Fatal(err)
	}
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(auth.token, claims, func(_ *jwt.Token) (any, error) {
		return &key.PublicKey, nil
	}, jwt.WithValidMethods([]string{"PS256"}), jwt.WithIssuer("test-account"),
		jwt.WithAudience("https://oauth-login.cloud.huawei.com/oauth2/v3/token"))
	if err != nil || !parsed.Valid || parsed.Header["kid"] != "test-key" ||
		claims.ExpiresAt.Unix()-claims.IssuedAt.Unix() != 3600 {
		t.Fatal("service-account JWT does not match the Huawei PS256 contract")
	}
	original := huaweiHTTPClient
	t.Cleanup(func() { huaweiHTTPClient = original })
	for _, tc := range []struct {
		name, body string
		status     int
		success    bool
	}{
		{"accepted", `{"code":"80000000"}`, 200, true},
		{"providerRejected", `{"code":"80300007","msg":"do not expose provider text"}`, 200, false},
		{"serverError", `{"code":"80000000"}`, 503, false},
		{"malformed", `not JSON`, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			huaweiHTTPClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if request.URL.String() != "https://push-api.cloud.huawei.com/v3/123456/messages:send" ||
					request.Header.Get("Authorization") != "Bearer "+auth.token || request.Header.Get("push-type") != "0" {
					t.Fatal("unexpected Huawei endpoint or authentication headers")
				}
				var body struct {
					Payload struct {
						Notification struct {
							Category    string `json:"category"`
							ProfileID   string `json:"profileId"`
							ClickAction struct {
								URI string `json:"uri"`
							} `json:"clickAction"`
						} `json:"notification"`
					} `json:"payload"`
					Target struct {
						Tokens []string `json:"token"`
					} `json:"target"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				click, err := url.Parse(body.Payload.Notification.ClickAction.URI)
				if err != nil || click.Scheme != "yourtj" || click.Host != "push" ||
					click.Query().Get("route") != "/chat?userId=3&recipientId=7" ||
					body.Payload.Notification.Category != "IM" || len(body.Payload.Notification.ProfileID) != 64 ||
					len(body.Target.Tokens) != 1 || body.Target.Tokens[0] != "test-device" {
					t.Fatal("notification lost its recipient scope, route or device target")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			})}
			err := sendHuawei(t.Context(), auth, "test-device", &nativePayload{
				Title: "YourTJ", Body: "Sample", Channel: "yourtj_chat_v2", RecipientId: 7,
				Route: "/chat?userId=3&recipientId=7",
			})
			if (err == nil) != tc.success {
				t.Fatal("provider outcome was misclassified")
			}
			if err != nil && strings.Contains(err.Error(), "do not expose") {
				t.Fatal("provider response leaked")
			}
		})
	}
	if err := os.WriteFile(path, []byte(`{"project_id":"../bad","key_id":"x","sub_account":"x"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := authorizeHuawei(HuaweiConfig{CredentialsPath: path}); err == nil {
		t.Fatal("invalid project ID accepted")
	}
}
