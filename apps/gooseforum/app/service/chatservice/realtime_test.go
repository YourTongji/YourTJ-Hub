package chatservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/realtimeservice"
)

func TestCommittedChatMutationsPublishToBothMembers(t *testing.T) {
	setupMarkReadTestDB(t)
	sender, err := realtimeservice.DefaultHub.Subscribe(markReadTestSender)
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	reader, err := realtimeservice.DefaultHub.Subscribe(markReadTestMember)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if _, err := SendMessage(markReadTestSender, markReadTestSender, "invalid", 1); err == nil {
		t.Fatal("self send should fail")
	}
	select {
	case event := <-sender.Events():
		t.Fatalf("failed write published %+v", event)
	default:
	}
	select {
	case event := <-reader.Events():
		t.Fatalf("failed write published %+v", event)
	default:
	}
	convID, err := SendMessage(markReadTestSender, markReadTestMember, "committed", 1)
	if err != nil {
		t.Fatal(err)
	}
	if convID != markReadTestConvID {
		t.Fatalf("convID=%d", convID)
	}
	for _, sub := range []*realtimeservice.Subscription{sender, reader} {
		event := <-sub.Events()
		if event.Type != realtimeservice.EventChatChanged || event.ConvID != convID {
			t.Fatalf("event=%+v", event)
		}
		<-sub.Events() // unread.changed
	}
	if err := MarkRead(markReadTestMember, convID); err != nil {
		t.Fatal(err)
	}
	for _, sub := range []*realtimeservice.Subscription{sender, reader} {
		event := <-sub.Events()
		if event.Type != realtimeservice.EventChatChanged || event.Change != "read" {
			t.Fatalf("read event=%+v", event)
		}
	}
}
