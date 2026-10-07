package agents

import (
	"time"

	"errors"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func builder() *gorm.DB {
	return db.Connect().Table(tableName)
}

// GetByUserID 根据机器人用户id获取 Agent 记录。
func GetByUserID(userID uint64) *Entity {
	return GetByUserIDWithDB(nil, userID)
}

// GetByUserIDWithDB reads through tx when called from a transaction, so the
// lookup observes the transaction snapshot and does not contend for another
// SQLite connection.
func GetByUserIDWithDB(tx *gorm.DB, userID uint64) *Entity {
	if tx == nil {
		tx = db.Connect()
	}
	var entity Entity
	err := tx.Table(tableName).Where(fieldUserId+" = ?", userID).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// GetByTokenPrefix 按令牌前缀查找 Agent 记录（前缀非敏感，可建索引）。
func GetByTokenPrefix(prefix string) *Entity {
	var entity Entity
	err := builder().Where(fieldTokenPrefix+" = ?", prefix).First(&entity).Error
	if err != nil {
		return nil
	}
	return &entity
}

// UpdateColumns updates only fields owned by the current operation. This avoids
// stale full-row snapshots reverting a concurrent disable or token rotation.
func UpdateColumns(tx *gorm.DB, userID uint64, values map[string]any) error {
	if tx == nil {
		tx = db.Connect()
	}
	result := tx.Model(&Entity{}).Where(fieldUserId+" = ?", userID).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var count int64
		if err := tx.Table(tableName).Where(fieldUserId+" = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// UpdateTokenCAS replaces the credential columns only when the current
// token_prefix still matches oldPrefix. It returns the number of affected
// rows: 1 on success, 0 when a concurrent rotation already changed the
// prefix. This makes concurrent rotations fail loudly instead of silently
// discarding one of the two new tokens.
func UpdateTokenCAS(userID uint64, oldPrefix, newPrefix, hash string) (int64, error) {
	result := builder().
		Where(fieldUserId+" = ?", userID).
		Where(fieldTokenPrefix+" = ?", oldPrefix).
		Updates(map[string]any{
			fieldTokenPrefix: newPrefix,
			fieldTokenHash:   hash,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// RevokeCredential disables the agent and clears its token hash. The
// non-secret token prefix is retained as the unique index and the rotation
// CAS anchor; a revoked credential can never validate again, and re-enabling
// requires an explicit rotation first.
func RevokeCredential(userID uint64) error { return RevokeCredentialTx(nil, userID) }
func RevokeCredentialTx(tx *gorm.DB, userID uint64) error {
	return UpdateColumns(tx, userID, map[string]any{
		fieldEnabled:              StatusDisabled,
		fieldTokenHash:            "",
		"webhook_enabled":         false,
		"events_enabled":          false,
		"endpoint_generation":     gorm.Expr("endpoint_generation + 1"),
		"subscription_generation": gorm.Expr("subscription_generation + 1"),
		"config_version":          gorm.Expr("config_version + 1"),
	})
}

// List 按创建时间倒序返回全部 Agent 记录。
func List() []*Entity {
	var entities []*Entity
	builder().Order(fieldCreatedAt + " desc").Find(&entities)
	return entities
}

// TouchLastUsedAt 更新最近使用时间。
func TouchLastUsedAt(userID uint64, lastUsedAt time.Time) error {
	return builder().
		Where(fieldUserId+" = ?", userID).
		Update(fieldLastUsedAt, lastUsedAt).Error
}

// GetTx locks the configuration row before event sequence or send authorization.
func GetTx(tx *gorm.DB, userID uint64, lock bool) (*Entity, error) {
	var row Entity
	q := tx.Where("user_id = ?", userID)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&row).Error
	return &row, err
}
func ReserveEventSeqTx(tx *gorm.DB, userID uint64) (uint64, error) {
	row, err := GetTx(tx, userID, true)
	if err != nil {
		return 0, err
	}
	seq := row.EventSeq + 1
	err = UpdateColumns(tx, userID, map[string]any{"event_seq": seq})
	return seq, err
}

var ErrConfigConflict = errors.New("agent config conflict")

func UpdateConfigTx(tx *gorm.DB, userID, version uint64, columns map[string]any) error {
	columns["config_version"] = version + 1
	r := tx.Model(&Entity{}).Where("user_id = ? AND config_version = ?", userID, version).Updates(columns)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected != 1 {
		return ErrConfigConflict
	}
	return nil
}
func ListEnabledTx(tx *gorm.DB) ([]Entity, error) {
	var rows []Entity
	err := tx.Where("enabled = ? AND events_enabled = ?", StatusEnabled, true).Order("user_id asc").Find(&rows).Error
	return rows, err
}

// IsolateSnapshot disables copied credentials before an instance can serve.
func IsolateSnapshot(tx *gorm.DB) error {
	return tx.Model(&Entity{}).Where("1 = 1").Updates(map[string]any{"enabled": StatusDisabled, "token_hash": "", "secret_ciphertext": "", "previous_secret_ciphertext": "", "previous_secret_expires_at": nil, "webhook_enabled": false, "events_enabled": false, "endpoint_generation": gorm.Expr("endpoint_generation + 1"), "subscription_generation": gorm.Expr("subscription_generation + 1"), "config_version": gorm.Expr("config_version + 1")}).Error
}

func EnabledIDsTx(tx *gorm.DB) ([]uint64, error) {
	var ids []uint64
	err := tx.Model(&Entity{}).Where("enabled = ?", StatusEnabled).Order("user_id asc").Pluck("user_id", &ids).Error
	return ids, err
}
func LockTx(tx *gorm.DB, agentID uint64) (*Entity, error) { return GetTx(tx, agentID, true) }

// Worker-owned endpoint diagnostics cannot pause a replacement endpoint.
func UpdateWebhookGenerationTx(tx *gorm.DB, agentID, generation uint64, columns map[string]any) error {
	return tx.Model(&Entity{}).Where("user_id = ? AND endpoint_generation = ?", agentID, generation).Updates(columns).Error
}
