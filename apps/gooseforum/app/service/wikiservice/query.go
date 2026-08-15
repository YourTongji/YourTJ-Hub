package wikiservice

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

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
// GitHub SSOT 后内容/标题直接来自 wiki_pages 投影列（不再查修订表）。
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

	result := make([]TreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byNamespace[ns.Name]
		items := make([]TreePage, 0, len(pages))
		for _, page := range pages {
			items = append(items, TreePage{
				PageId: page.Id,
				Path:   page.Path,
				Title:  page.Title,
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
// 出现在公开导航树、首页与摘要。
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
		if t.Status != 1 || t.VisibilityStatus != topics.VisibilityActive ||
			t.ProcessStatus != topics.ProcessStatusNormal {
			continue
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// BuildTreeAPI 构建公开导航树（契约形状）：path 为 namespace 内相对路径。
func BuildTreeAPI() WikiTreeResult {
	return WikiTreeResult{Namespaces: buildTree("", true)}
}

// buildTree 构建导航树；relative=true 时 path 相对 namespace。
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
				Title:  page.Title,
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
	result := make([]AdminTreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byNamespace[ns.Name]
		items := make([]AdminTreePage, 0, len(pages))
		for _, page := range pages {
			items = append(items, AdminTreePage{
				PageId:    page.Id,
				Path:      page.Path,
				Title:     page.Title,
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

// BuildNamespaceSummaries 返回 namespace 摘要列表（页面数 + 最近更新时间）。
func BuildNamespaceSummaries() []NamespaceSummary {
	namespaces := wikiNamespaces.List()
	pages := filterPublicPages(wikiPages.ListAll())
	byNamespace := make(map[string][]*wikiPages.Entity)
	for _, p := range pages {
		byNamespace[p.Namespace] = append(byNamespace[p.Namespace], p)
	}
	summaries := make([]NamespaceSummary, 0, len(namespaces))
	for _, ns := range namespaces {
		nsPages := byNamespace[ns.Name]
		updated := ns.UpdatedAt
		firstPath := ""
		for _, p := range nsPages {
			if firstPath == "" {
				firstPath = p.Path
			}
			if p.UpdatedAt.After(updated) {
				updated = p.UpdatedAt
			}
		}
		summaries = append(summaries, NamespaceSummary{
			Name:          ns.Name,
			Description:   ns.Description,
			SortOrder:     ns.SortOrder,
			PageCount:     int64(len(nsPages)),
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

// BuildHome 组装 wiki 首页数据（最近更新 = 页面投影更新时间降序前 10）。
func BuildHome() HomeData {
	pages := filterPublicPages(wikiPages.ListAll())
	summaries := BuildNamespaceSummaries()

	// 最近更新：按页面投影 updated_at 降序取 10。
	all := make([]*wikiPages.Entity, 0, len(pages))
	all = append(all, pages...)
	for i := 1; i < len(all); i++ {
		for j := i; j > 0 && all[j].UpdatedAt.After(all[j-1].UpdatedAt); j-- {
			all[j], all[j-1] = all[j-1], all[j]
		}
	}
	if len(all) > 10 {
		all = all[:10]
	}

	recentPages := make([]RecentPage, 0, len(all))
	for _, item := range all {
		recentPages = append(recentPages, RecentPage{
			PageId:    item.Id,
			Path:      item.Path,
			Title:     item.Title,
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return HomeData{Namespaces: summaries, Recent: recentPages}
}

// Contributor 贡献者条目（GitHub SSOT：来源为仓库 git log 贡献者快照）。
type Contributor struct {
	UserId       uint64    `json:"userId"`
	Username     string    `json:"username"`
	AvatarUrl    string    `json:"avatarUrl"`
	Count        int       `json:"count"`
	LastEditedAt time.Time `json:"lastEditedAt"`
}

// BuildContributors 返回页面贡献者（读 wiki_pages.contributors_json 缓存；
// 由同步器从 git log 生成，GitHub 贡献者无论坛账号，userId/avatarUrl 为空）。
func BuildContributors(pageId uint64) []Contributor {
	page := wikiPages.Get(pageId)
	if page.Id == 0 || page.ContributorsJSON == "" {
		return []Contributor{}
	}
	var raw []gitContributor
	if err := json.Unmarshal([]byte(page.ContributorsJSON), &raw); err != nil {
		return []Contributor{}
	}
	lastEdited := time.Time{}
	if page.LastCommitAt != nil {
		lastEdited = *page.LastCommitAt
	}
	result := make([]Contributor, 0, len(raw))
	for _, c := range raw {
		result = append(result, Contributor{
			Username:     c.Name,
			Count:        c.Count,
			LastEditedAt: lastEdited,
		})
	}
	return result
}

// PageDetail 详情页数据（渲染 wiki_pages 投影快照）。
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
	// GitHub 外链（前端「编辑此页」/「历史」按钮；由 forum 控制器注入仓库配置）。
	EditUrl    string `json:"editUrl,omitempty"`
	HistoryUrl string `json:"historyUrl,omitempty"`
}

// TocItem 目录条目（渲染到前端）。
type TocItem struct {
	Level int    `json:"level"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// LoadPageDetail 加载页面详情（topic 可见性由调用方把关）。
func LoadPageDetail(page *wikiPages.Entity, topic *topics.Entity) (PageDetail, error) {
	detail := PageDetail{
		Id:                  page.Id,
		TopicId:             topic.Id,
		Namespace:           page.Namespace,
		Path:                page.Path,
		Title:               page.Title,
		Content:             page.RenderedHTML,
		UpdatedAt:           page.UpdatedAt.Format(time.RFC3339),
		LikeCount:           topic.LikeCount,
		ViewCount:           topic.ViewCount,
		PostCount:           topic.PostCount,
		PublishedRevisionNo: page.PublishedRevisionNo,
	}
	if page.Toc != "" {
		var items []TocItem
		if err := json.Unmarshal([]byte(page.Toc), &items); err == nil {
			detail.Toc = items
		}
	}
	return detail, nil
}

// DecodeTOC 解析 toc JSON 为条目（通用工具，供渲染/测试复用）。
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
