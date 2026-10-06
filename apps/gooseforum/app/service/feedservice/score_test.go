package feedservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"testing"
	"time"
)

func TestDailyCapsOneUserAcrossAgeBandsAndUnionsKinds(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	in := RankInput{Author: 1, FirstPublicAt: now.Add(-40 * time.Hour), Actions: []TimedAction{{2, "reply", now.Add(-20 * time.Hour)}, {2, "reply", now.Add(-14 * time.Hour)}, {2, "reply", now.Add(-8 * time.Hour)}, {2, "reply", now.Add(-time.Hour)}, {2, "like", now.Add(-time.Hour)}, {2, "bookmark", now.Add(-time.Hour)}}}
	r := ScoreRank(in, now)
	if r.Engaged != 1 || r.Daily != 0 {
		t.Fatalf("one person across kinds passed the engagement gate: %+v", r)
	}
	if r.Replies != 1.5 || r.ReplyPeople != .7 {
		t.Fatalf("reply cap was applied per band or chose newest three: %+v", r)
	}
	in.Actions = append(in.Actions, TimedAction{3, "like", now.Add(-time.Hour)})
	if got := ScoreRank(in, now); got.Daily <= 0 || got.Engaged != 2 {
		t.Fatalf("two distinct participants did not qualify: %+v", got)
	}
}

func TestAgeBoundaryScheduledBeforeRefresh(t *testing.T) {
	now := time.Now()
	first := now.Add(-6*time.Hour + time.Second)
	r := ScoreRank(RankInput{Author: 1, FirstPublicAt: first}, now)
	if !r.NextDue.Equal(now.Add(time.Second)) {
		t.Fatalf("next due %s skipped fresh boundary", r.NextDue)
	}
	if next := ScoreRank(RankInput{Author: 1, FirstPublicAt: first}, now.Add(time.Second)); next.Hot >= r.Hot {
		t.Fatal("score did not decay at the exact bucket boundary")
	}
}

func TestAuthorCannotSupplyDailyEngagementOrViews(t *testing.T) {
	now := time.Now()
	in := RankInput{Author: 1, FirstPublicAt: now}
	for i := 0; i < 12; i++ {
		in.Actions = append(in.Actions, TimedAction{1, "view", now}, TimedAction{1, "reply", now})
	}
	if got := ScoreRank(in, now); got.Daily != 0 || got.Views != 0 || got.Replies != 0 {
		t.Fatalf("author supplied credit: %+v", got)
	}
}

func TestRankIsDeterministicAndExplorationCannotDuplicate(t *testing.T) {
	pool := []Candidate{}
	for i := 1; i <= 80; i++ {
		pool = append(pool, Candidate{ID: uint64(i), Author: uint64(i%7 + 1), Sources: 32, Features: Features{0, 5000, 6000, 0, 0, 0, 0}})
	}
	a := Rank(pool, feedconfig.Default().Weights, 17, 80)
	b := Rank(pool, feedconfig.Default().Weights, 17, 80)
	seen := map[uint64]bool{}
	for i, c := range a {
		if c != b[i] || seen[c.ID] {
			t.Fatal("replay diverged or duplicated a topic")
		}
		seen[c.ID] = true
		if (i%20 == 3 || i%20 == 10) && (!c.Explore || c.Pool < 1) {
			t.Fatal("exploration slot missing provenance")
		}
	}
}

func TestIdenticalInterleavingHasNoWinners(t *testing.T) {
	pool := []Candidate{{ID: 1}, {ID: 2}, {ID: 3}}
	for _, c := range InterleaveCandidates(pool, feedconfig.Default().Weights, feedconfig.Default().Weights, 7) {
		if c.Team != "shared" {
			t.Fatal("identical lists fabricated a team win")
		}
	}
}

func BenchmarkRank240(b *testing.B) {
	pool := make([]Candidate, 240)
	for i := range pool {
		pool[i] = Candidate{ID: uint64(i + 1), Author: uint64(i%80 + 1), Features: Features{0, 6000, 7000, 4000, 3000, 0, 0}}
	}
	w := feedconfig.Default().Weights
	b.ReportAllocs()
	for b.Loop() {
		Rank(pool, w, 17, 120)
	}
}

func TestSmallExperimentDoesNotFabricateConfidenceIntervals(t *testing.T) {
	stats := newStats()
	for _, v := range []string{"control", "treatment"} {
		m := Moment{}
		m.add(0)
		stats[v]["active_days"] = m
	}
	result := effects(stats, feed.ExperimentPeriod{AssignedControl: 1, AssignedTreatment: 1})
	e := result.Effects["active_days"]
	if e.IntervalAvailable || e.Lower != nil || e.Upper != nil {
		t.Fatal("small sample exposed a zero-width interval")
	}
}

// Issue 1056: revived peer discussion outranks untouched new publications.
func TestHotReviewExamplesOrderBThenCThenDThenA(t *testing.T) {
	now := time.Now()
	input := func(age, lastAge time.Duration, likes, replies int) RankInput {
		in := RankInput{Author: 1, FirstPublicAt: now.Add(-age)}
		for i := 0; i < max(likes, (replies+2)/3); i++ {
			n := min(3, max(0, replies-i*3))
			last := now.Add(-lastAge)
			in.Participants = append(in.Participants, Participant{UserID: uint64(i + 2), Replies: uint32(n), Liked: i < likes, LastPublicReplyAt: &last})
		}
		return in
	}
	a := ScoreRank(input(time.Hour, 0, 0, 0), now).Hot
	b := ScoreRank(input(3*time.Hour, 30*time.Minute, 2, 5), now).Hot
	c := ScoreRank(input(10*24*time.Hour, 3*time.Hour, 40, 80), now).Hot
	d := ScoreRank(input(2*24*time.Hour, 25*time.Hour, 10, 30), now).Hot
	if (b <= c || c <= d || d <= a) || a != 6000 {
		t.Fatalf("Hot examples: B=%d C=%d D=%d A=%d", b, c, d, a)
	}
}
