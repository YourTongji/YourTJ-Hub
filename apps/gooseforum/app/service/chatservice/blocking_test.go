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
