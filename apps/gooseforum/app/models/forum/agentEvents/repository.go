package agentEvents

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

func PublicationTx(tx *gorm.DB, instance string, postID uint64) (Publication, error) {
	var p Publication
	err := tx.Where("instance_id = ? AND post_id = ?", instance, postID).Take(&p).Error
	return p, err
}
func SetPublicationTx(tx *gorm.DB, p Publication) error {
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "post_id"}}, DoUpdates: clause.AssignmentColumns([]string{"version"})}).Create(&p).Error
}
func CreateIntentTx(tx *gorm.DB, i *Intent) error { return tx.Create(i).Error }
func UpdateIntentTx(tx *gorm.DB, id string, status, lastError string) error {
	return tx.Model(&Intent{}).Where("id = ?", id).Updates(map[string]any{"status": status, "last_error": lastError}).Error
}
func BindTaskTx(tx *gorm.DB, id string, taskID uint64) error {
	return tx.Model(&Intent{}).Where("id = ?", id).Update("task_id", taskID).Error
}
func IntentTx(tx *gorm.DB, instance, id string) (Intent, error) {
	var i Intent
	err := tx.Where("instance_id = ? AND id = ?", instance, id).Take(&i).Error
	return i, err
}
func EventTx(tx *gorm.DB, instance string, agentID uint64, id string) (Entity, error) {
	var e Entity
	err := tx.Where("instance_id = ? AND agent_id = ? AND id = ?", instance, agentID, id).Take(&e).Error
	return e, err
}
func CreateEventTx(tx *gorm.DB, e *Entity) error { return tx.Create(e).Error }
func ListTx(tx *gorm.DB, instance string, agentID, after uint64, limit int) ([]Entity, error) {
	rows := make([]Entity, 0)
	err := tx.Where("instance_id = ? AND agent_id = ? AND seq > ?", instance, agentID, after).Order("seq").Limit(limit).Find(&rows).Error
	return rows, err
}
func ReplayFloorTx(tx *gorm.DB, instance string, agentID uint64, now time.Time) (uint64, error) {
	var floor uint64
	err := tx.Model(&Entity{}).Where("instance_id = ? AND agent_id = ?", instance, agentID).Select("COALESCE(MIN(CASE WHEN expires_at > ? THEN seq END) - 1, MAX(seq), 0)", now).Scan(&floor).Error
	if err != nil {
		return 0, err
	}
	var retained int64
	if err := tx.Model(&Entity{}).Where("instance_id = ? AND agent_id = ? AND expires_at > ?", instance, agentID, now).Count(&retained).Error; err != nil {
		return 0, err
	}
	if retained > 0 {
		return floor, nil
	}
	var state ReplayState
	if err := tx.Where("instance_id = ? AND agent_id = ?", instance, agentID).Take(&state).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	return max(floor, state.PurgedThrough), nil
}
func AckTx(tx *gorm.DB, instance string, agentID uint64, ids []string, now time.Time) error {
	return tx.Model(&Entity{}).Where("instance_id = ? AND agent_id = ? AND id IN ? AND acked_at IS NULL", instance, agentID, ids).Update("acked_at", now).Error
}
func WithdrawTx(tx *gorm.DB, e *Entity, now time.Time) error {
	if e.WithdrawnAt != nil {
		return nil
	}
	e.WithdrawnAt = &now
	e.ActorID = 0
	e.Reasons = []string{}
	e.TopicID = 0
	e.PostID = 0
	e.PostNo = 0
	e.ReplyToPostID = 0
	e.ResultingPostID = 0
	e.ResultingTopicID = 0
	return tx.Model(&Entity{}).Where("id = ? AND instance_id = ?", e.ID, e.InstanceID).Updates(map[string]any{"withdrawn_at": now, "actor_id": 0, "reasons": "[]", "topic_id": 0, "post_id": 0, "post_no": 0, "reply_to_post_id": 0, "resulting_post_id": 0, "resulting_topic_id": 0}).Error
}
func RecordResultTx(tx *gorm.DB, instance string, agentID uint64, eventID string, topicID, postID uint64) error {
	return tx.Model(&Entity{}).Where("instance_id = ? AND agent_id = ? AND id = ?", instance, agentID, eventID).Updates(map[string]any{"resulting_topic_id": topicID, "resulting_post_id": postID}).Error
}

