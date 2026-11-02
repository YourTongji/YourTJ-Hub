package safefetch

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeResolver map[string][]netip.Addr

func (r fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	addresses, ok := r[host]
	if !ok {
		return nil, errors.New("host not found")
	}
	return addresses, nil
}

func publicAddr() netip.Addr { return netip.MustParseAddr("93.184.216.34") }

func mappedDialer(target string, calls *atomic.Int32) DialContextFunc {
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		calls.Add(1)
		return (&net.Dialer{}).DialContext(ctx, network, target)
	}
}

func TestFetchReturnsBoundedHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<title>Example</title>")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:    fakeResolver{"preview.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &calls),
	})
	result, err := client.Fetch(t.Context(), "http://preview.example/page#fragment")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if result.StatusCode != http.StatusOK || string(result.Body) != "<title>Example</title>" {
		t.Fatalf("Fetch() result = %#v", result)
	}
	if result.FinalURL.Fragment != "" {
		t.Fatalf("Fetch() final fragment = %q, want empty", result.FinalURL.Fragment)
	}
	if calls.Load() != 1 {
		t.Fatalf("dial calls = %d, want 1", calls.Load())
	}
}

func TestFetchJSONReturnsBoundedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accept := r.Header.Get("Accept"); accept != "application/json" {
			t.Errorf("Accept header = %q, want application/json", accept)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, `{"resultCount":1,"results":[]}`)
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:    fakeResolver{"api.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &calls),
	})
	result, err := client.FetchJSON(t.Context(), "http://api.example/lookup?id=1")
	if err != nil {
		t.Fatalf("FetchJSON() error = %v", err)
	}
	if result.StatusCode != http.StatusOK || string(result.Body) != `{"resultCount":1,"results":[]}` {
		t.Fatalf("FetchJSON() result = %#v", result)
	}
	if result.ContentType != "application/json" {
		t.Fatalf("FetchJSON() content type = %q", result.ContentType)
	}
	if calls.Load() != 1 {
		t.Fatalf("dial calls = %d, want 1", calls.Load())
	}
}

// Apple 的固定 Lookup 端点用 text/javascript 返回 JSON（实测），这条回归钉住它：
// 只允许 application/json 时，平台端点会被判为 content_type 失败，卡片静默退回
// 通用形态，而任何 fixture 都看不出来。
func TestFetchJSONAcceptsTheLegacyJavaScriptMediaType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = io.WriteString(w, `{"resultCount":1}`)
	}))
	t.Cleanup(server.Close)

	client := New(Config{
		Resolver:    fakeResolver{"itunes.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &atomic.Int32{}),
	})
	result, err := client.FetchJSON(t.Context(), "http://itunes.example/lookup?id=1")
	if err != nil {
		t.Fatalf("FetchJSON() error = %v", err)
	}
	if string(result.Body) != `{"resultCount":1}` {
		t.Fatalf("FetchJSON() body = %q", result.Body)
	}
}

// JSON 抓取只放宽媒体类型，不放宽其它策略：HTML 页面不能从这条路径溜进来，DNS 与
// 拨号校验也必须和页面抓取完全一致。
func TestFetchJSONKeepsDocumentPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<title>Example</title>")
	}))
	t.Cleanup(server.Close)

	client := New(Config{
		Resolver:    fakeResolver{"api.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &atomic.Int32{}),
	})
	if _, err := client.FetchJSON(t.Context(), "http://api.example/page"); !IsClass(err, ErrorContentType) {
		t.Fatalf("FetchJSON() on HTML error = %v, want content_type", err)
	}
	if _, err := client.Fetch(t.Context(), "http://api.example/page"); err != nil {
		t.Fatalf("Fetch() on HTML error = %v, want accepted", err)
	}

	var dialCalls atomic.Int32
	private := New(Config{
		Resolver: fakeResolver{"private.example": {netip.MustParseAddr("127.0.0.1")}},
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			dialCalls.Add(1)
			return nil, errors.New("must not dial")
		},
	})
	if _, err := private.FetchJSON(t.Context(), "http://private.example/lookup"); !IsClass(err, ErrorBlocked) {
		t.Fatalf("FetchJSON() private DNS error = %v, want blocked", err)
	}
	if dialCalls.Load() != 0 {
		t.Fatalf("dial calls = %d, want 0", dialCalls.Load())
	}
}

