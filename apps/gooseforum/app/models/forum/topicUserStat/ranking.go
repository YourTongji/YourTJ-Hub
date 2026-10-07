package topicUserStat

import (
	"context"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

// RankParticipant is an internal, rebuildable business projection. Anonymous
// authors remain deduplicated here and never enter the public poster list.
type RankParticipant struct {
	TopicID           uint64 `gorm:"primaryKey;autoIncrement:false"`
	UserId            uint64 `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	RankReplyCount    uint32
	LastPublicReplyAt *time.Time
}

func (RankParticipant) TableName() string { return "topic_rank_participant" }
func RankParticipants(ctx context.Context, topicID uint64) ([]RankParticipant, error) {
	var rows []RankParticipant
	err := db.ConnectContext(ctx).Where("topic_id = ? AND rank_reply_count > 0", topicID).Where("user_id IN (?)", users.EligibleIDsQuery(ctx)).Limit(50001).Find(&rows).Error
	return rows, err
}

// BackfillRankProjectionTx also reconciles publication changes: unchanged actors
// cause no writes, avoiding a full participant rewrite for each new reply.
func BackfillRankProjectionTx(tx *gorm.DB, topicID uint64, rows []ReplierStat) error {

	var old []RankParticipant
	if err := tx.Where("topic_id = ?", topicID).Find(&old).Error; err != nil {
		return err
	}
	previous := map[uint64]RankParticipant{}
	for _, row := range old {
		previous[row.UserId] = row
	}
	changed := []RankParticipant{}
	for _, r := range rows {
		before, exists := previous[r.UserID]
		delete(previous, r.UserID)
		sameTime := before.LastPublicReplyAt == nil && r.LastPublicReplyAt == nil || before.LastPublicReplyAt != nil && r.LastPublicReplyAt != nil && before.LastPublicReplyAt.Equal(*r.LastPublicReplyAt)
		if exists && before.RankReplyCount == r.RankReplyCount && sameTime {
			continue
		}
		changed = append(changed, RankParticipant{TopicID: topicID, UserId: r.UserID, RankReplyCount: r.RankReplyCount, LastPublicReplyAt: r.LastPublicReplyAt})
	}
	removed := []uint64{}
	for uid := range previous {
		removed = append(removed, uid)
	}
	for len(removed) > 0 {
		n := min(500, len(removed))
		if err := tx.Where("topic_id = ? AND user_id IN ?", topicID, removed[:n]).Delete(&RankParticipant{}).Error; err != nil {
			return err
		}
		removed = removed[n:]
	}
	if len(changed) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "topic_id"}, {Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"rank_reply_count", "last_public_reply_at"})}).CreateInBatches(&changed, 500).Error
}
