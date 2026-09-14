package statusservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func uptimeFixture() (map[string]any, map[string]any) {
	page := map[string]any{"config": map[string]any{"slug": "a", "published": true, "customCSS": "private-css"}, "publicGroupList": []any{map[string]any{"monitorList": []any{map[string]any{"id": 5, "name": "Community", "type": "http", "url": "private-target"}}}}}
	beats := map[string]any{"heartbeatList": map[string]any{"5": []any{
		map[string]any{"time": "2026-09-14 12:00:00.041", "status": 1, "ping": 628, "msg": "private-message"},
		map[string]any{"time": "2026-09-14T13:59:59.999+02:00", "status": 0, "ping": nil},
	}, "99": []any{map[string]any{"status": 0, "msg": "private-monitor"}}}, "uptimeList": map[string]any{"5_24": 0.9995}}
	return page, beats
}

func mockUptime(t *testing.T, page, beats map[string]any) (*Service, *atomic.Bool, *atomic.Int32) {
	t.Helper()
	p, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(beats)
	if err != nil {
		t.Fatal(err)
	}
	var failed atomic.Bool
	var calls atomic.Int32
	upstream := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.Header.Get("Authorization") != "" {
			t.Error("uptime must use public read-only access")
		}
		if failed.Load() {
			http.Error(w, "private-provider-error", http.StatusBadGateway)
			return
		}
		switch r.URL.Path {
		case "/api/status-page/a":
			_, _ = w.Write(p)
		case "/api/status-page/heartbeat/a":
			_, _ = w.Write(b)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(upstream.Close)
	s := New(Config{Enabled: true, UptimeURL: upstream.URL, UptimeSlug: "a"})
	s.http.Transport = upstream.Client().Transport
	s.now = func() time.Time { return time.Date(2026, 9, 14, 12, 0, 1, 0, time.UTC) }
	return s, &failed, &calls
}

func TestUptimePublicProjectionCacheAndFailure(t *testing.T) {
	p, b := uptimeFixture()
	s, failed, calls := mockUptime(t, p, b)
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { s.Get(context.Background(), "24h", "1h") })
	}
	wg.Wait()
	got := s.Get(context.Background(), "7d", "7d").Uptime
	if got.State != "ok" || calls.Load() != 2 || len(got.Data.Monitors) != 1 {
		t.Fatalf("public monitors or shared cache incorrect: %+v, calls=%d", got, calls.Load())
	}
	m := got.Data.Monitors[0]
	if *m.Uptime24h != 99.95 || m.Current.Status != "up" || *m.Current.Ping != 628 || m.History[0].Status != "down" || m.Current.Time != "2026-09-14T12:00:00.041Z" {
		t.Fatalf("bad conversion or timestamp ordering: %+v", m)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-") {
		t.Fatalf("non-public fields leaked: %s", raw)
	}
	clock := s.now().Add(31 * time.Second)
	s.now = func() time.Time { return clock }
	failed.Store(true)
	if stale := s.Get(context.Background(), "24h", "1h").Uptime; stale.State != "stale" || stale.FetchedAt != got.FetchedAt {
		t.Fatal("uptime failure should preserve marked stale data")
	}
	n := calls.Load()
	s.Get(context.Background(), "30d", "6h")
	if calls.Load() != n {
		t.Fatal("failed cache should be shared across ranges")
	}
	clock = clock.Add(16 * time.Minute)
	if expired := s.Get(context.Background(), "24h", "1h").Uptime; expired.State != "unavailable" || expired.Data != nil {
		t.Fatal("expired uptime remained available")
	}
	failed.Store(false)
	clock = clock.Add(31 * time.Second)
	if s.Get(context.Background(), "24h", "1h").Uptime.State != "ok" {
		t.Fatal("uptime did not recover")
	}
}