// 图片抓取只接受图片媒体类型；DNS 与拨号校验和页面抓取一致。
func TestFetchImageKeepsDocumentPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/page" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<title>Example</title>")
			return
		}
		if accept := r.Header.Get("Accept"); !strings.HasPrefix(accept, "image/") {
			t.Errorf("Accept header = %q, want an image type", accept)
		}
		w.Header().Set("Content-Type", "image/webp")
		_, _ = io.WriteString(w, "RIFF")
	}))
	t.Cleanup(server.Close)

	client := New(Config{
		Resolver:    fakeResolver{"cdn.example": {publicAddr()}},
		DialContext: mappedDialer(server.Listener.Addr().String(), &atomic.Int32{}),
	})
	result, err := client.FetchImage(t.Context(), "http://cdn.example/a.webp")
	if err != nil || result.ContentType != "image/webp" || string(result.Body) != "RIFF" {
		t.Fatalf("FetchImage() = %#v, %v", result, err)
	}
	if _, err := client.FetchImage(t.Context(), "http://cdn.example/page"); !IsClass(err, ErrorContentType) {
		t.Fatalf("FetchImage() on HTML error = %v, want content_type", err)
	}

	private := New(Config{
		Resolver: fakeResolver{"private.example": {netip.MustParseAddr("10.0.0.1")}},
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			return nil, errors.New("must not dial")
		},
	})
	if _, err := private.FetchImage(t.Context(), "http://private.example/a.webp"); !IsClass(err, ErrorBlocked) {
		t.Fatalf("FetchImage() private DNS error = %v, want blocked", err)
	}
}

// PreferIPv4 只调整已校验地址的拨号顺序：先试 IPv4，拨号失败再换下一个。
func TestPreferIPv4DialsValidatedAddressesInOrder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		_, _ = io.WriteString(w, "RIFF")
	}))
	t.Cleanup(server.Close)
	v4, v6 := publicAddr(), netip.MustParseAddr("2001:4860:4860::8888")

	newClient := func(prefer bool, addresses []netip.Addr, failing netip.Addr) (*Client, *[]string) {
		var mu sync.Mutex
		attempts := []string{}
		client := New(Config{
			Resolver: fakeResolver{"cdn.example": addresses},
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				host, _, _ := net.SplitHostPort(address)
				mu.Lock()
				attempts = append(attempts, host)
				mu.Unlock()
				if host == failing.String() {
					return nil, errors.New("stalled route")
				}
				return (&net.Dialer{}).DialContext(ctx, network, server.Listener.Addr().String())
			},
			PreferIPv4: prefer,
		})
		return client, &attempts
	}

	client, attempts := newClient(true, []netip.Addr{v6, v4}, v6)
	if _, err := client.FetchImage(t.Context(), "http://cdn.example/a.webp"); err != nil {
		t.Fatalf("FetchImage() with PreferIPv4 = %v", err)
	}
	if got := strings.Join(*attempts, ","); got != v4.String() {
		t.Fatalf("dial attempts = %q, want IPv4 first and only", got)
	}

	client, attempts = newClient(true, []netip.Addr{v4, v6}, v4)
	if _, err := client.FetchImage(t.Context(), "http://cdn.example/a.webp"); err != nil {
		t.Fatalf("FetchImage() after IPv4 dial failure = %v", err)
	}
	if got := strings.Join(*attempts, ","); got != v4.String()+","+v6.String() {
		t.Fatalf("dial attempts = %q, want fallback to the validated IPv6 address", got)
	}

	client, attempts = newClient(false, []netip.Addr{v6, v4}, v6)
	if _, err := client.FetchImage(t.Context(), "http://cdn.example/a.webp"); err == nil {
		t.Fatal("FetchImage() without PreferIPv4 succeeded, want the first address only")
	}
	if got := strings.Join(*attempts, ","); got != v6.String() {
		t.Fatalf("dial attempts = %q, want only the first resolved address", got)
	}
}

