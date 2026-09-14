package statusservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var errProvider = errors.New("status source unavailable")

func (s *Service) request(ctx context.Context, method, endpoint string, headers http.Header, body any, result any) error {
	var input io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return errProvider
		}
		input = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, input)
	if err != nil {
		return errProvider
	}
	req.Header = headers.Clone()
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := s.http.Do(req)
	if err != nil {
		return errProvider
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return errProvider
	}
	const limit = 2 << 20
	b, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil || len(b) > limit {
		return errProvider
	}
	if err := json.Unmarshal(b, result); err != nil {
		return errProvider
	}
	return nil
}

func (s *Service) rpc(ctx context.Context, method string, params any, result any) error {
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	err := s.request(ctx, http.MethodPost, strings.TrimRight(s.config.KomariURL, "/")+"/api/rpc2", nil,
		map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}, &envelope)
	if err != nil || (len(envelope.Error) > 0 && string(envelope.Error) != "null") || len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return errProvider
	}
	if json.Unmarshal(envelope.Result, result) != nil {
		return errProvider
	}
	return nil
}

type komariRecord struct {
	CPU struct {
		Usage *float64 `json:"usage"`
	} `json:"cpu"`
	RAM struct {
		Total int64
		Used  *int64
	} `json:"ram"`
	Disk struct {
		Total int64
		Used  *int64
	} `json:"disk"`
	Network   struct{ Up, Down *int64 } `json:"network"`
	Uptime    *int64                    `json:"uptime"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

type komariHistoryRecord struct {
	Client   string
	Time     time.Time
	CPU      *float64
	RAM      int64
	RAMTotal int64 `json:"ram_total"`
}

func decodeHistory(raw json.RawMessage, id string) ([]komariHistoryRecord, error) {
	var records []komariHistoryRecord
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errProvider
	}
	if raw[0] == '{' {
		var nodes map[string]json.RawMessage
		if json.Unmarshal(raw, &nodes) != nil {
			return nil, errProvider
		}
		raw = nodes[id]
		if len(raw) == 0 {
			return []komariHistoryRecord{}, nil
		}
	}
	if json.Unmarshal(raw, &records) != nil || records == nil {
		return nil, errProvider
	}
	return records, nil
}

func (s *Service) fetchServer(ctx context.Context, period string) (*Server, error) {
	if !validOrigin(s.config.KomariURL) || !validNodeID(s.config.KomariNodeID) {
		return nil, errProvider
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	id := s.config.KomariNodeID
	hours := serverHours(period)
	start := s.now().Add(-time.Duration(hours) * time.Hour)
	var node struct {
		UUID, Name, Region string
		CPUCores           int   `json:"cpu_cores"`
		MemoryTotal        int64 `json:"mem_total"`
		Hidden             bool
	}
	var recent []komariRecord
	var history struct {
		Records json.RawMessage
	}
	var nodeErr, recentErr, historyErr error
	var wg sync.WaitGroup
	wg.Go(func() { nodeErr = s.rpc(ctx, "common:getNodes", map[string]any{"uuid": id}, &node) })
	wg.Go(func() { recentErr = s.rpc(ctx, "public:getClientRecentRecords", map[string]any{"uuid": id}, &recent) })
	wg.Go(func() {
		historyErr = s.rpc(ctx, "common:getRecords", map[string]any{"uuid": id, "type": "load", "hours": hours, "maxCount": 120}, &history)
	})
	wg.Wait()
	if nodeErr != nil || recentErr != nil || node.UUID != id || node.Hidden || node.Name == "" {
		return nil, errProvider
	}
	var historyRecords []komariHistoryRecord
	if historyErr == nil {
		historyRecords, historyErr = decodeHistory(history.Records, id)
	}
	result := &Server{Name: node.Name, Region: node.Region, CPUCores: node.CPUCores, History: []LoadPoint{}, HistoryAvailable: historyErr == nil}
	// Recent records are not guaranteed to arrive in timestamp order.
	sort.Slice(recent, func(i, j int) bool { return recent[i].UpdatedAt.Before(recent[j].UpdatedAt) })
	if len(recent) > 0 {
		r := recent[len(recent)-1]
		if r.CPU.Usage == nil || r.RAM.Used == nil || r.Disk.Used == nil || r.Network.Up == nil || r.Network.Down == nil || r.Uptime == nil {
			return nil, errProvider
		}
		if r.UpdatedAt.IsZero() || r.UpdatedAt.After(s.now().Add(time.Minute)) || !validPercent(*r.CPU.Usage) || r.RAM.Total <= 0 || r.Disk.Total <= 0 || *r.RAM.Used < 0 || *r.Disk.Used < 0 || *r.Network.Up < 0 || *r.Network.Down < 0 || *r.Uptime < 0 {
			return nil, errProvider
		}
		result.Current = &Sample{ObservedAt: r.UpdatedAt.UTC().Format(time.RFC3339), CPU: *r.CPU.Usage, MemoryUsed: *r.RAM.Used, MemoryTotal: r.RAM.Total, DiskUsed: *r.Disk.Used, DiskTotal: r.Disk.Total, NetworkUp: *r.Network.Up, NetworkDown: *r.Network.Down, Uptime: *r.Uptime}
	}
	if historyErr == nil {
		for _, r := range historyRecords {
			// Older agents omit historical capacities. The chart discloses this
			// normalization against the node's current memory capacity.
			if r.RAMTotal == 0 {
				r.RAMTotal = node.MemoryTotal
			}
			if (r.Client != "" && r.Client != id) || r.CPU == nil || !validPercent(*r.CPU) || r.RAMTotal <= 0 || r.RAM < 0 || r.Time.Before(start) || r.Time.After(s.now().Add(time.Minute)) {
				continue
			}
			result.History = append(result.History, LoadPoint{Time: r.Time.UTC().Format(time.RFC3339), CPU: *r.CPU, MemoryPercent: min(100, float64(r.RAM)/float64(r.RAMTotal)*100)})
		}
		sort.Slice(result.History, func(i, j int) bool { return result.History[i].Time < result.History[j].Time })
		if len(result.History) > 120 {
			// Preserve the full time span even if a provider ignores maxCount.
			points := make([]LoadPoint, 120)
			for i := range points {
				points[i] = result.History[i*(len(result.History)-1)/(len(points)-1)]
			}
			result.History = points
		}
	}
	return result, nil
}

func validPercent(n float64) bool { return n >= 0 && n <= 100 }

// Umami v2 returns {value:n}; v3 returns n. Missing/null fields remain nil at
// their use sites, so an authentication error or changed schema cannot become 0.
type umamiCount int64

func (n *umamiCount) UnmarshalJSON(b []byte) error {
	var value int64
	if len(b) > 0 && b[0] == '{' {
		var wrapped struct {
			Value *int64 `json:"value"`
		}
		if err := json.Unmarshal(b, &wrapped); err != nil || wrapped.Value == nil {
			return errProvider
		}
		value = *wrapped.Value
	} else if err := json.Unmarshal(b, &value); err != nil {
		return errProvider
	}
	if value < 0 {
		return errProvider
	}
	*n = umamiCount(value)
	return nil
}

func (s *Service) fetchTraffic(ctx context.Context, period string) (*Traffic, error) {
	if !validOrigin(s.config.UmamiURL) || !validShareID(s.config.UmamiShareID) {
		return nil, errProvider
	}
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	base := strings.TrimRight(s.config.UmamiURL, "/")
	var share struct {
		WebsiteID string `json:"websiteId"`
		Token     string `json:"token"`
	}
	if err := s.request(ctx, http.MethodGet, base+"/api/share/"+s.config.UmamiShareID, nil, nil, &share); err != nil || !validNodeID(share.WebsiteID) || share.Token == "" {
		return nil, errProvider
	}
	headers := http.Header{"X-Umami-Share-Token": {share.Token}, "X-Umami-Share-Context": {"overview"}}
	end := s.now().UTC()
	hours := map[string]int{"24h": 24, "7d": 168, "30d": 720}[period]
	start := end.Add(-time.Duration(hours) * time.Hour)
	unit := "day"
	if period == "24h" {
		unit = "hour"
	}
	query := url.Values{"startAt": {strconv.FormatInt(start.UnixMilli(), 10)}, "endAt": {strconv.FormatInt(end.UnixMilli(), 10)}, "timezone": {"UTC"}, "unit": {unit}}
	endpoint := base + "/api/websites/" + share.WebsiteID
	var stats struct{ Pageviews, Visitors, Visits, Bounces, Totaltime *umamiCount }
	var active struct{ Visitors *umamiCount }
	type bucket struct {
		X string
		Y int64
	}
	var series struct{ Pageviews, Sessions []bucket }
	var statsErr, activeErr, seriesErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		statsErr = s.request(ctx, http.MethodGet, endpoint+"/stats?"+query.Encode(), headers, nil, &stats)
	})
	wg.Go(func() { activeErr = s.request(ctx, http.MethodGet, endpoint+"/active", headers, nil, &active) })
	wg.Go(func() {
		seriesErr = s.request(ctx, http.MethodGet, endpoint+"/pageviews?"+query.Encode(), headers, nil, &series)
	})
	wg.Wait()
	if statsErr != nil || stats.Pageviews == nil || stats.Visitors == nil || stats.Visits == nil {
		return nil, errProvider
	}
	result := &Traffic{StartAt: start.Format(time.RFC3339), EndAt: end.Format(time.RFC3339), Visitors: int64(*stats.Visitors), Pageviews: int64(*stats.Pageviews), Visits: int64(*stats.Visits), Series: []TrafficPoint{}}
	if *stats.Visits > 0 {
		if stats.Bounces != nil {
			n := min(100, float64(*stats.Bounces)/float64(*stats.Visits)*100)
			result.BounceRate = &n
		}
		if stats.Totaltime != nil {
			n := float64(*stats.Totaltime) / float64(*stats.Visits)
			result.AverageDuration = &n
		}
	}
	if activeErr == nil && active.Visitors != nil {
		n := int64(*active.Visitors)
		result.ActiveVisitors = &n
	}
	if seriesErr == nil && series.Pageviews != nil && series.Sessions != nil {
		points := make(map[string]TrafficPoint)
		valid := true
		for i, buckets := range [][]bucket{series.Pageviews, series.Sessions} {
			for _, b := range buckets {
				timestamp, err := parseBucket(b.X)
				if err != nil || b.Y < 0 {
					valid = false
					break
				}
				key := timestamp.UTC().Format(time.RFC3339)
				p := points[key]
				p.Time = key
				if i == 0 {
					p.Pageviews = b.Y
				} else {
					p.Visitors = b.Y
				}
				points[key] = p
			}
		}
		if valid && len(points) <= 750 {
			result.SeriesAvailable = true
			for _, point := range points {
				result.Series = append(result.Series, point)
			}
			sort.Slice(result.Series, func(i, j int) bool { return result.Series[i].Time < result.Series[j].Time })
		}
	}
	return result, nil
}

func parseBucket(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid bucket time")
}
