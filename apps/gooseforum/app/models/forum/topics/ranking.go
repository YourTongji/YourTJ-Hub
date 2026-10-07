package topics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userFollow"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const rankColumns = "id,user_id,persona_uid,category_id,first_public_at,first_public_estimated,rank_score,daily_score,rank_ready,rank_params_hash,rank_scored_at,last_public_reply_at,reply_count,like_count,updated_at,rank_source,rank_components"

// Candidate recall reads IDs only. Validate first-post visibility after the
// candidate LIMIT in RankTopics/PublicTopics, so planner selectivity estimates
// cannot turn a small recall into thousands of correlated post lookups.
func rankingCandidateQuery(ctx context.Context) *gorm.DB {
	return builder().WithContext(ctx).Where("status = 1 AND process_status = 0 AND visibility_status = ? AND deleted_at IS NULL AND topic_type = ?", VisibilityActive, TopicTypeForum)
}
func publicRankingQuery(ctx context.Context) *gorm.DB {
	return rankingCandidateQuery(ctx).Where(firstPostVisibleSQL, ProcessStatusNormal).Where("user_id IN (?)", users.EligibleIDsQuery(ctx))
}

// RankTopic reads no content, excerpts, poster JSON or image lists.
func RankTopic(ctx context.Context, id uint64) (Entity, error) {
	var row Entity
	err := publicRankingQuery(ctx).Select(rankColumns).First(&row, "id = ?", id).Error
	return row, err
}
func RankTopics(ctx context.Context, ids []uint64) ([]Entity, error) {
	if len(ids) == 0 {
		return []Entity{}, nil
	}
	var rows []Entity
	err := publicRankingQuery(ctx).Select(rankColumns).Where("id IN ?", ids).Limit(300).Find(&rows).Error
	return rows, err
}
func PublicTopics(ctx context.Context, ids []uint64) ([]Entity, error) {
	if len(ids) == 0 {
		return []Entity{}, nil
	}
	var rows []Entity
	err := publicRankingQuery(ctx).Where("id IN ?", ids).Limit(120).Find(&rows).Error
	return rows, err
}

type Recall struct {
	Viewer   uint64
	Source   string
	Authors  []uint64
	Category uint64
	After    time.Time
	Hash     string
	Limit    int
}

func RecallRankIDs(ctx context.Context, q Recall) ([]uint64, error) {
	limit := min(max(q.Limit, 1), 200)
	b := rankingCandidateQuery(ctx).Select("topics.id")
	if !q.After.IsZero() {
		b = b.Where("first_public_at >= ?", q.After)
	}
	switch q.Source {
	case "following":
		b = b.Where("persona_uid = '' AND user_id IN (?)", userFollow.ActiveFollowedIDsQuery(ctx, q.Viewer))
	case "hot":
		b = b.Where("rank_ready = ? AND rank_params_hash = ? AND rank_score > 0 AND (rank_due_at IS NULL OR rank_due_at >= ?)", true, q.Hash, time.Now().Add(-10*time.Minute)).Order("rank_score DESC")
	case "daily":
		b = b.Where("rank_ready = ? AND rank_params_hash = ? AND daily_score > 0 AND (rank_due_at IS NULL OR rank_due_at >= ?)", true, q.Hash, time.Now().Add(-10*time.Minute)).Order("daily_score DESC")
	case "author":
		if len(q.Authors) == 0 {
			return []uint64{}, nil
		}
		b = b.Where("persona_uid = '' AND user_id IN ?", q.Authors)
	case "category":
		// Materialize the small recent pool before quality sorting. A nested
		// IN+ORDER BY could choose a global rank-index scan on PostgreSQL.
		var recent []struct {
			ID        uint64
			RankScore int64
		}
		err := rankingCandidateQuery(ctx).Select("topics.id,rank_score").Where("EXISTS (SELECT 1 FROM topic_category_index idx WHERE idx.topic_id = topics.id AND idx.category_id = ? AND idx.effective = 1)", q.Category).Order("first_public_at DESC,id DESC").Limit(200).Find(&recent).Error
		if err != nil {
			return nil, err
		}
		sort.Slice(recent, func(i, j int) bool {
			if recent[i].RankScore == recent[j].RankScore {
				return recent[i].ID > recent[j].ID
			}
			return recent[i].RankScore > recent[j].RankScore
		})
		ids := make([]uint64, min(limit, len(recent)))
		for i := range ids {
			ids[i] = recent[i].ID
		}
		return ids, nil
	case "explore":
		b = b.Where("rank_engaged < 3")
	}
	var ids []uint64
	if q.Source != "hot" && q.Source != "daily" {
		b = b.Order("first_public_at DESC")
	}
	err := b.Order("id DESC").Limit(limit).Find(&ids).Error
	return ids, err
}