func ListIntentsTx(tx *gorm.DB, instance string, limit int) ([]Intent, error) {
	rows := make([]Intent, 0)
	err := tx.Where("instance_id = ?", instance).Order("created_at DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

// ContentEventsTx returns only retained references for a specific lifecycle
// target. The caller already owns the corresponding content lock.
func ContentEventsTx(tx *gorm.DB, instance string, topicID, postID uint64) ([]Entity, error) {
	rows := make([]Entity, 0)
	q := tx.Where("instance_id = ? AND withdrawn_at IS NULL", instance)
	if postID != 0 {
		q = q.Where("post_id = ?", postID)
	} else {
		q = q.Where("topic_id = ?", topicID)
	}
	err := q.Order("agent_id ASC").Order("seq ASC").Find(&rows).Error
	return rows, err
}
func CancelPostIntentsTx(tx *gorm.DB, instance string, postID uint64) error {
	return tx.Model(&Intent{}).Where("instance_id = ? AND post_id = ?", instance, postID).Updates(map[string]any{"status": "cancelled", "last_error": "source_withdrawn", "actor_id": 0}).Error
}

func PageIntentsForAgentTx(tx *gorm.DB, instance string, agentID uint64, page, size int) ([]Intent, int64, error) {
	condition := "EXISTS (SELECT 1 FROM json_each(recipients) WHERE CAST(json_extract(value, '$.agentId') AS INTEGER) = ?)"
	if tx.Name() == "postgres" {
		condition = "EXISTS (SELECT 1 FROM jsonb_array_elements(recipients::jsonb) AS target WHERE (target->>'agentId')::bigint = ?)"
	}
	q := tx.Model(&Intent{}).Where("instance_id = ?", instance).Where(condition, agentID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]Intent, 0)
	err := q.Order("created_at DESC").Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error
	return rows, total, err
}

func ExpiringEventsTx(tx *gorm.DB, instance string, now time.Time, limit int) ([]Entity, error) {
	var rows []Entity
	err := tx.Where("instance_id = ? AND expires_at <= ? AND (actor_id <> 0 OR post_id <> 0 OR withdrawn_at IS NULL)", instance, now).Order("agent_id ASC").Order("seq ASC").Limit(limit).Find(&rows).Error
	return rows, err
}
func ExpireIntentsTx(tx *gorm.DB, instance string, now time.Time, limit int) error {
	var ids []string
	if err := tx.Model(&Intent{}).Where("instance_id = ? AND expires_at <= ? AND status <> 'expired'", instance, now).Order("created_at ASC").Limit(limit).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	// The source-version unique index covers live references only, allowing
	// expired diagnostics to redact their source columns without collisions.
	return tx.Model(&Intent{}).Where("instance_id = ? AND id IN ?", instance, ids).Updates(map[string]any{"status": "expired", "last_error": "retention_expired", "actor_id": 0, "post_id": 0, "version": 0, "previous_version": 0}).Error
}

// PurgeMetadataTx removes expired diagnostics in bounded batches. The caller
// supplies a cutoff beyond the event retention window (currently 30 days).
func PurgeMetadataTx(tx *gorm.DB, instance string, before time.Time, batch int) error {
	var rows []Entity
	if err := tx.Where("instance_id = ? AND expires_at < ?", instance, before).Order("agent_id ASC, seq ASC").Limit(batch).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		state := ReplayState{InstanceID: instance, AgentID: row.AgentID, PurgedThrough: row.Seq}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "agent_id"}}, DoUpdates: clause.Assignments(map[string]any{"purged_through": gorm.Expr("CASE WHEN agent_event_replay_states.purged_through > ? THEN agent_event_replay_states.purged_through ELSE ? END", row.Seq, row.Seq)})}).Create(&state).Error; err != nil {
			return err
		}
		if err := tx.Where("instance_id = ? AND id = ?", instance, row.ID).Delete(&Entity{}).Error; err != nil {
			return err
		}
	}
	var ids []string
	if err := tx.Model(&Intent{}).Where("instance_id = ? AND expires_at < ? AND status = 'expired'", instance, before).Order("created_at ASC").Limit(batch).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("instance_id = ? AND id IN ?", instance, ids).Delete(&Intent{}).Error
}

func ActorEventsTx(tx *gorm.DB, instance string, userID uint64) ([]Entity, error) {
	var rows []Entity
	err := tx.Where("instance_id = ? AND (actor_id = ? OR agent_id = ?) AND withdrawn_at IS NULL", instance, userID, userID).Order("agent_id ASC").Order("seq ASC").Find(&rows).Error
	return rows, err
}
func CancelActorIntentsTx(tx *gorm.DB, instance string, userID uint64) error {
	return tx.Model(&Intent{}).Where("instance_id = ? AND actor_id = ?", instance, userID).Updates(map[string]any{"status": "cancelled", "last_error": "actor_unavailable", "actor_id": 0}).Error
}

func AgentEventsTx(tx *gorm.DB, instance string, agentID uint64) ([]Entity, error) {
	var rows []Entity
	err := tx.Where("instance_id = ? AND agent_id = ? AND withdrawn_at IS NULL", instance, agentID).Order("seq ASC").Find(&rows).Error
	return rows, err
}