func TestFetchRejectsPrivateDNSBeforeDial(t *testing.T) {
	var calls atomic.Int32
	client := New(Config{
		Resolver: fakeResolver{
			"private.example": {netip.MustParseAddr("127.0.0.1")},
			"localhost":       {netip.MustParseAddr("127.0.0.1")},
		},
		DialContext: func(context.Context, string, string) (net.Conn, error) {
			calls.Add(1)
			return nil, errors.New("must not dial")
		},
	})
	for _, target := range []string{"http://private.example/", "http://localhost/"} {
		_, err := client.Fetch(t.Context(), target)
		if !IsClass(err, ErrorBlocked) {
			t.Fatalf("Fetch(%q) error = %v, want blocked", target, err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("dial calls = %d, want 0", calls.Load())
	}
}

func TestFetchRejectsNonPublicLiteralTargetsBeforeDial(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/",
		"http://[::1]/",
		"http://10.0.0.1/",
		"http://172.16.0.1/",
		"http://192.168.0.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://[fc00::1]/",
		"http://[fe80::1]/",
		"http://224.0.0.1/",
		"http://0.1.2.3/",
		"http://192.88.99.1/",
		"http://[64:ff9b:1::1]/",
		"http://[100::1]/",
		"http://[2001::1]/",
		"http://[2002::1]/",
	}
	for _, target := range tests {
		t.Run(target, func(t *testing.T) {
			var calls atomic.Int32
			client := New(Config{DialContext: func(context.Context, string, string) (net.Conn, error) {
				calls.Add(1)
				return nil, errors.New("must not dial")
			}})
			_, err := client.Fetch(t.Context(), target)
			if !IsClass(err, ErrorBlocked) {
				t.Fatalf("Fetch() error = %v, want blocked", err)
			}
			if calls.Load() != 0 {
				t.Fatalf("dial calls = %d, want 0", calls.Load())
			}
		})
	}
}

func TestFetchDialsOnlyTheValidatedAddress(t *testing.T) {
	var dialed string
	client := New(Config{
		Resolver: fakeResolver{"rebind.example": {publicAddr()}},
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, errors.New("stop after observing dial target")
		},
	})
	_, _ = client.Fetch(t.Context(), "https://rebind.example/article")
	if dialed != "93.184.216.34:443" {
		t.Fatalf("dial target = %q, want checked IP instead of hostname", dialed)
	}
}

func TestFetchRechecksRedirectDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://private.example/admin", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver: fakeResolver{
			"public.example":  {publicAddr()},
			"private.example": {netip.MustParseAddr("10.0.0.1")},
		},
		DialContext: mappedDialer(server.Listener.Addr().String(), &calls),
	})
	_, err := client.Fetch(t.Context(), "http://public.example/")
	if !IsClass(err, ErrorBlocked) {
		t.Fatalf("Fetch() redirect error = %v, want blocked", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("dial calls = %d, want only public hop", calls.Load())
	}
}

func TestFetchEnforcesRedirectLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		step := strings.Count(r.URL.Path, "next")
		if step < 4 {
			http.Redirect(w, r, strings.TrimRight(r.URL.Path, "/")+"/next", http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "text/html")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"redirect.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		MaxRedirects: 3,
	})
	_, err := client.Fetch(t.Context(), "http://redirect.example/")
	if !IsClass(err, ErrorRedirect) {
		t.Fatalf("Fetch() error = %v, want redirect", err)
	}
}

func TestFetchCapsDecompressedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		writer := gzip.NewWriter(w)
		_, _ = writer.Write([]byte(strings.Repeat("x", 1025)))
		_ = writer.Close()
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"large.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		MaxBodyBytes: 1024,
	})
	_, err := client.Fetch(t.Context(), "http://large.example/")
	if !IsClass(err, ErrorTooLarge) {
		t.Fatalf("Fetch() error = %v, want too_large", err)
	}
}

