package chatservice

import (
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// A blocked pair cannot create a message in either direction.
func TestSendMessageRejectsBlockedPair(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.BlockEntity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.Exec("INSERT INTO user_blocks (owner_id,target_user_id) VALUES (?,?)", markReadTestMember, markReadTestSender).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Exec("DELETE FROM user_blocks") })
	var before int64
	conn.Model(&messages.Entity{}).Count(&before)
	for _, pair := range [][2]uint64{{markReadTestSender, markReadTestMember}, {markReadTestMember, markReadTestSender}} {
		if _, err := SendMessage(pair[0], pair[1], "blocked contact", 1); err == nil {
			t.Fatalf("blocked pair %v could send", pair)
		}
	}
	var after int64
	conn.Model(&messages.Entity{}).Count(&after)
	if before != after {
		t.Fatal("blocked send changed messages")
	}
}

func TestCommittedRetryStillSucceedsAfterRecipientBlocksSender(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	const key = "committed-before-block"
	original, err := SendMessage(markReadTestSender, markReadTestMember, "delivered once", 1, key)
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&users.BlockEntity{OwnerID: markReadTestMember, TargetUserID: markReadTestSender}).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		conn.Where("owner_id = ? AND target_user_id = ?", markReadTestMember, markReadTestSender).Delete(&users.BlockEntity{})
	})
	replayed, err := SendMessage(markReadTestSender, markReadTestMember, "delivered once", 1, key)
	if err != nil || replayed != original {
		t.Fatalf("committed retry = %d, %v; want %d", replayed, err, original)
	}
	if _, err := SendMessage(markReadTestSender, markReadTestMember, "another message", 1, "new-key"); err == nil {
		t.Fatal("block allowed new write")
	}
	if _, err := SendMessage(markReadTestSender, markReadTestMember, "changed content", 1, key); err == nil {
		t.Fatal("block allowed conflicting retry")
	}
	var count int64
	if err := conn.Model(&messages.Entity{}).Where("sender_id = ? AND client_message_id = ?", markReadTestSender, key).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("retry duplicated message: %d", count)
	}
}
