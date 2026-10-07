package posts

import (
	"context"
	"fmt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/feedconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/feed"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"strings"
	"time"

	"gorm.io/gorm"
)

type PublicReplyFact struct {
	ID            uint64
	UserID        uint64
	FirstPublicAt time.Time
}

func DailyReplyFacts(ctx context.Context, topicID uint64, after time.Time) ([]PublicReplyFact, error) {
	var rows []PublicReplyFact
	inner := builder().WithContext(ctx).Select("id,user_id,first_public_at,ROW_NUMBER() OVER (PARTITION BY user_id ORDER BY first_public_at,id) AS reply_rank").Where("topic_id = ? AND post_no > 1 AND process_status = 0 AND visibility_status = ? AND first_public_at >= ? AND deleted_at IS NULL", topicID, VisibilityActive, after).Where("user_id IN (?)", users.EligibleIDsQuery(ctx))
	err := builder().WithContext(ctx).Table("(?) ranked", inner).Select("id,user_id,first_public_at").Where("reply_rank <= 3").Order("first_public_at,id").Limit(50001).Scan(&rows).Error
	return rows, err
}

type RankReplier struct {
	UserID            uint64
	Replies           uint32
	LastPublicReplyAt *time.Time
}

type aggregateTime struct{ time.Time }

func (aggregateTime) GormDataType() string { return "time" }
func (t *aggregateTime) Scan(value any) error {
	if value == nil {
		return nil
	}
	if v, ok := value.(time.Time); ok {
		t.Time = v
		return nil
	}
	text, ok := value.(string)
	if !ok {
		if bytes, yes := value.([]byte); yes {
			text = string(bytes)
		} else {
			return fmt.Errorf("unexpected aggregate timestamp %T", value)
		}
	}
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05.999999999 -0700 MST"} {
		if at, e := time.Parse(layout, text); e == nil {
			t.Time = at
			return nil
		}
	}
	return fmt.Errorf("invalid aggregate timestamp")
}
func RankRepliers(ctx context.Context, topicID uint64) ([]RankReplier, error) {
	var facts []struct {
		UserID            uint64
		Replies           uint32
		LastPublicReplyAt aggregateTime
	}
	err := builder().WithContext(ctx).Select("user_id,COUNT(*) AS replies,MAX(COALESCE(first_public_at,created_at)) AS last_public_reply_at").Where("topic_id = ? AND post_no > 1 AND process_status = 0 AND visibility_status = ? AND deleted_at IS NULL", topicID, VisibilityActive).Where("user_id IN (?)", users.EligibleIDsQuery(ctx)).Group("user_id").Limit(50001).Scan(&facts).Error
	if err != nil {
		return nil, err
	}
	rows := make([]RankReplier, 0, len(facts))
	for _, r := range facts {
		at := r.LastPublicReplyAt.Time
		rows = append(rows, RankReplier{UserID: r.UserID, Replies: r.Replies, LastPublicReplyAt: &at})
	}
	return rows, nil
}

func BackfillFirstPublicBatch(ctx context.Context, after uint64) (uint64, bool, error) {
	var rows []Entity
	err := builder().WithContext(ctx).Select("id").Where("id > ?", after).Order("id").Limit(200).Find(&rows).Error
	if err != nil || len(rows) == 0 {
		return after, true, err
	}
	ids := make([]uint64, len(rows))
	for i, r := range rows {
		ids[i] = r.Id
	}
	err = builder().WithContext(ctx).Where("id IN ? AND first_public_at IS NULL AND process_status = 0 AND visibility_status = ?", ids, VisibilityActive).UpdateColumns(map[string]any{"first_public_at": gorm.Expr("created_at"), "first_public_estimated": true}).Error
	return rows[len(rows)-1].Id, false, err
}

// ProfileReplies is bounded before merging with action streams.
func ProfileReplies(ctx context.Context, uid uint64, after time.Time) ([]PublicReplyFact, error) {
	var rows []PublicReplyFact
	err := builder().WithContext(ctx).Select("topic_id AS id,user_id,first_public_at").Where("user_id = ? AND post_no > 1 AND first_public_at >= ? AND process_status = 0 AND visibility_status = ? AND deleted_at IS NULL", uid, after, VisibilityActive).Order("first_public_at DESC").Limit(50).Scan(&rows).Error
	return rows, err
}

