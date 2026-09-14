package statusservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testNode = "e643a364-0372-43b2-a341-b05d858866ad"

func mockService(t *testing.T, mappedHistory ...bool) (*Service, *atomic.Bool, *atomic.Int32) {
	t.Helper()
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	var fail atomic.Bool
	var calls atomic.Int32
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if fail.Load() {
			http.Error(w, "secret-provider-error", http.StatusBadGateway)
			return
		}
		var data any
		switch r.URL.Path {
		case "/api/share/public123":
			data = map[string]any{"websiteId": testNode, "token": "fixture-secret"}
		case "/api/rpc2":
			var rpc struct {
				Method string
				Params struct {
					UUID     string
					MaxCount int
					Hours    int
				}
			}
			if err := json.NewDecoder(r.Body).Decode(&rpc); err != nil {
				t.Error(err)
			}
			if rpc.Params.UUID != testNode {
				t.Errorf("unexpected node: %s", rpc.Params.UUID)
			}
			switch rpc.Method {
			case "common:getNodes":
				data = map[string]any{"uuid": testNode, "name": "Community", "region": "KR", "cpu_cores": 2, "mem_total": 1024, "ipv4": "private-ip", "token": "private-token", "remark": "private-remark"}
			case "public:getClientRecentRecords":
				data = []any{map[string]any{"cpu": map[string]any{"usage": 12.4}, "ram": map[string]any{"total": 1024, "used": 512}, "disk": map[string]any{"total": 4096, "used": 1024}, "network": map[string]any{"up": 512, "down": 256}, "uptime": 3600, "updated_at": now.Format(time.RFC3339)}}
				if len(mappedHistory) > 1 && mappedHistory[1] {
					delete(data.([]any)[0].(map[string]any), "network")
				}
			case "common:getRecords":
				if rpc.Params.MaxCount != 120 {
					t.Error("unbounded history request")
				}
				if rpc.Params.Hours != 1 && rpc.Params.Hours != 6 && rpc.Params.Hours != 24 && rpc.Params.Hours != 168 {
					t.Errorf("unsupported history duration %d", rpc.Params.Hours)
				}
				start := now.Add(-time.Duration(rpc.Params.Hours) * time.Hour)
				var records []any
				// Include out-of-window data and more than maxCount to verify that
				// the projection stays bounded without truncating the older span.
				for i := -1; i <= 120; i++ {
					records = append(records, map[string]any{"time": start.Add(time.Duration(i) * now.Sub(start) / 120).Format(time.RFC3339), "cpu": 12.4, "ram": 512, "ram_total": 1024})
				}
				data = map[string]any{"records": records}
				if len(mappedHistory) > 0 && mappedHistory[0] {
					data = map[string]any{"records": map[string]any{testNode: []any{map[string]any{"client": testNode, "time": now.Format(time.RFC3339), "cpu": 12.4, "ram": 512, "ram_total": 0}}, "another-node": []any{map[string]any{"cpu": 99}}}}
				}
			default:
				t.Errorf("unexpected RPC %s", rpc.Method)
			}
			data = map[string]any{"jsonrpc": "2.0", "id": 1, "result": data}
		default:
			if r.Header.Get("X-Umami-Share-Token") != "fixture-secret" || r.Header.Get("X-Umami-Share-Context") != "overview" {
				t.Error("missing shared analytics context")
			}
			if r.Header.Get("Authorization") != "" {
				t.Error("admin credentials must not be used")
			}
			if !strings.HasSuffix(r.URL.Path, "/active") && r.URL.Query().Get("timezone") != "UTC" {
				t.Error("analytics bucket timezone must be explicit")
			}
			switch {
			case strings.HasSuffix(r.URL.Path, "/stats"):
				data = map[string]any{"pageviews": 100, "visitors": 10, "visits": 20, "bounces": 5, "totaltime": 1200}
			case strings.HasSuffix(r.URL.Path, "/active"):
				data = map[string]any{"visitors": 0}
			case strings.HasSuffix(r.URL.Path, "/pageviews"):
				data = map[string]any{"pageviews": []any{map[string]any{"x": "2026-09-14 11:00:00", "y": 100}}, "sessions": []any{map[string]any{"x": "2026-09-14 11:00:00", "y": 10}}}
			default:
				t.Errorf("unexpected endpoint: %s", r.URL.Path)
			}
		}
		if err := json.NewEncoder(w).Encode(data); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(upstream.Close)
	s := New(Config{Enabled: true, UmamiURL: upstream.URL, UmamiShareID: "public123", KomariURL: upstream.URL, KomariNodeID: testNode})
	s.http.Transport = upstream.Client().Transport
	s.now = func() time.Time { return now }
	return s, &fail, &calls
}

