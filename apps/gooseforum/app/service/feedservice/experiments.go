package feedservice

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var assignments = &dailySeen

func assignDefault(ctx context.Context, uid uint64) (bool, string, error) {
	cfg := feedconfig.Current()
	now := time.Now()
	if uid == 0 || !cfg.Enabled || !cfg.Metrics || cfg.Rollout == 0 || !feedconfig.RankReady() {
		return false, "", nil
	}
	key := memoryKey{User: uid, Name: cfg.Hash}
	if e, ok := assignments.get(key, now); ok {
		return e.Trace == "treatment", e.Trace, nil
	}
	variant := "control"
	if bucket(uid, cfg.Salt+":entry") < cfg.Rollout {
		variant = "treatment"
	}
	row := feed.Assignment{Experiment: cfg.Experiment, UserID: uid, Variant: variant, Hash: cfg.Hash, CreatedAt: now, ExpiresAt: now.Add(30 * 24 * time.Hour)}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := feed.LockOwnerTx(tx, uid); err != nil {
			return err
		}
		p := feed.ExperimentPeriod{ID: cfg.Experiment, Hash: cfg.Hash, StartedAt: now, EnrollUntil: now.Add(14 * 24 * time.Hour), AnalyzeAt: now.Add(22 * 24 * time.Hour)}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&p).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&p, "id = ?", cfg.Experiment).Error; err != nil {
			return err
		}
		if p.Aborted != "" {
			variant = "unassigned"
			return nil
		}
		if p.Hash != cfg.Hash {
			variant = "unassigned"
			return tx.Model(&p).UpdateColumn("aborted", "parameters changed").Error
		}
		existing := tx.First(&row, "experiment = ? AND user_id = ? AND expires_at > ?", cfg.Experiment, uid, now)
		if existing.Error == nil {
			variant = row.Variant
			return nil
		}
		if !errors.Is(existing.Error, gorm.ErrRecordNotFound) {
			return existing.Error
		}
		if !p.EnrollUntil.After(now) {
			variant = "unassigned"
			return nil
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		col := "assigned_control"
		if variant == "treatment" {
			col = "assigned_treatment"
		}
		if err := tx.Model(&p).UpdateColumn(col, gorm.Expr(col+" + 1")).Error; err != nil {
			return err
		}
		return addMetric(tx, feed.Serve{CreatedAt: now, Feed: "entry", Hash: cfg.Hash, Experiment: cfg.Experiment, Variant: variant, Capability: "v2", RankHash: feedconfig.Current().RankHash}, "assigned", 1)
	})
	if err != nil {
		return false, "", err
	}
	assignments.put(key, memoryEntry{At: now, Expires: now.Add(15 * time.Second), Trace: variant})
	return variant == "treatment", variant, nil
}

type Moment struct {
	N          int64   `json:"n"`
	Sum        float64 `json:"sum"`
	SumSquares float64 `json:"sumSquares"`
}

func (m *Moment) add(x float64) { m.N++; m.Sum += x; m.SumSquares += x * x }
func (m Moment) mean() float64 {
	if m.N == 0 {
		return 0
	}
	return m.Sum / float64(m.N)
}
func (m Moment) varianceOfMean() float64 {
	if m.N < 2 {
		return 0
	}
	return math.Max(0, (m.SumSquares-m.Sum*m.Sum/float64(m.N))/float64(m.N-1)) / float64(m.N)
}

type PeriodStats map[string]map[string]Moment

func newStats() PeriodStats { return PeriodStats{"control": {}, "treatment": {}} }

type Effect struct {
	Control           Moment   `json:"control"`
	Treatment         Moment   `json:"treatment"`
	Difference        float64  `json:"difference"`
	IntervalAvailable bool     `json:"intervalAvailable"`
	Lower             *float64 `json:"lower95"`
	Upper             *float64 `json:"upper95"`
}
type ExperimentResult struct {
	Period            string            `json:"period"`
	Method            string            `json:"method"`
	AssignedControl   int64             `json:"assignedControl"`
	AssignedTreatment int64             `json:"assignedTreatment"`
	MissingControl    int64             `json:"missingControl"`
	MissingTreatment  int64             `json:"missingTreatment"`
	ClosedControl     int64             `json:"closedControl"`
	ClosedTreatment   int64             `json:"closedTreatment"`
	Effects           map[string]Effect `json:"effects"`
	Limitation        string            `json:"limitation"`
}

