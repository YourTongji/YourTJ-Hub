package feedservice

import (
	"math"
	"math/rand/v2"
	"sort"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
)

type Participant struct {
	UserID            uint64
	Replies           uint32
	LastPublicReplyAt *time.Time
	Liked             bool
}
type TimedAction struct {
	UserID uint64
	Kind   string
	At     time.Time
}
type RankInput struct {
	Author        uint64
	FirstPublicAt time.Time
	Participants  []Participant
	Actions       []TimedAction
}
type RankResult struct {
	Hot               int64
	Daily             int64
	NextDue           time.Time
	HotPeople         int
	HotReplyEffective int
	HotLikers         int
	Fresh             float64
	Active            float64
	Engaged           int
	Views             float64
	Replies           float64
	ReplyPeople       float64
	Likes             float64
	Bookmarks         float64
}

func normalized(x, cap float64) float64 {
	return math.Log1p(math.Min(math.Max(0, x), cap)) / math.Log1p(cap)
}

// ScoreRank is the only Hot/Daily scoring implementation, shared by worker,
// explain and replay. Daily reply caps apply across the entire 24-hour window.
func ScoreRank(in RankInput, now time.Time) RankResult {
	return ScoreRankWithRules(in, now, feedconfig.DefaultRules())
}
func ScoreRankWithRules(in RankInput, now time.Time, rules feedconfig.MaterializedRules) RankResult {
	out := RankResult{NextDue: now.Add(time.Hour)}
	boundary := func(at time.Time, ds ...time.Duration) {
		for _, d := range ds {
			b := at.Add(d)
			if b.After(now) && b.Before(out.NextDue) {
				out.NextDue = b
			}
		}
	}
	engaged := map[uint64]bool{}
	likers := 0
	effective := 0
	var last time.Time
	for _, p := range in.Participants {
		if p.UserID == 0 || p.UserID == in.Author {
			continue
		}
		if p.Liked {
			likers++
			engaged[p.UserID] = true
		}
		if p.Replies > 0 {
			effective += int(min(p.Replies, rules.ReplyPerUser))
			engaged[p.UserID] = true
			if p.LastPublicReplyAt != nil && p.LastPublicReplyAt.After(last) && !p.LastPublicReplyAt.After(now) {
				last = *p.LastPublicReplyAt
			}
		}
	}
	fresh := 0.
	age := now.Sub(in.FirstPublicAt)
	if !in.FirstPublicAt.IsZero() && age >= 0 {
		for i, hours := range rules.FreshHours {
			if age < time.Duration(hours)*time.Hour {
				fresh = rules.FreshValues[i]
				break
			}
		}
	}
	for _, hours := range rules.FreshHours {
		boundary(in.FirstPublicAt, time.Duration(hours)*time.Hour)
	}
	active := 0.
	if effective > 0 && !last.IsZero() {
		for i, hours := range rules.ActiveHours {
			if now.Sub(last) < time.Duration(hours)*time.Hour {
				active = rules.ActiveValues[i]
				break
			}
		}
		for _, hours := range rules.ActiveHours {
			boundary(last, time.Duration(hours)*time.Hour)
		}
	}
	out.HotPeople = len(engaged)
	out.HotReplyEffective = effective
	out.HotLikers = likers
	out.Fresh = fresh
	out.Active = active
	out.Hot = int64(math.Round(1000 * (rules.HotLikeWeight*normalized(float64(likers), rules.HotCaps[0]) + rules.HotReplyWeight*normalized(float64(effective), rules.HotCaps[1]) + fresh*math.Min(1, .2+.4*float64(len(engaged))) + active)))
	replies := map[uint64][]time.Time{}
	dailyEngaged := map[uint64]bool{}
	viewers := map[uint64]bool{}
	for _, a := range in.Actions {
		if a.UserID == 0 || a.UserID == in.Author {
			continue
		}
		weight := ageWeightWithRules(now.Sub(a.At), rules)
		if weight == 0 {
			continue
		}
		boundary(a.At, 3*time.Hour, 12*time.Hour, 24*time.Hour)
		switch a.Kind {
		case "view":
			out.Views += weight
			viewers[a.UserID] = true
		case "reply":
			replies[a.UserID] = append(replies[a.UserID], a.At)
			dailyEngaged[a.UserID] = true
		case "like":
			out.Likes += weight
			dailyEngaged[a.UserID] = true
		case "bookmark":
			out.Bookmarks += weight
			dailyEngaged[a.UserID] = true
		}
	}
	for _, times := range replies {
		sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
		times = times[:min(len(times), int(rules.ReplyPerUser))]
		for _, at := range times {
			out.Replies += ageWeightWithRules(now.Sub(at), rules)
		}
		out.ReplyPeople += ageWeightWithRules(now.Sub(times[len(times)-1]), rules)
	}
	out.Engaged = len(dailyEngaged)
	if len(dailyEngaged) >= rules.EngagementMinimum || len(viewers) >= rules.ViewMinimum {
		weighted := 0.
		for i, value := range []float64{out.Views, out.ReplyPeople, out.Replies, out.Likes, out.Bookmarks} {
			weighted += rules.DailyWeights[i] * normalized(value, rules.DailyCaps[i])
		}
		out.Daily = int64(math.Round(100000 * weighted))
	}
	// Refresh smooth view/action decay and detect projection drift, but never skip
	// a discontinuous age boundary. Quiet old topics refresh hourly.
	refresh := time.Hour
	if len(dailyEngaged) > 0 || len(viewers) > 0 {
		refresh = 30 * time.Minute
	}
	if !last.IsZero() && now.Sub(last) < time.Hour {
		refresh = 10 * time.Minute
	}
	if now.Add(refresh).Before(out.NextDue) {
		out.NextDue = now.Add(refresh)
	}
	if age >= 7*24*time.Hour && (last.IsZero() || now.Sub(last) >= 24*time.Hour) && len(dailyEngaged) == 0 && len(viewers) == 0 {
		out.NextDue = time.Time{}
	}
	return out
}

