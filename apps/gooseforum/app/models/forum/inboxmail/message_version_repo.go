package inboxmail

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	// ErrVersionNotFound 目标 Message Version 不存在。
	ErrVersionNotFound = errors.New("inboxmail: message version not found")
	// ErrPublishedVersionImmutable 已发布版本不可原地修改或删除。
	ErrPublishedVersionImmutable = errors.New("inboxmail: published message version is immutable")
	// ErrUnscopedVersionMutation 模型层更新缺少主键，无法确认目标行是否为草稿。
	ErrUnscopedVersionMutation = errors.New("inboxmail: message version mutation requires a primary key")
	// ErrDraftVersionRequired 操作只允许作用于草稿版本。
	ErrDraftVersionRequired = errors.New("inboxmail: operation requires a draft message version")
	// ErrMessageCodeRequired Message code 是 Trigger Registry 的稳定引用键，不能为空。
	ErrMessageCodeRequired = errors.New("inboxmail: message code is required")
)

// CreateMessageTx 创建逻辑消息。code 全局唯一；重复 code 返回数据库唯一约束错误
// （gorm.ErrDuplicatedKey），调用方据此做幂等创建。空 code 在此直接拒绝，
// chk_inbox_message_code 作为绕过 Go 层写入的兜底。
func CreateMessageTx(tx *gorm.DB, code, name string, createdBy uint64) (MessageEntity, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return MessageEntity{}, ErrMessageCodeRequired
	}
	message := MessageEntity{
		Code:      code,
		Name:      name,
		Status:    MessageStatusActive,
		CreatedBy: createdBy,
	}
	if err := tx.Create(&message).Error; err != nil {
		return MessageEntity{}, err
	}
	return message, nil
}

// CreateDraftVersionTx 为消息创建下一个版本的草稿（version_no = max+1）。
// 并发创建相同版本号时数据库唯一索引会拒绝后到者，调用方需重试。
func CreateDraftVersionTx(tx *gorm.DB, messageID uint64, content VersionContent, createdBy uint64) (MessageVersionEntity, error) {
	if messageID == 0 {
		return MessageVersionEntity{}, ErrVersionNotFound
	}
	versionNo, err := nextVersionNoTx(tx, messageID)
	if err != nil {
		return MessageVersionEntity{}, err
	}
	contentHash, err := HashVersionContent(content)
	if err != nil {
		return MessageVersionEntity{}, err
	}
	version := MessageVersionEntity{
		MessageId:     messageID,
		VersionNo:     versionNo,
		Status:        StatusDraft,
		Title:         content.Title,
		Summary:       content.Summary,
		BodyMarkdown:  content.BodyMarkdown,
		Blocks:        content.Blocks,
		SchemaVersion: normalizeSchemaVersion(content.SchemaVersion),
		ContentHash:   contentHash,
		CreatedBy:     createdBy,
	}
	if err := tx.Create(&version).Error; err != nil {
		return MessageVersionEntity{}, err
	}
	return version, nil
}

