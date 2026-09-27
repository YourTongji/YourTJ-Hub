package routes

import (
	"fmt"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
)

func TestChatSendRetryUsesClientMessageID(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	sender := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, sender)
	body := fmt.Sprintf(`{"peerId":%d,"content":"retry safely","msgType":1,"clientMessageId":"client-1"}`, peer.Id)
	for range 2 {
		rec := serveJSON(router, "/api/forum/chat/send", body, token)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"code":0`) {
			t.Fatalf("send: %d %s", rec.Code, rec.Body)
		}
	}
	var count int64
	conn.Model(&messages.Entity{}).Where("sender_id = ?", sender.Id).Count(&count)
	if count != 1 {
		t.Fatalf("retry created %d messages", count)
	}
	rec := serveJSON(router, "/api/forum/chat/send", strings.Replace(body, "retry safely", "changed", 1), token)
	if strings.Contains(rec.Body.String(), `"code":0`) {
		t.Fatal("changed body reused key")
	}
}