// ActorTopicIDs returns a bounded batch for identity-driven rank invalidation.
func ActorTopicIDs(ctx context.Context, uid, after uint64) ([]uint64, error) {
	var ids []uint64
	err := builder().Unscoped().WithContext(ctx).Select("DISTINCT topic_id").Where("user_id = ? AND topic_id > ?", uid, after).Order("topic_id").Limit(200).Find(&ids).Error
	return ids, err
}

// FirstPeerReplyTx uses actual public time; authors and estimated legacy replies
// cannot supply the first-response metric for a new publication cohort.
func FirstPeerReplyTx(tx *gorm.DB, topicID, author uint64) (*time.Time, error) {
	var row Entity
	err := tx.Model(&Entity{}).Select("first_public_at").Where("topic_id = ? AND post_no > 1 AND user_id <> ? AND process_status = 0 AND visibility_status = ? AND first_public_at IS NOT NULL AND first_public_estimated = ?", topicID, author, VisibilityActive, false).Where("user_id IN (?)", users.EligibleIDsQuery(tx.Statement.Context)).Order("first_public_at,id").Limit(1).Find(&row).Error
	return row.FirstPublicAt, err
}

// The post owner supplies topic IDs. The feed owner only writes its schedule;
// rank recomputation does not depend on a later, best-effort projection refresh.
func markPostProjectionTx(tx *gorm.DB, id uint64) error {
	if !feedconfig.Current().Ranking {
		return nil
	}
	var row Entity
	if err := tx.Session(&gorm.Session{NewDB: true}).Unscoped().Model(&Entity{}).Select("topic_id").Where("id = ?", id).Find(&row).Error; err != nil {
		return err
	}
	return feed.MarkProjectionTx(tx, row.TopicId)
}

// NewPublicRepliesAmong checks at most 300 topic/cutoff pairs in one statement.
// EXISTS uses the public-time index and never hydrates reply bodies. The caller's
// deadline also bounds pathological long chains of ineligible replies.
func NewPublicRepliesAmong(ctx context.Context, viewer uint64, cutoffs map[uint64]time.Time) (map[uint64]bool, error) {
	return newPublicRepliesAmong(builder().WithContext(ctx), viewer, cutoffs)
}
func newPublicRepliesAmong(conn *gorm.DB, viewer uint64, cutoffs map[uint64]time.Time) (map[uint64]bool, error) {
	result := map[uint64]bool{}
	if len(cutoffs) == 0 {
		return result, nil
	}
	if len(cutoffs) > 300 {
		return nil, fmt.Errorf("reply cutoff batch exceeds bound")
	}
	parts := make([]string, 0, len(cutoffs))
	args := make([]any, 0, 2*len(cutoffs))
	for id, at := range cutoffs {
		if conn.Name() == "postgres" {
			parts = append(parts, "SELECT CAST(? AS BIGINT) AS topic_id, CAST(? AS TIMESTAMPTZ) AS content_at")
		} else {
			parts = append(parts, "SELECT ? AS topic_id, ? AS content_at")
		}
		args = append(args, id, at.UTC())
	}
	comparison := "posts.first_public_at > marks.content_at"
	if conn.Name() == "sqlite" {
		comparison = "julianday(posts.first_public_at) > julianday(marks.content_at)"
	}
	eligibleAuthor := users.FeedAuthorsQuery(conn, viewer).Where("users.id = posts.user_id").Limit(1)
	authorExists := "EXISTS (?)"
	exists := "EXISTS (?)"
	if conn.Name() == "postgres" {
		// Optimization fences keep both probes correlated: topic/time index first,
		// then author PK and block keys. Neither public replies nor all users should
		// be scanned/hash-joined globally before the finite cutoff batch.
		authorExists = "EXISTS (? OFFSET 0)"
		exists = "EXISTS (? OFFSET 0)"
	}
	reply := conn.Session(&gorm.Session{NewDB: true}).Table("posts").Select("1").Where("posts.topic_id = marks.topic_id AND post_no > 1 AND user_id <> ? AND process_status = 0 AND visibility_status = ? AND deleted_at IS NULL AND "+comparison, viewer, VisibilityActive).Where(authorExists, eligibleAuthor).Limit(1)

	var ids []uint64
	err := conn.Table("(?) marks", conn.Raw(strings.Join(parts, " UNION ALL "), args...)).Select("marks.topic_id").Where(exists, reply).Scan(&ids).Error
	for _, id := range ids {
		result[id] = true
	}
	return result, err
}
