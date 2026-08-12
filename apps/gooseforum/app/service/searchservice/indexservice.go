package searchservice

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/meilisearch/meilisearch-go"
	"github.com/samber/lo"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

// IndexBuildResult summarizes a Meilisearch rebuild.
type IndexBuildResult struct {
	ProcessedCount int    `json:"processedCount"`
	FailedCount    int    `json:"failedCount"`
	TotalBatches   int    `json:"totalBatches"`
	IndexName      string `json:"indexName"`
	GhostRemoved   int    `json:"ghostRemoved"`
}

// convertTopicToSearchDocument maps a topic and its first post to a search document.
func convertTopicToSearchDocument(topic *topics.Entity, firstPost *posts.Entity) TopicSearchDocument {
	searchContent := ""
	if firstPost != nil {
		searchContent = markdown2html.ExtractSearchContent(firstPost.Content)
	}
	return TopicSearchDocument{
		ID:            topic.Id,
		Title:         topic.Title,
		SearchContent: searchContent,
		Category:      topic.CategoryIds,
		TopicStatus:   topic.Status,
		ProcessStatus: topic.ProcessStatus,
		CreatedAt:     topic.CreatedAt.Unix(),
		UpdatedAt:     topic.UpdatedAt.Unix(),
	}
}

// isTopicPubliclySearchable 判断话题当前是否应出现在公共搜索：
// 已发布、未封禁/待审、未软删、可见性正常。
// 供索引构建（isIndexable）与聚合搜索防御过滤（issue #132）复用，
// 保证"索引事件未落地"的窗口期也不泄露非公开话题。
func isTopicPubliclySearchable(topic *topics.Entity) bool {
	if topic == nil {
		return false
	}
	return topic.Status == 1 &&
		topic.ProcessStatus == topics.ProcessStatusNormal &&
		!topic.DeletedAt.Valid &&
		topic.VisibilityStatus == topics.VisibilityActive
}

func BuildSingleTopicSearchDocument(topic *topics.Entity, firstPost *posts.Entity) (*meilisearch.TaskInfo, error) {
	if !meiliconnect.IsAvailable() {
		return nil, nil
	}
	if topic == nil {
		return nil, nil
	}

	client := meiliconnect.GetClient()
	index := client.Index(TopicIndex)
	var task *meilisearch.TaskInfo
	var err error
	pk := "id"
	// 仅"已发布、未封禁、未软删、可见性正常"的主题进入索引；其余一律删除文档。
	// 用户删除（visibility_status=USER_DELETED）与封禁（process_status=1）都走删除分支。
	isIndexable := topic.Status == 1 &&
		topic.ProcessStatus == 0 &&
		!topic.DeletedAt.Valid &&
		topic.VisibilityStatus == topics.VisibilityActive &&
		firstPost != nil &&
		firstPost.Id > 0 &&
		!firstPost.DeletedAt.Valid &&
		firstPost.ProcessStatus == posts.ProcessStatusNormal &&
		firstPost.VisibilityStatus == posts.VisibilityActive
	if isIndexable {
		doc := convertTopicToSearchDocument(topic, firstPost)
		task, err = index.AddDocuments(doc, &meilisearch.DocumentOptions{PrimaryKey: &pk})
		if err != nil {
			slog.Warn(fmt.Sprintf("Meilisearch 处理主题 ID:%v 失败: %v\n", doc.ID, err))
			return nil, fmt.Errorf("add search document: %w", err)
		}
		slog.Info(fmt.Sprintf("处理主题 ID:%v, TaskUID: %v\n", doc.ID, getTaskUID(task)))
	} else {
		// DeleteDocument 删除单个文档；index.Delete(uid) 会误删整个索引
		task, err = index.DeleteDocument(cast.ToString(topic.Id), nil)
		if err != nil {
			slog.Warn(fmt.Sprintf("Meilisearch 删除文档失败: %v, Error: %v\n", topic.Id, err))
			return nil, fmt.Errorf("delete search document: %w", err)
		}
		slog.Info(fmt.Sprintf("删除主题 ID:%v, TaskUID: %v\n", topic.Id, getTaskUID(task)))
	}
	return task, nil
}

