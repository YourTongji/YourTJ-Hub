package feed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func NewID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func MarkTx(tx *gorm.DB, topicID uint64) error {
	if topicID == 0 {
		return nil
	}
	return MarkManyTx(tx, []uint64{topicID})
}
func MarkManyTx(tx *gorm.DB, ids []uint64) error {
	if !feedconfig.Current().Ranking || len(ids) == 0 {
		return nil
	}
	now := time.Now()
	rows := make([]Schedule, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, Schedule{TopicID: id, Version: 1, Generation: NewID(), DueAt: now, Dirty: true})
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "topic_id"}}, DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("topic_rank_schedule.version + 1"), "due_at": now, "dirty": true})}).CreateInBatches(&rows, 50).Error
}

var ErrClosed = errors.New("feed owner closed")

// LockOwnerTx serializes raw ingestion with the account-close fence.
func LockOwnerTx(tx *gorm.DB, uid uint64) error { _, err := lockOwnerVersionTx(tx, uid); return err }
func lockOwnerVersionTx(tx *gorm.DB, uid uint64) (uint64, error) {
	if uid == 0 {
		return 0, ErrClosed
	}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("feed_owner.version + 1"), "expires_at": time.Now().Add(30 * 24 * time.Hour)}), Where: clause.Where{Exprs: []clause.Expression{clause.Eq{Column: "feed_owner.closed", Value: false}}}}).Create(&Owner{Version: 1, UserID: uid, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}).Error; err != nil {
		return 0, err
	}
	var o Owner
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&o, "user_id = ?", uid).Error; err != nil {
		return 0, err
	}
	if o.Closed {
		return 0, ErrClosed
	}
	return o.Version, nil
}

type attributionKey struct{}
type Attribution struct {
	UserID   uint64
	ServeID  string
	TopicID  uint64
	Position int
	Source   string
}

func WithAttribution(ctx context.Context, a Attribution) context.Context {
	return context.WithValue(ctx, attributionKey{}, a)
}

func EventTx(tx *gorm.DB, uid, topicID, objectID uint64, kind string, active bool) error {
	c := feedconfig.Current()
	if !c.Metrics || uid == 0 {
		return nil
	}
	version, err := lockOwnerVersionTx(tx, uid)
	if errors.Is(err, ErrClosed) && (kind == "public_topic" || kind == "public_reply") {
		return nil
	}
	if err != nil {
		return err
	}
	now := time.Now()
	e := Event{ID: NewID(), UserID: uid, TopicID: topicID, ObjectID: objectID, Kind: kind, Active: active, SourceVersion: version, Source: "unattributed", Position: -1, CreatedAt: now, ExpiresAt: now.Add(time.Duration(c.RetentionDays) * 24 * time.Hour)}
	if a, ok := tx.Statement.Context.Value(attributionKey{}).(Attribution); ok && a.TopicID == topicID && a.UserID == uid {
		e.ServeID = a.ServeID
		e.Position = a.Position
		e.Source = a.Source
	}
	if (kind == "public_topic" || kind == "public_reply") && e.ServeID == "" {
		var intent Event
		if err := tx.Where("user_id = ? AND topic_id = ? AND object_id = ? AND kind = ? AND expires_at > ?", uid, topicID, objectID, "intent", now).Order("created_at DESC").First(&intent).Error; err == nil {
			e.ServeID = intent.ServeID
			e.Position = intent.Position
			e.Source = intent.Source
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	}
	return tx.Create(&e).Error
}

func CreditTx(tx *gorm.DB, uid, topicID uint64, kind string, active bool, now time.Time) error {
	if !feedconfig.Current().Ranking {
		return nil
	}
	if !feedconfig.Current().Metrics {
		if err := LockOwnerTx(tx, uid); err != nil {
			return err
		}
	}
	row := ActionCredit{TopicID: topicID, UserID: uid, Kind: kind, CreditAt: now, ExpiresAt: now.Add(24 * time.Hour), Active: active}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "topic_id"}, {Name: "user_id"}, {Name: "kind"}}, DoUpdates: clause.Assignments(map[string]any{"active": active, "credit_at": gorm.Expr("CASE WHEN topic_action_credit.expires_at <= ? AND ? THEN ? ELSE topic_action_credit.credit_at END", now, active, now), "expires_at": gorm.Expr("CASE WHEN topic_action_credit.expires_at <= ? AND ? THEN ? ELSE topic_action_credit.expires_at END", now, active, row.ExpiresAt)})}).Create(&row).Error
}

func CloseTx(tx *gorm.DB, uid uint64) error {
	// Existing installations and isolated owner tests may predate feed schema.
	if !tx.Migrator().HasTable(&Owner{}) {
		return nil
	}
	var owner Owner
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&owner, "user_id = ?", uid).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && owner.Closed {
		return nil
	}
	if tx.Migrator().HasTable(&ExperimentPeriod{}) {
		var rows []Assignment
		if e := tx.Where("user_id = ? AND expires_at > ?", uid, time.Now()).Find(&rows).Error; e != nil {
			return e
		}
		for _, r := range rows {
			col := "closed_control"
			if r.Variant == "treatment" {
				col = "closed_treatment"
			}
			if e := tx.Model(&ExperimentPeriod{}).Where("id = ?", r.Experiment).UpdateColumn(col, gorm.Expr(col+" + 1")).Error; e != nil {
				return e
			}
		}
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]any{"closed": true, "expires_at": time.Now().Add(30 * 24 * time.Hour)})}).Create(&Owner{UserID: uid, Closed: true, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}).Error
}

func MarkActorTx(tx *gorm.DB, uid uint64) error {
	if uid == 0 || !feedconfig.Current().Ranking {
		return nil
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]any{"version": gorm.Expr("feed_actor_work.version + 1"), "phase": 0, "cursor": 0, "expires_at": time.Now().Add(30 * 24 * time.Hour)})}).Create(&ActorWork{UserID: uid, Version: 1, ExpiresAt: time.Now().Add(30 * 24 * time.Hour)}).Error
}
