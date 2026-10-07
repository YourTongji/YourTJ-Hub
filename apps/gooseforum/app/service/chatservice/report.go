package chatservice

import (
	"errors"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/imUserChatConfigs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/reports"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrReportMessage = errors.New("message unavailable for reporting")

// ReportMessage discloses only the selected received message, never a conversation.
// Membership is checked against the database, including archived conversations.
// Locking the message serializes duplicate reports without locking unrelated chats.
// It returns the open report and whether this call created it (a duplicate open
// report from the same reporter is not an error and is not created again).
func ReportMessage(reporterID, messageID uint64, reason, note string) (reports.Entity, bool, error) {
	switch reason {
	case reports.ReasonSpam, reports.ReasonAbuse, reports.ReasonIllegal, reports.ReasonIrrelevant, reports.ReasonOther:
	default:
		return reports.Entity{}, false, ErrReportMessage
	}
	var report reports.Entity
	var created bool
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		var message messages.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&message, messageID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrReportMessage
			}
			return err
		}
		if reporterID == 0 || message.SenderId == reporterID {
			return ErrReportMessage
		}
		var count int64
		if err := tx.Model(&imUserChatConfigs.Entity{}).
			Where("conv_id = ? AND user_id = ? AND peer_id = ?", message.ConvId, reporterID, message.SenderId).
			Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			return ErrReportMessage
		}
		var err error
		report, created, err = reports.CreateOpenTx(tx, reports.Entity{
			TargetType: reports.TargetChatMessage, TargetId: messageID, ReporterId: reporterID,
			Reason: reason, Note: boundedReportText(note, 300),
			EvidenceSnapshot: reports.EvidenceSnapshotData{
				TargetType: reports.TargetChatMessage, TargetID: messageID,
				AuthorID: message.SenderId, Title: "Private message",
				Excerpt: boundedReportText(messages.DisplayContent(message.Content, message.MsgType), 4000), CreatedAt: time.Now(),
			},
		})
		return err
	})
	if err != nil {
		return reports.Entity{}, false, err
	}
	return report, created, nil
}

func boundedReportText(value string, limit int) string {
	text := []rune(strings.TrimSpace(value))
	if len(text) > limit {
		text = text[:limit]
	}
	return string(text)
}
