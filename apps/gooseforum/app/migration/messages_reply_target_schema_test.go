package migration

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type legacyMessageSchema struct {
	Id              uint64    `gorm:"primaryKey;column:id;autoIncrement;not null"`
	ConvId          uint64    `gorm:"column:conv_id;not null;default:0"`
	SenderId        uint64    `gorm:"column:sender_id;not null;default:0"`
	Content         string    `gorm:"column:content;type:text;not null"`
	MsgType         int8      `gorm:"column:msg_type;not null;default:1"`
	IsRead          int       `gorm:"column:is_read;not null;default:0"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	ClientMessageID *string   `gorm:"column:client_message_id;type:varchar(64)"`
}

func (legacyMessageSchema) TableName() string { return "messages" }

func TestMessageReplyTargetUpgradePreservesLegacyRows(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&legacyMessageSchema{}); err != nil {
		t.Fatalf("create legacy messages table: %v", err)
	}
	if err := db.Create(&legacyMessageSchema{
		Id: 701, ConvId: 702, SenderId: 703, Content: "old reply", MsgType: 1, CreatedAt: time.Unix(704, 0),
	}).Error; err != nil {
		t.Fatalf("insert legacy message: %v", err)
	}

	if err := db.AutoMigrate(&messages.Entity{}); err != nil {
		t.Fatalf("upgrade messages table: %v", err)
	}
	if !db.Migrator().HasColumn(&messages.Entity{}, "ReplyToMessageID") {
		t.Fatal("reply_to_message_id column missing after upgrade")
	}
	var upgraded messages.Entity
	if err := db.First(&upgraded, 701).Error; err != nil {
		t.Fatalf("read legacy message after upgrade: %v", err)
	}
	var replyTo sql.NullInt64
	if err := db.Raw("SELECT reply_to_message_id FROM messages WHERE id = ?", upgraded.Id).Scan(&replyTo).Error; err != nil {
		t.Fatalf("read legacy reply target: %v", err)
	}
	if upgraded.Content != "old reply" || replyTo.Valid {
		t.Fatalf("upgraded legacy message content=%q replyTo=%v, want preserved content and null target", upgraded.Content, replyTo)
	}
}
