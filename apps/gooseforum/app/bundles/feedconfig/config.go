// Package feedconfig owns the immutable, non-secret feed parameter contract.
package feedconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/spf13/cast"
)

const CandidateCap = 240
const FallbackCap = 60
const SnapshotSize = 120
const PageSize = 20
const RawRetentionDays = 30

type Weights struct {
	Follow   float64 `json:"follow"`
	Recency  float64 `json:"recency"`
	Quality  float64 `json:"quality"`
	Daily    float64 `json:"daily"`
	Category float64 `json:"category"`
	Author   float64 `json:"author"`
	NewReply float64 `json:"newReply"`
}

type Config struct {
	Personalization FixedPolicy       `json:"personalization"`
	Rules           MaterializedRules `json:"rules"`
	Limits          ResourceLimits    `json:"limits"`
	RankHash        string            `json:"rankHash"`
	Algorithm       string            `json:"algorithm"`
	Enabled         bool              `json:"enabled"`
	Ranking         bool              `json:"ranking"`
	Metrics         bool              `json:"metrics"`
	Rollout         int               `json:"rollout"`
	RetentionDays   int               `json:"retentionDays"`
	JobsPerSecond   int               `json:"jobsPerSecond"`
	QueryTimeoutMS  int               `json:"queryTimeoutMs"`
	SampleRate      float64           `json:"sampleRate"`
	Experiment      string            `json:"experiment"`
	Salt            string            `json:"salt"`
	Weights         Weights           `json:"weights"`
	Comparison      bool              `json:"comparison"`
	Interleaving    bool              `json:"interleaving"`
	Alternative     Weights           `json:"alternative"`
	Hash            string            `json:"-"`
}

func Default() Config {
	w := Weights{1, .8, 1, .5, .6, .5, .4}
	return Config{Personalization: DefaultPolicy(), Rules: DefaultRules(), Limits: HardLimits(), Algorithm: "rules-v3-hot-daily-v1", Rollout: 20, RetentionDays: 30, JobsPerSecond: 20, QueryTimeoutMS: 250, SampleRate: .05, Experiment: "default-entry-v1", Salt: "default-entry-v1", Weights: w, Alternative: w}
}