func TestUptimeMissingAndNonHealthyStates(t *testing.T) {
	for _, state := range []int{0, 1, 2, 3, 99} {
		t.Run(strconv.Itoa(state), func(t *testing.T) {
			p, b := uptimeFixture()
			beat := b["heartbeatList"].(map[string]any)["5"].([]any)[0].(map[string]any)
			beat["status"], beat["ping"] = state, 0
			b["uptimeList"].(map[string]any)["5_24"] = 0
			s, _, _ := mockUptime(t, p, b)
			m := s.Get(context.Background(), "24h", "1h").Uptime.Data.Monitors[0]
			want := map[int]string{0: "down", 1: "up", 2: "pending", 3: "maintenance", 99: "unknown"}[state]
			if m.Current.Status != want || *m.Current.Ping != 0 || *m.Uptime24h != 0 {
				t.Fatal("states or real zero were lost")
			}
		})
	}
	for _, variant := range []string{"missingStatus", "invalidTime", "missingMonitor", "missingHeartbeatList", "unpublished", "invalidRatio"} {
		t.Run(variant, func(t *testing.T) {
			p, b := uptimeFixture()
			beat := b["heartbeatList"].(map[string]any)["5"].([]any)[0].(map[string]any)
			switch variant {
			case "missingStatus":
				delete(beat, "status")
			case "invalidTime":
				beat["time"] = "bad-time"
			case "missingMonitor":
				delete(b["heartbeatList"].(map[string]any), "5")
			case "missingHeartbeatList":
				delete(b, "heartbeatList")
			case "unpublished":
				p["config"].(map[string]any)["published"] = false
			case "invalidRatio":
				b["uptimeList"].(map[string]any)["5_24"] = 2
			}
			s, _, _ := mockUptime(t, p, b)
			got := s.Get(context.Background(), "24h", "1h").Uptime
			if variant == "unpublished" || variant == "missingHeartbeatList" {
				if got.State != "unavailable" {
					t.Fatal("bad page accepted")
				}
				return
			}
			m := got.Data.Monitors[0]
			if variant == "invalidRatio" {
				if m.Uptime24h != nil {
					t.Fatal("invalid ratio accepted")
				}
				return
			}
			if m.Current != nil && m.Current.Status != "unknown" {
				t.Fatal("missing data became healthy")
			}
		})
	}
}

func TestUptimeHistoryBoundsAndConfiguration(t *testing.T) {
	p, b := uptimeFixture()
	var history []any
	for i := 0; i < 120; i++ {
		history = append(history, map[string]any{"time": time.Date(2026, 9, 14, 10, i, 0, 0, time.UTC).Format(time.RFC3339), "status": 1, "ping": 0})
	}
	b["heartbeatList"].(map[string]any)["5"] = history
	s, _, calls := mockUptime(t, p, b)
	got := s.Get(context.Background(), "24h", "1h").Uptime
	if len(got.Data.Monitors[0].History) != 100 || got.Data.Monitors[0].History[0].Time != "2026-09-14T10:20:00Z" {
		t.Fatal("recent history was not bounded to the latest 100 checks")
	}
	for _, slug := range []string{"../private", "a?token=x", "a/b", strings.Repeat("a", 101)} {
		bad := New(Config{Enabled: true, UptimeURL: s.config.UptimeURL, UptimeSlug: slug})
		if bad.Get(context.Background(), "24h", "1h").Uptime.State != "unavailable" {
			t.Fatal("invalid slug accepted")
		}
	}
	if calls.Load() != 2 {
		t.Fatal("invalid slugs reached the provider")
	}
	var monitors []any
	for i := 1; i <= 51; i++ {
		monitors = append(monitors, map[string]any{"id": i, "name": "monitor", "type": "http"})
	}
	p["publicGroupList"].([]any)[0].(map[string]any)["monitorList"] = monitors
	bounded, _, _ := mockUptime(t, p, b)
	if bounded.Get(context.Background(), "24h", "1h").Uptime.State != "unavailable" {
		t.Fatal("unbounded monitor list accepted")
	}
}
