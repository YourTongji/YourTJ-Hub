package wikiservice

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

// 修订状态字符串（契约枚举：approved 为写即发布的当前状态；pending/rejected/superseded
// 仅为 v19 前遗留数据的兼容展示，新写入一律 approved）。
const (
	StatusStringPending    = "pending"
	StatusStringApproved   = "approved"
	StatusStringRejected   = "rejected"
	StatusStringSuperseded = "superseded"
)

// RevisionStatusString 将修订状态 int8 映射为契约字符串。
func RevisionStatusString(s int8) string {
	switch s {
	case wikiPageRevisions.StatusApproved:
		return StatusStringApproved
	case wikiPageRevisions.StatusRejected:
		return StatusStringRejected
	case wikiPageRevisions.StatusSuperseded:
		return StatusStringSuperseded
	default:
		return StatusStringPending
	}
}

// TreePage 导航树中的一页。
type TreePage struct {
	PageId uint64 `json:"pageId"`
	Path   string `json:"path"`
	Title  string `json:"title"`
	Active bool   `json:"active"`
}

// TreeNamespace 导航树中的一个 namespace 分组。
type TreeNamespace struct {
	Name  string     `json:"name"`
	Label string     `json:"label"`
	Pages []TreePage `json:"pages"`
}

// WikiTreeResult 公开导航树响应（契约包裹层）。
type WikiTreeResult struct {
	Namespaces []TreeNamespace `json:"namespaces"`
}

// BuildTree 构建 wiki 导航树（按 namespace 分组，当前页 active）。
func BuildTree(activePath string) []TreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []TreeNamespace{}
	}
	allPages := filterPublicPages(wikiPages.ListAll())
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byNamespace[page.Namespace] = append(byNamespace[page.Namespace], page)
	}
	titles := pageTitles(allPages)

	result := make([]TreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byNamespace[ns.Name]
		items := make([]TreePage, 0, len(pages))
		for _, page := range pages {
			items = append(items, TreePage{
				PageId: page.Id,
				Path:   page.Path,
				Title:  titles[page.Id],
				Active: page.Path == activePath,
			})
		}
		result = append(result, TreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Pages: items,
		})
	}
	return result
}

// filterPublicPages 过滤出 topic 仍公开的页面：删除/隐藏的 wiki 页面不得
// 出现在公开导航树、首页与摘要（review：此前仅靠 wiki_pages 自身软删，
// topic 被治理删除而页面行仍可见时树/首页会泄漏）。
func filterPublicPages(pages []*wikiPages.Entity) []*wikiPages.Entity {
	if len(pages) == 0 {
		return pages
	}
	ids := make([]uint64, 0, len(pages))
	for _, p := range pages {
		ids = append(ids, p.TopicId)
	}
	topicMap := topics.GetMapByIds(ids)
	filtered := make([]*wikiPages.Entity, 0, len(pages))
	for _, p := range pages {
		t, ok := topicMap[p.TopicId]
		if !ok {
			continue
		}
		if t.Status != 1 || t.VisibilityStatus != topics.VisibilityActive {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

func pageTitles(pages []*wikiPages.Entity) map[uint64]string {
	result := make(map[uint64]string, len(pages))
	ids := make([]uint64, 0, len(pages))
	for _, p := range pages {
		ids = append(ids, p.Id)
	}
	// 批量取最新修订：approved 优先，纯 pending（草稿）页面回退取 pending 标题
	// （review N+1：此前每页各一次 GetLatestApproved/GetLatestPending）。
	latest := wikiPageRevisions.LatestByPages(ids, wikiPageRevisions.StatusApproved, wikiPageRevisions.StatusPending)
	for _, p := range pages {
		if rev, ok := latest[p.Id]; ok && rev.Id != 0 {
			result[p.Id] = rev.Title
		}
	}
	return result
}

// BuildTreeAPI 构建公开导航树（契约形状）：path 为 namespace 内相对路径，
// active = 页面存在至少一条 approved 修订（纯 pending 页面为草稿）。
func BuildTreeAPI() WikiTreeResult {
	return WikiTreeResult{Namespaces: buildTree("", true)}
}

// buildTree 构建导航树；relative=true 时 path 相对 namespace 且 active 语义
// 为「存在 approved 修订」，否则 path 为完整路径且 active 为当前页高亮。
func buildTree(activePath string, contractShape bool) []TreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []TreeNamespace{}
	}
	allPages := filterPublicPages(wikiPages.ListAll())
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byNamespace[page.Namespace] = append(byNamespace[page.Namespace], page)
	}
	titles := pageTitles(allPages)
	approved := pageApprovedSet(allPages)

	result := make([]TreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byNamespace[ns.Name]
		items := make([]TreePage, 0, len(pages))
		for _, page := range pages {
			path := page.Path
			if contractShape {
				path = strings.TrimPrefix(path, ns.Name+"/")
			}
			items = append(items, TreePage{
				PageId: page.Id,
				Path:   path,
				Title:  titles[page.Id],
				Active: approved[page.Id].approved,
			})
		}
		result = append(result, TreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Pages: items,
		})
	}
	return result
}