// UpdateDraftContentTx 只更新草稿版本的内容。条件更新（WHERE status='draft'）
// 是最后一道防线：目标已发布时返回 ErrPublishedVersionImmutable，不存在时返回
// ErrVersionNotFound。
func UpdateDraftContentTx(tx *gorm.DB, versionID uint64, content VersionContent) error {
	contentHash, err := HashVersionContent(content)
	if err != nil {
		return err
	}
	// Blocks 必须先自行序列化：GORM 的 Updates(map) 不会对值应用 serializer:json，
	// 直接传结构体会让两种方言的 json 列驱动报错（eventNotification 同款处理）。
	blocks, err := json.Marshal(content.Blocks)
	if err != nil {
		return fmt.Errorf("inboxmail: marshal blocks: %w", err)
	}
	result := tx.Table(messageVersionTableName).
		Where("id = ? AND status = ?", versionID, StatusDraft).
		Updates(map[string]any{
			"title":          content.Title,
			"summary":        content.Summary,
			"body_markdown":  content.BodyMarkdown,
			"blocks":         blocks,
			"schema_version": normalizeSchemaVersion(content.SchemaVersion),
			"content_hash":   contentHash,
			"updated_at":     time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return versionMutationError(tx, versionID)
	}
	return nil
}

// PublishVersionTx 把草稿发布为不可变版本：写入发布人、发布时间与重新计算的内容哈希。
// 重复发布同一版本是幂等的（返回已发布行，不修改内容），Worker 重试与事件重放安全。
func PublishVersionTx(tx *gorm.DB, versionID uint64, publishedBy uint64, publishedAt time.Time) (MessageVersionEntity, error) {
	current, err := GetVersionByIDTx(tx, versionID)
	if err != nil {
		return MessageVersionEntity{}, err
	}
	if current.Status == StatusPublished {
		return current, nil
	}
	if current.Status != StatusDraft {
		return MessageVersionEntity{}, fmt.Errorf("%w: id=%d status=%s", ErrDraftVersionRequired, versionID, current.Status)
	}
	contentHash, err := HashVersionContent(current.Content())
	if err != nil {
		return MessageVersionEntity{}, err
	}
	if publishedAt.IsZero() {
		publishedAt = time.Now()
	}
	result := tx.Table(messageVersionTableName).
		Where("id = ? AND status = ?", versionID, StatusDraft).
		Updates(map[string]any{
			"status":       StatusPublished,
			"content_hash": contentHash,
			"published_at": publishedAt,
			"published_by": publishedBy,
			"updated_at":   time.Now(),
		})
	if result.Error != nil {
		return MessageVersionEntity{}, result.Error
	}
	if result.RowsAffected == 0 {
		// 并发发布：另一方已冻结该版本，返回其落库状态即为成功。
		after, err := GetVersionByIDTx(tx, versionID)
		if err != nil {
			return MessageVersionEntity{}, err
		}
		if after.Status == StatusPublished {
			return after, nil
		}
		return MessageVersionEntity{}, fmt.Errorf("%w: id=%d status=%s", ErrDraftVersionRequired, versionID, after.Status)
	}
	return GetVersionByIDTx(tx, versionID)
}

// GetVersionByIDTx 读取单个版本；不存在时返回 gorm.ErrRecordNotFound。
func GetVersionByIDTx(tx *gorm.DB, versionID uint64) (MessageVersionEntity, error) {
	var version MessageVersionEntity
	if err := tx.Table(messageVersionTableName).Where("id = ?", versionID).Take(&version).Error; err != nil {
		return MessageVersionEntity{}, err
	}
	return version, nil
}

// GetLatestPublishedVersionTx 返回消息的最新已发布版本（草稿修订不算）；
// 不存在时返回 gorm.ErrRecordNotFound。
func GetLatestPublishedVersionTx(tx *gorm.DB, messageID uint64) (MessageVersionEntity, error) {
	var version MessageVersionEntity
	err := tx.Table(messageVersionTableName).
		Where("message_id = ? AND status = ?", messageID, StatusPublished).
		Order("version_no DESC").
		Take(&version).Error
	if err != nil {
		return MessageVersionEntity{}, err
	}
	return version, nil
}

func nextVersionNoTx(tx *gorm.DB, messageID uint64) (int, error) {
	var row struct{ MaxNo int }
	if err := tx.Table(messageVersionTableName).
		Select("COALESCE(MAX(version_no), 0) AS max_no").
		Where("message_id = ?", messageID).
		Scan(&row).Error; err != nil {
		return 0, err
	}
	return row.MaxNo + 1, nil
}

func versionMutationError(tx *gorm.DB, versionID uint64) error {
	var persistedStatus string
	if err := tx.Session(&gorm.Session{NewDB: true}).
		Table(messageVersionTableName).
		Where("id = ?", versionID).
		Limit(1).
		Pluck("status", &persistedStatus).Error; err != nil {
		return err
	}
	switch persistedStatus {
	case "":
		return fmt.Errorf("%w: id=%d", ErrVersionNotFound, versionID)
	case StatusPublished:
		return fmt.Errorf("%w: id=%d", ErrPublishedVersionImmutable, versionID)
	default:
		return fmt.Errorf("%w: id=%d status=%s", ErrDraftVersionRequired, versionID, persistedStatus)
	}
}