type Features [7]uint16
type Candidate struct {
	ID           uint64   `json:"id"`
	Author       uint64   `json:"author"`
	Sources      uint8    `json:"sources"`
	Features     Features `json:"features"`
	Repeat       bool     `json:"repeat,omitempty"`
	Reason       string   `json:"reason,omitempty"`
	Base         int64    `json:"base"`
	Adjusted     int64    `json:"adjusted"`
	Explore      bool     `json:"explore,omitempty"`
	Pool         int      `json:"pool,omitempty"`
	Team         string   `json:"team,omitempty"`
	SoftFiltered bool     `json:"softFiltered,omitempty"`
	Fallback     bool     `json:"fallback,omitempty"`
}

func Quantize(v float64) uint16 { return uint16(math.Round(math.Min(1, math.Max(0, v)) * 10000)) }
func ScoreFeatures(f Features, w feedconfig.Weights) int64 {
	sum := 0.
	for i, v := range []float64{w.Follow, w.Recency, w.Quality, w.Daily, w.Category, w.Author, w.NewReply} {
		sum += float64(f[i]) * v
	}
	if f[0] == 0 {
		sum *= .75
	}
	return int64(math.Round(sum))
}

// Rank greedily applies author diversity to bounded candidates. random is a
// deterministic PRNG whose seed is recorded in the snapshot/sample.
func Rank(pool []Candidate, w feedconfig.Weights, seed uint64, limit int) []Candidate {
	if limit > feedconfig.SnapshotSize {
		limit = feedconfig.SnapshotSize
	}
	pool = append([]Candidate(nil), pool...)
	out := make([]Candidate, 0, min(limit, len(pool)))
	authors := map[uint64]int{}
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	random := rng.IntN
	for len(pool) > 0 && len(out) < limit {
		best := -1
		var value int64
		for i, c := range pool {
			base := ScoreFeatures(c.Features, w)
			factor := math.Max(.25, math.Pow(.5, float64(authors[c.Author])))
			if c.Repeat {
				factor *= .7
			}
			adjusted := int64(math.Round(float64(base) * factor))
			pool[i].Base = base
			pool[i].Adjusted = adjusted
			if best < 0 || adjusted > value || adjusted == value && c.ID > pool[best].ID {
				best = i
				value = adjusted
			}
		}
		if slot := len(out) % 20; slot == 3 || slot == 10 {
			explore := []int{}
			for i, c := range pool {
				if c.Sources&32 != 0 && !c.SoftFiltered {
					explore = append(explore, i)
				}
			}
			if len(explore) > 0 {
				best = explore[random(len(explore))]
				pool[best].Explore = true
				pool[best].Pool = len(explore)
			}
		}
		chosen := pool[best]
		out = append(out, chosen)
		authors[chosen.Author]++
		pool = append(pool[:best], pool[best+1:]...)
	}
	return out
}

