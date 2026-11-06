package nativepushservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/golang-jwt/jwt/v5"
)

// HuaweiConfig keeps the HarmonyOS service-account key on the server only.
type HuaweiConfig struct{ CredentialsPath string }

func (c HuaweiConfig) Enabled() bool { return c.CredentialsPath != "" }
func LoadHuaweiConfig() HuaweiConfig {
	return HuaweiConfig{CredentialsPath: strings.TrimSpace(preferences.GetString("push.huawei.credentials_path", ""))}
}

type huaweiAuthorization struct{ projectID, token string }

var huaweiProjectID = regexp.MustCompile(`^[0-9]{1,64}$`)
var huaweiHTTPClient = &http.Client{
	Timeout:       10 * time.Second,
	CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
}

func authorizeHuawei(cfg HuaweiConfig) (*huaweiAuthorization, error) {
	file, err := os.Open(cfg.CredentialsPath)
	if err != nil {
		return nil, errors.New("huawei service-account file unavailable")
	}
	defer func() { _ = file.Close() }()
	var key struct {
		ProjectID  string `json:"project_id"`
		KeyID      string `json:"key_id"`
		SubAccount string `json:"sub_account"`
		PrivateKey string `json:"private_key"`
	}
	if json.NewDecoder(io.LimitReader(file, 65536)).Decode(&key) != nil ||
		!huaweiProjectID.MatchString(key.ProjectID) || key.KeyID == "" || key.SubAccount == "" {
		return nil, errors.New("invalid huawei service-account configuration")
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key.PrivateKey))
	if err != nil || privateKey.N.BitLen() < 2048 {
		return nil, errors.New("invalid huawei service-account signing key")
	}
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:   key.SubAccount,
		Audience: jwt.ClaimStrings{"https://oauth-login.cloud.huawei.com/oauth2/v3/token"},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodPS256, claims)
	token.Header["kid"] = key.KeyID
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return nil, errors.New("huawei service-account signing failed")
	}
	return &huaweiAuthorization{projectID: key.ProjectID, token: signed}, nil
}

func huaweiProfileID(userID uint64) string {
	digest := sha256.Sum256(fmt.Appendf(nil, "yourtj:%d", userID))
	return hex.EncodeToString(digest[:])
}

func sendHuawei(ctx context.Context, auth *huaweiAuthorization, deviceToken string, msg *nativePayload) error {
	category := "MARKETING"
	if msg.Channel == "yourtj_chat_v2" {
		category = "IM"
	}
	body := map[string]any{
		"payload": map[string]any{"notification": map[string]any{
			"category": category, "title": msg.Title, "body": msg.Body,
			"profileId": huaweiProfileID(msg.RecipientId), "foregroundShow": false,
			"clickAction": map[string]any{
				"actionType": 1, "uri": "yourtj://push?route=" + url.QueryEscape(msg.Route),
			},
		}},
		"target":      map[string]any{"token": []string{deviceToken}},
		"pushOptions": map[string]any{"ttl": 3600},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://push-api.cloud.huawei.com/v3/"+auth.projectID+"/messages:send", bytes.NewReader(raw))
	if err != nil {
		return errors.New("huawei request construction failed")
	}
	req.Header.Set("Authorization", "Bearer "+auth.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("push-type", "0")
	resp, err := huaweiHTTPClient.Do(req)
	if err != nil {
		return errors.New("huawei push request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	var response struct {
		Code string `json:"code"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&response) != nil {
		return errors.New("invalid huawei push response")
	}
	if resp.StatusCode == http.StatusOK && response.Code == "80000000" {
		return nil
	}
	// Invalid-token codes can also mean a configuration mismatch; retain ownership for retries.
	return errors.New("huawei push rejected")
}
