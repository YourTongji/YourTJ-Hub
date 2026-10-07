package agentWebhook

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

const (
	Pending   = "pending"
	Running   = "running"
	Accepted  = "accepted"
	RetryWait = "retry_wait"
	Dead      = "dead"
	Cancelled = "cancelled"
)

type Delivery struct {
	ID                 uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	InstanceID         string     `gorm:"type:varchar(128);not null;uniqueIndex:uniq_agent_delivery,priority:1;index:idx_agent_delivery_list,priority:1" json:"instanceId"`
	EventID            string     `gorm:"type:varchar(96);not null;uniqueIndex:uniq_agent_delivery,priority:2" json:"eventId"`
	AgentID            uint64     `gorm:"not null;index:idx_agent_delivery_list,priority:2" json:"agentId"`
	EndpointGeneration uint64     `gorm:"not null;uniqueIndex:uniq_agent_delivery,priority:3" json:"endpointGeneration"`
	SchemaVersion      uint32     `gorm:"not null;default:1" json:"schemaVersion"`
	Status             string     `gorm:"type:varchar(16);not null;default:'pending';index" json:"status"`
	Reason             string     `gorm:"type:varchar(64);not null;default:''" json:"reason"`
	TaskID             uint64     `gorm:"not null;default:0" json:"taskId"`
	Body               string     `gorm:"type:text;not null" json:"-"`
	Round              uint32     `gorm:"not null;default:1" json:"round"`
	AttemptCount       uint32     `gorm:"not null;default:0" json:"attemptCount"`
	TotalAttempts      uint32     `gorm:"not null;default:0" json:"totalAttempts"`
	Deadline           time.Time  `gorm:"not null" json:"deadline"`
	ExpiresAt          time.Time  `gorm:"not null;index" json:"expiresAt"`
	NextRunAt          *time.Time `json:"nextRunAt"`
	PermitExpiresAt    *time.Time `json:"-"`
	LastAttemptID      string     `gorm:"type:varchar(36);not null;default:''" json:"-"`
	CreatedBy          uint64     `gorm:"not null;default:0" json:"createdBy"`
	LastRedeliveredBy  uint64     `gorm:"not null;default:0" json:"lastRedeliveredBy"`
	AcceptedAt         *time.Time `json:"acceptedAt"`
	CreatedAt          time.Time  `gorm:"autoCreateTime;index" json:"createdAt"`
	UpdatedAt          time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (*Delivery) TableName() string { return "agent_webhook_deliveries" }

type Attempt struct {
	ID           string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	InstanceID   string     `gorm:"type:varchar(128);not null;index:idx_agent_attempt,priority:1" json:"instanceId"`
	DeliveryID   uint64     `gorm:"not null;index:idx_agent_attempt,priority:2" json:"deliveryId"`
	Round        uint32     `gorm:"not null" json:"round"`
	Number       uint32     `gorm:"not null" json:"number"`
	HTTPStatus   int        `gorm:"not null;default:0" json:"httpStatus"`
	ErrorClass   string     `gorm:"type:varchar(64);not null;default:''" json:"errorClass"`
	DurationMS   int64      `gorm:"not null;default:0" json:"durationMs"`
	AuthorizedAt time.Time  `gorm:"not null" json:"authorizedAt"`
	CompletedAt  *time.Time `json:"completedAt"`
}

func (*Attempt) TableName() string { return "agent_webhook_attempts" }
func CreateTx(tx *gorm.DB, row *Delivery) (bool, error) {
	r := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "instance_id"}, {Name: "event_id"}, {Name: "endpoint_generation"}}, DoNothing: true}).Create(row)
	return r.RowsAffected == 1, r.Error
}
func GetTx(tx *gorm.DB, instanceID string, id uint64, lock bool) (*Delivery, error) {
	var row Delivery
	q := tx.Where("instance_id = ? AND id = ?", instanceID, id)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&row).Error
	return &row, err
}
func UpdateTx(tx *gorm.DB, instanceID string, id uint64, values map[string]any) error {
	return tx.Model(&Delivery{}).Where("instance_id = ? AND id = ?", instanceID, id).Updates(values).Error
}
func AddAttemptTx(tx *gorm.DB, row *Attempt) error { return tx.Create(row).Error }
func CompleteAttemptTx(tx *gorm.DB, instanceID, id string, values map[string]any) error {
	return tx.Model(&Attempt{}).Where("instance_id = ? AND id = ?", instanceID, id).Updates(values).Error
}
func ListTx(tx *gorm.DB, instanceID string, agentID uint64, offset, limit int) ([]Delivery, int64, error) {
	var rows []Delivery
	var n int64
	q := tx.Model(&Delivery{}).Where("instance_id = ? AND agent_id = ?", instanceID, agentID)
	if err := q.Count(&n).Error; err != nil {
		return nil, 0, err
	}
	err := q.Order("id desc").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, n, err
}
func AttemptsTx(tx *gorm.DB, instanceID string, id uint64) ([]Attempt, error) {
	var rows []Attempt
	err := tx.Where("instance_id = ? AND delivery_id = ?", instanceID, id).Order("authorized_at asc").Limit(100).Find(&rows).Error
	return rows, err
}
func CancelGenerationTx(tx *gorm.DB, instanceID string, agentID, generation uint64, reason string) error {
	return tx.Model(&Delivery{}).Where("instance_id = ? AND agent_id = ? AND endpoint_generation <> ? AND status IN ?", instanceID, agentID, generation, []string{Pending, RetryWait}).Updates(map[string]any{"status": Cancelled, "reason": reason, "body": ""}).Error
}