func RankBackfillBatch(ctx context.Context, after uint64, limit int) ([]uint64, error) {
	var ids []uint64
	err := publicRankingQuery(ctx).Select("id").Where("id > ?", after).Order("id").Limit(min(limit, 200)).Find(&ids).Error
	return ids, err
}

func BackfillFirstPublic(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return builder().WithContext(ctx).Where("id IN ? AND first_public_at IS NULL", ids).UpdateColumns(map[string]any{"first_public_at": gorm.Expr("created_at"), "first_public_estimated": true}).Error
}

// WriteRankTx uses columns directly: GORM must not refresh updated_at.
func WriteRankTx(tx *gorm.DB, id uint64, hot, daily int64, hash string, now time.Time) error {
	return writeRankTx(tx, id, hot, daily, hash, now, true)
}

// Hidden topics retain the need to rebuild participants when restored. A reply
// invalidation consumed while the parent is hidden must never validate old data.
func ClearRankTx(tx *gorm.DB, id uint64, hash string, now time.Time) error {
	return writeRankTx(tx, id, 0, 0, hash, now, false)
}
func writeRankTx(tx *gorm.DB, id uint64, hot, daily int64, hash string, now time.Time, ready bool) error {
	return tx.Model(&Entity{}).Where("id = ?", id).UpdateColumns(map[string]any{"rank_score": hot, "daily_score": daily, "rank_ready": ready, "rank_params_hash": hash, "rank_scored_at": now}).Error
}

func AllRankReady(ctx context.Context, hash string) (bool, error) {
	var ids []uint64
	err := publicRankingQuery(ctx).Select("id").Where("rank_ready = ? OR rank_params_hash <> ?", false, hash).Limit(1).Find(&ids).Error
	return len(ids) == 0, err
}

func RankSource(e Entity) string {
	data := fmt.Sprintf("%d:%d:%d:%s:%s:%s:%v", e.UserId, e.ReplyCount, e.LikeCount, e.UpdatedAt.UTC().Format(time.RFC3339Nano), rankSourceTime(e.LastPublicReplyAt), rankSourceTime(e.FirstPublicAt), e.CategoryIds)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:16])
}

// Aggregate scans and normal model reads can use different location names for
// the same instant. A watermark must survive that database round trip.
func rankSourceTime(at *time.Time) string {
	if at == nil {
		return ""
	}
	return at.UTC().Format(time.RFC3339Nano)
}
func ReconcileRankBatch(ctx context.Context, after uint64) ([]Entity, error) {
	var rows []Entity
	err := publicRankingQuery(ctx).Select(rankColumns).Where("id > ?", after).Order("id").Limit(200).Find(&rows).Error
	return rows, err
}

// ActorTopicIDs returns a bounded batch for identity-driven rank invalidation.
func ActorTopicIDs(ctx context.Context, uid, after uint64) ([]uint64, error) {
	var ids []uint64
	err := builder().Unscoped().WithContext(ctx).Select("id").Where("user_id = ? AND id > ?", uid, after).Order("id").Limit(200).Find(&ids).Error
	return ids, err
}

// NewTopicFactsTx returns only authoritative public metadata for at most one page.
func NewTopicFactsTx(tx *gorm.DB, ids []uint64) ([]Entity, error) {
	var rows []Entity
	if len(ids) == 0 {
		return rows, nil
	}
	err := tx.Model(&Entity{}).Select("id,user_id,first_public_at,first_public_estimated").Where("id IN ? AND status = 1 AND process_status = 0 AND visibility_status = ? AND topic_type = ?", ids[:min(len(ids), 20)], VisibilityActive, TopicTypeForum).Where(firstPostVisibleSQL, ProcessStatusNormal).Where("user_id IN (?)", users.EligibleIDsQuery(tx.Statement.Context)).Find(&rows).Error
	return rows, err
}

// ResetRankReadinessTx keeps a controlled rebuild in warming until each queued
// batch has actually been recomputed; old same-hash scores cannot promote it.
func ResetRankReadinessTx(tx *gorm.DB, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Model(&Entity{}).Where("id IN ?", ids).UpdateColumn("rank_ready", false).Error
}