func effects(stats PeriodStats, p feed.ExperimentPeriod) ExperimentResult {
	r := ExperimentResult{Period: p.ID, Method: "user-unit difference of means; large-sample normal interval from n/sum/sumSquares", AssignedControl: p.AssignedControl, AssignedTreatment: p.AssignedTreatment, ClosedControl: p.ClosedControl, ClosedTreatment: p.ClosedTreatment, Effects: map[string]Effect{}, Limitation: "Observed users include zero outcomes, tab changes and fallback. Deleted/missing trajectories remain in the assigned denominator and are reported separately. Intervals exclude missing users; small samples and missingness preclude rollout decisions. These sufficient statistics cannot reproduce a clustered bootstrap."}
	for _, name := range []string{"public_contributions", "active_days", "d1", "d7", "corrected_actions"} {
		a, b := stats["control"][name], stats["treatment"][name]
		d := b.mean() - a.mean()
		margin := 1.96 * math.Sqrt(a.varianceOfMean()+b.varianceOfMean())
		effect := Effect{Control: a, Treatment: b, Difference: d}
		if a.N >= 30 && b.N >= 30 {
			lower, upper := d-margin, d+margin
			effect.Lower = &lower
			effect.Upper = &upper
			effect.IntervalAvailable = true
		}
		r.Effects[name] = effect
	}
	r.MissingControl = p.AssignedControl - stats["control"]["active_days"].N
	r.MissingTreatment = p.AssignedTreatment - stats["treatment"]["active_days"].N
	return r
}

// analyzePeriod consumes one batch of <=200 users, not a whole raw table. The
// cursor, sufficient statistics and final result commit atomically.
func analyzePeriod(ctx context.Context) error {
	now := time.Now()
	conn := db.ConnectContext(ctx)
	var p feed.ExperimentPeriod
	if err := conn.Where("aborted = '' AND result = '' AND analyze_at <= ?", now).Order("analyze_at").First(&p).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	return conn.Transaction(func(tx *gorm.DB) error {
		progress := feed.PeriodProgress{ID: p.ID, Stats: "{}", ExpiresAt: p.StartedAt.Add(30 * 24 * time.Hour)}
		if e := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&progress).Error; e != nil {
			return e
		}
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&progress, "id = ?", p.ID).Error; e != nil {
			return e
		}
		stats := newStats()
		if progress.Stats != "{}" {
			if e := json.Unmarshal([]byte(progress.Stats), &stats); e != nil {
				return e
			}
		}
		var rows []feed.Assignment
		if e := tx.Where("experiment = ? AND user_id > ? AND expires_at > ?", p.ID, progress.Cursor, now).Order("user_id").Limit(200).Find(&rows).Error; e != nil {
			return e
		}
		if len(rows) == 0 {
			data, e := json.Marshal(effects(stats, p))
			if e != nil {
				return e
			}
			if e = tx.Model(&p).UpdateColumn("result", string(data)).Error; e != nil {
				return e
			}
			return tx.Delete(&progress).Error
		}
		uids := make([]uint64, len(rows))
		for i, r := range rows {
			uids[i] = r.UserID
		}
		var daily []feed.UserDaily
		if e := tx.Where("user_id IN ? AND expires_at > ? AND day >= ? AND day <= ?", uids, now, localDay(p.StartedAt), localDay(p.AnalyzeAt)).Find(&daily).Error; e != nil {
			return e
		}
		byUser := map[uint64][]feed.UserDaily{}
		for _, d := range daily {
			byUser[d.UserID] = append(byUser[d.UserID], d)
		}
		for _, r := range rows {
			values := map[string]float64{"public_contributions": 0, "active_days": 0, "d1": 0, "d7": 0, "corrected_actions": 0}
			first := localDay(r.CreatedAt)
			last := localDay(r.CreatedAt.AddDate(0, 0, 7))
			d1 := localDay(r.CreatedAt.AddDate(0, 0, 1))
			for _, d := range byUser[r.UserID] {
				if d.Day < first || d.Day > last {
					continue
				}
				values["public_contributions"] += float64(d.PublicTopics + d.PublicReplies)
				values["corrected_actions"] += float64(d.Likes + d.Bookmarks)
				if d.Active {
					values["active_days"]++
					if d.Day == d1 {
						values["d1"] = 1
					}
					if d.Day == last {
						values["d7"] = 1
					}
				}
			}
			for name, value := range values {
				m := stats[r.Variant][name]
				m.add(value)
				stats[r.Variant][name] = m
			}
		}
		progress.Cursor = rows[len(rows)-1].UserID
		data, e := json.Marshal(stats)
		if e != nil {
			return e
		}
		progress.Stats = string(data)
		return tx.Save(&progress).Error
	})
}
func abortPeriods(ctx context.Context, reason string) error {
	return db.ConnectContext(ctx).Model(&feed.ExperimentPeriod{}).Where("aborted = '' AND result = ''").UpdateColumn("aborted", reason).Error
}

func abortMismatchedPeriods(ctx context.Context, hash string) error {
	return db.ConnectContext(ctx).Model(&feed.ExperimentPeriod{}).Where("aborted = '' AND result = '' AND hash <> ?", hash).UpdateColumn("aborted", "parameters changed").Error
}
