package eventhandlers

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
)

// eventhandlersTestResolver 把所有域名解析成公网地址，让投递停在测试拨号器上。
type eventhandlersTestResolver struct{}

func (eventhandlersTestResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
}

// captureHttpNotify 把 HTTP 通知配置指向本地接收端，返回收到的请求体通道。
func captureHttpNotify(t *testing.T, events ...string) <-chan []byte {
	t.Helper()
	return captureHttpNotifyStatus(t, func(int) int { return http.StatusNoContent }, events...)
}

// captureHttpNotifyStatus 同 captureHttpNotify，status 按第 n 次（从 1 起）请求决定响应码。
// 端点地址用普通域名 + httptest 真实端口：通知目标策略照旧判定（内网放行、环回拒绝），
// 只有拨号目标被测试客户端改到 loopback 接收端。
func captureHttpNotifyStatus(t *testing.T, status func(n int) int, events ...string) <-chan []byte {
	t.Helper()
	received := make(chan []byte, 8)
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		received <- body
		w.WriteHeader(status(int(count.Add(1))))
	}))
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// 清理是 LIFO：先注册恢复，接收端随后才关闭，恢复发生在 server.Close 之后。
	t.Cleanup(httpnotifyservice.SetNotifyClientForTest(safefetch.New(safefetch.Config{
		Resolver: eventhandlersTestResolver{},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			_, dialPort, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort("127.0.0.1", dialPort))
		},
	})))
	t.Cleanup(server.Close)
	if err := dbconnect.Connect().AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatal(err)
	}
	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	entity.PageType = pageConfig.HttpNotify
	entity.Config = jsonopt.Encode(pageConfig.HttpNotifyStorageConfig{Enabled: true, Endpoints: []pageConfig.HttpNotifyStorageEndpoint{{
		Id: "capture", Name: "capture", Enabled: true, URL: "http://notify.test:" + port, Events: events, TimeoutSeconds: 2,
	}}})
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearHttpNotifyConfigCache()
	t.Cleanup(func() {
		entity.Config = jsonopt.Encode(pageConfig.HttpNotifyStorageConfig{Endpoints: []pageConfig.HttpNotifyStorageEndpoint{}})
		pageConfig.CreateOrSave(&entity)
		hotdataserve.ClearHttpNotifyConfigCache()
	})
	return received
}

func waitHttpNotify(t *testing.T, received <-chan []byte) []byte {
	t.Helper()
	select {
	case body := <-received:
		return body
	case <-time.After(3 * time.Second):
		t.Fatal("http notify was not delivered")
		return nil
	}
}

// 匿名楼层被举报时，举报 webhook 不得携带真实作者身份（issue #1049 / #524）。
func TestReportCreatedWebhookOmitsAnonymousPostAuthor(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &topics.Entity{}, &posts.Entity{}); err != nil {
		t.Fatal(err)
	}
	author := users.MakeUser("anon_report_author", "pass1234", "anon_report_author@example.com")
	if err := users.Create(author); err != nil {
		t.Fatal(err)
	}
	topic := topics.Entity{UserId: 9999, Status: 1, Title: "anonymous report"}
	if err := conn.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	post := posts.Entity{TopicId: topic.Id, PostNo: 2, UserId: author.Id, Content: "anonymous floor", IsAnonymous: true}
	if err := conn.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	received := captureHttpNotify(t, "moderation.report.created")

	if err := handleHttpNotifyReportCreated(context.Background(), &ReportCreatedEvent{
		ReportId: 1, TargetType: "post", TargetId: post.Id, TopicId: topic.Id, ReporterId: 9998, Reason: "abuse",
	}); err != nil {
		t.Fatal(err)
	}
	body := waitHttpNotify(t, received)
	if strings.Contains(string(body), "anon_report_author") {
		t.Fatalf("anonymous author username leaked: %s", body)
	}
	var envelope struct {
		Data struct {
			Post *struct {
				UserID uint64 `json:"userId"`
				User   struct {
					ID uint64 `json:"id"`
				} `json:"user"`
			} `json:"post"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Data.Post == nil {
		t.Fatalf("post payload missing: %s", body)
	}
	if envelope.Data.Post.UserID != 0 || envelope.Data.Post.User.ID != 0 {
		t.Fatalf("anonymous author id leaked: %s", body)
	}
}
