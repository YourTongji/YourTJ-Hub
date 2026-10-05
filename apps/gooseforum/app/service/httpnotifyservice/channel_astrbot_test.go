package httpnotifyservice

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// pushLiteStub 复刻 astrbot_plugin_push_lite v0.2.2 的 /send：Authorization 整串比对，
// 缺 content/umo 返回 400，成功返回 queued。
func pushLiteStub(t *testing.T, token string, got chan<- map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost || r.URL.Path != "/send" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<h1>Not Found</h1>"))
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"Forbidden","details":"403 Forbidden: 无效令牌"}`))
			return
		}
		var data map[string]string
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil || data["content"] == "" || data["umo"] == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"Bad Request","details":"400 Bad Request: 缺少字段"}`))
			return
		}
		got <- data
		_, _ = w.Write([]byte(`{"status":"queued","message_id":"m1","queue_size":1}`))
	}))
	t.Cleanup(server.Close)
	return server
}

func TestSendTestAstrBotPushesTextToTargetSession(t *testing.T) {
	got := make(chan map[string]string, 1)
	server := pushLiteStub(t, "tok", got)
	endpoint := pageConfig.HttpNotifyEndpoint{
		ChannelType: pageConfig.HttpNotifyChannelAstrBot,
		URL:         server.URL, // 只填到端口：自动补 /send
		Target:      "aiocqhttp:GroupMessage:123456",
		Secret:      "tok",
	}
	if err := SendTest(endpoint, "https://hub.example.test", time.Unix(1_800_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	data := <-got
	if data["umo"] != "aiocqhttp:GroupMessage:123456" || data["message_type"] != "text" {
		t.Fatalf("body = %v", data)
	}
	content := data["content"]
	for _, want := range []string{"【测试】【新举报 · 回复】垃圾广告", "举报说明：", "\n\n版主工作台：https://hub.example.test/moderation?tab=reports\n"} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q:\n%s", want, content)
		}
	}
	if strings.Count(content, "https://") != 1 {
		t.Fatalf("sample actions share the workbench URL and must collapse into one link:\n%s", content)
	}

	endpoint.URL = server.URL + "/send"
	endpoint.Secret = "wrong"
	if err := SendTest(endpoint, "", time.Now()); err == nil || !strings.Contains(err.Error(), "无效令牌") {
		t.Fatalf("bad token = %v", err)
	}
	endpoint.Secret = "tok"
	endpoint.URL = server.URL + "/other"
	if err := SendTest(endpoint, "", time.Now()); err == nil || err.Error() != "404 Not Found" {
		t.Fatalf("wrong path = %v", err)
	}
	endpoint.URL = server.URL
	endpoint.Target = " "
	if err := SendTest(endpoint, "", time.Now()); err == nil || !strings.Contains(err.Error(), "SID") {
		t.Fatalf("missing target = %v", err)
	}
}

func TestAstrBotRejectsResponsesThatAreNotQueued(t *testing.T) {
	reply := `{"status":"ok","queue_size":0}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	endpoint := pageConfig.HttpNotifyEndpoint{ChannelType: pageConfig.HttpNotifyChannelAstrBot, URL: server.URL + "/send", Target: "s", Secret: "t"}
	if err := SendTest(endpoint, "", time.Now()); err == nil || !strings.Contains(err.Error(), "queued") {
		t.Fatalf("non-queued status = %v", err)
	}
	reply = "<html>AstrBot</html>"
	if err := SendTest(endpoint, "", time.Now()); err == nil || !strings.Contains(err.Error(), "not JSON") {
		t.Fatalf("html reply = %v", err)
	}
}

func TestNewJSONPostRejectsURLsWithoutHost(t *testing.T) {
	for _, raw := range []string{"http:///send", "http://:9966/send", "https://"} {
		_, err := newJSONPost(pageConfig.HttpNotifyEndpoint{URL: raw}, nil)
		if err == nil || !strings.Contains(err.Error(), "missing a host") {
			t.Fatalf("%q: err = %v", raw, err)
		}
	}
}

func TestAstrBotTextRendersApprovalsAndEvents(t *testing.T) {
	endpoint := pageConfig.HttpNotifyEndpoint{ChannelType: pageConfig.HttpNotifyChannelAstrBot, Target: "s"}
	decode := func(body []byte) string {
		t.Helper()
		var data map[string]string
		if err := json.Unmarshal(body, &data); err != nil {
			t.Fatal(err)
		}
		return data["content"]
	}

	body, err := astrbotChannel{}.encode(endpoint, Alternative{Event: EventReportPostCreated}, testApproval(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	text := decode(body)
	// 用户文本折叠为单行：换行与控制字符不能伪造出额外的行。
	for _, want := range []string{"【新举报 · 回复】辱骂/人身攻击 · 匿名", "摘要：line1 line2\n", "作者：匿名（不披露身份）", "时间：2026-10-04 20:00\n", "封禁并结案：https://hub.example.test/moderation/action?token=t1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("approval text missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\x00") {
		t.Fatalf("control characters must be stripped: %q", text)
	}

	comment := map[string]any{
		"baseUri":        "https://hub.example.test",
		"contentPreview": "同问\n期末考范围",
		"topic":          map[string]any{"title": "高数复习", "url": "/post/7", "user": map[string]any{"displayName": "楼主"}},
		"user":           map[string]any{"displayName": "小明", "username": "xm"},
		"post":           map[string]any{"url": "/post/7#post-3"},
	}
	body, err = astrbotChannel{}.encode(endpoint, Alternative{Event: EventCommentCreated, Data: comment}, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if text := decode(body); text != "【新回复】高数复习\n回复者：小明（@xm）\n内容：同问 期末考范围\nhttps://hub.example.test/post/7#post-3" {
		t.Fatalf("comment text:\n%s", text)
	}
}
