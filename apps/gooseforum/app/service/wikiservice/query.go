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

// 修订状态字符串（OpenAPI 契约枚举：approved/pending/rejected/superseded）。
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

// ParseRevisionStatus 将契约状态字符串映射为 int8；未知返回 false。
func ParseRevisionStatus(s string) (int8, bool) {
	switch s {
	case StatusStringPending:
		return wikiPageRevisions.StatusPending, true
	case StatusStringApproved:
		return wikiPageRevisions.StatusApproved, true
	case StatusStringRejected:
		return wikiPageRevisions.StatusRejected, true
	case StatusStringSuperseded:
		return wikiPageRevisions.StatusSuperseded, true
	default:
		return 0, false
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
	allPages := wikiPages.ListAll()
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

func pageTitles(pages []*wikiPages.Entity) map[uint64]string {
	result := make(map[uint64]string, len(pages))
	for _, p := range pages {
		rev := wikiPageRevisions.GetLatestApproved(p.Id)
		if rev.Id != 0 {
			result[p.Id] = rev.Title
			continue
		}
		// 纯 pending（草稿）页面：取最新 pending 标题，保证导航树有标题可显示。
		if pending := wikiPageRevisions.GetLatestPending(p.Id); pending.Id != 0 {
			result[p.Id] = pending.Title
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
	allPages := wikiPages.ListAll()
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byNamespace[page.Namespace] = append(byNamespace[page.Namespace], page)
	}
	titles := pageTitles(allPages)
	approved := approvedPageSet(allPages)

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
				Active: approved[page.Id],
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

// approvedPageSet 返回存在至少一条 approved 修订的页面集合。
func approvedPageSet(pages []*wikiPages.Entity) map[uint64]bool {
	result := make(map[uint64]bool, len(pages))
	for _, p := range pages {
		rev := wikiPageRevisions.GetLatestApproved(p.Id)
		result[p.Id] = rev.Id != 0
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

// BuildAdminTree 构建管理端导航树（含 sortOrder；path 相对 namespace）。
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
				Path:      strings.TrimPrefix(page.Path, ns.Name+"/"),
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
	pages := wikiPages.ListAll()
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, p := range pages {
		byNamespace[p.Namespace] = append(byNamespace[p.Namespace], p)
	}
	summaries := make([]NamespaceSummary, 0, len(namespaces))
	for _, ns := range namespaces {
		nsPages := byNamespace[ns.Name]
		updated := ns.UpdatedAt
		count := int64(0)
		firstPath := ""
		for _, p := range nsPages {
			rev := wikiPageRevisions.GetLatestApproved(p.Id)
			if rev.Id != 0 {
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
	pages := wikiPages.ListAll()

	summaries := BuildNamespaceSummaries()
	// 最近更新：全部页面最新 approved 修订，按时间降序取 10。
	type recent struct {
		page *wikiPages.Entity
		rev  wikiPageRevisions.Entity
	}
	all := make([]recent, 0, len(pages))
	for _, p := range pages {
		rev := wikiPageRevisions.GetLatestApproved(p.Id)
		if rev.Id != 0 {
			all = append(all, recent{page: p, rev: rev})
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
	revisions := wikiPageRevisions.ListByPage(pageId)
	if len(revisions) == 0 {
		return []RevisionView{}
	}
	editorIDs := make([]uint64, 0, len(revisions))
	for _, r := range revisions {
		if r.Status != wikiPageRevisions.StatusApproved {
			continue
		}
		editorIDs = append(editorIDs, r.EditorId)
	}
	userMap := users.GetMapByIds(editorIDs)
	result := make([]RevisionView, 0, len(editorIDs))
	for _, r := range revisions {
		if r.Status != wikiPageRevisions.StatusApproved {
			continue
		}
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
	Id         uint64       `json:"id"`
	TopicId    uint64       `json:"topicId"`
	Namespace  string       `json:"namespace"`
	Path       string       `json:"path"`
	Title      string       `json:"title"`
	Content    string       `json:"content"`
	Toc        []TocItem    `json:"toc"`
	UpdatedAt  string       `json:"updatedAt"`
	EditorId   uint64       `json:"editorId"`
	EditorName string       `json:"editorName"`
	LikeCount  uint64       `json:"likeCount"`
	ViewCount  uint64       `json:"viewCount"`
	PostCount  uint64       `json:"postCount"`
	Liked      bool         `json:"liked"`
	Bookmarked bool         `json:"bookmarked"`
	Watched    bool         `json:"watched"`
	CanEdit    bool         `json:"canEdit"`
	CanReview  bool         `json:"canReview"`
	Pending    *PendingView `json:"pending"`
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
		Id:        page.Id,
		TopicId:   topic.Id,
		Namespace: page.Namespace,
		Path:      page.Path,
		Title:     rev.Title,
		Content:   rev.RenderedHTML,
		UpdatedAt: rev.CreatedAt.Format(time.RFC3339),
		EditorId:  rev.EditorId,
		LikeCount: topic.LikeCount,
		ViewCount: topic.ViewCount,
		PostCount: topic.PostCount,
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

// PendingView 待审中的编辑（编辑者/审核者可见）。
type PendingView struct {
	Title      string `json:"title"`
	Content    string `json:"content"`
	UpdatedAt  string `json:"updatedAt"`
	EditorId   uint64 `json:"editorId"`
	EditorName string `json:"editorName"`
}

// LoadPending 返回页面当前 pending 修订（无则 nil）。
func LoadPending(pageId uint64) *PendingView {
	rev := wikiPageRevisions.GetLatestPending(pageId)
	if rev.Id == 0 {
		return nil
	}
	editorName := ""
	if rev.EditorId != 0 {
		if u, ok := users.GetMapByIds([]uint64{rev.EditorId})[rev.EditorId]; ok && u != nil {
			editorName = u.Username
		}
	}
	return &PendingView{
		Title:      rev.Title,
		Content:    rev.Content,
		UpdatedAt:  rev.CreatedAt.Format(time.RFC3339),
		EditorId:   rev.EditorId,
		EditorName: editorName,
	}
}

// AdminRevision 审核队列条目（契约形状：updatedAt 为 RFC3339 字符串）。
type AdminRevision struct {
	RevisionId uint64 `json:"revisionId"`
	PageId     uint64 `json:"pageId"`
	Path       string `json:"path"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	EditorId   uint64 `json:"editorId"`
	EditorName string `json:"editorName"`
	UpdatedAt  string `json:"updatedAt"`
}

// ListAdminRevisions 分页返回指定状态的修订队列（附 path/编辑者）。
func ListAdminRevisions(status int8, page, pageSize int) []AdminRevision {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	revisions := wikiPageRevisions.ListByStatus(status, page, pageSize)
	if len(revisions) == 0 {
		return []AdminRevision{}
	}
	editorIDs := make([]uint64, 0, len(revisions))
	for _, r := range revisions {
		editorIDs = append(editorIDs, r.EditorId)
	}
	pageMap := make(map[uint64]*wikiPages.Entity)
	for _, p := range wikiPages.ListAll() {
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
			Path:       path,
			Title:      r.Title,
			Content:    r.Content,
			EditorId:   r.EditorId,
			EditorName: editorName,
			UpdatedAt:  r.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
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
