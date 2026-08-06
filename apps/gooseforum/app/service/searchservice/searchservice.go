package searchservice

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/connect/meiliconnect"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/meilisearch/meilisearch-go"
	"github.com/samber/lo"
)

// ErrSearchUnavailable 表示搜索服务不可用
var ErrSearchUnavailable = errors.New("search service unavailable")

// SearchRequest is a topic search request.
type SearchRequest struct {
	Query      string   `json:"query"`
	Categories []uint64 `json:"categories"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
}

// SearchResult is one search hit.
type SearchResult struct {
	ID    uint64 `json:"id"`
	Title string `json:"title"`
}

// SearchResponse is the topic search response.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int64          `json:"total"`
}

// SearchTopics returns topic IDs and titles directly from Meilisearch.
func SearchTopics(req SearchRequest) (*SearchResponse, error) {
	if !meiliconnect.IsAvailable() {
		return searchTopicsFromDatabase(req)
	}

	client := meiliconnect.GetClient()
	index := client.Index(TopicIndex)

	searchReq := &meilisearch.SearchRequest{
		Query:  req.Query,
		Limit:  int64(req.Limit),
		Offset: int64(req.Offset),
	}

	if len(req.Categories) > 0 {
		filters := lo.Map(req.Categories, func(categoryID uint64, _ int) string {
			return fmt.Sprintf("category = %d", categoryID)
		})
		filterStr := fmt.Sprintf("(%s)", strings.Join(filters, " OR "))
		searchReq.Filter = filterStr
	}

	searchReq.AttributesToRetrieve = []string{"id", "title"}

	searchResp, err := index.Search(req.Query, searchReq)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	results := lo.FilterMap(searchResp.Hits, func(hit meilisearch.Hit, _ int) (SearchResult, bool) {
		itemResult := SearchResult{}
		if err := hit.Decode(&itemResult); err != nil {
			slog.Error("failed to decode search hit", "err", err)
			return SearchResult{}, false
		}
		return itemResult, itemResult.ID > 0
	})

	return &SearchResponse{
		Results: results,
		Total:   searchResp.EstimatedTotalHits,
	}, nil
}

// searchTopicsFromDatabase 在未配置 Meilisearch 时退化为数据库 LIKE 搜索，
// 保证搜索功能开箱可用；仅匹配公开主题（status=1 且 process_status=0）
// 的标题与摘要，按置顶权重和最近更新排序。
func searchTopicsFromDatabase(req SearchRequest) (*SearchResponse, error) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		return &SearchResponse{Results: []SearchResult{}, Total: 0}, nil
	}
	keyword := "%" + query + "%"
	condition := "status = ? AND process_status = ? AND (title LIKE ? OR excerpt LIKE ?)"
	tableName := (&topics.Entity{}).TableName()

	var total int64
	countDB := db.Connect().Table(tableName).Where(condition, 1, 0, keyword, keyword)
	if err := countDB.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("搜索计数失败: %w", err)
	}

	entities := make([]topics.Entity, 0)
	resultDB := db.Connect().Table(tableName).
		Where(condition, 1, 0, keyword, keyword).
		Order("pin_weight DESC, updated_at DESC").
		Limit(req.Limit).
		Offset(req.Offset).
		Find(&entities)
	if resultDB.Error != nil {
		return nil, fmt.Errorf("搜索失败: %w", resultDB.Error)
	}

	results := lo.Map(entities, func(item topics.Entity, _ int) SearchResult {
		return SearchResult{ID: item.Id, Title: item.Title}
	})
	return &SearchResponse{Results: results, Total: total}, nil
}