func TestFetchRejectsOversizedContentLengthBeforeReading(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "2048")
		_, _ = io.WriteString(w, strings.Repeat("x", 2048))
	}))
	t.Cleanup(server.Close)

	client := New(Config{
		Resolver:     fakeResolver{"large.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &atomic.Int32{}),
		MaxBodyBytes: 1024,
	})
	_, err := client.Fetch(t.Context(), "http://large.example/")
	if !IsClass(err, ErrorTooLarge) {
		t.Fatalf("Fetch() error = %v, want too_large", err)
	}
}

func TestFetchHonorsTotalTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "text/html")
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{
		Resolver:     fakeResolver{"slow.example": {publicAddr()}},
		DialContext:  mappedDialer(server.Listener.Addr().String(), &calls),
		TotalTimeout: 20 * time.Millisecond,
	})
	_, err := client.Fetch(t.Context(), "http://slow.example/")
	if !IsClass(err, ErrorTimeout) {
		t.Fatalf("Fetch() error = %v, want timeout", err)
	}
}

func TestWebhookRejectsUnsafeTargetsAndPrivateDNS(t *testing.T) {
	for _, raw := range []string{"http://example.com/hook", "https://example.com:444/hook", "https://example.com/hook?token=x", "https://user:secret@example.com/hook", "https://example.com/hook#x", "https://[::ffff:8.8.8.8]/", "https://127.1/hook", "https://0177.0.0.1/hook"} {
		if ValidateWebhookURL(raw) == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
	var calls atomic.Int32
	client := New(Config{Resolver: fakeResolver{"mixed.example": {publicAddr(), netip.MustParseAddr("127.0.0.1")}}, DialContext: func(context.Context, string, string) (net.Conn, error) {
		calls.Add(1)
		return nil, errors.New("must not dial")
	}})
	_, err := client.PostJSON(t.Context(), "https://mixed.example/hook", []byte(`{}`), nil)
	if !IsClass(err, ErrorBlocked) || calls.Load() != 0 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestWebhookJSONTLSRedirectIsNotFollowedAndDNSRevalidated(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Ambient-Credential") != "" || r.Header.Get("Webhook-Delivery-Id") != "19" {
			t.Error("invalid JSON POST or leaked ambient credential")
		}
		w.Header().Set("Location", "https://127.0.0.1/private")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	certificate := server.Certificate()
	host := certificate.DNSNames[0]
	host = strings.Replace(host, "*.", "hook.", 1)
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	var calls atomic.Int32
	resolver := fakeResolver{host: {publicAddr()}}
	client := New(Config{Resolver: resolver, DialContext: mappedDialer(server.Listener.Addr().String(), &calls)})
	client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	headers := http.Header{"X-Ambient-Credential": []string{"never-forward-this"}, "Webhook-Id": []string{"evt_test"}, "Webhook-Delivery-Id": []string{"19"}}
	result, err := client.PostJSON(t.Context(), "https://"+host+"/hook", []byte(`{"id":"evt_test"}`), headers)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusTemporaryRedirect || result.RetryAfter != "120" || calls.Load() != 1 || received.Load() != 1 {
		t.Fatalf("result=%#v dials=%d requests=%d", result, calls.Load(), received.Load())
	}
	resolver[host] = []netip.Addr{netip.MustParseAddr("127.0.0.1")}
	if _, err := client.PostJSON(t.Context(), "https://"+host+"/hook", []byte(`{}`), nil); !IsClass(err, ErrorBlocked) {
		t.Fatalf("DNS rebind=%v", err)
	}
	if calls.Load() != 1 {
		t.Fatal("rebound host was dialed")
	}
}

// The static Authorization header is the only caller credential PostJSON puts on
// the wire: any other caller header stays local, and no Authorization is sent
// unless the caller set it explicitly.
func TestPostJSONForwardsOnlyExplicitAuthorization(t *testing.T) {
	type seenHeaders struct{ authorization, ambient string }
	var seen seenHeaders
	var received atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		seen = seenHeaders{authorization: r.Header.Get("Authorization"), ambient: r.Header.Get("X-Ambient-Credential")}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	certificate := server.Certificate()
	host := strings.Replace(certificate.DNSNames[0], "*.", "hook.", 1)
	pool := x509.NewCertPool()
	pool.AddCert(certificate)
	var calls atomic.Int32
	client := New(Config{Resolver: fakeResolver{host: {publicAddr()}}, DialContext: mappedDialer(server.Listener.Addr().String(), &calls)})
	client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
	headers := http.Header{"Authorization": []string{"Bearer agt_static_token"}, "X-Ambient-Credential": []string{"never-forward-this"}, "Webhook-Id": []string{"evt_test"}}
	result, err := client.PostJSON(t.Context(), "https://"+host+"/hook", []byte(`{"id":"evt_test"}`), headers)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusNoContent || seen.authorization != "Bearer agt_static_token" || seen.ambient != "" {
		t.Fatalf("result=%#v headers=%#v", result, seen)
	}
	if _, err := client.PostJSON(t.Context(), "https://"+host+"/hook", []byte(`{"id":"evt_none"}`), nil); err != nil {
		t.Fatal(err)
	}
	if seen.authorization != "" {
		t.Fatalf("Authorization = %q without an explicit header", seen.authorization)
	}
	if received.Load() != 2 {
		t.Fatalf("requests=%d", received.Load())
	}
}

func TestWebhookRejectsMappedPublicAddressFromDNS(t *testing.T) {
	var calls atomic.Int32
	client := New(Config{Resolver: fakeResolver{"mapped.example": {netip.MustParseAddr("::ffff:8.8.8.8")}}, DialContext: func(context.Context, string, string) (net.Conn, error) {
		calls.Add(1)
		return nil, errors.New("must not dial")
	}})
	if _, err := client.PostJSON(t.Context(), "https://mapped.example/hook", []byte(`{}`), nil); !IsClass(err, ErrorBlocked) || calls.Load() != 0 {
		t.Fatalf("mapped DNS err=%v dials=%d", err, calls.Load())
	}
}

func TestWebhookRetainsResponseMetadataWhenBodyFails(t *testing.T) {
	for _, mode := range []string{"content_length", "chunked", "truncated"} {
		for _, status := range []int{http.StatusOK, http.StatusGone, http.StatusTooManyRequests} {
			t.Run(fmt.Sprintf("%s/%d", mode, status), func(t *testing.T) {
				server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Retry-After", "3600")
					switch mode {
					case "content_length":
						w.Header().Set("Content-Length", "65")
					case "truncated":
						w.Header().Set("Content-Length", "16")
					}
					w.WriteHeader(status)
					if mode == "chunked" {
						w.(http.Flusher).Flush()
					}
					if mode == "truncated" {
						_, _ = w.Write([]byte("short"))
					} else {
						_, _ = w.Write([]byte(strings.Repeat("x", 65)))
					}
				}))
				defer server.Close()
				certificate := server.Certificate()
				host := strings.Replace(certificate.DNSNames[0], "*.", "hook.", 1)
				pool := x509.NewCertPool()
				pool.AddCert(certificate)
				var calls atomic.Int32
				client := New(Config{Resolver: fakeResolver{host: {publicAddr()}}, DialContext: mappedDialer(server.Listener.Addr().String(), &calls), MaxBodyBytes: 64})
				client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
				result, err := client.PostJSON(t.Context(), "https://"+host+"/hook", []byte(`{}`), nil)
				if err == nil || (mode != "truncated" && !IsClass(err, ErrorTooLarge)) {
					t.Fatalf("body limit/read error = %v", err)
				}
				if result.StatusCode != status || result.RetryAfter != "3600" || result.Duration <= 0 {
					t.Fatalf("lost response metadata: %#v, err=%v", result, err)
				}
			})
		}
	}
}

