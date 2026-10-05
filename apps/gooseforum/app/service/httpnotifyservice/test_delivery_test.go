package httpnotifyservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// verifiedFeishuIcons 已逐个在飞书卡片图标库核对过的 token；写错的 token 在客户端
// 静默不显示，新增图标时须先核对再加入这里。
var verifiedFeishuIcons = map[string]bool{
	"approval_outlined": true, "report_outlined": true, "member_outlined": true, "tag_outlined": true,
	"time_outlined": true, "chat_outlined": true, "lock_outlined": true, "warning_outlined": true,
	"info_outlined": true, "safe_outlined": true, "admin_outlined": true, "yes_outlined": true,
	"no_outlined": true, "ban_outlined": true, "invisible_outlined": true, "close_outlined": true,
}

func collectFeishuIcons(node any, tokens map[string]bool) {
	switch value := node.(type) {
	case map[string]any:
		if value["tag"] == "standard_icon" {
			tokens[value["token"].(string)] = true
		}
		for _, child := range value {
			collectFeishuIcons(child, tokens)
		}
	case []any:
		for _, child := range value {
			collectFeishuIcons(child, tokens)
		}
	}
}

func decodeFeishuCard(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var decoded struct {
		Card map[string]any `json:"card"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded.Card
}

func TestFeishuCardUsesVerifiedIconsAndNoEmoji(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	review := testApproval()
	review.Approval.Kind = ApprovalKindReview
	review.Approval.Reason = "ai"
	review.Approval.Edited = true
	review.Approval.Actions = []ApprovalAction{{Action: ApprovalActionApprove, URL: "/a"}, {Action: ApprovalActionReject, URL: "/r"}}
	chat := testApproval()
	chat.Approval.TargetType = "chat_message"
	noBase := testApproval()
	noBase.BaseURI = ""
	course := testApproval()
	course.Approval.Actions = []ApprovalAction{{Action: ApprovalActionHide, URL: "/h"}, {Action: ApprovalActionDismiss, URL: "/d"}}
	testCard, err := feishuChannel{}.testBody(pageConfig.HttpNotifyEndpoint{}, "https://hub.example.test", now)
	if err != nil {
		t.Fatal(err)
	}
	bodies := [][]byte{testCard}
	for _, payload := range []*ApprovalPayload{testApproval(), review, chat, noBase, course} {
		body, err := feishuChannel{}.encode(pageConfig.HttpNotifyEndpoint{}, Alternative{}, payload, now)
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
	}
	for _, body := range bodies {
		card := decodeFeishuCard(t, body)
		tokens := map[string]bool{}
		collectFeishuIcons(card, tokens)
		for token := range tokens {
			if !verifiedFeishuIcons[token] {
				t.Errorf("icon token %q is not in the verified list", token)
			}
		}
		for _, r := range string(body) {
			if r >= 0x2600 && (unicode.Is(unicode.So, r) || r >= 0x1F000) {
				t.Fatalf("card contains an emoji/symbol %q: %s", r, body)
			}
		}
		header := card["header"].(map[string]any)
		if tags, _ := header["text_tag_list"].([]any); len(tags) > 3 {
			t.Fatalf("feishu shows at most 3 header tags: %v", tags)
		}
	}

	// 审核卡片：标题栏用审批图标与橙色，标签依次为原因与「编辑后」，按钮自动换行并带图标。
	reviewCard := decodeFeishuCard(t, bodies[2])
	header := reviewCard["header"].(map[string]any)
	if header["template"] != "orange" || header["icon"].(map[string]any)["token"] != "approval_outlined" {
		t.Fatalf("review header = %v", header)
	}
	tags := header["text_tag_list"].([]any)
	if tags[0].(map[string]any)["text"].(map[string]any)["content"] != "AI 审核存疑" || tags[1].(map[string]any)["text"].(map[string]any)["content"] != "编辑后" {
		t.Fatalf("review tags = %v", tags)
	}
	text := string(bodies[2])
	if !strings.Contains(text, `"background_style":"grey-50"`) || !strings.Contains(text, `"flex_mode":"flow"`) || !strings.Contains(text, `"primary_filled"`) {
		t.Fatalf("review card layout: %s", text)
	}
	// 测试卡片带「测试」标签，按钮只指向版主工作台。
	sample := string(testCard)
	if !strings.Contains(sample, `"content":"测试"`) || strings.Contains(sample, "/moderation/action") || !strings.Contains(sample, "https://hub.example.test/moderation?tab=reports") {
		t.Fatalf("test card: %s", sample)
	}
}

func TestSendTestGenericSignsWebhookTestEvent(t *testing.T) {
	type captured struct {
		header http.Header
		body   []byte
	}
	received := make(chan captured, 1)
	status := http.StatusNoContent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- captured{r.Header.Clone(), body}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	endpoint := pageConfig.HttpNotifyEndpoint{Id: "ep1", Name: "测试", URL: server.URL + "/hook", Secret: "s3cret", TimeoutSeconds: 2}
	now := time.Unix(1_800_000_000, 0)
	if err := SendTest(endpoint, "https://hub.example.test", now); err != nil {
		t.Fatal(err)
	}
	got := <-received
	if got.header.Get("X-Goose-Event") != EventWebhookTest {
		t.Fatalf("event header = %q", got.header.Get("X-Goose-Event"))
	}
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write([]byte(strconv.FormatInt(now.Unix(), 10) + "."))
	mac.Write(got.body)
	if got.header.Get("X-Goose-Signature") != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("test delivery must carry a verifiable signature")
	}
	var envelope Envelope
	if err := json.Unmarshal(got.body, &envelope); err != nil || envelope.Event != EventWebhookTest {
		t.Fatalf("envelope = %s", got.body)
	}

	status = http.StatusInternalServerError
	err := SendTest(endpoint, "", now)
	<-received
	if err == nil || err.Error() != "500 Internal Server Error" {
		t.Fatalf("server error = %v", err)
	}
}

func TestSendTestFeishuReportsBusinessErrorsWithoutURL(t *testing.T) {
	reply := `{"code":0,"msg":"success"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)
	endpoint := pageConfig.HttpNotifyEndpoint{ChannelType: pageConfig.HttpNotifyChannelFeishu, URL: server.URL + "/open-apis/bot/v2/hook/secret-token"}
	if err := SendTest(endpoint, "https://hub.example.test", time.Now()); err != nil {
		t.Fatal(err)
	}
	reply = `{"code":19021,"msg":"sign match fail"}`
	err := SendTest(endpoint, "https://hub.example.test", time.Now())
	if err == nil || !strings.Contains(err.Error(), "19021") {
		t.Fatalf("business error = %v", err)
	}
	server.Close()
	err = SendTest(endpoint, "https://hub.example.test", time.Now())
	if err == nil || strings.Contains(err.Error(), "secret-token") || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("network error must not echo the webhook url: %v", err)
	}
}
