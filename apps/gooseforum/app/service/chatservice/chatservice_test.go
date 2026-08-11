package chatservice

import (
	"testing"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/models/chat/imConversations"
	"github.com/leancodebox/GooseForum/app/models/chat/imUserChatConfigs"
	"github.com/leancodebox/GooseForum/app/models/chat/messages"
)

// mark-read 测试使用的固定 ID；选取与会话/用户无关的数以避免与其他测试冲突。
const (
	markReadTestConvID  = uint64(5001)
	markReadTestSender  = uint64(1) // member 发送方
	markReadTestMember  = uint64(2) // member 接收方，有未读
	markReadTestNonUser = uint64(3) // 用户 3 不属于 conv 5001
	markReadTestMsgID   = uint64(6001)
)

func setupMarkReadTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&imConversations.Entity{},
		&imUserChatConfigs.Entity{},
		&messages.Entity{},
	); err != nil {
		t.Fatalf("autoMigrate chat tables: %v", err)
	}

	// 清空成员校验缓存，避免前序用例残留的命中覆盖本次 seed 后的真实状态。
	imUserChatConfigs.InvalidateConversationAccess(markReadTestSender, markReadTestConvID)
	imUserChatConfigs.InvalidateConversationAccess(markReadTestMember, markReadTestConvID)
	imUserChatConfigs.InvalidateConversationAccess(markReadTestNonUser, markReadTestConvID)

	if err := conn.Where("conv_id = ?", markReadTestConvID).Delete(&messages.Entity{}).Error; err != nil {
		t.Fatalf("cleanup messages: %v", err)
	}
	if err := conn.Where("conv_id = ?", markReadTestConvID).Delete(&imUserChatConfigs.Entity{}).Error; err != nil {
		t.Fatalf("cleanup configs: %v", err)
	}
	if err := conn.Where("id = ?", markReadTestConvID).Delete(&imConversations.Entity{}).Error; err != nil {
		t.Fatalf("cleanup conversations: %v", err)
	}

	now := time.Date(2026, 8, 11, 10, 0, 0, 0, time.UTC)
	if err := conn.Create(&imConversations.Entity{
		Id: markReadTestConvID, Type: 1,
		LastMsgContent: "hello", LastMsgTime: now,
	}).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := conn.Create(&[]imUserChatConfigs.Entity{
		{UserId: markReadTestSender, PeerId: markReadTestMember, ConvId: markReadTestConvID, UnreadCount: 0, UpdatedAt: now},
		{UserId: markReadTestMember, PeerId: markReadTestSender, ConvId: markReadTestConvID, UnreadCount: 1, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatalf("create configs: %v", err)
	}
	if err := conn.Create(&messages.Entity{
		Id: markReadTestMsgID, ConvId: markReadTestConvID, SenderId: markReadTestSender,
		Content: "hello", MsgType: 1, IsRead: 0, CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create message: %v", err)
	}
}

func unreadCountOr(cfg *imUserChatConfigs.Entity) any {
	if cfg == nil {
		return "<nil config>"
	}
	return cfg.UnreadCount
}

func TestMarkReadRejectsNonMember(t *testing.T) {
	setupMarkReadTestDB(t)

	err := MarkRead(markReadTestNonUser, markReadTestConvID)
	if err == nil {
		t.Fatal("MarkRead(non-member) returned nil, want membership error")
	}

	// 越权调用不得触碰该会话任何状态：peer 未读数必须保持 1。
	cfg := imUserChatConfigs.GetConfig(markReadTestMember, markReadTestSender)
	if cfg == nil || cfg.UnreadCount != 1 {
		t.Fatalf("peer UnreadCount = %v, want 1 (non-member must not affect state)", unreadCountOr(cfg))
	}

	// 越权调用不得把消息 is_read 翻为 1。
	var msg messages.Entity
	if err := db.Connect().Where("id = ?", markReadTestMsgID).First(&msg).Error; err != nil {
		t.Fatalf("load message: %v", err)
	}
	if msg.IsRead != 0 {
		t.Fatalf("message IsRead = %d, want 0 (non-member must not affect state)", msg.IsRead)
	}
}

func TestMarkReadMemberSucceeds(t *testing.T) {
	setupMarkReadTestDB(t)

	if err := MarkRead(markReadTestMember, markReadTestConvID); err != nil {
		t.Fatalf("MarkRead(member) error = %v", err)
	}

	// 成员调用必须清零未读。
	cfg := imUserChatConfigs.GetConfig(markReadTestMember, markReadTestSender)
	if cfg == nil || cfg.UnreadCount != 0 {
		t.Fatalf("peer UnreadCount = %v, want 0 after member marks read", unreadCountOr(cfg))
	}

	// 成员调用必须把消息置为已读。
	var msg messages.Entity
	if err := db.Connect().Where("id = ?", markReadTestMsgID).First(&msg).Error; err != nil {
		t.Fatalf("load message: %v", err)
	}
	if msg.IsRead != 1 {
		t.Fatalf("message IsRead = %d, want 1 after member marks read", msg.IsRead)
	}
}