func Decode(settings map[string]any) (Config, error) {
	c := Default()
	var decodeErr error
	get := func(path ...string) (any, bool) {
		var cur any = settings
		for _, key := range path {
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			cur, ok = m[key]
			if !ok {
				return nil, false
			}
		}
		return cur, true
	}
	b := func(dst *bool, path ...string) {
		if v, ok := get(path...); ok {
			value, err := cast.ToBoolE(v)
			if err != nil {
				decodeErr = errors.Join(decodeErr, err)
			}
			*dst = value
		}
	}
	i := func(dst *int, path ...string) {
		if v, ok := get(path...); ok {
			value, err := cast.ToIntE(v)
			if err != nil {
				decodeErr = errors.Join(decodeErr, err)
			}
			*dst = value
		}
	}
	f := func(dst *float64, path ...string) {
		if v, ok := get(path...); ok {
			value, err := cast.ToFloat64E(v)
			if err != nil {
				decodeErr = errors.Join(decodeErr, err)
			}
			*dst = value
		}
	}
	s := func(dst *string, path ...string) {
		if v, ok := get(path...); ok {
			value, err := cast.ToStringE(v)
			if err != nil {
				decodeErr = errors.Join(decodeErr, err)
			}
			*dst = value
		}
	}
	b(&c.Enabled, "feed", "for_you", "enabled")
	b(&c.Ranking, "ranking", "enabled")
	b(&c.Metrics, "feed", "metrics", "enabled")
	i(&c.Rollout, "feed", "for_you", "rollout_percent")
	i(&c.RetentionDays, "feed", "metrics", "raw_retention_days")
	i(&c.JobsPerSecond, "ranking", "jobs_per_second")
	i(&c.QueryTimeoutMS, "ranking", "query_timeout_ms")
	f(&c.SampleRate, "feed", "metrics", "candidate_sample_rate")
	s(&c.Experiment, "feed", "experiments", "period")
	s(&c.Salt, "feed", "experiments", "salt")
	b(&c.Comparison, "feed", "experiments", "weight_comparison_enabled")
	b(&c.Interleaving, "feed", "experiments", "interleaving_enabled")
	weights := func(w *Weights, key string) {
		for name, dst := range map[string]*float64{"follow": &w.Follow, "recency": &w.Recency, "quality": &w.Quality, "daily": &w.Daily, "category": &w.Category, "author": &w.Author, "new_reply": &w.NewReply} {
			f(dst, "feed", key, name)
		}
	}
	weights(&c.Weights, "weights")
	weights(&c.Alternative, "alternative_weights")
	f(&c.Rules.HotLikeWeight, "ranking", "hot", "like_weight")
	f(&c.Rules.HotReplyWeight, "ranking", "hot", "reply_weight")
	for key, dst := range map[string]*float64{"view_weight": &c.Rules.DailyWeights[0], "replier_weight": &c.Rules.DailyWeights[1], "reply_weight": &c.Rules.DailyWeights[2], "like_weight": &c.Rules.DailyWeights[3], "bookmark_weight": &c.Rules.DailyWeights[4]} {
		f(dst, "ranking", "daily", key)
	}
	if decodeErr != nil {
		return Config{}, decodeErr
	}
	if math.IsNaN(c.SampleRate) || math.IsInf(c.SampleRate, 0) {
		return Config{}, fmt.Errorf("sample rate must be finite")
	}
	if c.Rollout < 0 || c.Rollout > 100 || c.RetentionDays < 1 || c.RetentionDays > 30 || c.JobsPerSecond < 1 || c.JobsPerSecond > 20 || c.QueryTimeoutMS < 1 || c.QueryTimeoutMS > 250 || c.SampleRate < 0 || c.SampleRate > .05 || len(c.Experiment) > 64 || c.Experiment == "" || len(c.Salt) > 128 || c.Salt == "" {
		return Config{}, fmt.Errorf("feed parameters exceed supported bounds")
	}
	for _, w := range []Weights{c.Weights, c.Alternative} {
		for _, v := range []float64{w.Follow, w.Recency, w.Quality, w.Daily, w.Category, w.Author, w.NewReply} {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 10 {
				return Config{}, fmt.Errorf("feed weights must be between zero and ten")
			}
		}
	}
	if c.Enabled && c.Metrics && c.Rollout > 0 && c.RetentionDays != 30 {
		return Config{}, fmt.Errorf("default-entry experiment requires 30-day retention")
	}
	if c.Comparison && c.Interleaving {
		return Config{}, fmt.Errorf("only one weight comparison mode may be active")
	}
	sum := 0.
	for _, v := range c.Rules.DailyWeights {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 1 {
			return Config{}, fmt.Errorf("daily weights must be finite in [0,1]")
		}
		sum += v
	}
	if math.Abs(sum-1) > 1e-8 {
		return Config{}, fmt.Errorf("daily weights must sum to one")
	}
	for _, v := range []float64{c.Rules.HotLikeWeight, c.Rules.HotReplyWeight} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > 50 {
			return Config{}, fmt.Errorf("hot weights must be finite in [0,50]")
		}
	}
	rankData, err := json.Marshal(c.Rules)
	if err != nil {
		return Config{}, err
	}
	rankDigest := sha256.Sum256(rankData)
	c.RankHash = hex.EncodeToString(rankDigest[:16])
	data, err := json.Marshal(c)
	if err != nil {
		return Config{}, err
	}
	hash := sha256.Sum256(data)
	c.Hash = hex.EncodeToString(hash[:16])
	return c, nil
}

var mu sync.Mutex
var previous = Default()
var previousRevision uint64

func init() {
	preferences.AddValidator(func(settings map[string]any) error { _, err := Decode(settings); return err })
}