// pageApprovedSet 批量返回页面是否已发布（存在 approved 修订）。
// 一次 SQL 查询取全部页面的最新 approved/pending 修订，替代逐页 GetLatestApproved
// （review N+1：公开导航树热路径，页面规模增长后是放大的 DoS 面）。
type pageApproved struct {
	approved bool
	pending  bool
}

func pageApprovedSet(pages []*wikiPages.Entity) map[uint64]*pageApproved {
	pageIDs := make([]uint64, 0, len(pages))
	for _, p := range pages {
		pageIDs = append(pageIDs, p.Id)
	}
	result := make(map[uint64]*pageApproved, len(pageIDs))
	latest := wikiPageRevisions.LatestByPages(pageIDs, wikiPageRevisions.StatusApproved, wikiPageRevisions.StatusPending)
	for _, id := range pageIDs {
		rev, ok := latest[id]
		if !ok {
			result[id] = &pageApproved{}
			continue
		}
		result[id] = &pageApproved{
			approved: rev.Status == wikiPageRevisions.StatusApproved,
			pending:  rev.Status == wikiPageRevisions.StatusPending,
		}
	}
	return result
}

// AdminTreePage 管理端导航树中的一页。
type AdminTreePage struct {
	PageId    uint64 `json:"pageId"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	SortOrder int    `json:"sortOrder"`
}

// AdminTreeNamespace 管理端导航树中的一个 namespace 分组。
type AdminTreeNamespace struct {
	Name  string          `json:"name"`
	Label string          `json:"label"`
	Pages []AdminTreePage `json:"pages"`
}

// BuildAdminTree 构建管理端导航树（含 sortOrder；path 为完整路径，含 namespace 段）。
func BuildAdminTree() []AdminTreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []AdminTreeNamespace{}
	}
	allPages := wikiPages.ListAll()
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byNamespace[page.Namespace] = append(byNamespace[page.Namespace], page)
	}
	titles := pageTitles(allPages)
	result := make([]AdminTreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byNamespace[ns.Name]
		items := make([]AdminTreePage, 0, len(pages))
		for _, page := range pages {
			items = append(items, AdminTreePage{
				PageId:    page.Id,
				Path:      page.Path,
				Title:     titles[page.Id],
				SortOrder: page.SortOrder,
			})
		}
		result = append(result, AdminTreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Pages: items,
		})
	}
	return result
}

// NamespaceSummary 首页 namespace 卡。
type NamespaceSummary struct {
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	SortOrder     int       `json:"sortOrder"`
	PageCount     int64     `json:"pageCount"`
	UpdatedAt     time.Time `json:"updatedAt"`
	FirstPagePath string    `json:"firstPagePath"`
}

// BuildNamespaceSummaries 返回 namespace 摘要列表（含 approved 页面数与最近更新时间）。
func BuildNamespaceSummaries() []NamespaceSummary {
	namespaces := wikiNamespaces.List()
	pages := filterPublicPages(wikiPages.ListAll())
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, p := range pages {
		byNamespace[p.Namespace] = append(byNamespace[p.Namespace], p)
	}
	// 批量取各页最新 approved 修订（review N+1：此前逐页 GetLatestApproved）。
	pageIDs := make([]uint64, 0, len(pages))
	for _, p := range pages {
		pageIDs = append(pageIDs, p.Id)
	}
	latest := wikiPageRevisions.LatestByPages(pageIDs, wikiPageRevisions.StatusApproved)
	summaries := make([]NamespaceSummary, 0, len(namespaces))
	for _, ns := range namespaces {
		nsPages := byNamespace[ns.Name]
		updated := ns.UpdatedAt
		count := int64(0)
		firstPath := ""
		for _, p := range nsPages {
			rev, ok := latest[p.Id]
			if ok && rev.Id != 0 {
				if firstPath == "" {
					firstPath = p.Path
				}
				count++
				if rev.CreatedAt.After(updated) {
					updated = rev.CreatedAt
				}
			}
		}
		summaries = append(summaries, NamespaceSummary{
			Name:          ns.Name,
			Description:   ns.Description,
			SortOrder:     ns.SortOrder,
			PageCount:     count,
			UpdatedAt:     updated,
			FirstPagePath: firstPath,
		})
	}
	return summaries
}

// RecentPage 首页最近更新条目。
type RecentPage struct {
	PageId     uint64 `json:"pageId"`
	Path       string `json:"path"`
	Title      string `json:"title"`
	UpdatedAt  string `json:"updatedAt"`
	EditorId   uint64 `json:"editorId"`
	EditorName string `json:"editorName"`
}

// HomeData 首页数据。
type HomeData struct {
	Namespaces []NamespaceSummary `json:"namespaces"`
	Recent     []RecentPage       `json:"recent"`
}

// BuildHome 组装 wiki 首页数据。
func BuildHome() HomeData {
	pages := filterPublicPages(wikiPages.ListAll())

	summaries := BuildNamespaceSummaries()
	// 最近更新：全部页面最新 approved 修订，按时间降序取 10。
	// 批量取（review N+1：此前逐页 GetLatestApproved）。
	pageIDs := make([]uint64, 0, len(pages))
	for _, p := range pages {
		pageIDs = append(pageIDs, p.Id)
	}
	latest := wikiPageRevisions.LatestByPages(pageIDs, wikiPageRevisions.StatusApproved)
	type recent struct {
		page *wikiPages.Entity
		rev  wikiPageRevisions.Entity
	}
	all := make([]recent, 0, len(pages))
	for _, p := range pages {
		rev, ok := latest[p.Id]
		if ok && rev.Id != 0 {
			all = append(all, recent{page: p, rev: *rev})
		}
	}
	// 简单排序（修订时间降序）。
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].rev.CreatedAt.After(all[j-1].rev.CreatedAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if len(all) > 10 {
		all = all[:10]
	}

	editorIDs := make([]uint64, 0, len(all))
	for _, item := range all {
		editorIDs = append(editorIDs, item.rev.EditorId)
	}
	userMap := users.GetMapByIds(editorIDs)

	recentPages := make([]RecentPage, 0, len(all))
	for _, item := range all {
		editorName := ""
		if u, ok := userMap[item.rev.EditorId]; ok && u != nil {
			editorName = u.Username
		}
		recentPages = append(recentPages, RecentPage{
			PageId:     item.page.Id,
			Path:       item.page.Path,
			Title:      item.rev.Title,
			UpdatedAt:  item.rev.CreatedAt.Format(time.RFC3339),
			EditorId:   item.rev.EditorId,
			EditorName: editorName,
		})
	}

	return HomeData{Namespaces: summaries, Recent: recentPages}
}

// RevisionView 修订历史条目。
type RevisionView struct {
	RevisionId uint64 `json:"revisionId"`
	PageId     uint64 `json:"pageId"`
	RevisionNo int    `json:"revisionNo"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	EditorId   uint64 `json:"editorId"`
	EditorName string `json:"editorName"`
	UpdatedAt  string `json:"updatedAt"`
}

