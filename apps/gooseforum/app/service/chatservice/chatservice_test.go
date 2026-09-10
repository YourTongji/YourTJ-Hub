package chatservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imConversations"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
)

// mark-read 测试使用的固定 ID；选取与会话/用户无关的数以避免与其他测试冲突。
const (
	markReadTestConvID    = uint64(5001)
	markReadTestSender    = uint64(1)    // member 发送方
	markReadTestMember    = uint64(2)    // member 接收方，有未读
	markReadTestNonUser   = uint64(3)    // 用户 3 不属于 conv 5001
	markReadTestMsgID     = uint64(6001) // sender 发出的消息
	markReadTestPeerMsgID = uint64(6002) // member 发出的消息
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
	// 注：该断言不能独立捕获“缺失成员校验”这一原始 bug——ClearUnread 按调用者
	// user_id 匹配，用户 3 在 seed 中无 config 行，修复前该 UPDATE 也是 0 行，未读数
	// 本就保持 1。真正让本用例 RED 的是上方 err != nil 与下方 IsRead 断言；此断言仅
	// 防御未来回归把未读清到错误目标（如 peer）。
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

// TestMarkReadSenderDoesNotMarkOwnMessage 锁定 MarkMessagesRead 的
// "sender_id != readerId" 排除分支：发送者标记已读时不得把自己发出的消息翻为已读。
// 缺少该用例时，删除此过滤条件的回归会让既有两个测试仍全绿（假绿空间）。
func TestMarkReadSenderDoesNotMarkOwnMessage(t *testing.T) {
	setupMarkReadTestDB(t)

	// 再 seed 一条来自接收方(member 2)的消息，用于证明成员标记已读路径仍有效。
	peerMsgTime := time.Date(2026, 8, 11, 10, 0, 1, 0, time.UTC)
	if err := db.Connect().Create(&messages.Entity{
		Id: markReadTestPeerMsgID, ConvId: markReadTestConvID, SenderId: markReadTestMember,
		Content: "from peer", MsgType: 1, IsRead: 0, CreatedAt: peerMsgTime,
	}).Error; err != nil {
		t.Fatalf("create peer message: %v", err)
	}

	if err := MarkRead(markReadTestSender, markReadTestConvID); err != nil {
		t.Fatalf("MarkRead(sender) error = %v", err)
	}

	// 发送者自己发出的消息不得被标记为已读。
	var selfMsg messages.Entity
	if err := db.Connect().Where("id = ?", markReadTestMsgID).First(&selfMsg).Error; err != nil {
		t.Fatalf("load sender message: %v", err)
	}
	if selfMsg.IsRead != 0 {
		t.Fatalf("sender's own message IsRead = %d, want 0 (sender must not mark own message read)", selfMsg.IsRead)
	}

	// 成员路径仍应有效：对方(peer)发出的消息被标记为已读。
	var peerMsg messages.Entity
	if err := db.Connect().Where("id = ?", markReadTestPeerMsgID).First(&peerMsg).Error; err != nil {
		t.Fatalf("load peer message: %v", err)
	}
	if peerMsg.IsRead != 1 {
		t.Fatalf("peer message IsRead = %d, want 1 (member mark-read still works)", peerMsg.IsRead)
	}
}