// BuildMeilisearchIndex rebuilds the Meilisearch topic index.
func BuildMeilisearchIndex() (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, errors.New("meilisearch 服务不可用，请检查配置或连接状态")
	}

	fmt.Println("开始构建 Meilisearch 主题索引...")

	client := meiliconnect.GetClient()
	indexName := TopicIndex
	index := client.Index(indexName)

	fmt.Println("配置索引设置...")
	if err := configureIndex(index); err != nil {
		return nil, fmt.Errorf("配置索引失败: %w", err)
	}

	var topicStartID uint64
	limit := 100
	processedCount := 0
	failedCount := 0
	totalBatches := 0
	expectedIDs := make(map[string]struct{})

	for {
		topicList := topics.QueryById(topicStartID, limit)
		if len(topicList) == 0 {
			break
		}
		lo.ForEach(topicList, func(topic *topics.Entity, _ int) {
			firstPost := posts.Get(topic.FirstPostId)
			if firstPost.Id == 0 {
				firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
			}
			// 先登记应存在于索引的文档 ID：即使本次写入失败，幽灵清理也不得
			// 删除数据库仍要求保留的文档（写入失败由 failedCount 暴露）。
			if topic.Status == 1 && topic.ProcessStatus == 0 {
				expectedIDs[cast.ToString(topic.Id)] = struct{}{}
			}
			task, err := BuildSingleTopicSearchDocument(topic, &firstPost)
			if err != nil {
				failedCount++
				slog.Warn("failed to build topic search document", "topicId", topic.Id, "err", err)
				return
			}
			fmt.Printf("处理主题 ID:%v, TaskUID: %v\n", topic.Id, getTaskUID(task))
			processedCount++
		})
		topicStartID = topicList[len(topicList)-1].Id

		totalBatches++
		if len(topicList) < limit {
			break
		}
	}

	// 幽灵清理删除候选在入队前按数据库最新状态复核，跳过 snapshot 之后
	// 新创建或恢复为可索引的文档（PR #151 review P1 竞态：线上事件处理器的
	// upsert 可能晚于 snapshot 到达，不能把它判定为幽灵删除）。
	revalidateTopicGhost := func(id string) (bool, error) {
		topicID := cast.ToUint64(id)
		if topicID == 0 {
			return false, nil
		}
		topic, err := topics.GetWithError(topicID)
		if err != nil {
			// 记录不存在 → 确实是幽灵；其他错误（如 DB 瞬时故障）→ 保守保留，
			// 宁可不删也不误删有效文档。
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return false, nil
			}
			return true, err
		}
		return topic.Status == 1 && topic.ProcessStatus == 0, nil
	}
	deletedIDs, err := cleanupGhostDocuments(index, expectedIDs, revalidateTopicGhost)
	if err != nil {
		return nil, fmt.Errorf("清理主题索引幽灵文档失败: %w", err)
	}
	ghostRemoved := len(deletedIDs)

	// replay：删除任务入队后、执行前，事件处理器仍可能为新文档入队 upsert；
	// Meilisearch 同索引任务按入队顺序执行，因此把删除入队期间重新变为可索引
	// 的文档重新入队 upsert（排在 delete 之后），确保有效文档最终不丢失。
	replayedCount := 0
	for _, id := range deletedIDs {
		topicID := cast.ToUint64(id)
		if topicID == 0 {
			continue
		}
		topic := topics.Get(topicID)
		if topic.Id == 0 || !(topic.Status == 1 && topic.ProcessStatus == 0) {
			continue
		}
		firstPost := posts.Get(topic.FirstPostId)
		if firstPost.Id == 0 {
			firstPost, _ = posts.GetByTopicPostNoAtOrAfter(topic.Id, 1)
		}
		if _, err := BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
			failedCount++
			slog.Warn("failed to restore topic search document after ghost cleanup", "topicId", topic.Id, "err", err)
			continue
		}
		replayedCount++
	}

	result := &IndexBuildResult{
		ProcessedCount: processedCount,
		FailedCount:    failedCount,
		TotalBatches:   totalBatches,
		IndexName:      indexName,
		GhostRemoved:   ghostRemoved,
	}

	fmt.Printf("\n=== Meilisearch 索引构建完成 ===\n")
	fmt.Printf("处理批次: %d\n", result.TotalBatches)
	fmt.Printf("成功索引: %d 个主题\n", result.ProcessedCount)
	fmt.Printf("失败数量: %d 个主题\n", result.FailedCount)
	fmt.Printf("提交幽灵文档删除任务: %d 个\n", result.GhostRemoved)
	fmt.Printf("清理期间恢复索引文档: %d 个\n", replayedCount)
	fmt.Printf("索引名称: %s\n", result.IndexName)

	return result, nil
}

// configureIndex applies searchable, filterable, sortable and displayed fields.
func configureIndex(index meilisearch.IndexManager) error {
	searchableAttributes := []string{
		"title",
		"searchContent",
	}
	_, err := index.UpdateSearchableAttributes(&searchableAttributes)
	if err != nil {
		return fmt.Errorf("设置可搜索字段失败: %w", err)
	}

	filterableAttributes := []any{
		"category",
	}
	_, err = index.UpdateFilterableAttributes(&filterableAttributes)
	if err != nil {
		return fmt.Errorf("设置可过滤字段失败: %w", err)
	}

	sortableAttributes := []string{
		"createdAt",
		"updatedAt",
	}
	_, err = index.UpdateSortableAttributes(&sortableAttributes)
	if err != nil {
		return fmt.Errorf("设置可排序字段失败: %w", err)
	}

	displayedAttributes := []string{"id", "title"}
	_, err = index.UpdateDisplayedAttributes(&displayedAttributes)
	if err != nil {
		return fmt.Errorf("设置显示字段失败: %w", err)
	}

	fmt.Println("索引配置完成:")
	fmt.Printf("- 可搜索字段: %v\n", searchableAttributes)
	fmt.Printf("- 可过滤字段: %v\n", filterableAttributes)
	fmt.Printf("- 可排序字段: %v\n", sortableAttributes)

	return nil
}

// getTaskUID returns nil when no task was created.
func getTaskUID(task *meilisearch.TaskInfo) any {
	if task == nil {
		return nil
	}
	return task.TaskUID
}
