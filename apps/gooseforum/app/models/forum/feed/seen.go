package feed

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SeenState records a displayed content cutoff, not a detail visit or a metric.
// One row per viewer/topic; expiry and closure use the raw-data lifecycle.
type SeenState struct {
	UserID          uint64    `gorm:"primaryKey;autoIncrement:false"`
	TopicID         uint64    `gorm:"primaryKey;autoIncrement:false"`
	LastSeenAt      time.Time `gorm:"not null"`
	SeenContentAt   time.Time `gorm:"not null"`
	LastSeenProofID string    `gorm:"size:32;not null"`
	ExpiresAt       time.Time `gorm:"not null;index"`
}

func (SeenState) TableName() string { return "feed_seen_state" }

func SeenAmong(conn *gorm.DB, uid uint64, ids []uint64, now time.Time) (map[uint64]time.Time, error) {
	result := map[uint64]time.Time{}
	if len(ids) == 0 {
		return result, nil
	}
	if len(ids) > 300 {
		return nil, fmt.Errorf("seen batch exceeds bound")
	}
	var rows []SeenState
	err := conn.Select("topic_id,seen_content_at").Where("user_id = ? AND topic_id IN ? AND expires_at > ?", uid, ids, now.UTC()).Find(&rows).Error
	for _, row := range rows {
		result[row.TopicID] = row.SeenContentAt
	}
	return result, err
}

// The caller locks the live owner's fence. Duplicate or older proofs cannot
// move the cutoff backwards or extend retention after an ACK is lost.
func UpsertSeenTx(tx *gorm.DB, rows []SeenState) error {
	if len(rows) == 0 {
		return nil
	}
	if len(rows) > 120 {
		return fmt.Errorf("seen write exceeds bound")
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "topic_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_seen_at", "seen_content_at", "last_seen_proof_id", "expires_at"}),
		Where:     clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "excluded.seen_content_at > feed_seen_state.seen_content_at AND excluded.last_seen_proof_id <> feed_seen_state.last_seen_proof_id"}}},
	}).CreateInBatches(&rows, 120).Error
}