func TestWebhookRawURLPolicyMatchesRequestParsing(t *testing.T) {
	for _, raw := range []string{"https://allowed.example&sol;@receiver.example:8443/hook", "https://allowed.example&sol;@receiver.example/hook"} {
		t.Run(raw, func(t *testing.T) {
			if ValidateWebhookURL(raw) == nil {
				t.Error("HTML entity changed the validated URL authority")
			}
			var calls atomic.Int32
			client := New(Config{Resolver: fakeResolver{"allowed.example": {publicAddr()}, "receiver.example": {publicAddr()}}, DialContext: func(context.Context, string, string) (net.Conn, error) {
				calls.Add(1)
				return nil, errors.New("unexpected dial")
			}})
			_, err := client.PostJSON(t.Context(), raw, []byte(`{}`), nil)
			if !IsClass(err, ErrorBlocked) || calls.Load() != 0 {
				t.Fatalf("unsafe raw authority err=%v dials=%d", err, calls.Load())
			}
		})
	}
}

// 通知目标放行 http、任意端口和 RFC1918/ULA 私网（企业内网通知是合法场景），
// 但仍必须挡住本机、链路本地、保留段与 IPv4-mapped 字面量（issue #1280）。
func TestNotifyURLPolicyAcceptsIntranetAndRejectsLocalTargets(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1:8080/hook",
		"http://localhost:8080/",
		"http://intranet.localhost/hook",
		"http://169.254.169.254/latest/meta-data/",
		"http://[::1]:9000/",
		"http://0.0.0.0:80/",
		"http://[::ffff:127.0.0.1]/",
		"http://[fe80::1]/",
		"http://224.0.0.1/",
		"http://127.1/hook",
		"http://[fe80::1%25eth0]/",
		"http://user:pw@192.168.1.10/hook",
		"http://192.168.1.10/hook#fragment",
		"ftp://192.168.1.10/hook",
		"http:///hook",
		"http://:9966/send",
	}
	for _, raw := range blocked {
		if err := ValidateNotifyURL(raw); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	allowed := []string{
		"https://open.feishu.cn/open-apis/bot/v2/hook/xxx",
		"http://192.168.1.10:9966/send",
		"http://10.1.2.3/hook",
		"http://[fd00::5]:9966/send",
		"http://notify.corp.example/hook?token=x",
	}
	for _, raw := range allowed {
		if err := ValidateNotifyURL(raw); err != nil {
			t.Errorf("rejected %q: %v", raw, err)
		}
	}
}

