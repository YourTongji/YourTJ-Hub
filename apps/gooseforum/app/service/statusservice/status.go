// Package statusservice publishes an allowlisted projection of public Umami,
// Komari and Uptime Kuma data. It never exposes raw provider responses or credentials.
package statusservice

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/google/uuid"
)

const (
	cacheTTL       = 30 * time.Second
	staleTTL       = 15 * time.Minute
	requestTimeout = 8 * time.Second
)

type Config struct {
	Enabled      bool
	UmamiURL     string
	UmamiShareID string
	KomariURL    string
	KomariNodeID string
	UptimeURL    string
	UptimeSlug   string
}

func LoadConfig() Config {
	return Config{
		Enabled:      preferences.GetBool("status.enabled", false),
		UmamiURL:     preferences.GetString("status.umami_url"),
		UmamiShareID: preferences.GetString("status.umami_share_id"),
		KomariURL:    preferences.GetString("status.komari_url"),
		KomariNodeID: preferences.GetString("status.komari_node_id"),
		UptimeURL:    preferences.GetString("status.uptime_url"),
		UptimeSlug:   preferences.GetString("status.uptime_slug"),
	}
}

type Source[T any] struct {
	State     string `json:"state"`
	FetchedAt string `json:"fetchedAt,omitempty"`
	Data      *T     `json:"data"`
}

type Snapshot struct {
	Range        string          `json:"range"`
	ServerRange  string          `json:"serverRange"`
	RefreshAfter int             `json:"refreshAfter"`
	Server       Source[Server]  `json:"server"`
	Traffic      Source[Traffic] `json:"traffic"`
	Uptime       Source[Uptime]  `json:"uptime"`
}

type Server struct {
	Name             string      `json:"name"`
	Region           string      `json:"region"`
	CPUCores         int         `json:"cpuCores"`
	Current          *Sample     `json:"current"`
	History          []LoadPoint `json:"history"`
	HistoryAvailable bool        `json:"historyAvailable"`
}

type Sample struct {
	ObservedAt  string  `json:"observedAt"`
	CPU         float64 `json:"cpu"`
	MemoryUsed  int64   `json:"memoryUsed"`
	MemoryTotal int64   `json:"memoryTotal"`
	DiskUsed    int64   `json:"diskUsed"`
	DiskTotal   int64   `json:"diskTotal"`
	NetworkUp   int64   `json:"networkUp"`
	NetworkDown int64   `json:"networkDown"`
	Uptime      int64   `json:"uptime"`
}

type LoadPoint struct {
	Time          string  `json:"time"`
	CPU           float64 `json:"cpu"`
	MemoryPercent float64 `json:"memoryPercent"`
}

type Traffic struct {
	StartAt         string         `json:"startAt"`
	EndAt           string         `json:"endAt"`
	Visitors        int64          `json:"visitors"`
	Pageviews       int64          `json:"pageviews"`
	Visits          int64          `json:"visits"`
	BounceRate      *float64       `json:"bounceRate"`
	AverageDuration *float64       `json:"averageDuration"`
	ActiveVisitors  *int64         `json:"activeVisitors"`
	Series          []TrafficPoint `json:"series"`
	SeriesAvailable bool           `json:"seriesAvailable"`
}

type TrafficPoint struct {
	Time      string `json:"time"`
	Pageviews int64  `json:"pageviews"`
	Visitors  int64  `json:"visitors"`
}

// A failed refresh also gets the short TTL. Cancellable waiters share one fetch;
// an abandoned request cannot poison the cache or start detached network work.
type sourceCache[T any] struct {
	mu          sync.Mutex
	pending     chan struct{}
	value       *T
	fetchedAt   time.Time
	attemptedAt time.Time
	failed      bool
}

