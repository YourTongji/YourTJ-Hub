package datamigration

import (
	"encoding/json"
	"fmt"
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/postservice"
	"gorm.io/gorm"
)

// topicPostStatsBackfillBatch 单批扫描的话题数，控制全量回填的内存占用。
const topicPostStatsBackfillBatch = 500

// TopicPostStatsBackfillResult 汇总存量派生统计回填结果（issue #554）。
type TopicPostStatsBackfillResult struct {
	// TopicsScanned 本次回填扫描的话题总数。
	TopicsScanned int
	// PostersRepaired 回填前后 topics.posters JSON 发生变化的话题数（观测指标）。
	PostersRepaired int
	Failed          int
	LastFailed      string
}

// BackfillTopicPostStats 从 active posts 绝对重建全部话题的
// topic_user_stat 与 topics.posters（issue #554）。
//
// 根因：legacy articles.posters 只存楼主，v5 迁移原样拷入 topics.posters；
// 增量路径（SyncTopicPostStats）只在新回复/删回复时修复，存量多回复话题
// 的首页参与人列长期只显示楼主。
//
// 策略：派生口径的单一所有者是 postservice.RebuildTopicPostStats——先清空
// topic_user_stat，再从 active posts（visibility=ACTIVE，匿名楼层排除，
// issue #524）重建计数、first/last 指针与 posters（楼主置前 + 非匿名回复者
// top-3）。绝对重建对任何存量形态（楼主-only/空/零计数残留）幂等。
func BackfillTopicPostStats() TopicPostStatsBackfillResult {
	return BackfillTopicPostStatsWithDB(db.Connect())
}

// BackfillTopicPostStatsWithDB 使用指定连接执行回填，便于测试注入。
// 逐话题独立、绝对写：单话题失败记录后继续，最终由调用方
// （app_migration v29）按 Failed>0 阻断启动、下次启动重跑。
func BackfillTopicPostStatsWithDB(conn *gorm.DB) TopicPostStatsBackfillResult {
	result := TopicPostStatsBackfillResult{}
	lastID := uint64(0)
	for {
		var batch []topics.Entity
		if err := conn.Unscoped().
			Where("id > ?", lastID).
			Order("id asc").
			Limit(topicPostStatsBackfillBatch).
			Find(&batch).Error; err != nil {
			result.Failed++
			result.LastFailed = "topics_scan:" + err.Error()
			slog.Error("backfill topic post stats scan failed", "afterId", lastID, "err", err)
			return result
		}
		if len(batch) == 0 {
			break
		}
		for _, topic := range batch {
			result.TopicsScanned++
			lastID = topic.Id
			repaired, err := backfillTopicPostStatsForOne(conn, topic)
			if err != nil {
				result.Failed++
				result.LastFailed = fmt.Sprintf("topic %d: %v", topic.Id, err)
				slog.Error("backfill topic post stats failed", "topicId", topic.Id, "err", err)
				continue
			}
			if repaired {
				result.PostersRepaired++
			}
		}
		if len(batch) < topicPostStatsBackfillBatch {
			break
		}
	}
	slog.Info("backfill topic post stats done",
		"topicsScanned", result.TopicsScanned,
		"postersRepaired", result.PostersRepaired,
		"failed", result.Failed,
		"lastFailed", result.LastFailed)
	return result
}

// backfillTopicPostStatsForOne 对单个话题执行绝对重建，返回 posters 是否
// 发生修复。回填不得扰动 topics.updated_at（首页默认排序键）：派生写路径
// ReplacePostStats 走 GORM Updates 会自动触碰 updated_at，这里以快照恢复。
func backfillTopicPostStatsForOne(conn *gorm.DB, topic topics.Entity) (bool, error) {
	// 与 contentdeleteservice 的重建载入一致：Unscoped + post_no/id 升序，
	// 由 RebuildTopicPostStats 按 visibility=ACTIVE 过滤。
	var activePosts []*posts.Entity
	if err := conn.Unscoped().
		Where("topic_id = ?", topic.Id).
		Order("post_no asc").
		Order("id asc").
		Find(&activePosts).Error; err != nil {
		return false, fmt.Errorf("load posts: %w", err)
	}

	postersBefore := postersJSON(topic.Posters)
	updatedAtBefore := topic.UpdatedAt

	if err := postservice.RebuildTopicPostStats(topic, activePosts); err != nil {
		return false, fmt.Errorf("rebuild stats: %w", err)
	}

	var after topics.Entity
	if err := conn.Unscoped().Where("id = ?", topic.Id).First(&after).Error; err != nil {
		return false, fmt.Errorf("reload topic: %w", err)
	}
	if !updatedAtBefore.IsZero() && !after.UpdatedAt.Equal(updatedAtBefore) {
		if err := conn.Model(&topics.Entity{}).Unscoped().
			Where("id = ?", topic.Id).
			UpdateColumn("updated_at", updatedAtBefore).Error; err != nil {
			return false, fmt.Errorf("restore updated_at: %w", err)
		}
	}
	return postersJSON(after.Posters) != postersBefore, nil
}

// postersJSON 序列化 posters 为可比对字符串；解析失败按 nil 处理
// （存量库该列由 GORM JSON serializer 写入，正常恒为合法 JSON）。
func postersJSON(posters []topics.Poster) string {
	if len(posters) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(posters)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}
