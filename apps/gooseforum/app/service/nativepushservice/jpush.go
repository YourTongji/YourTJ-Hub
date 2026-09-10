package nativepushservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

// JPush aggregates Android OEM offline channels without requiring Google services.
// MasterSecret is server-only; the app receives only the public AppKey.
type JPushConfig struct{ AppKey, MasterSecret string }

func (c JPushConfig) Enabled() bool { return c.AppKey != "" && c.MasterSecret != "" }
func LoadJPushConfig() JPushConfig {
	return JPushConfig{
		AppKey:       strings.TrimSpace(preferences.GetString("push.jpush.app_key", "")),
		MasterSecret: strings.TrimSpace(preferences.GetString("push.jpush.master_secret", "")),
	}
}

var jpushHTTPClient = &http.Client{Timeout: 10 * time.Second}

func sendJPush(ctx context.Context, cfg JPushConfig, registrationID string, msg *nativePayload) error {
	// Do not use broadcasts, aliases, analytics IDs or user profile data.
	body := map[string]any{
		"platform": "android",
		"audience": map[string]any{"registration_id": []string{registrationID}},
		"notification": map[string]any{"android": map[string]any{
			"alert": msg.Body, "title": msg.Title, "channel_id": "yourtj_activity",
			"extras": map[string]string{"route": msg.Route},
			// OEM notification clicks start this dedicated, allowlisted receiver activity.
			"intent": map[string]string{"url": "intent:#Intent;action=tj.yourtj.forum_app.OPEN_PUSH;component=tj.yourtj.forum_app/tj.yourtj.forum_app.PushOpenActivity;end"},
		}},
		"options": map[string]any{"time_to_live": 3600},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.jpush.cn/v3/push", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.AppKey, cfg.MasterSecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err := jpushHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("jpush request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	// Parse only the provider error code; never include its body or token in logs.
	var failure struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	_ = json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&failure)
	// 1011 can also mean an AppKey/test-mode mismatch or a temporarily unmatched
	// target. It is not proof of uninstall, so retain the registration for retry.
	return fmt.Errorf("jpush rejected notification: status=%d code=%d", resp.StatusCode, failure.Error.Code)
}
