package statusservice

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Uptime struct {
	StatusPageURL string          `json:"statusPageUrl"`
	Monitors      []UptimeMonitor `json:"monitors"`
}

type UptimeMonitor struct {
	ID        int64             `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Uptime24h *float64          `json:"uptime24h"`
	Current   *UptimeHeartbeat  `json:"current"`
	History   []UptimeHeartbeat `json:"history"`
}

type UptimeHeartbeat struct {
	Time   string   `json:"time"`
	Status string   `json:"status"`
	Ping   *float64 `json:"ping"`
}

func validUptimeSlug(value string) bool {
	return len(value) > 0 && len(value) <= 100 && strings.IndexFunc(value, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_'
	}) == -1
}

func (s *Service) fetchUptime(ctx context.Context) (*Uptime, error) {
	if !validOrigin(s.config.UptimeURL) || !validUptimeSlug(s.config.UptimeSlug) {
		return nil, errProvider
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	base := strings.TrimRight(s.config.UptimeURL, "/")
	var page struct {
		Config struct {
			Published bool
			Slug      string
		}
		PublicGroupList []struct {
			MonitorList []struct {
				ID         int64
				Name, Type string
			}
		}
	}
	var beats struct {
		HeartbeatList map[string][]struct {
			Time   string
			Status *int
			Ping   *float64
		}
		UptimeList map[string]*float64
	}
	var pageErr, beatsErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		pageErr = s.request(ctx, http.MethodGet, base+"/api/status-page/"+s.config.UptimeSlug, nil, nil, &page)
	})
	wg.Go(func() {
		beatsErr = s.request(ctx, http.MethodGet, base+"/api/status-page/heartbeat/"+s.config.UptimeSlug, nil, nil, &beats)
	})
	wg.Wait()
	if pageErr != nil || beatsErr != nil || !page.Config.Published || page.Config.Slug != s.config.UptimeSlug || page.PublicGroupList == nil || beats.HeartbeatList == nil {
		return nil, errProvider
	}
	result := &Uptime{StatusPageURL: base + "/status/" + s.config.UptimeSlug, Monitors: []UptimeMonitor{}}
	seen := make(map[int64]bool)
	for _, group := range page.PublicGroupList {
		for _, item := range group.MonitorList {
			if item.ID <= 0 || item.Name == "" || seen[item.ID] {
				continue
			}
			if len(result.Monitors) >= 50 {
				return nil, errProvider
			}
			seen[item.ID] = true
			id := strconv.FormatInt(item.ID, 10)
			monitor := UptimeMonitor{ID: item.ID, Name: item.Name, Type: item.Type, History: []UptimeHeartbeat{}}
			if ratio := beats.UptimeList[id+"_24"]; ratio != nil && *ratio >= 0 && *ratio <= 1 {
				percent := *ratio * 100
				monitor.Uptime24h = &percent
			}
			invalidTime := false
			for _, beat := range beats.HeartbeatList[id] {
				when, err := parseUptimeTime(beat.Time)
				if err != nil || when.After(s.now().Add(time.Minute)) {
					invalidTime = true
					continue
				}
				state := "unknown"
				if beat.Status != nil {
					if value, ok := map[int]string{0: "down", 1: "up", 2: "pending", 3: "maintenance"}[*beat.Status]; ok {
						state = value
					}
				}
				ping := beat.Ping
				if ping != nil && (*ping < 0 || math.IsNaN(*ping) || math.IsInf(*ping, 0)) {
					ping = nil
				}
				monitor.History = append(monitor.History, UptimeHeartbeat{Time: when.UTC().Format(time.RFC3339Nano), Status: state, Ping: ping})
			}
			sort.SliceStable(monitor.History, func(i, j int) bool {
				a, _ := time.Parse(time.RFC3339Nano, monitor.History[i].Time)
				b, _ := time.Parse(time.RFC3339Nano, monitor.History[j].Time)
				return a.Before(b)
			})
			if len(monitor.History) > 100 {
				monitor.History = monitor.History[len(monitor.History)-100:]
			}
			// Malformed timestamps cannot promote an older healthy beat to current.
			if len(monitor.History) > 0 && !invalidTime {
				latest := monitor.History[len(monitor.History)-1]
				monitor.Current = &latest
			}
			result.Monitors = append(result.Monitors, monitor)
		}
	}
	return result, nil
}

func parseUptimeTime(raw string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed, nil
	}
	// Uptime Kuma's public API serializes database timestamps in UTC without a zone.
	return time.Parse("2006-01-02 15:04:05", raw)
}
