package feedservice

import (
	"context"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
)

type MetricRow struct {
	WeightVariant string `json:"weightVariant"`
	Capability    string `json:"capability"`
	RankHash      string `json:"rankHash"`
	Day           string `json:"day"`
	Feed          string `json:"feed"`
	Hash          string `json:"hash"`
	Experiment    string `json:"experiment"`
	Variant       string `json:"variant"`
	Metric        string `json:"metric"`
	Count         int64  `json:"count"`
}
type Summary struct {
	Enabled        bool            `json:"enabled"`
	RankingReady   bool            `json:"rankingReady"`
	MetricsEnabled bool            `json:"metricsEnabled"`
	Rollout        int             `json:"rolloutPercent"`
	Retention      int             `json:"rawRetentionDays"`
	ParamsHash     string          `json:"paramsHash"`
	Rows           []MetricRow     `json:"rows"`
	Truncated      bool            `json:"truncated"`
	Periods        []PeriodSummary `json:"periods"`
	Health         map[string]any  `json:"health"`
}

var summaryMu sync.Mutex
var cachedSummary Summary
var summaryAt time.Time

// The admin view reads anonymous aggregates only. It never reconstructs or
// exposes individual traces, candidate samples, assignments or activity days.
func GetSummary(ctx context.Context) (Summary, error) {
	summaryMu.Lock()
	defer summaryMu.Unlock()
	if time.Since(summaryAt) < 60*time.Second {
		return cachedSummary, nil
	}
	cfg := feedconfig.Current()
	var rows []feed.MetricsDaily
	if err := db.ConnectContext(ctx).Where("day >= ?", localDay(time.Now().Add(-7*24*time.Hour))).Order("day DESC").Order("feed,hash,rank_hash,experiment,variant,weight_variant,capability,metric").Limit(1001).Find(&rows).Error; err != nil {
		return Summary{}, err
	}
	s := Summary{Enabled: cfg.Enabled, RankingReady: feedconfig.RankReady(), MetricsEnabled: cfg.Metrics, Rollout: cfg.Rollout, Retention: cfg.RetentionDays, ParamsHash: cfg.Hash, Rows: []MetricRow{}, Truncated: len(rows) > 1000, Health: TelemetryHealth()}
	s.Health["parameters"] = cfg
	for _, r := range rows[:min(len(rows), 1000)] {
		s.Rows = append(s.Rows, MetricRow{WeightVariant: r.WeightVariant, Day: r.Day, Feed: r.Feed, Hash: r.Hash, Experiment: r.Experiment, Variant: r.Variant, Metric: r.Metric, Count: r.Count, Capability: r.Capability, RankHash: r.RankHash})
	}
	var periods []feed.ExperimentPeriod
	if err := db.ConnectContext(ctx).Order("started_at DESC").Limit(20).Find(&periods).Error; err != nil {
		return Summary{}, err
	}
	s.Periods = []PeriodSummary{}
	for _, p := range periods {
		s.Periods = append(s.Periods, PeriodSummary{ID: p.ID, Hash: p.Hash, EnrollUntil: p.EnrollUntil, AnalyzeAt: p.AnalyzeAt, Aborted: p.Aborted, AssignedControl: p.AssignedControl, AssignedTreatment: p.AssignedTreatment, Result: p.Result})
	}
	cachedSummary = s
	summaryAt = time.Now()
	return s, nil
}

type PeriodSummary struct {
	ID                string    `json:"id"`
	Hash              string    `json:"hash"`
	EnrollUntil       time.Time `json:"enrollUntil"`
	AnalyzeAt         time.Time `json:"analyzeAt"`
	Aborted           string    `json:"aborted"`
	AssignedControl   int64     `json:"assignedControl"`
	AssignedTreatment int64     `json:"assignedTreatment"`
	Result            string    `json:"result"`
}

// ExplainRank prints current aggregate inputs/results, never raw actor facts.
func ExplainRank(ctx context.Context, id uint64) (map[string]any, error) {
	cfg := feedconfig.Current()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.QueryTimeoutMS)*time.Millisecond)
	defer cancel()
	topic, err := topics.RankTopic(ctx, id)
	if err != nil {
		return nil, err
	}
	input, err := rankInput(ctx, id, topic)
	if err != nil {
		return nil, err
	}
	return map[string]any{"topicId": id, "paramsHash": cfg.Hash, "rankHash": cfg.RankHash, "materializedHot": topic.RankScore, "materializedDaily": topic.DailyScore, "current": ScoreRankWithRules(input, time.Now(), cfg.Rules), "firstPublicEstimated": topic.FirstPublicEstimated, "limitation": "Current public state only; this does not reconstruct a historical candidate pool or causal effect."}, nil
}
