package chatservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"gorm.io/gorm"
)

func seedIncoming(t *testing.T, id uint64, sender uint64) {
	t.Helper()
	if err := db.Connect().Create(&messages.Entity{
		Id: id, ConvId: markReadTestConvID, SenderId: sender,
		Content: "message", MsgType: 1, CreatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed message %d: %v", id, err)
	}
}

func TestMarkVisibleReadOnlyTouchesRequestedIncomingIDs(t *testing.T) {
	setupMarkReadTestDB(t)
	seedIncoming(t, 6003, markReadTestSender)
	seedIncoming(t, 6004, markReadTestSender)
	seedIncoming(t, 6005, markReadTestMember)
	if err := db.Connect().Model(&imUserChatConfigs.Entity{}).
		Where("user_id = ? AND conv_id = ?", markReadTestMember, markReadTestConvID).
		Update("unread_count", 3).Error; err != nil {
		t.Fatal(err)
	}

	result, err := MarkVisibleRead(markReadTestMember, markReadTestConvID,
		[]uint64{6004, markReadTestMsgID, 6004})
	if err != nil {
		t.Fatalf("MarkVisibleRead: %v", err)
	}
	if len(result.AcknowledgedMessageIds) != 2 ||
		result.AcknowledgedMessageIds[0] != 6004 ||
		result.AcknowledgedMessageIds[1] != markReadTestMsgID ||
		result.UnreadCount != 1 {
		t.Fatalf("acknowledgement = %+v, want IDs [6004 6001], unread 1", result)
	}
	var got []messages.Entity
	if err := db.Connect().Where("conv_id = ?", markReadTestConvID).Order("id").Find(&got).Error; err != nil {
		t.Fatal(err)
	}
	status := map[uint64]int{}
	for _, item := range got {
		status[item.Id] = item.IsRead
	}
	if status[6001] != 1 || status[6004] != 1 || status[6003] != 0 || status[6005] != 0 {
		t.Fatalf("read states = %v", status)
	}

	again, err := MarkVisibleRead(markReadTestMember, markReadTestConvID, []uint64{6004})
	if err != nil || again.UnreadCount != 1 {
		t.Fatalf("idempotent read = %+v, %v", again, err)
	}
}

func TestMarkVisibleReadRejectsEntireInvalidBatch(t *testing.T) {
	setupMarkReadTestDB(t)
	seedIncoming(t, 6005, markReadTestMember)
	for _, ids := range [][]uint64{
		{markReadTestMsgID, 99999},
		{markReadTestMsgID, 6005},
		{markReadTestMsgID, 0},
		{},
	} {
		if _, err := MarkVisibleRead(markReadTestMember, markReadTestConvID, ids); err == nil {
			t.Fatalf("MarkVisibleRead(%v) succeeded", ids)
		}
	}
	tooMany := make([]uint64, 101)
	for i := range tooMany {
		tooMany[i] = markReadTestMsgID
	}
	if _, err := MarkVisibleRead(markReadTestMember, markReadTestConvID, tooMany); err == nil {
		t.Fatal("over-limit batch succeeded")
	}
	if _, err := MarkVisibleRead(markReadTestNonUser, markReadTestConvID, []uint64{markReadTestMsgID}); err == nil {
		t.Fatal("non-member read succeeded")
	}
	var incoming messages.Entity
	if err := db.Connect().First(&incoming, markReadTestMsgID).Error; err != nil {
		t.Fatal(err)
	}
	if incoming.IsRead != 0 || imUserChatConfigs.GetConfig(markReadTestMember, markReadTestSender).UnreadCount != 1 {
		t.Fatal("invalid batch changed read state")
	}
}

func TestSendMessageRollsBackWhenMessageInsertFails(t *testing.T) {
	setupMarkReadTestDB(t)
	const sender, peer = uint64(8101), uint64(8102)
	callback := "test:reject_chat_message_create"
	if err := db.Connect().Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "messages" {
			_ = tx.AddError(errors.New("injected message insert failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Connect().Callback().Create().Remove(callback); err != nil {
			t.Errorf("remove chat failure callback: %v", err)
		}
	})

	if _, err := SendMessage(sender, peer, "should roll back", 1); err == nil {
		t.Fatal("SendMessage succeeded despite message insert failure")
	}
	if config := imUserChatConfigs.GetConfig(sender, peer); config != nil {
		t.Fatalf("orphan sender config: %+v", config)
	}
	if config := imUserChatConfigs.GetConfig(peer, sender); config != nil {
		t.Fatalf("orphan recipient config: %+v", config)
	}
}

func TestChatOperationsDoNotRecountUnreadBacklog(t *testing.T) {
	for _, operation := range []string{"send", "visible-read", "read-states"} {
		t.Run(operation, func(t *testing.T) {
			setupMarkReadTestDB(t)
			conn := db.Connect()
			backlog := make([]messages.Entity, 256)
			for i := range backlog {
				backlog[i] = messages.Entity{Id: uint64(7000 + i), ConvId: markReadTestConvID, SenderId: markReadTestSender, Content: "unread backlog", MsgType: 1, CreatedAt: time.Now()}
			}
			if err := conn.Create(&backlog).Error; err != nil {
				t.Fatal(err)
			}
			if err := conn.Model(&imUserChatConfigs.Entity{}).Where("user_id = ? AND conv_id = ?", markReadTestMember, markReadTestConvID).Update("unread_count", len(backlog)+1).Error; err != nil {
				t.Fatal(err)
			}
			counts := 0
			const callback = "test:observe_unread_recounts"
			if err := conn.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
				if tx.Statement.Table == "messages" && strings.Contains(strings.ToLower(tx.Statement.SQL.String()), "count(") {
					counts++
				}
			}); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = conn.Callback().Query().Remove(callback) })
			want := uint(len(backlog) + 1)
			switch operation {
			case "send":
				if _, err := SendMessage(markReadTestSender, markReadTestMember, "new", 1); err != nil {
					t.Fatal(err)
				}
				want++
			case "visible-read":
				for range 2 {
					result, err := MarkVisibleRead(markReadTestMember, markReadTestConvID, []uint64{markReadTestMsgID, markReadTestMsgID})
					if err != nil || result.UnreadCount != want-1 {
						t.Fatalf("visible read: %+v %v", result, err)
					}
				}
				want--
			case "read-states":
				result, err := GetMessageReadStates(markReadTestMember, markReadTestConvID, []uint64{markReadTestMsgID})
				if err != nil || result.UnreadCount != want {
					t.Fatalf("read states: %+v %v", result, err)
				}
			}
			if got := imUserChatConfigs.GetConfig(markReadTestMember, markReadTestSender).UnreadCount; got != want {
				t.Fatalf("unread counter = %d, want %d", got, want)
			}
			if counts != 0 {
				t.Fatalf("%s scanned the unread backlog %d times", operation, counts)
			}
		})
	}
}
