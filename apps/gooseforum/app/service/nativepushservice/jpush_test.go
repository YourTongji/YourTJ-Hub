package nativepushservice

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestJPushRequestAndFailure(t *testing.T) {
	original := jpushHTTPClient
	t.Cleanup(func() { jpushHTTPClient = original })
	for _, tc := range []struct {
		status int
		body   string
		stale  bool
	}{
		{200, `{"msg_id":"1"}`, false}, {400, `{"error":{"code":1011}}`, false},
		{401, `{"error":{"code":1004}}`, false}, {503, `unavailable`, false},
	} {
		jpushHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.jpush.cn/v3/push" {
				t.Errorf("unexpected endpoint %s", r.URL)
			}
			user, password, ok := r.BasicAuth()
			if !ok || user != "app" || password != "secret" {
				t.Error("missing provider authentication")
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["platform"] != "android" {
				t.Error("wrong platform")
			}
			audience := payload["audience"].(map[string]any)["registration_id"].([]any)
			if len(audience) != 1 || audience[0] != "device" {
				t.Error("must target exactly the registered device")
			}
			android := payload["notification"].(map[string]any)["android"].(map[string]any)
			if android["channel_id"] != "yourtj_activity" || android["extras"].(map[string]any)["route"] != "/p/42" {
				t.Error("lost channel or route")
			}
			intent := android["intent"].(map[string]any)["url"].(string)
			if !strings.Contains(intent, "action=tj.yourtj.forum_app.OPEN_PUSH;") || !strings.Contains(intent, "component=tj.yourtj.forum_app/tj.yourtj.forum_app.PushOpenActivity;") {
				t.Error("OEM intent must carry action and full component")
			}
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
		})}
		err := sendJPush(context.Background(), JPushConfig{"app", "secret"}, "device", &nativePayload{Title: "YourTJ", Body: "New reply", Route: "/p/42"})
		if (err == nil) != (tc.status == 200) {
			t.Errorf("status %d: error=%v", tc.status, err)
		}
		if errors.Is(err, errTokenStale) != tc.stale {
			t.Errorf("unexpected stale classification: %v", err)
		}
	}
}
func TestProviderPreservesLegacyRegistrations(t *testing.T) {
	for _, tc := range []struct{ platform, provider, want string }{
		{"ios", "", "apns"}, {"android", "", "fcm"}, {"android", "jpush", "jpush"},
	} {
		d := pushDevice.Entity{Platform: tc.platform, Provider: tc.provider}
		if d.DeliveryProvider() != tc.want {
			t.Fatal("provider mismatch")
		}
	}
}

func TestJPushWorkerRoutesAndPreservesUnmatchedRegistration(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)
	preferences.Set("push.jpush.app_key", "app")
	preferences.Set("push.jpush.master_secret", "secret")
	if err := pushDevice.Upsert(1, "android", "jpush-device", time.Now(), "jpush"); err != nil {
		t.Fatal(err)
	}
	if err := pushDevice.Upsert(1, "android", "legacy-fcm", time.Now()); err != nil {
		t.Fatal(err)
	}
	notification := eventNotification.Entity{UserId: 1, EventType: eventNotification.EventTypeComment,
		Payload: eventNotification.NotificationPayload{TopicId: 42, TopicTitle: "Course review", PostNo: 2}}
	if err := eventNotification.Create(&notification); err != nil {
		t.Fatal(err)
	}
	original := jpushHTTPClient
	t.Cleanup(func() { jpushHTTPClient = original })
	sends := 0
	jpushHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sends++
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		recipients := payload["audience"].(map[string]any)["registration_id"].([]any)
		if len(recipients) != 1 || recipients[0] != "jpush-device" {
			t.Fatal("worker used the wrong provider token")
		}
		return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"error":{"code":1011}}`)), Header: make(http.Header)}, nil
	})}
	if err := RunPushTask(context.Background(), makeNativePushTask(t, 1, notification.Id)); err != nil {
		t.Fatal(err)
	}
	devices := pushDevice.ListByUser(1)
	if sends != 1 || len(devices) != 2 {
		t.Fatal("JPush unmatched target must not erase device registrations")
	}
}