func TestPublicProjectionAndConcurrentCache(t *testing.T) {
	s, _, calls := mockService(t)
	var wg sync.WaitGroup
	for range 12 {
		wg.Go(func() { s.Get(context.Background(), "24h", "1h") })
	}
	wg.Wait()
	if n := calls.Load(); n != 7 {
		t.Fatalf("concurrent requests made %d upstream calls, want 7", n)
	}
	result := s.Get(context.Background(), "24h", "1h")
	if result.Server.State != "ok" || result.Traffic.State != "ok" {
		t.Fatalf("sources not connected: %+v", result)
	}
	if result.Server.Data.Current.CPU != 12.4 || result.Server.Data.History[0].MemoryPercent != 50 {
		t.Fatal("incorrect resource conversion")
	}
	traffic := result.Traffic.Data
	if *traffic.ActiveVisitors != 0 || *traffic.BounceRate != 25 || *traffic.AverageDuration != 60 || traffic.Series[0].Time != "2026-09-14T11:00:00Z" {
		t.Fatalf("incorrect traffic conversion: %+v", traffic)
	}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"fixture-secret", "private-", "ipv4", "remark", "token", s.config.UmamiURL, testNode} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("public response exposes %s", forbidden)
		}
	}
}

func TestFailureCacheExpiresAndRecovers(t *testing.T) {
	s, fail, calls := mockService(t)
	initial := s.Get(context.Background(), "24h", "1h")
	clock := s.now()
	s.now = func() time.Time { return clock }
	fail.Store(true)
	clock = clock.Add(31 * time.Second)
	stale := s.Get(context.Background(), "24h", "1h")
	if stale.Server.State != "stale" || stale.Traffic.State != "stale" || stale.Server.FetchedAt != initial.Server.FetchedAt {
		t.Fatalf("last successful data not preserved: %+v", stale)
	}
	count := calls.Load()
	s.Get(context.Background(), "24h", "1h")
	if calls.Load() != count {
		t.Fatal("failed source was hammered within retry TTL")
	}
	clock = clock.Add(16 * time.Minute)
	expired := s.Get(context.Background(), "24h", "1h")
	if expired.Server.State != "unavailable" || expired.Server.Data != nil || expired.Traffic.Data != nil {
		t.Fatal("expired data still exposed")
	}
	fail.Store(false)
	clock = clock.Add(31 * time.Second)
	if recovered := s.Get(context.Background(), "24h", "1h"); recovered.Server.State != "ok" || recovered.Traffic.State != "ok" {
		t.Fatal("sources failed to recover")
	}
}

func TestTrafficRangesAndIndependentSources(t *testing.T) {
	s, _, calls := mockService(t)
	for _, period := range []string{"24h", "7d", "30d"} {
		result := s.Get(context.Background(), period, "1h")
		if result.Range != period || result.Traffic.State != "ok" {
			t.Fatal("range unavailable")
		}
		start, _ := time.Parse(time.RFC3339, result.Traffic.Data.StartAt)
		if int(s.now().Sub(start).Hours()) != map[string]int{"24h": 24, "7d": 168, "30d": 720}[period] {
			t.Fatal("wrong time range")
		}
	}
	if calls.Load() != 15 {
		t.Fatalf("range change should reuse node cache, got %d calls", calls.Load())
	}
	s2 := New(Config{Enabled: true, KomariURL: s.config.KomariURL, KomariNodeID: testNode, UmamiURL: "https://bad.invalid?token=private", UmamiShareID: "public123"})
	s2.http.Transport = s.http.Transport
	s2.now = s.now
	result := s2.Get(context.Background(), "24h", "1h")
	if result.Server.State != "ok" || result.Traffic.State != "unavailable" {
		t.Fatal("traffic failure hides server data")
	}
}

