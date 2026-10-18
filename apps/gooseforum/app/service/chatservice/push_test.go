package chatservice

import (
	"errors"
	"fmt"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

// 一个无法建立 savepoint 的事务（此处用已回滚的事务代表）必须跳过该通道，
// 既不能让发送失败，也不能在没有回滚点的前提下继续写入。
func TestChatPushSavepointUnavailableSkipsChannel(t *testing.T) {
	conn := db.Connect()
	tx := conn.Begin()
	if tx.Error != nil {
		t.Fatalf("begin: %v", tx.Error)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	enqueued := false
	err := enqueueChatPushBestEffort(tx, "chat_push_test", "web", 1, 2, func(*gorm.DB, uint64, uint64) error {
		enqueued = true
		return nil
	})
	if err != nil {
		t.Fatalf("unavailable savepoint must not fail the send: %v", err)
	}
	if enqueued {
		t.Fatal("enqueue must be skipped when the savepoint cannot be established")
	}
}

func TestChatPushCommittedReplay(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Where("type LIKE ?", "%push.%").Delete(&taskQueue.Entity{}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []uint64{markReadTestSender, markReadTestMember} {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
		if err := conn.Create(&users.EntityComplete{Id: id, Username: "push-test-" + string(rune(id)), Email: ""}).Error; err != nil {
			t.Fatal(err)
		}
	}
	preferences.Set("push.jpush.app_key", "test")
	preferences.Set("push.jpush.master_secret", "test")
	t.Cleanup(func() { preferences.Set("push.jpush.app_key", ""); preferences.Set("push.jpush.master_secret", "") })
	if _, err := SendMessage(markReadTestSender, markReadTestMember, "private text", 1, "push-replay"); err != nil {
		t.Fatal(err)
	}
	if _, err := SendMessage(markReadTestSender, markReadTestMember, "private text", 1, "push-replay"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "nativepush.%").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("want one committed chat push after replay, got %d", count)
	}
}

func TestForwardPushCommitReplayRollback(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&taskQueue.Entity{}); err != nil {
		t.Fatal(err)
	}
	conn.Where("type LIKE ?", "nativepush.%").Delete(&taskQueue.Entity{})
	for _, id := range []uint64{1, 2, 3} {
		conn.Unscoped().Where("id = ?", id).Delete(&users.EntityComplete{})
		if err := conn.Create(&users.EntityComplete{Id: id, Username: fmt.Sprintf("forward-push-%d", id)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	preferences.Set("push.jpush.app_key", "test")
	preferences.Set("push.jpush.master_secret", "test")
	t.Cleanup(func() { preferences.Set("push.jpush.app_key", ""); preferences.Set("push.jpush.master_secret", "") })
	if err := conn.Create(&messages.Entity{Id: markReadTestPeerMsgID, ConvId: markReadTestConvID, SenderId: 2, Content: "second source", MsgType: 1}).Error; err != nil {
		t.Fatal(err)
	}
	request := ForwardRequest{ConvID: markReadTestConvID, PeerID: 3, MessageIDs: []uint64{markReadTestMsgID, markReadTestPeerMsgID}, Mode: "individual", ClientForwardID: "push-forward-replay"}
	if _, err := ForwardMessages(1, request); err != nil {
		t.Fatal(err)
	}
	if _, err := ForwardMessages(1, request); err != nil {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "nativepush.%").Count(&count)
	if count != 2 {
		t.Fatalf("individual+replay queued %d want 2", count)
	}
	request.Mode = "merged"
	request.ClientForwardID = "push-forward-merged"
	if _, err := ForwardMessages(1, request); err != nil {
		t.Fatal(err)
	}
	conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "nativepush.%").Count(&count)
	if count != 3 {
		t.Fatalf("merged queued %d want 3", count)
	}
	creates := 0
	if err := conn.Callback().Create().Before("gorm:create").Register("test_forward_push_rollback", func(tx *gorm.DB) {
		if tx.Statement.Table == "messages" {
			creates++
			if creates == 2 {
				_ = tx.AddError(errors.New("forced second forward failure"))
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove("test_forward_push_rollback") })
	request.Mode = "individual"
	request.ClientForwardID = "push-forward-rollback"
	if _, err := ForwardMessages(1, request); err == nil {
		t.Fatal("expected failing outer forward transaction")
	}
	conn.Model(&taskQueue.Entity{}).Where("type LIKE ?", "nativepush.%").Count(&count)
	if count != 3 {
		t.Fatalf("rollback leaked tasks: %d", count)
	}
}
