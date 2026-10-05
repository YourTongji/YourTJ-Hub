// Package agentWrites owns transactional Agent creation deduplication. It stores
// resource references rather than rendered content, so retries use live visibility.
package agentWrites

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrConflict = errors.New("agent write idempotency conflict")
var ErrIncomplete = errors.New("agent write result unavailable")

type Entry struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement" json:"-"`
	InstanceID    string    `gorm:"size:128;not null;uniqueIndex:uniq_agent_write,priority:1;check:chk_agent_write_instance,instance_id <> ''" json:"-"`
	AgentID       uint64    `gorm:"not null;uniqueIndex:uniq_agent_write,priority:2" json:"-"`
	Operation     string    `gorm:"size:32;not null;uniqueIndex:uniq_agent_write,priority:3" json:"-"`
	TargetID      uint64    `gorm:"not null;uniqueIndex:uniq_agent_write,priority:4" json:"-"`
	RequestKey    string    `gorm:"size:256;not null;uniqueIndex:uniq_agent_write,priority:5;check:chk_agent_write_key,request_key <> ''" json:"-"`
	Digest        string    `gorm:"size:64;not null;check:chk_agent_write_digest,digest <> ''" json:"-"`
	SourceEventID string    `gorm:"size:80;not null;default:'';index" json:"-"`
	TopicID       uint64    `gorm:"not null;default:0" json:"-"`
	PostID        uint64    `gorm:"not null;default:0" json:"-"`
	CreatedAt     time.Time `gorm:"not null;autoCreateTime" json:"-"`
	ExpiresAt     time.Time `gorm:"not null;index" json:"-"`
}

func (Entry) TableName() string { return "agent_write_idempotency" }

func scope(tx *gorm.DB, e Entry) *gorm.DB {
	return tx.Where("instance_id = ? AND agent_id = ? AND operation = ? AND target_id = ? AND request_key = ?", e.InstanceID, e.AgentID, e.Operation, e.TargetID, e.RequestKey)
}

func LookupTx(tx *gorm.DB, e Entry, now time.Time) (*Entry, error) {
	var found Entry
	err := scope(tx, e).Where("expires_at > ?", now).Take(&found).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if found.Digest != e.Digest {
		return nil, ErrConflict
	}
	if found.TopicID == 0 || found.PostID == 0 {
		return nil, ErrIncomplete
	}
	return &found, nil
}

// ReserveTx waits on the unique constraint for an in-flight creator. The winner
// reserves, writes content and completes the reference in this same transaction;
// rollback releases the key. A committed reservation is never left incomplete.
func ReserveTx(tx *gorm.DB, e *Entry, now time.Time) (replay *Entry, err error) {
	if err = scope(tx, *e).Where("expires_at <= ?", now).Delete(&Entry{}).Error; err != nil {
		return nil, err
	}
	result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "agent_id"}, {Name: "operation"}, {Name: "target_id"}, {Name: "request_key"}}, DoNothing: true}).Create(e)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 1 {
		return nil, nil
	}
	return LookupTx(tx, *e, now)
}

func CompleteTx(tx *gorm.DB, e *Entry, topicID, postID uint64) error {
	if e == nil {
		return nil
	}
	result := tx.Model(&Entry{}).Where("id = ? AND instance_id = ? AND topic_id = 0 AND post_id = 0", e.ID, e.InstanceID).Updates(map[string]any{"topic_id": topicID, "post_id": postID})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrIncomplete
	}
	return nil
}

func PurgeExpiredTx(tx *gorm.DB, instanceID string, now time.Time, limit int) error {
	var ids []uint64
	if limit < 1 || limit > 500 {
		limit = 500
	}
	if err := tx.Model(&Entry{}).Where("instance_id = ? AND expires_at <= ?", instanceID, now).Order("id").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("instance_id = ? AND id IN ?", instanceID, ids).Delete(&Entry{}).Error
}