func TestCheckNotifyTargetValidatesDNSAnswers(t *testing.T) {
	for _, tc := range []struct {
		name    string
		address string
		blocked bool
	}{
		{"loopback", "127.0.0.1", true},
		{"link local metadata", "169.254.169.254", true},
		{"mapped loopback", "::ffff:127.0.0.1", true},
		{"intranet", "192.168.0.9", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := New(Config{Resolver: fakeResolver{"notify.example": {netip.MustParseAddr(tc.address)}}, DialContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, errors.New("must not dial")
			}})
			err := client.CheckNotifyTarget(t.Context(), "http://notify.example:9966/hook")
			if tc.blocked {
				if !IsClass(err, ErrorBlocked) {
					t.Fatalf("err = %v, want blocked", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("intranet target rejected: %v", err)
			}
		})
	}
}

func TestDoNotifyNeverDialsLoopback(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received.Add(1)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	// 真实拨号器 + 解析到 loopback 的域名：策略必须在此之前拦住，接收端一次都不能被打到。
	client := New(Config{Resolver: fakeResolver{"notify.example": {netip.MustParseAddr("127.0.0.1")}}})
	request, err := http.NewRequest(http.MethodPost, "http://notify.example:"+port(server)+"/hook", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.DoNotify(t.Context(), request, time.Second)
	if response != nil {
		_ = response.Body.Close()
	}
	if !IsClass(err, ErrorBlocked) {
		t.Fatalf("err = %v, want blocked", err)
	}
	if received.Load() != 0 {
		t.Fatalf("loopback receiver saw %d requests", received.Load())
	}
}

func TestDoNotifyPostsCallerRequestThroughPinnedDialer(t *testing.T) {
	type captured struct{ host, authorization, body string }
	var seen captured
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = captured{host: r.Host, authorization: r.Header.Get("Authorization"), body: string(body)}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{Resolver: fakeResolver{"notify.example": {publicAddr()}}, DialContext: mappedDialer(server.Listener.Addr().String(), &calls)})
	request, err := http.NewRequest(http.MethodPost, "http://notify.example:9966/send", strings.NewReader(`{"event":"topic.published"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer bot-token")
	response, err := client.DoNotify(t.Context(), request, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusAccepted || calls.Load() != 1 {
		t.Fatalf("response=%d dials=%d", response.StatusCode, calls.Load())
	}
	if seen.host != "notify.example:9966" || seen.authorization != "Bearer bot-token" || seen.body != `{"event":"topic.published"}` {
		t.Fatalf("received %#v", seen)
	}
}

// 内网私网接收端必须真的能投递：把 DoNotify 的拨号判定换成“仅公网”时这条用例会红（issue #1280）。
func TestDoNotifyDeliversToIntranetReceiver(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)

	var calls atomic.Int32
	client := New(Config{Resolver: fakeResolver{"notify.example": {netip.MustParseAddr("192.168.0.9")}}, DialContext: mappedDialer(server.Listener.Addr().String(), &calls)})
	request, err := http.NewRequest(http.MethodPost, "http://notify.example:9966/hook", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.DoNotify(t.Context(), request, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusNoContent || calls.Load() != 1 {
		t.Fatalf("response=%d dials=%d", response.StatusCode, calls.Load())
	}
}

// 投递不跟随重定向：3xx 原样交给通道判定，目标接收端只收到一次请求。
func TestDoNotifyDoesNotFollowRedirects(t *testing.T) {
	var received atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		http.Redirect(w, r, "http://notify.example:9966/elsewhere", http.StatusFound)
	}))
	t.Cleanup(server.Close)

	client := New(Config{Resolver: fakeResolver{"notify.example": {netip.MustParseAddr("192.168.0.9")}}, DialContext: mappedDialer(server.Listener.Addr().String(), &atomic.Int32{})})
	request, err := http.NewRequest(http.MethodPost, "http://notify.example:9966/hook", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.DoNotify(t.Context(), request, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusFound || received.Load() != 1 {
		t.Fatalf("response=%d requests=%d", response.StatusCode, received.Load())
	}
}

func port(server *httptest.Server) string {
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		panic(err)
	}
	return port
}

func TestFetchRequestPolicyChecksEveryHop(t *testing.T) {
	for _, destination := range []string{"https://unapproved.example.com/step", "http://approved.example.com/end", "https://approved.example.com/step"} {
		t.Run(destination, func(t *testing.T) {
			var received atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				received.Add(1)
				switch r.URL.Path {
				case "/start":
					http.Redirect(w, r, destination, http.StatusFound)
				case "/step":
					http.Redirect(w, r, "https://approved.example.com/end", http.StatusFound)
				default:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{}`))
				}
			}))
			defer server.Close()
			pool := x509.NewCertPool()
			pool.AddCert(server.Certificate())
			client := New(Config{Resolver: fakeResolver{"approved.example.com": {publicAddr()}, "unapproved.example.com": {publicAddr()}}, DialContext: mappedDialer(server.Listener.Addr().String(), &atomic.Int32{})})
			client.httpClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}
			ctx := WithURLPolicy(t.Context(), func(raw string) bool { return strings.HasPrefix(raw, "https://approved.example.com/") })
			_, err := client.FetchJSON(ctx, "https://approved.example.com/start")
			if destination == "https://approved.example.com/step" {
				if err != nil || received.Load() != 3 {
					t.Fatalf("err=%v requests=%d", err, received.Load())
				}
			} else if !IsClass(err, ErrorBlocked) || received.Load() != 1 {
				t.Fatalf("err=%v requests=%d", err, received.Load())
			}
			before := received.Load()
			if _, err := client.FetchJSON(ctx, "https://unapproved.example.com/start"); !IsClass(err, ErrorBlocked) || received.Load() != before {
				t.Fatalf("initial target err=%v requests=%d", err, received.Load())
			}
		})
	}
}
