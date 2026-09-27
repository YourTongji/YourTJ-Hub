package routes

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

func TestChatSendDoesNotExposeStorageErrors(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	sender := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	const callback = "test:private_chat_storage_failure"
	if err := conn.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "messages" {
			_ = tx.AddError(errors.New("ERROR: value too long for type character varying(255) (SQLSTATE 22001) private-db-detail"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })
	body := fmt.Sprintf(`{"peerId":%d,"content":"private message","msgType":1}`, peer.Id)
	rec := serveJSON(router, "/api/forum/chat/send", body, contractSessionToken(t, sender))
	if strings.Contains(rec.Body.String(), `"code":0`) || !strings.Contains(rec.Body.String(), "chat.send.failed") {
		t.Fatalf("unexpected send response: %s", rec.Body)
	}
	for _, detail := range []string{"SQLSTATE", "varchar", "varying", "private-db-detail"} {
		if strings.Contains(rec.Body.String(), detail) {
			t.Fatalf("storage error exposed: %s", rec.Body)
		}
	}
}
