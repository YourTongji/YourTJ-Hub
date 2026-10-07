package chatservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
)

func TestMessageReportMembershipSnapshotAndRetry(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&reports.Entity{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Where("target_type = ?", reports.TargetChatMessage).Delete(&reports.Entity{}) })
	for _, reporter := range []uint64{0, markReadTestSender, markReadTestNonUser} {
		if _, _, err := ReportMessage(reporter, markReadTestMsgID, "abuse", "note"); err == nil {
			t.Fatalf("reporter %d could disclose a message", reporter)
		}
	}
	if _, _, err := ReportMessage(markReadTestMember, markReadTestMsgID, "中文说明", ""); err == nil {
		t.Fatal("invalid reason accepted")
	}
	for attempt := range 2 {
		report, created, err := ReportMessage(markReadTestMember, markReadTestMsgID, "abuse", strings.Repeat("字", 400))
		if err != nil {
			t.Fatal(err)
		}
		// 仅首次创建返回 created=true（举报通知只为新举报发送，issue #1049）。
		if created != (attempt == 0) || report.Id == 0 {
			t.Fatalf("attempt %d: created=%v report=%d", attempt, created, report.Id)
		}
	}
	var rows []reports.Entity
	if err := conn.Where("target_type = ?", reports.TargetChatMessage).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("retry created %d reports", len(rows))
	}
	got := rows[0]
	if got.TopicId != 0 || got.EvidenceSnapshot.Excerpt != "hello" || got.EvidenceSnapshot.AuthorID != markReadTestSender || len([]rune(got.Note)) != 300 {
		t.Fatalf("wrong evidence: %+v", got)
	}
	if len(reports.CursorPage(reports.CursorPageQuery{TargetType: reports.TargetChatMessage, ExcludePrivateMessages: true})) != 0 {
		t.Fatal("private evidence leaked into moderator query")
	}
}

func TestForwardReportUsesReadableSnapshotAndForwarder(t *testing.T) {
	setupMarkReadTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&reports.Entity{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Where("target_type = ?", reports.TargetChatMessage).Delete(&reports.Entity{}) })
	raw, _ := (&messages.ForwardedBundle{Version: 1, Messages: []messages.ForwardedEntry{{SenderName: "Original", Content: "Selected body", MsgType: 1}}}).Encode()
	conn.Model(&messages.Entity{}).Where("id = ?", markReadTestMsgID).Updates(map[string]any{"content": raw, "msg_type": messages.ForwardType})
	if _, _, err := ReportMessage(markReadTestMember, markReadTestMsgID, "abuse", ""); err != nil {
		t.Fatal(err)
	}
	var got reports.Entity
	conn.Where("target_type = ? AND target_id = ?", reports.TargetChatMessage, markReadTestMsgID).First(&got)
	if got.EvidenceSnapshot.Excerpt != "[Chat history]\nOriginal: Selected body" || got.EvidenceSnapshot.AuthorID != markReadTestSender {
		t.Fatalf("wrong forward evidence: %+v", got.EvidenceSnapshot)
	}
}
