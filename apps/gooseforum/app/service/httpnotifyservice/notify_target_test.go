package httpnotifyservice

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// notifyTestHost 是测试里的普通域名：生产策略照旧判定地址，只替换 resolver 应答
// 与拨号目标，使投递能打到 httptest 的 loopback 接收端。
const notifyTestHost = "notify.test"

type notifyTestResolver struct{ addrs []netip.Addr }

func (r notifyTestResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	if len(r.addrs) == 0 {
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	}
	return r.addrs, nil
}

// useLocalNotifyServer 把 notifyFetch 换成能拨到 server 的客户端，返回以
// notify.test 为主机的 base URL（端口取 server 的真实端口）。测试结束恢复。
func useLocalNotifyServer(t *testing.T, server *httptest.Server) string {
	t.Helper()
	previous := notifyFetch
	t.Cleanup(func() { notifyFetch = previous })
	address := server.Listener.Addr().String()
	_, port, err := net.SplitHostPort(address)
	if err != nil {
		t.Fatal(err)
	}
	notifyFetch = safefetch.New(safefetch.Config{
		Resolver: notifyTestResolver{},
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, address)
		},
	})
	return "http://" + notifyTestHost + ":" + port
}

// localNotifyTargets 是必须被通知策略挡住的目标：本机、链路本地（云元数据）、
// unspecified、IPv4-mapped 与组播地址。
var localNotifyTargets = []string{
	"http://127.0.0.1:8080/hook",
	"http://localhost:8080/",
	"http://169.254.169.254/latest/meta-data/",
	"http://[::1]:9000/",
	"http://0.0.0.0:80/",
	"http://[::ffff:127.0.0.1]/",
	"http://[fe80::1]/",
	"http://224.0.0.1/",
}

func TestNotifyRequestsRejectLocalTargets(t *testing.T) {
	for _, raw := range localNotifyTargets {
		t.Run(raw, func(t *testing.T) {
			if _, err := newJSONPost(pageConfig.HttpNotifyEndpoint{URL: raw}, nil); err == nil {
				t.Fatal("build request accepted a local target")
			} else if strings.Contains(err.Error(), "127.0.0.1") || strings.Contains(err.Error(), "169.254") {
				t.Fatalf("error leaks the resolved target: %v", err)
			}
			if err := SendTest(pageConfig.HttpNotifyEndpoint{URL: raw}, "https://hub.example.test", time.Now()); err == nil {
				t.Fatal("test delivery accepted a local target")
			}
			if err := ValidateEndpointTarget(t.Context(), raw); err == nil {
				t.Fatal("save-time validation accepted a local target")
			}
		})
	}
}

func TestNotifyRequestsAcceptIntranetTargets(t *testing.T) {
	for _, raw := range []string{"https://open.feishu.cn/open-apis/bot/v2/hook/xxx", "http://192.168.1.10:9966/send", "http://10.1.2.3/hook"} {
		if _, err := newJSONPost(pageConfig.HttpNotifyEndpoint{URL: raw}, nil); err != nil {
			t.Fatalf("%q rejected: %v", raw, err)
		}
	}
}

func TestValidateEndpointTargetChecksDNSAnswers(t *testing.T) {
	previous := notifyFetch
	t.Cleanup(func() { notifyFetch = previous })
	for _, tc := range []struct {
		name    string
		address string
		blocked bool
	}{
		{"loopback", "127.0.0.1", true},
		{"link local", "169.254.169.254", true},
		{"intranet", "192.168.0.9", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			notifyFetch = safefetch.New(safefetch.Config{Resolver: notifyTestResolver{addrs: []netip.Addr{netip.MustParseAddr(tc.address)}}})
			err := ValidateEndpointTarget(t.Context(), "http://notify.example:9966/hook")
			if tc.blocked != (err != nil) {
				t.Fatalf("err = %v, blocked want %v", err, tc.blocked)
			}
		})
	}
}

// blockedResolver 模拟 DNS 黑洞：一直阻塞到 ctx 结束。保存校验必须用自己的短
// 预算中止它，否则会撞上服务端 10s 写超时。
type blockedResolver struct{}

func (blockedResolver) LookupNetIP(ctx context.Context, _, _ string) ([]netip.Addr, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestValidateEndpointTargetBoundsDNSCheck(t *testing.T) {
	previous := notifyFetch
	t.Cleanup(func() { notifyFetch = previous })
	// 端点超时是 15s；保存校验必须远早于它返回，否则管理端只看到断连。
	notifyFetch = safefetch.New(safefetch.Config{Resolver: blockedResolver{}, ConnectTimeout: maxTimeoutSeconds * time.Second})

	for _, tc := range []struct {
		name   string
		ctx    func(t *testing.T) context.Context
		budget time.Duration
	}{
		{
			name:   "save budget stops the DNS wait",
			ctx:    func(t *testing.T) context.Context { return t.Context() },
			budget: saveTargetCheckTimeout,
		},
		{
			name: "shorter caller deadline wins",
			ctx: func(t *testing.T) context.Context {
				ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
				t.Cleanup(cancel)
				return ctx
			},
			budget: 50 * time.Millisecond,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			started := time.Now()
			err := ValidateEndpointTarget(tc.ctx(t), "http://notify.example:9966/hook")
			elapsed := time.Since(started)
			if err == nil {
				t.Fatal("save-time validation accepted a host whose DNS never answers")
			}
			if elapsed >= 2*tc.budget {
				t.Fatalf("validation took %v, want under %v", elapsed, 2*tc.budget)
			}
		})
	}
}

func TestSendTestReachesUseLocalNotifyServerReceiver(t *testing.T) {
	received := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Goose-Event") != EventWebhookTest {
			t.Errorf("event header = %q", r.Header.Get("X-Goose-Event"))
		}
		received <- struct{}{}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	base := useLocalNotifyServer(t, server)
	if err := SendTest(pageConfig.HttpNotifyEndpoint{URL: base + "/hook"}, "", time.Now()); err != nil {
		t.Fatal(err)
	}
	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("test delivery did not reach the receiver")
	}
}
