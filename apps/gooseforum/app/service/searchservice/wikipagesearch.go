package searchservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/meiliconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/meilisearch/meilisearch-go"
)

// WikiPageIndex 是 wiki 站内局内搜索的 Meilisearch 索引名。
// 文档粒度：每段一文档（document per paragraph），搜索结果天然是「点」级，
// 直接携带段落锚点（anchor: "s-<n>"）供前端精准定位。
const WikiPageIndex = "wiki_pages"

// wikiParaAnchorDB 与 wikiservice.ParaAnchor 同形的 DB 投影解析结构
// （searchservice 不能被 wikiservice import，避免 import 循环；JSON 形状对齐）。
type wikiParaAnchorDB struct {
	Index       int    `json:"index"`
	Anchor      string `json:"anchor"`
	HeadingId   string `json:"headingId"`
	HeadingText string `json:"headingText"`
	Text        string `json:"text"`
}

// WikiPageDocument 每段一文档的 Meilisearch 文档结构。
type WikiPageDocument struct {
	ID        string `json:"id"` // "<pageId>-<paraIndex>"
	PageId    uint64 `json:"pageId"`
	IsPublic  bool   `json:"isPublic"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	Namespace string `json:"namespace"`
	Heading   string `json:"heading"`
	Anchor    string `json:"anchor"`
	Paragraph string `json:"paragraph"`
	SortOrder int    `json:"sortOrder"`
}

// WikiPageHit 段落级公开命中（由索引过滤公开页面，wikiservice 仍会再次校验可见性并聚合）。
type WikiPageHit struct {
	PageId    uint64  `json:"pageId"`
	Path      string  `json:"path"`
	Title     string  `json:"title"`
	Namespace string  `json:"namespace"`
	Heading   string  `json:"heading"`
	Anchor    string  `json:"anchor"`
	Snippet   string  `json:"snippet"` // 高亮片段（<mark> 包裹命中词）
	Score     float64 `json:"score"`
}

// WikiPageSearchResponse 段落级搜索结果。
type WikiPageSearchResponse struct {
	Hits  []WikiPageHit `json:"hits"`
	Total int64         `json:"total"`
}

// configureWikiPageIndex 配置 wiki_pages 索引的可搜索/可过滤/显示字段。
func configureWikiPageIndex(index meilisearch.IndexManager) error {
	searchable := []string{"title", "heading", "paragraph"}
	if _, err := index.UpdateSearchableAttributes(&searchable); err != nil {
		return fmt.Errorf("设置可搜索字段失败: %w", err)
	}
	filterable := []any{"pageId", "namespace", "isPublic"}
	if _, err := index.UpdateFilterableAttributes(&filterable); err != nil {
		return fmt.Errorf("设置可过滤字段失败: %w", err)
	}
	displayed := []string{
		"id", "pageId", "isPublic", "path", "title", "namespace", "heading", "anchor", "paragraph", "sortOrder",
	}
	if _, err := index.UpdateDisplayedAttributes(&displayed); err != nil {
		return fmt.Errorf("设置显示字段失败: %w", err)
	}
	return nil
}

// decodeWikiParaAnchors 解析 wiki_pages.para_anchors JSON。
func decodeWikiParaAnchors(raw string) []wikiParaAnchorDB {
	if raw == "" {
		return nil
	}
	var anchors []wikiParaAnchorDB
	if err := json.Unmarshal([]byte(raw), &anchors); err != nil {
		return nil
	}
	return anchors
}

// wikiPagePubliclySearchable 判断页面当前是否应出现在公开 wiki 搜索结果
// （对齐 wikiservice.filterPublicPages：已发布、未封禁/待审、未软删、可见性正常）。
func wikiPagePubliclySearchable(page *wikiPages.Entity) bool {
	if page == nil || page.Id == 0 {
		return false
	}
	topic := topics.Get(page.TopicId)
	return isTopicPubliclySearchable(&topic)
}

// BuildWikiPageIndex 全量重建 wiki 段落索引（对齐 BuildMeilisearchIndex 模式）。
// 只索引公开页面（topic 可见性过滤），删除文档先清空索引（重建语义）。
func BuildWikiPageIndex() (*IndexBuildResult, error) {
	if !meiliconnect.IsAvailable() {
		return nil, errors.New("meilisearch 服务不可用，请检查配置或连接状态")
	}
	client := meiliconnect.GetClient()
	indexName := WikiPageIndex
	index := client.Index(indexName)

	if err := configureWikiPageIndex(index); err != nil {
		return nil, fmt.Errorf("配置 wiki 索引失败: %w", err)
	}
	deleteTask, err := index.DeleteAllDocuments(&meilisearch.DocumentOptions{})
	if err != nil {
		return nil, fmt.Errorf("清空 wiki 索引失败: %w", err)
	}

	// 清空与写入必须顺序执行：DeleteAllDocuments 是异步任务，直接 AddDocuments
	// 会与删除任务竞争，导致文档残留/丢失。等待删除任务完成后才批量写入。
	if err := meiliconnect.WaitForTask(deleteTask); err != nil {
		return nil, fmt.Errorf("等待 wiki 索引清空失败: %w", err)
	}

	pages, err := wikiPages.ListAll()
	if err != nil {
		return nil, fmt.Errorf("读取 wiki 页面列表失败: %w", err)
	}
	documents := make([]WikiPageDocument, 0, len(pages)*3)
	processedCount := 0
	for _, page := range pages {
		if !wikiPagePubliclySearchable(page) {
			continue
		}
		documents = append(documents, wikiPageDocuments(page)...)
		processedCount++
	}
	totalBatches := 0
	for start := 0; start < len(documents); start += 500 {
		end := start + 500
		if end > len(documents) {
			end = len(documents)
		}
		batch := documents[start:end]
		if _, err := index.AddDocuments(batch, &meilisearch.DocumentOptions{PrimaryKey: strPtr("id")}); err != nil {
			return nil, fmt.Errorf("写入 wiki 索引批次失败: %w", err)
		}
		totalBatches++
	}
	slog.Info("wiki page index built", "index", indexName, "pages", processedCount, "documents", len(documents), "batches", totalBatches)
	return &IndexBuildResult{
		ProcessedCount: processedCount,
		TotalBatches:   totalBatches,
		IndexName:      indexName,
	}, nil
}

// wikiPageDocuments 把单页投影为段落文档列表（从 para_anchors 读段落纯文本，
// 无需二次解析 HTML）。
func wikiPageDocuments(page *wikiPages.Entity) []WikiPageDocument {
	anchors := decodeWikiParaAnchors(page.ParaAnchors)
	if len(anchors) == 0 {
		return nil
	}
	docs := make([]WikiPageDocument, 0, len(anchors))
	for _, a := range anchors {
		docs = append(docs, WikiPageDocument{
			ID:        fmt.Sprintf("%d-%d", page.Id, a.Index),
			PageId:    page.Id,
			IsPublic:  true,
			Path:      page.Path,
			Title:     page.Title,
			Namespace: page.Namespace,
			Heading:   a.HeadingText,
			Anchor:    a.Anchor,
			Paragraph: a.Text,
			SortOrder: a.Index,
		})
	}
	return docs
}

// IndexWikiPageDocuments 增量索引单页的段落文档（wiki sync 后调用）。
// 先按 pageId 删除该页现有文档并等待删除任务完成，再写入当前文档：
// 页面段落数可能减少（如 3 段缩为 2 段），旧 <pageId>-3 文档若不清理，
// 搜索会命中已删除文本并把用户带到不存在的 #s-3 锚点。无段落（纯标题页）
// 同样清空该页索引，避免整页旧索引继续存在。
func IndexWikiPageDocuments(pageID uint64) error {
	if !meiliconnect.IsAvailable() {
		return nil
	}
	page := wikiPages.Get(pageID)
	if page.Id == 0 || !wikiPagePubliclySearchable(&page) {
		// 页面不可见 → 索引侧全删（与 BuildSingleTopicSearchDocument 删除分支对称）。
		return DeleteWikiPageDocuments(pageID)
	}
	client := meiliconnect.GetClient()
	index := client.Index(WikiPageIndex)
	// 先按 pageId 删除该页现有文档并等待完成，再写入当前段落文档：
	// 删除与写入必须顺序执行（异步任务竞争会残留旧文档）。
	if err := DeleteWikiPageDocuments(pageID); err != nil {
		return err
	}
	docs := wikiPageDocuments(&page)
	if len(docs) == 0 {
		return nil
	}
	if _, err := index.AddDocuments(docs, &meilisearch.DocumentOptions{PrimaryKey: strPtr("id")}); err != nil {
		return fmt.Errorf("索引 wiki 页面 %d: %w", pageID, err)
	}
	return nil
}

// DeleteWikiPageDocuments 删除单页全部段落文档（页面软删/不可见/段落清空时）。
// 删除是异步任务：等待完成，保证调用方随后写入或搜索不会读到旧文档。
func DeleteWikiPageDocuments(pageID uint64) error {
	if !meiliconnect.IsAvailable() {
		return nil
	}
	client := meiliconnect.GetClient()
	index := client.Index(WikiPageIndex)
	filter := "pageId = " + fmt.Sprintf("%d", pageID)
	task, err := index.DeleteDocumentsByFilter([]string{filter}, nil)
	if err != nil {
		return fmt.Errorf("删除 wiki 页面 %d 索引: %w", pageID, err)
	}
	if err := meiliconnect.WaitForTask(task); err != nil {
		return fmt.Errorf("等待 wiki 页面 %d 索引删除: %w", pageID, err)
	}
	return nil
}

// SearchWikiPageHits 在 wiki_pages 索引执行段落级搜索，只返回标记为公开的文档。
func SearchWikiPageHits(query string, limit int) (*WikiPageSearchResponse, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return &WikiPageSearchResponse{Hits: []WikiPageHit{}, Total: 0}, nil
	}
	if len([]rune(query)) > 100 {
		return nil, errors.New("query too long (max 100 characters)")
	}
	if !meiliconnect.IsAvailable() {
		return nil, ErrSearchUnavailable
	}
	if limit <= 0 || limit > 50 {
		limit = 30
	}
	index := meiliconnect.GetClient().Index(WikiPageIndex)
	searchResp, err := index.Search(query, &meilisearch.SearchRequest{
		Limit:                 int64(limit),
		Filter:                "isPublic = true",
		AttributesToRetrieve:  []string{"id", "pageId", "path", "title", "namespace", "heading", "anchor", "paragraph"},
		AttributesToHighlight: []string{"title", "heading", "paragraph"},
		HighlightPreTag:       "<mark>",
		HighlightPostTag:      "</mark>",
		ShowRankingScore:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("wiki search: %w", err)
	}

	type rawHit struct {
		PageId    uint64 `json:"pageId"`
		Path      string `json:"path"`
		Title     string `json:"title"`
		Namespace string `json:"namespace"`
		Heading   string `json:"heading"`
		Anchor    string `json:"anchor"`
		Paragraph string `json:"paragraph"`
	}
	type rawFormatted struct {
		Title     string `json:"title"`
		Heading   string `json:"heading"`
		Paragraph string `json:"paragraph"`
	}

	hits := make([]WikiPageHit, 0, len(searchResp.Hits))
	for _, hit := range searchResp.Hits {
		var doc rawHit
		if err := hit.DecodeInto(&doc); err != nil {
			slog.Warn("wiki search: decode hit failed", "err", err)
			continue
		}
		if doc.PageId == 0 {
			continue
		}
		var formatted rawFormatted
		if raw, ok := hit["_formatted"]; ok {
			_ = json.Unmarshal(raw, &formatted)
		}
		snippet := formatted.Paragraph
		if snippet == "" {
			snippet = doc.Paragraph
		}
		score := float64(0)
		if raw, ok := hit["_rankingScore"]; ok {
			_ = json.Unmarshal(raw, &score)
		}
		hits = append(hits, WikiPageHit{
			PageId:    doc.PageId,
			Path:      doc.Path,
			Title:     formatted.Title,
			Namespace: doc.Namespace,
			Heading:   doc.Heading,
			Anchor:    doc.Anchor,
			Snippet:   snippet,
			Score:     score,
		})
	}
	return &WikiPageSearchResponse{Hits: hits, Total: searchResp.EstimatedTotalHits}, nil
}

// CountWikiPages 统计页面级命中数（distinct on pageId）。
// 段落索引按段建文档，段落级 EstimatedTotalHits 会把同一页的多段命中重复计数；
// 页面级 total 应去重（review P2）。distinct 在 Meilisearch 中要求该属性可过滤；
// isPublic 同时排除旧软删文档和尚未完成清理的非公开文档。
func CountWikiPages(query string) (int64, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, nil
	}
	if len([]rune(query)) > 100 {
		return 0, errors.New("query too long (max 100 characters)")
	}
	if !meiliconnect.IsAvailable() {
		return 0, ErrSearchUnavailable
	}
	index := meiliconnect.GetClient().Index(WikiPageIndex)
	searchResp, err := index.Search(query, &meilisearch.SearchRequest{
		Limit:    1,
		Distinct: "pageId",
		Filter:   "isPublic = true",
	})
	if err != nil {
		return 0, fmt.Errorf("wiki page count: %w", err)
	}
	return searchResp.EstimatedTotalHits, nil
}

func strPtr(s string) *string {
	return &s
}
