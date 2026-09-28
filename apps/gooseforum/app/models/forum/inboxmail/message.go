package inboxmail

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Message 状态：active 可用于新建 Campaign/Trigger，disabled 停止新引用，
// archived 表示内容已下线但历史投递仍可回溯。
const (
	MessageStatusActive   = "active"
	MessageStatusDisabled = "disabled"
	MessageStatusArchived = "archived"
)

// Message Version 状态：只有 draft 可编辑；published 之后整行不可变。
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
)

// Block 是受控 typed block 的信封：Type 是 Special Block Registry 的键（#771），
// Payload 由该 block 类型自行解释。禁止任意 HTML/JS/iframe，渲染器对未知 Type
// 必须有 fallback。
type Block struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// VersionContent 是 Message Version 的可编辑内容载荷（Markdown 为真源）。
type VersionContent struct {
	Title         string
	Summary       string
	BodyMarkdown  string
	Blocks        []Block
	SchemaVersion int
}

// MessageEntity 是逻辑消息（模板）身份：稳定 code 供 Trigger Registry 与管理端
// 引用，具体内容永远存放在不可变的 Message Version 中。
type MessageEntity struct {
	Id        uint64    `gorm:"primaryKey;column:id;autoIncrement;not null" json:"id"`
	Code      string    `gorm:"column:code;type:varchar(128);not null;uniqueIndex:uniq_inbox_message_code;check:chk_inbox_message_code,code <> ''" json:"code"`
	Name      string    `gorm:"column:name;type:varchar(128);not null;default:''" json:"name"`
	Status    string    `gorm:"column:status;type:varchar(16);not null;default:'active';index:idx_inbox_message_status" json:"status"`
	CreatedBy uint64    `gorm:"column:created_by;not null;default:0" json:"createdBy"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *MessageEntity) TableName() string {
	return messageTableName
}

// MessageVersionEntity 是发布内容的不可变快照。status=published 后
// title/summary/body_markdown/blocks/content_hash/schema_version/version_no
// 均不得原地修改：修订必须新建一行，历史 Delivery 始终引用固定版本。
//
// 不可变性由三层共同保证：1) 本类型的 BeforeUpdate/BeforeDelete hook；
// 2) 仓储函数 PublishVersionTx / UpdateDraftContentTx 的条件更新；
// 3) （message_id, version_no）唯一索引防止重复版本号。
type MessageVersionEntity struct {
	Id            uint64     `gorm:"primaryKey;column:id;autoIncrement;not null" json:"id"`
	MessageId     uint64     `gorm:"column:message_id;not null;default:0;uniqueIndex:uniq_inbox_message_version_no,priority:1;index:idx_inbox_message_version_message" json:"messageId"`
	VersionNo     int        `gorm:"column:version_no;not null;default:1;uniqueIndex:uniq_inbox_message_version_no,priority:2" json:"versionNo"`
	Status        string     `gorm:"column:status;type:varchar(16);not null;default:'draft';index:idx_inbox_message_version_status" json:"status"`
	Title         string     `gorm:"column:title;type:varchar(255);not null;default:''" json:"title"`
	Summary       string     `gorm:"column:summary;type:varchar(500);not null;default:''" json:"summary"`
	BodyMarkdown  string     `gorm:"column:body_markdown;type:text;not null;default:''" json:"bodyMarkdown"`
	Blocks        []Block    `gorm:"column:blocks;type:json;serializer:json" json:"blocks"`
	SchemaVersion int        `gorm:"column:schema_version;not null;default:1" json:"schemaVersion"`
	ContentHash   string     `gorm:"column:content_hash;type:varchar(64);not null;default:''" json:"contentHash"`
	PublishedAt   *time.Time `gorm:"column:published_at;type:timestamp" json:"publishedAt,omitempty"`
	PublishedBy   uint64     `gorm:"column:published_by;not null;default:0" json:"publishedBy"`
	CreatedBy     uint64     `gorm:"column:created_by;not null;default:0" json:"createdBy"`
	CreatedAt     time.Time  `gorm:"column:created_at;autoCreateTime;<-:create" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (itself *MessageVersionEntity) TableName() string {
	return messageVersionTableName
}

// Content 还原可哈希/可渲染的内容载荷。
func (itself MessageVersionEntity) Content() VersionContent {
	return VersionContent{
		Title:         itself.Title,
		Summary:       itself.Summary,
		BodyMarkdown:  itself.BodyMarkdown,
		Blocks:        itself.Blocks,
		SchemaVersion: itself.SchemaVersion,
	}
}

// BeforeUpdate / BeforeDelete 拒绝任何针对已发布（或无法定位目标行）的原地修改。
// 生命周期状态迁移（发布）由 PublishVersionTx 通过条件更新完成，不经过模型 hook。
func (itself *MessageVersionEntity) BeforeUpdate(tx *gorm.DB) error {
	return itself.guardPublishedMutation(tx)
}

func (itself *MessageVersionEntity) BeforeDelete(tx *gorm.DB) error {
	return itself.guardPublishedMutation(tx)
}

func (itself *MessageVersionEntity) guardPublishedMutation(tx *gorm.DB) error {
	if itself.Status == StatusPublished {
		return fmt.Errorf("%w: id=%d", ErrPublishedVersionImmutable, itself.Id)
	}
	if itself.Id == 0 {
		// 无主键的批量模型更新无法确认目标行是否为草稿；宁可拒绝，也不冒改写
		// 已发布历史的风险（草稿修改请用 UpdateDraftContentTx）。
		return fmt.Errorf("%w: model update without primary key", ErrUnscopedVersionMutation)
	}
	var persistedStatus string
	if err := tx.Session(&gorm.Session{NewDB: true}).
		Table(messageVersionTableName).
		Where("id = ?", itself.Id).
		Limit(1).
		Pluck("status", &persistedStatus).Error; err != nil {
		return err
	}
	if persistedStatus == StatusPublished {
		return fmt.Errorf("%w: id=%d", ErrPublishedVersionImmutable, itself.Id)
	}
	return nil
}

// HashVersionContent 计算内容哈希（SHA-256 hex）。哈希前先做规范化：
// Block payload 解码为通用值后重新序列化，Go 的 JSON 编码对 map key 排序，
// 因此不同序列化路径产生的等价内容得到同一哈希，幂等比较不会误判为内容变更。
func HashVersionContent(content VersionContent) (string, error) {
	canonical, err := canonicalizeContent(content)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("inboxmail: marshal canonical version content: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

type canonicalVersionContent struct {
	Title         string           `json:"title"`
	Summary       string           `json:"summary"`
	BodyMarkdown  string           `json:"bodyMarkdown"`
	SchemaVersion int              `json:"schemaVersion"`
	Blocks        []canonicalBlock `json:"blocks"`
}

type canonicalBlock struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

func canonicalizeContent(content VersionContent) (canonicalVersionContent, error) {
	blocks := make([]canonicalBlock, 0, len(content.Blocks))
	for _, block := range content.Blocks {
		var payload any
		if len(block.Payload) > 0 {
			// UseNumber 保留数字字面量精度：默认解码为 float64 会让
			// 9007199254740992/9007199254740993 这类大整数在哈希里碰撞。
			decoder := json.NewDecoder(bytes.NewReader(block.Payload))
			decoder.UseNumber()
			if err := decoder.Decode(&payload); err != nil {
				return canonicalVersionContent{}, fmt.Errorf("inboxmail: block %q payload is not valid JSON: %w", block.Type, err)
			}
		}
		blocks = append(blocks, canonicalBlock{Type: block.Type, Payload: payload})
	}
	return canonicalVersionContent{
		Title:         content.Title,
		Summary:       content.Summary,
		BodyMarkdown:  content.BodyMarkdown,
		SchemaVersion: normalizeSchemaVersion(content.SchemaVersion),
		Blocks:        blocks,
	}, nil
}

// DefaultContentSchemaVersion 是 blocks/正文契约的初始 schema 版本。
const DefaultContentSchemaVersion = 1

func normalizeSchemaVersion(schemaVersion int) int {
	if schemaVersion <= 0 {
		return DefaultContentSchemaVersion
	}
	return schemaVersion
}