// SelectCandidates preserves the recorded primary/relaxed/fallback phases in
// live ranking and offline replay. Repeat is relaxed only when a page is short.
func SelectCandidates(pool []Candidate, w feedconfig.Weights, seed uint64) []Candidate {
	primary, relaxed, fallback := []Candidate{}, []Candidate{}, []Candidate{}
	for _, c := range pool {
		if c.Fallback {
			fallback = append(fallback, c)
		} else if c.SoftFiltered {
			relaxed = append(relaxed, c)
		} else {
			primary = append(primary, c)
		}
	}
	out := Rank(primary, w, seed, 120)
	if len(out) < 20 {
		combined := append(append([]Candidate{}, primary...), relaxed...)
		for i := range combined {
			combined[i].Repeat = false
		}
		out = Rank(combined, w, seed, 120)
	}
	if len(out) < 20 {
		combined := append(append(append([]Candidate{}, primary...), relaxed...), fallback...)
		for i := range combined {
			combined[i].Repeat = false
		}
		out = Rank(combined, w, seed, 120)
	}
	return out
}

// InterleaveCandidates scores both teams against the same selected-author
// counts. Exploration is selected once from their common pool and is shared.
func InterleaveCandidates(pool []Candidate, a, b feedconfig.Weights, seed uint64) []Candidate {
	eligible := []Candidate{}
	for _, c := range pool {
		if !c.Fallback && !c.SoftFiltered {
			eligible = append(eligible, c)
		}
	}
	if len(eligible) < 20 {
		eligible = nil
		for _, c := range pool {
			if !c.Fallback {
				c.Repeat = false
				eligible = append(eligible, c)
			}
		}
	}
	if len(eligible) < 20 {
		eligible = append([]Candidate(nil), pool...)
		for i := range eligible {
			eligible[i].Repeat = false
		}
	}
	out := []Candidate{}
	authors := map[uint64]int{}
	rng := rand.New(rand.NewPCG(seed, seed^0x517cc1b727220a95))
	turn := rng.IntN(2)
	bestFor := func(w feedconfig.Weights) int {
		best := -1
		var score int64
		for i, c := range eligible {
			factor := math.Max(.25, math.Pow(.5, float64(authors[c.Author])))
			if c.Repeat {
				factor *= .7
			}
			v := int64(math.Round(float64(ScoreFeatures(c.Features, w)) * factor))
			if best < 0 || v > score || v == score && c.ID > eligible[best].ID {
				best = i
				score = v
			}
		}
		return best
	}
	for len(eligible) > 0 && len(out) < 120 {
		ai, bi := bestFor(a), bestFor(b)
		chosen := ai
		team := "A"
		if turn == 1 {
			chosen = bi
			team = "B"
		}
		if ai == bi {
			team = "shared"
		} else {
			turn = 1 - turn
		}
		if slot := len(out) % 20; slot == 3 || slot == 10 {
			indices := []int{}
			for i, c := range eligible {
				if c.Sources&32 != 0 && !c.SoftFiltered {
					indices = append(indices, i)
				}
			}
			if len(indices) > 0 {
				chosen = indices[rng.IntN(len(indices))]
				eligible[chosen].Explore = true
				eligible[chosen].Pool = len(indices)
				team = "shared"
			}
		}
		c := eligible[chosen]
		c.Team = team
		w := a
		if team == "B" {
			w = b
		}
		c.Base = ScoreFeatures(c.Features, w)
		factor := math.Max(.25, math.Pow(.5, float64(authors[c.Author])))
		if c.Repeat {
			factor *= .7
		}
		c.Adjusted = int64(math.Round(float64(c.Base) * factor))
		authors[c.Author]++
		out = append(out, c)
		eligible = append(eligible[:chosen], eligible[chosen+1:]...)
	}
	return out
}

func ageWeightWithRules(age time.Duration, rules feedconfig.MaterializedRules) float64 {
	if age < 0 {
		return 0
	}
	for i, hours := range rules.DailyHours {
		if age < time.Duration(hours)*time.Hour {
			return rules.AgeWeights[i]
		}
	}
	return 0
}
