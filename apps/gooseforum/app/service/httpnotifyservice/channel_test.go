package httpnotifyservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

func testApproval() *ApprovalPayload {
	return &ApprovalPayload{BaseURI: "https://hub.example.test", Approval: Approval{
		ID: "report:9", Kind: ApprovalKindReport, TargetType: "post", TargetID: 5, ReportID: 9,
		Reason: "abuse", Note: "<at id=all></at> **bold** [x](https://evil.test)",
		Title: "话题", Excerpt: "line1\n\nline2\x00", Anonymous: true, CreatedAt: "2026-10-04T12:00:00Z",
		ModerationURL: "/moderation?tab=reports",
		Actions:       []ApprovalAction{{Action: ApprovalActionBan, URL: "/moderation/action?token=t1"}},
	}}
}

func TestFeishuEncodeSignsAndRendersUserTextAsPlainText(t *testing.T) {
	endpoint := pageConfig.HttpNotifyEndpoint{ChannelType: pageConfig.HttpNotifyChannelFeishu, Secret: "feishu-secret"}
	now := time.Unix(1_800_000_000, 0)
	body, err := feishuChannel{}.encode(endpoint, Alternative{Event: EventReportPostCreated}, testApproval(), now)
	if err != nil {
		t.Fatal(err)
	}
	// 时间按 UTC+8 显示，不随服务器（容器常为 UTC）时区变化。
	if !strings.Contains(string(body), "时间：2026-10-04 20:00") {
		t.Fatalf("card time must render in UTC+8: %s", body)
	}
	var decoded struct {
		MsgType   string         `json:"msg_type"`
		Timestamp string         `json:"timestamp"`
		Sign      string         `json:"sign"`
		Card      map[string]any `json:"card"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("1800000000\nfeishu-secret"))
	if decoded.MsgType != "interactive" || decoded.Timestamp != "1800000000" || decoded.Sign != base64.StdEncoding.EncodeToString(mac.Sum(nil)) {
		t.Fatalf("signature fields = %q %q %q", decoded.MsgType, decoded.Timestamp, decoded.Sign)
	}
	if decoded.Card["schema"] != "2.0" {
		t.Fatalf("card schema = %v", decoded.Card["schema"])
	}
	text := string(body)
	if strings.Contains(text, `"markdown"`) || strings.Contains(text, `"lark_md"`) {
		t.Fatalf("user text must not be rendered as markdown: %s", text)
	}
	if !strings.Contains(text, "https://hub.example.test/moderation/action?token=t1") || !strings.Contains(text, `"open_url"`) {
		t.Fatalf("action button missing absolute open_url: %s", text)
	}
	if strings.Contains(text, `\u0000`) || !strings.Contains(text, "line1 line2") {
		t.Fatalf("excerpt not normalized: %s", text)
	}
	if !strings.Contains(text, "匿名") {
		t.Fatalf("anonymous marker missing: %s", text)
	}

	// 未配置签名密钥时不携带 timestamp/sign。
	unsigned, err := feishuChannel{}.encode(pageConfig.HttpNotifyEndpoint{}, Alternative{}, testApproval(), now)
	if err != nil || strings.Contains(string(unsigned), `"sign"`) {
		t.Fatalf("unsigned body = %s err=%v", unsigned, err)
	}
	// 站点地址未配置时不生成相对链接按钮。
	noBase := testApproval()
	noBase.BaseURI = ""
	relative, _ := feishuChannel{}.encode(pageConfig.HttpNotifyEndpoint{}, Alternative{}, noBase, now)
	if strings.Contains(string(relative), `"button"`) {
		t.Fatalf("buttons rendered without base uri: %s", relative)
	}
}

func TestFeishuCheckResponseTreatsBusinessErrorsAsFailures(t *testing.T) {
	cases := []struct {
		status int
		body   string
		ok     bool
	}{
		{200, `{"code":0,"msg":"success","data":{}}`, true},
		{200, `{"StatusCode":0,"StatusMessage":"success"}`, true},
		{200, `{"code":19021,"msg":"sign match fail or timestamp is not within one hour from current time"}`, false},
		{200, `{"StatusCode":9499,"StatusMessage":"Bad Request"}`, false},
		{200, `ok`, false},
		{200, `{}`, false},
		{500, `{"code":0}`, false},
	}
	for _, tc := range cases {
		resp := &http.Response{StatusCode: tc.status, Status: http.StatusText(tc.status), Body: io.NopCloser(strings.NewReader(tc.body))}
		if err := (feishuChannel{}).checkResponse(resp); (err == nil) != tc.ok {
			t.Fatalf("status %d body %s: err = %v, want ok=%v", tc.status, tc.body, err, tc.ok)
		}
	}
}

func TestSelectAlternativeDeliversOncePerEndpoint(t *testing.T) {
	msg := Message{
		Alternatives: []Alternative{{Event: EventReportPostCreated, Data: "specific"}, {Event: EventReportCreated, Data: "legacy"}},
		Approval:     testApproval(),
	}
	both := pageConfig.HttpNotifyEndpoint{Enabled: true, URL: "http://example.test", Events: []string{EventReportCreated, EventReportPostCreated}}
	if alt, ok := selectAlternative(both, msg); !ok || alt.Data != "specific" {
		t.Fatalf("endpoint subscribed to both got %+v %v, want the specific event once", alt, ok)
	}
	legacy := pageConfig.HttpNotifyEndpoint{Enabled: true, URL: "http://example.test", Events: []string{EventReportCreated}}
	if alt, ok := selectAlternative(legacy, msg); !ok || alt.Data != "legacy" {
		t.Fatalf("legacy endpoint got %+v %v", alt, ok)
	}
	feishu := pageConfig.HttpNotifyEndpoint{Enabled: true, URL: "http://example.test", ChannelType: pageConfig.HttpNotifyChannelFeishu,
		Events: []string{EventTopicPublished, EventReportCreated}}
	if endpointAccepts(feishu, EventTopicPublished) {
		t.Fatal("feishu channel must not accept non-approval events")
	}
	if _, ok := selectAlternative(feishu, Message{Alternatives: []Alternative{{Event: EventReportCreated}}}); ok {
		t.Fatal("feishu channel must skip messages without an approval payload")
	}
	if alt, ok := selectAlternative(feishu, msg); !ok || alt.Event != EventReportCreated {
		t.Fatalf("feishu aggregate subscription got %+v %v", alt, ok)
	}
}

func TestDeliveryErrorMessageOmitsCredentialURL(t *testing.T) {
	err := &url.Error{Op: "Post", URL: "https://open.feishu.cn/open-apis/bot/v2/hook/secret-hook-token", Err: io.ErrUnexpectedEOF}
	if got := deliveryErrorMessage(err); strings.Contains(got, "secret-hook-token") || got != io.ErrUnexpectedEOF.Error() {
		t.Fatalf("delivery error = %q", got)
	}
	_, parseErr := url.Parse("https://open.feishu.cn/hook/secret-hook-token\x7f")
	if got := deliveryErrorMessage(parseErr); strings.Contains(got, "secret-hook-token") {
		t.Fatalf("parse error leaked url: %q", got)
	}
}

func TestDeliveryDeduperClaimsOncePerTTL(t *testing.T) {
	d := &deliveryDeduper{seen: map[string]time.Time{}}
	now := time.Unix(1_800_000_000, 0)
	if !d.claim("ep|report:1", now) || d.claim("ep|report:1", now.Add(time.Minute)) {
		t.Fatal("duplicate approval delivered twice within ttl")
	}
	if !d.claim("ep2|report:1", now) {
		t.Fatal("another endpoint must still receive the approval")
	}
	// 失败投递归还名额后可以重试；只归还自己那次登记，不删除之后的新登记。
	d.release("ep2|report:1", now.Add(time.Second))
	if d.claim("ep2|report:1", now.Add(time.Minute)) {
		t.Fatal("release with a different claim time must not drop the claim")
	}
	d.release("ep2|report:1", now)
	if !d.claim("ep2|report:1", now.Add(time.Minute)) {
		t.Fatal("a released claim must allow the next delivery attempt")
	}
	if !d.claim("ep|report:1", now.Add(dedupeTTL)) {
		t.Fatal("approval must be deliverable again after ttl")
	}
	for i := range maxDedupeEntries + 10 {
		d.claim("bulk|"+time.Duration(i).String(), now.Add(time.Duration(i)*time.Millisecond))
	}
	if len(d.seen) > maxDedupeEntries {
		t.Fatalf("deduper grew past its bound: %d", len(d.seen))
	}

	// 登记时间扎堆（突发批量举报，时间相同）时，超限也只淘汰一半，而不是几乎全部。
	burst := &deliveryDeduper{seen: map[string]time.Time{}}
	for i := range maxDedupeEntries + 1 {
		burst.claim("burst|"+strconv.Itoa(i), now)
	}
	if len(burst.seen) < maxDedupeEntries/2 {
		t.Fatalf("clustered claims evicted too many entries: %d left", len(burst.seen))
	}
	// 时间不同时淘汰最早的一半，最新的登记保留。
	ordered := &deliveryDeduper{seen: map[string]time.Time{}}
	for i := range maxDedupeEntries + 1 {
		ordered.claim("ordered|"+strconv.Itoa(i), now.Add(time.Duration(i)))
	}
	if ordered.claim("ordered|"+strconv.Itoa(maxDedupeEntries-1), now.Add(time.Minute)) {
		t.Fatal("the newest claims must survive pruning")
	}
}

// 飞书返回 HTTP 200 + code!=0 时必须计入失败计数，且 lastError 不含 webhook 地址。
func TestFeishuBusinessFailureCountsTowardCircuitBreaker(t *testing.T) {
	old := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", "httpnotify-feishu-test-key-0123456789")
	t.Cleanup(func() { preferences.Set("app.signingKey", old) })

	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		_, _ = w.Write([]byte(`{"code":19021,"msg":"sign match fail"}`))
	}))
	t.Cleanup(server.Close)
	hookURL := server.URL + "/open-apis/bot/v2/hook/secret-hook-token"
	sealedURL, err := securestore.EncryptPurpose(hookURL, securestore.HttpNotifyURLPurpose)
	if err != nil {
		t.Fatal(err)
	}
	if err := dbconnect.Connect().AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	entity.PageType = pageConfig.HttpNotify
	entity.Config = jsonopt.Encode(pageConfig.HttpNotifyStorageConfig{Enabled: true, Endpoints: []pageConfig.HttpNotifyStorageEndpoint{{
		Id: "feishu-1", Name: "飞书", ChannelType: pageConfig.HttpNotifyChannelFeishu, Enabled: true,
		URLEncrypted: sealedURL, Events: []string{EventReportPostCreated}, TimeoutSeconds: 2,
	}}})
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearHttpNotifyConfigCache()
	t.Cleanup(func() {
		entity.Config = jsonopt.Encode(pageConfig.HttpNotifyStorageConfig{Endpoints: []pageConfig.HttpNotifyStorageEndpoint{}})
		pageConfig.CreateOrSave(&entity)
		hotdataserve.ClearHttpNotifyConfigCache()
	})

	approval := testApproval()
	approval.Approval.ID = "report:feishu-failure-test"
	Publish(Message{Alternatives: []Alternative{{Event: EventReportPostCreated, Data: approval}}, Approval: approval, DedupeKey: approval.Approval.ID})
	select {
	case body := <-received:
		if !strings.Contains(string(body), `"msg_type":"interactive"`) {
			t.Fatalf("feishu body = %s", body)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("feishu endpoint was not called")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		stored := pageConfig.GetConfigByPageType(pageConfig.HttpNotify, pageConfig.HttpNotifyStorageConfig{})
		if len(stored.Endpoints) == 1 && stored.Endpoints[0].FailureCount == 1 {
			ep := stored.Endpoints[0]
			if strings.Contains(ep.LastError, "secret-hook-token") || !strings.Contains(ep.LastError, "19021") {
				t.Fatalf("lastError = %q", ep.LastError)
			}
			if ep.URL != "" || ep.URLEncrypted != sealedURL {
				t.Fatalf("delivery state write-back must keep the sealed url: %+v", ep)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("failure not recorded: %+v", stored.Endpoints)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