// Current captures all settings from one published preferences snapshot. Invalid
// edits retain the last valid feed parameters, including enable/disable state.
func Current() Config {
	revision := preferences.Revision()
	mu.Lock()
	defer mu.Unlock()
	if revision == previousRevision {
		return previous
	}
	c, err := Decode(preferences.All())
	if err == nil {
		previous = c
	}
	previousRevision = revision
	return previous
}

var rankReady atomic.Pointer[string]

func SetRankReady(ready bool) {
	if !ready {
		rankReady.Store(nil)
		return
	}
	h := Current().RankHash
	rankReady.Store(&h)
}
func RankReady() bool {
	c := Current()
	h := rankReady.Load()
	return c.Ranking && h != nil && *h == c.RankHash
}

// Versioned defaults capture every fixed coefficient and boundary in parameter
// history. Hot/Daily weights may change only through a controlled full rebuild.
type MaterializedRules struct {
	Version           string     `json:"version"`
	HotLikeWeight     float64    `json:"hotLikeWeight"`
	HotReplyWeight    float64    `json:"hotReplyWeight"`
	HotCaps           [2]float64 `json:"hotCaps"`
	ReplyPerUser      uint32     `json:"replyPerUser"`
	FreshHours        [6]int     `json:"freshHours"`
	FreshValues       [6]float64 `json:"freshValues"`
	ActiveHours       [4]int     `json:"activeHours"`
	ActiveValues      [4]float64 `json:"activeValues"`
	DailyHours        [3]int     `json:"dailyHours"`
	AgeWeights        [3]float64 `json:"ageWeights"`
	DailyCaps         [5]float64 `json:"dailyCaps"`
	DailyWeights      [5]float64 `json:"dailyWeights"`
	EngagementMinimum int        `json:"engagementMinimum"`
	ViewMinimum       int        `json:"viewMinimum"`
}

func DefaultRules() MaterializedRules {
	return MaterializedRules{"hot-daily-v1", 6, 14, [2]float64{50, 100}, 3, [6]int{6, 12, 24, 48, 72, 168}, [6]float64{30, 24, 18, 12, 8, 4}, [4]int{1, 6, 12, 24}, [4]float64{20, 12, 6, 2}, [3]int{3, 12, 24}, [3]float64{1, .7, .4}, [5]float64{300, 20, 50, 40, 15}, [5]float64{.30, .25, .15, .20, .10}, 2, 10}
}
func (r MaterializedRules) HotMax() float64 {
	return r.HotLikeWeight + r.HotReplyWeight + r.FreshValues[0] + r.ActiveValues[0]
}

type ResourceLimits struct{ Candidates, Fallback, Snapshot, Page, ConcurrentBuilds, BackgroundConnections, SnapshotBytes, ProfileBytes, PublicPoolBytes, RepeatBytes, InferredBytes, ViewBytes, LogBytes, SampleBytes, DayBytes int }

func HardLimits() ResourceLimits {
	return ResourceLimits{240, 60, 120, 20, 2, 1, 32 << 20, 4 << 20, 1 << 20, 4 << 20, 2 << 20, 2 << 20, 8 << 20, 2 << 20, 2 << 20}
}

// These rules are versioned in the full parameter snapshot for exact replay.
// Changing one is an algorithm change, rather than an independent live knob.
type FixedPolicy struct {
	OutOfNetwork, AuthorDiversity, AuthorFloor, Repeat, RecencyHalfLifeHours            float64
	ProfileDays, ProfileLimit, ProfileHalfLifeDays, MinimumProfile, Categories, Authors int
	ProfileLike, ProfileBookmark, ProfileReply                                          float64
	SnapshotTTLMinutes, ReuseSeconds, HardDeadlineMS, TraceHours, ViewCooldownMinutes   int
	ExplorationPositions                                                                [2]int
}

func DefaultPolicy() FixedPolicy {
	return FixedPolicy{.75, .5, .25, .7, 12, 60, 100, 14, 5, 5, 50, 1, 1.5, 2, 30, 30, 200, 6, 5, [2]int{4, 11}}
}