func TestServerRangesPreserveSpanAndIndependentCaches(t *testing.T) {
	s, _, calls := mockService(t)
	for period, hours := range map[string]int{"1h": 1, "6h": 6, "24h": 24, "7d": 168} {
		result := s.Get(context.Background(), "24h", period)
		if result.ServerRange != period || result.Range != "24h" || result.Server.State != "ok" {
			t.Fatalf("wrong range response: %+v", result)
		}
		history := result.Server.Data.History
		if len(history) != 120 || history[0].Time != s.now().Add(-time.Duration(hours)*time.Hour).Format(time.RFC3339) || history[119].Time != s.now().Format(time.RFC3339) {
			t.Fatalf("%s history did not preserve the selected span: %d points", period, len(history))
		}
		if result.Server.Data.Current.ObservedAt != s.now().Format(time.RFC3339) {
			t.Fatal("current metrics must still use the latest sample")
		}
	}
	if calls.Load() != 16 {
		t.Fatalf("four resource scopes should reuse traffic cache: %d upstream calls", calls.Load())
	}
	for _, period := range []string{"1h", "6h", "24h", "7d"} {
		s.Get(context.Background(), "7d", period)
	}
	if calls.Load() != 20 {
		t.Fatalf("traffic range changes should reuse all resource caches: %d upstream calls", calls.Load())
	}
}

func TestCancellationDoesNotPoisonCache(t *testing.T) {
	var cache sourceCache[int]
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		cache.get(ctx, time.Now, func(ctx context.Context) (*int, error) { close(started); <-ctx.Done(); return nil, ctx.Err() })
		close(done)
	}()
	<-started
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancellation did not stop fetch")
	}
	value := 42
	result := cache.get(context.Background(), time.Now, func(context.Context) (*int, error) { return &value, nil })
	if result.State != "ok" || *result.Data != 42 {
		t.Fatal("cancelled request poisoned cache")
	}
}

func TestUnconfiguredAndValidation(t *testing.T) {
	result := New(Config{}).Get(context.Background(), "24h", "1h")
	if result.Server.State != "unconfigured" || result.Traffic.State != "unconfigured" {
		t.Fatal("missing config must not imply zero/healthy")
	}
	for _, u := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com/api", "https://example.com?url=x", "https://example.com#x"} {
		if validOrigin(u) {
			t.Errorf("accepted invalid origin %s", u)
		}
	}
	for _, id := range []string{"..", "test/id123", "1234567", strings.Repeat("a", 51)} {
		if validShareID(id) {
			t.Errorf("accepted invalid share %s", id)
		}
	}
	for _, period := range []string{"", "365d", "http://localhost"} {
		if ValidRange(period) {
			t.Errorf("accepted range %s", period)
		}
	}
}

func TestUmamiNumericVersions(t *testing.T) {
	for _, raw := range []string{`0`, `42`, `{"value":42}`} {
		var n umamiCount
		if err := json.Unmarshal([]byte(raw), &n); err != nil {
			t.Fatalf("valid count %s: %v", raw, err)
		}
	}
	for _, raw := range []string{`{}`, `{"value":null}`, `-1`, `"42"`} {
		var n umamiCount
		if json.Unmarshal([]byte(raw), &n) == nil {
			t.Errorf("accepted malformed count %s", raw)
		}
	}
}

func TestProviderRedirectDoesNotForwardShareToken(t *testing.T) {
	var received atomic.Bool
	target := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received.Store(true) }))
	defer target.Close()
	redirect := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redirect.Close()
	s := New(Config{})
	s.http.Transport = redirect.Client().Transport
	var result any
	err := s.request(context.Background(), http.MethodGet, redirect.URL, http.Header{"X-Umami-Share-Token": {"secret"}}, nil, &result)
	if err == nil || received.Load() {
		t.Fatal("provider redirect followed")
	}
}
