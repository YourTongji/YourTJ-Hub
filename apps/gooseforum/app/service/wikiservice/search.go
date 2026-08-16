package wikiservice

import (
	"sort"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
)

// PageSearchResult 页面级 wiki 站内搜索结果（聚合段落命中）。
// Anchors 携带页面内全部命中段落锚点，供前端跳转首个命中并页内连续定位。
type PageSearchResult struct {
	Namespace string   `json:"namespace"`
	Path      string   `json:"path"`
	Title     string   `json:"title"`
	TitleHit  bool     `json:"titleHit"`
	Heading   string   `json:"heading,omitempty"`
	Anchors   []string `json:"anchors"`
	Snippet   string   `json:"snippet"`
	Score     float64  `json:"score"`
	HitType   string   `json:"hitType"` // "title" | "body"
}

// PageSearchResponse wiki 站内搜索响应。
type PageSearchResponse struct {
	Items []PageSearchResult `json:"items"`
	Total int64              `json:"total"`
}

// isPagePublic 判断页面是否对公开读可见（与 filterPublicPages 同一条件）：
// topic 已发布、未封禁/待审、未软删、可见性正常。
func isPagePublic(page *wikiPages.Entity) bool {
	if page == nil || page.Id == 0 {
		return false
	}
	topic := topics.Get(page.TopicId)
	if topic.Id == 0 {
		return false
	}
	return topic.Status == 1 &&
		topic.VisibilityStatus == topics.VisibilityActive &&
		topic.ProcessStatus == topics.ProcessStatusNormal
}

// SearchPages wiki 站内全文搜索：Meilisearch 段落级命中 → 过滤可见性 →
// 聚合为页面级结果（按 score 降序）。TitleHit=true 表示标题命中。
// total 是页面级结果数（distinct pageId，review P2）：段落索引按段建文档，
// 段落级 EstimatedTotalHits 会把同一页的多段命中重复计数，不能作为页面数。
func SearchPages(query string, limit int) (*PageSearchResponse, error) {
	if limit <= 0 || limit > 20 {
		limit = 12
	}
	resp, err := searchservice.SearchWikiPageHits(query, limit*3)
	if err != nil {
		return nil, err
	}
	pageTotal, err := searchservice.CountWikiPages(query)
	if err != nil {
		return nil, err
	}
	result := &PageSearchResponse{Items: []PageSearchResult{}, Total: pageTotal}
	if len(resp.Hits) == 0 {
		return result, nil
	}

	// 页面聚合：同 pageId 的段落命中合并为一条结果；先按 hit 顺序收集，最后
	// 按最高 score 降序。不可见页面（topic 已删/隐藏）直接剔除。
	byPage := make(map[uint64]*PageSearchResult)
	var order []uint64
	for _, hit := range resp.Hits {
		page := wikiPages.Get(hit.PageId)
		if page.Id == 0 || !isPagePublic(&page) {
			continue
		}
		aggregated, ok := byPage[hit.PageId]
		if !ok {
			aggregated = &PageSearchResult{
				Namespace: displayNamespaceName(&page),
				Path:      page.Path,
				Title:     hit.Title,
				TitleHit:  hit.Title != "" && containsHighlight(hit.Title),
				Anchors:   []string{},
				Snippet:   hit.Snippet,
				Score:     hit.Score,
				HitType:   "body",
			}
			if aggregated.TitleHit {
				aggregated.HitType = "title"
			}
			byPage[hit.PageId] = aggregated
			order = append(order, hit.PageId)
		}
		if hit.Score > aggregated.Score {
			aggregated.Score = hit.Score
		}
		if hit.Heading != "" && aggregated.Heading == "" {
			aggregated.Heading = hit.Heading
		}
		aggregated.Anchors = append(aggregated.Anchors, hit.Anchor)
	}

	for _, pageID := range order {
		result.Items = append(result.Items, *byPage[pageID])
	}
	sort.SliceStable(result.Items, func(i, j int) bool {
		return result.Items[i].Score > result.Items[j].Score
	})
	if len(result.Items) > limit {
		result.Items = result.Items[:limit]
	}
	return result, nil
}

// containsHighlight 判断字符串是否含 Meilisearch 高亮标记（命中词）。
func containsHighlight(s string) bool {
	return strings.Contains(s, "<mark>") || strings.Contains(s, "<em>")
}

// displayNamespaceName 输出命名空间显示名（wiki_namespaces.name，降级=URL key）。
func displayNamespaceName(page *wikiPages.Entity) string {
	// 模型层命名空间列存 URL key；显示名反查（与 LoadPageDetail 一致）。
	if ns := wikiNamespaces.GetBySlug(page.Namespace); ns.Id != 0 && ns.Name != "" {
		return ns.Name
	}
	return page.Namespace
}