// ListRevisions 返回某页面的公开修订历史（仅 approved，降序，附编辑者名）。
// 契约：pending/superseded/rejected 修订含未发布内容，仅管理端可见（蓝图风险项
// 「待审内容泄漏给公众」）；公开历史只展示已发布版本。
func ListRevisions(pageId uint64) []RevisionView {
	// SQL 层过滤 approved，避免把含 content/rendered_html 大字段的未发布修订拉进内存
	// （review：公开历史只展示已发布版本）。
	revisions := wikiPageRevisions.ListApprovedByPage(pageId)
	if len(revisions) == 0 {
		return []RevisionView{}
	}
	editorIDs := make([]uint64, 0, len(revisions))
	for _, r := range revisions {
		editorIDs = append(editorIDs, r.EditorId)
	}
	userMap := users.GetMapByIds(editorIDs)
	result := make([]RevisionView, 0, len(editorIDs))
	for _, r := range revisions {
		editorName := ""
		if u, ok := userMap[r.EditorId]; ok && u != nil {
			editorName = u.Username
		}
		result = append(result, RevisionView{
			RevisionId: r.Id,
			PageId:     r.PageId,
			RevisionNo: r.RevisionNo,
			Title:      r.Title,
			Content:    r.Content,
			Status:     RevisionStatusString(r.Status),
			EditorId:   r.EditorId,
			EditorName: editorName,
			UpdatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

// Contributor 贡献者条目。
type Contributor struct {
	UserId       uint64    `json:"userId"`
	Username     string    `json:"username"`
	AvatarUrl    string    `json:"avatarUrl"`
	Count        int       `json:"count"`
	LastEditedAt time.Time `json:"lastEditedAt"`
}

// BuildContributors 聚合某页面的贡献者（按编辑者分组计数）。
func BuildContributors(pageId uint64) []Contributor {
	revisions := wikiPageRevisions.ListByPage(pageId)
	if len(revisions) == 0 {
		return []Contributor{}
	}
	byEditor := make(map[uint64]*Contributor)
	for _, r := range revisions {
		if r.EditorId == 0 {
			continue
		}
		c, ok := byEditor[r.EditorId]
		if !ok {
			c = &Contributor{UserId: r.EditorId}
			byEditor[r.EditorId] = c
		}
		c.Count++
		if r.CreatedAt.After(c.LastEditedAt) {
			c.LastEditedAt = r.CreatedAt
		}
	}
	ids := make([]uint64, 0, len(byEditor))
	for id := range byEditor {
		ids = append(ids, id)
	}
	userMap := users.GetMapByIds(ids)
	result := make([]Contributor, 0, len(byEditor))
	for id, c := range byEditor {
		if u, ok := userMap[id]; ok && u != nil {
			c.Username = u.Username
			c.AvatarUrl = u.GetWebAvatarUrl()
		}
		result = append(result, *c)
	}
	return result
}

// PageDetail 详情页数据（公开渲染 approved 快照）。
type PageDetail struct {
	Id                  uint64    `json:"id"`
	TopicId             uint64    `json:"topicId"`
	Namespace           string    `json:"namespace"`
	Path                string    `json:"path"`
	Title               string    `json:"title"`
	Content             string    `json:"content"`
	Toc                 []TocItem `json:"toc"`
	UpdatedAt           string    `json:"updatedAt"`
	EditorId            uint64    `json:"editorId"`
	EditorName          string    `json:"editorName"`
	LikeCount           uint64    `json:"likeCount"`
	ViewCount           uint64    `json:"viewCount"`
	PostCount           uint64    `json:"postCount"`
	Liked               bool      `json:"liked"`
	Bookmarked          bool      `json:"bookmarked"`
	Watched             bool      `json:"watched"`
	PublishedRevisionNo int       `json:"publishedRevisionNo"`
	CanEdit             bool      `json:"canEdit"`
}

// TocItem 目录条目（渲染到前端）。
type TocItem struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// LoadPageDetail 加载页面详情（topic 可见性由调用方把关）。
func LoadPageDetail(page *wikiPages.Entity, topic *topics.Entity) (PageDetail, error) {
	rev := wikiPageRevisions.GetLatestApproved(page.Id)
	if rev.Id == 0 {
		// 无 approved 修订（理论上创建即 approved），返回空内容。
		rev = wikiPageRevisions.Entity{Title: topic.Title}
	}
	detail := PageDetail{
		Id:                  page.Id,
		TopicId:             topic.Id,
		Namespace:           page.Namespace,
		Path:                page.Path,
		Title:               rev.Title,
		Content:             rev.RenderedHTML,
		UpdatedAt:           rev.CreatedAt.Format(time.RFC3339),
		EditorId:            rev.EditorId,
		LikeCount:           topic.LikeCount,
		ViewCount:           topic.ViewCount,
		PostCount:           topic.PostCount,
		PublishedRevisionNo: page.PublishedRevisionNo,
	}
	if rev.Toc != "" {
		var items []TocItem
		if err := json.Unmarshal([]byte(rev.Toc), &items); err == nil {
			detail.Toc = items
		}
	}
	editorName := ""
	if rev.EditorId != 0 {
		if u, ok := users.GetMapByIds([]uint64{rev.EditorId})[rev.EditorId]; ok && u != nil {
			editorName = u.Username
		}
	}
	detail.EditorName = editorName
	return detail, nil
}

// AdminRevision 管理端版本历史条目（契约形状：updatedAt 为 RFC3339 字符串）。
type AdminRevision struct {
	RevisionId uint64 `json:"revisionId"`
	PageId     uint64 `json:"pageId"`
	RevisionNo int    `json:"revisionNo"`
	Path       string `json:"path"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Status     string `json:"status"`
	EditorId   uint64 `json:"editorId"`
	EditorName string `json:"editorName"`
	UpdatedAt  string `json:"updatedAt"`
}

// AdminRevisionPage 版本历史分页结果（契约形状：{list, page, pageSize, hasNext}）。
type AdminRevisionPage struct {
	List     []AdminRevision `json:"list"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
	HasNext  bool            `json:"hasNext"`
}

// ListAdminRevisions 分页返回版本历史（pageId>0 时只列该页；附 path/编辑者/版本号）。
// 写即发布后无审核队列：管理端版本历史 = 全部修订（含回滚截断后的剩余版本），
// 供 diff 对比与回滚选择目标版本。
func ListAdminRevisions(pageId uint64, page, pageSize int) AdminRevisionPage {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	// pageSize+1 探测 hasNext（与 topics_rep.PageForModeration 一致），
	// 避免额外的 COUNT 查询。
	revisions := wikiPageRevisions.ListRecent(pageId, page, pageSize+1)
	total := len(revisions)
	hasNext := total > pageSize
	if total > pageSize {
		revisions = revisions[:pageSize]
		total = pageSize
	}
	if len(revisions) == 0 {
		return AdminRevisionPage{
			List:     []AdminRevision{},
			Page:     page,
			PageSize: pageSize,
			HasNext:  false,
		}
	}
	editorIDs := make([]uint64, 0, len(revisions))
	pageIDs := make([]uint64, 0, len(revisions))
	for _, r := range revisions {
		editorIDs = append(editorIDs, r.EditorId)
		pageIDs = append(pageIDs, r.PageId)
	}
	pageMap := make(map[uint64]*wikiPages.Entity, len(pageIDs))
	for _, p := range wikiPages.ListByIDs(pageIDs) {
		pageMap[p.Id] = p
	}
	userMap := users.GetMapByIds(editorIDs)
	result := make([]AdminRevision, 0, len(revisions))
	for _, r := range revisions {
		editorName := ""
		if u, ok := userMap[r.EditorId]; ok && u != nil {
			editorName = u.Username
		}
		path := ""
		if p, ok := pageMap[r.PageId]; ok && p != nil {
			path = p.Path
		}
		result = append(result, AdminRevision{
			RevisionId: r.Id,
			PageId:     r.PageId,
			RevisionNo: r.RevisionNo,
			Path:       path,
			Title:      r.Title,
			Content:    r.Content,
			Status:     RevisionStatusString(r.Status),
			EditorId:   r.EditorId,
			EditorName: editorName,
			UpdatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}
	return AdminRevisionPage{
		List:     result,
		Page:     page,
		PageSize: pageSize,
		HasNext:  hasNext,
	}
}

// DecodeTOC 解析修订表 toc JSON 为条目。
func DecodeTOC(raw string) []markdown2html.HeadingItem {
	if raw == "" {
		return nil
	}
	var items []markdown2html.HeadingItem
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}
