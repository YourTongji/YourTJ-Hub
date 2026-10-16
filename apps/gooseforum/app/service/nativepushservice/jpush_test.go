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

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pushDevice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
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
			if android["channel_id"] != "yourtj_activity_v2" || android["extras"].(map[string]any)["route"] != "/p/42" {
				t.Error("lost channel or route")
			}
			intent := android["intent"].(map[string]any)["url"].(string)
			if !strings.Contains(intent, "action=tj.yourtj.forum_app.OPEN_PUSH;") || !strings.Contains(intent, "component=tj.yourtj.forum_app/tj.yourtj.forum_app.PushOpenActivity;") {
				t.Error("OEM intent must carry action and full component")
			}
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
		})}
		err := sendJPush(context.Background(), JPushConfig{"app", "secret"}, "device", &nativePayload{Channel: "yourtj_activity_v2", Title: "YourTJ", Body: "New reply", Route: "/p/42"})
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
	conn := db.Connect()
	if err := conn.AutoMigrate(&topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	conn.Unscoped().Where("id = ?", 42).Delete(&topics.Entity{})
	if err := conn.Create(&topics.Entity{Id: 42, Title: "Course review", Status: 1, FirstPostId: 4201}).Error; err != nil {
		t.Fatal(err)
	}
	conn.Unscoped().Where("id = ?", 4201).Delete(&posts.Entity{})
	if err := conn.Create(&posts.Entity{Id: 4201, TopicId: 42, PostNo: 1}).Error; err != nil {
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

func TestFCMRejectsFailureAndCarriesSourceEnvelope(t *testing.T) {
	original := fcmHTTPClient
	t.Cleanup(func() { fcmHTTPClient = original })
	for _, status := range []int{200, 400, 401, 404, 429, 503} {
		fcmHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			message := body["message"].(map[string]any)
			data := message["data"].(map[string]any)
			if data["sourceKind"] != "chat_message" || data["sourceId"] != "42" || data["recipientId"] != "7" {
				t.Fatal("lost source/recipient identity")
			}
			android := message["android"].(map[string]any)["notification"].(map[string]any)
			if android["channel_id"] != "yourtj_chat_v2" || android["tag"] != "chat_message:42" {
				t.Fatal("messages must have distinct notification tags")
			}
			return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
		})}
		err := sendFCM(context.Background(), "fake-access", FCMConfig{ProjectID: "test"}, "device", &nativePayload{Title: "Private message", Body: "Open messages", Route: "/chat?userId=3&recipientId=7", SourceKind: "chat_message", SourceId: 42, RecipientId: 7, Channel: "yourtj_chat_v2", Thread: "chat:9"})
		if (err == nil) != (status == 200) {
			t.Errorf("FCM status %d error %v", status, err)
		}
		if errors.Is(err, errTokenStale) != (status == 404) {
			t.Errorf("FCM stale status %d error %v", status, err)
		}
	}
}

func TestJPushChatWorkerUsesPrivateSafeSource(t *testing.T) {
	setupNativePushTestDB(t)
	clearNativePushConfig(t)
	preferences.Set("push.jpush.app_key", "app")
	preferences.Set("push.jpush.master_secret", "secret")
	conn := db.Connect()
	if err := conn.AutoMigrate(&messages.Entity{}, &imUserChatConfigs.Entity{}, &users.BlockEntity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.EntityComplete{Id: 2, Username: "sender"}).Error; err != nil {
		t.Fatal(err)
	}
	conn.Where("id = ?", 4301).Delete(&messages.Entity{})
	conn.Where("conv_id = ?", 4302).Delete(&imUserChatConfigs.Entity{})
	if err := conn.Create(&messages.Entity{Id: 4301, ConvId: 4302, SenderId: 2, Content: "PRIVATE MESSAGE BODY"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&imUserChatConfigs.Entity{UserId: 1, PeerId: 2, ConvId: 4302}).Error; err != nil {
		t.Fatal(err)
	}
	if err := pushDevice.Upsert(1, "android", "chat-device", time.Now(), "jpush"); err != nil {
		t.Fatal(err)
	}
	original := jpushHTTPClient
	t.Cleanup(func() { jpushHTTPClient = original })
	sends := 0
	jpushHTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		sends++
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "PRIVATE MESSAGE BODY") {
			t.Fatal("chat content leaked into provider payload")
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatal(err)
		}
		android := payload["notification"].(map[string]any)["android"].(map[string]any)
		extras := android["extras"].(map[string]any)
		if extras["sourceKind"] != "chat_message" || extras["sourceId"] != "4301" || extras["recipientId"] != "1" || extras["route"] != "/chat?userId=2&recipientId=1" || android["channel_id"] != "yourtj_chat_v2" {
			t.Fatalf("wrong private chat envelope: %+v", android)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"msg_id":"chat"}`)), Header: make(http.Header)}, nil
	})}
	raw, err := json.Marshal(PushTask{UserId: 1, SourceKind: "chat_message", SourceId: 4301})
	if err != nil {
		t.Fatal(err)
	}
	if err := RunPushTask(context.Background(), &taskQueue.Entity{Type: TaskTypeNativePush + "4301", TaskJson: string(raw)}); err != nil {
		t.Fatal(err)
	}
	if sends != 1 {
		t.Fatalf("chat worker requests = %d want 1", sends)
	}
}