// RedactBySourceEventsTx removes all retained payload copies, including accepted deliveries.
func RedactBySourceEventsTx(tx *gorm.DB, instanceID string, eventIDs []string) error {
	return redactBySourceEventsTx(tx, instanceID, eventIDs, "withdrawn")
}

// RedactExpiredSourceEventsTx preserves retention's diagnostic reason even
// when an already authorized send completes after the retained copy is erased.
func RedactExpiredSourceEventsTx(tx *gorm.DB, instanceID string, eventIDs []string) error {
	return redactBySourceEventsTx(tx, instanceID, eventIDs, "expired")
}

func redactBySourceEventsTx(tx *gorm.DB, instanceID string, eventIDs []string, reason string) error {
	if len(eventIDs) == 0 {
		return nil
	}
	q := tx.Model(&Delivery{}).Where("instance_id = ? AND event_id IN ?", instanceID, eventIDs)
	if err := q.Updates(map[string]any{"reason": reason, "body": ""}).Error; err != nil {
		return err
	}
	return tx.Model(&Delivery{}).Where("instance_id = ? AND event_id IN ? AND status IN ?", instanceID, eventIDs, []string{Pending, RetryWait, Dead}).Update("status", Cancelled).Error
}
func TasksForEventsTx(tx *gorm.DB, instanceID string, eventIDs []string) ([]uint64, error) {
	var ids []uint64
	if len(eventIDs) == 0 {
		return ids, nil
	}
	err := tx.Model(&Delivery{}).Where("instance_id = ? AND event_id IN ?", instanceID, eventIDs).Pluck("task_id", &ids).Error
	return ids, err
}

func ExpireTx(tx *gorm.DB, instanceID string, now time.Time) error {
	var ids []uint64
	if err := tx.Model(&Delivery{}).Where("instance_id = ? AND expires_at <= ? AND body <> ''", instanceID, now).Order("id asc").Limit(500).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return tx.Model(&Delivery{}).Where("instance_id = ? AND id IN ?", instanceID, ids).Updates(map[string]any{"status": Cancelled, "reason": "expired", "body": ""}).Error
}

func PendingCountTx(tx *gorm.DB, instanceID string, agentID uint64) (int64, error) {
	var n int64
	err := tx.Model(&Delivery{}).Where("instance_id = ? AND agent_id = ? AND status IN ?", instanceID, agentID, []string{Pending, Running, RetryWait}).Count(&n).Error
	return n, err
}
func PurgeAttemptsTx(tx *gorm.DB, instanceID string, before time.Time, batch int) error {
	var ids []string
	if err := tx.Model(&Attempt{}).Where("instance_id = ? AND authorized_at < ?", instanceID, before).Order("authorized_at asc").Limit(batch).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("instance_id = ? AND id IN ?", instanceID, ids).Delete(&Attempt{}).Error
}

// PurgeMetadataTx bounds retained delivery diagnostics after their expiry.
// Running permits remain fenced until their own completion/recovery.
func PurgeMetadataTx(tx *gorm.DB, instanceID string, before time.Time, batch int) error {
	var ids []uint64
	if err := tx.Model(&Delivery{}).Where("instance_id = ? AND expires_at < ? AND body = '' AND status <> ?", instanceID, before, Running).Order("id ASC").Limit(batch).Pluck("id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("instance_id = ? AND delivery_id IN ?", instanceID, ids).Delete(&Attempt{}).Error; err != nil {
		return err
	}
	return tx.Where("instance_id = ? AND id IN ?", instanceID, ids).Delete(&Delivery{}).Error
}