func (c *sourceCache[T]) get(ctx context.Context, now func() time.Time, fetch func(context.Context) (*T, error)) Source[T] {
	for {
		c.mu.Lock()
		if ctx.Err() != nil || (!c.attemptedAt.IsZero() && now().Sub(c.attemptedAt) < cacheTTL) {
			result := c.snapshot(now())
			c.mu.Unlock()
			return result
		}
		if pending := c.pending; pending != nil {
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return Source[T]{State: "unavailable"}
			case <-pending:
				continue
			}
		}
		c.pending = make(chan struct{})
		c.mu.Unlock()
		value, err := fetch(ctx)
		c.mu.Lock()
		if ctx.Err() == nil {
			c.attemptedAt = now()
			c.failed = err != nil
			if err == nil {
				c.value, c.fetchedAt = value, now()
			}
		}
		close(c.pending)
		c.pending = nil
		result := c.snapshot(now())
		c.mu.Unlock()
		return result
	}
}

func (c *sourceCache[T]) snapshot(now time.Time) Source[T] {
	if c.value == nil || now.Sub(c.fetchedAt) > staleTTL {
		return Source[T]{State: "unavailable"}
	}
	state := "ok"
	if c.failed || now.Sub(c.fetchedAt) >= cacheTTL {
		state = "stale"
	}
	return Source[T]{State: state, FetchedAt: c.fetchedAt.UTC().Format(time.RFC3339), Data: c.value}
}

type Service struct {
	config  Config
	http    *http.Client
	now     func() time.Time
	server  [4]sourceCache[Server]
	traffic [3]sourceCache[Traffic]
	uptime  sourceCache[Uptime]
}

func New(config Config) *Service {
	return &Service{config: config, now: time.Now, http: &http.Client{
		Timeout: requestTimeout,
		// Share tokens must never follow a redirect to another host.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

var defaultService = sync.OnceValue(func() *Service { return New(LoadConfig()) })

func Public(ctx context.Context, period, serverPeriod string) Snapshot {
	return defaultService().Get(ctx, period, serverPeriod)
}

func ValidRange(period string) bool { return period == "24h" || period == "7d" || period == "30d" }

func serverHours(period string) int {
	return map[string]int{"1h": 1, "6h": 6, "24h": 24, "7d": 168}[period]
}

func ValidServerRange(period string) bool { return serverHours(period) != 0 }

func (s *Service) Get(ctx context.Context, period, serverPeriod string) Snapshot {
	if !ValidRange(period) {
		period = "24h"
	}
	if !ValidServerRange(serverPeriod) {
		serverPeriod = "1h"
	}
	result := Snapshot{Range: period, ServerRange: serverPeriod, RefreshAfter: 30, Server: Source[Server]{State: "unconfigured"}, Traffic: Source[Traffic]{State: "unconfigured"}, Uptime: Source[Uptime]{State: "unconfigured"}}
	if !s.config.Enabled {
		return result
	}
	var wg sync.WaitGroup
	if s.config.UptimeURL != "" && s.config.UptimeSlug != "" {
		wg.Go(func() { result.Uptime = s.uptime.get(ctx, s.now, s.fetchUptime) })
	}
	if s.config.KomariURL != "" && s.config.KomariNodeID != "" {
		index := map[string]int{"1h": 0, "6h": 1, "24h": 2, "7d": 3}[serverPeriod]
		wg.Go(func() {
			result.Server = s.server[index].get(ctx, s.now, func(ctx context.Context) (*Server, error) { return s.fetchServer(ctx, serverPeriod) })
		})
	}
	if s.config.UmamiURL != "" && s.config.UmamiShareID != "" {
		index := map[string]int{"24h": 0, "7d": 1, "30d": 2}[period]
		wg.Go(func() {
			result.Traffic = s.traffic[index].get(ctx, s.now, func(ctx context.Context) (*Traffic, error) { return s.fetchTraffic(ctx, period) })
		})
	}
	wg.Wait()
	return result
}

// Provider origins come exclusively from operator configuration, never queries.
func validOrigin(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != "" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/")
}

func validNodeID(id string) bool { _, err := uuid.Parse(id); return err == nil }

func validShareID(id string) bool {
	if len(id) < 8 || len(id) > 50 {
		return false
	}
	return strings.IndexFunc(id, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9')
	}) == -1
}
