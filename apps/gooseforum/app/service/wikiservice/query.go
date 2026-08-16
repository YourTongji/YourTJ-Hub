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

const (
	// WikiTreeNodePage is a Markdown page from the content repository.
	WikiTreeNodePage = "page"
	// WikiTreeNodeDirectory is a non-clickable repository directory. Directories
	// do not require an index.md page to retain their navigation hierarchy.
	WikiTreeNodeDirectory = "directory"
)

// TreeNode is a recursive navigation node. Directory nodes have pageId 0.
type TreeNode struct {
	Kind     string     `json:"kind"`
	PageId   uint64     `json:"pageId"`
	Path     string     `json:"path"`
	Title    string     `json:"title"`
	Active   bool       `json:"active"`
	Children []TreeNode `json:"children"`
}

// TreeNamespace 导航树中的一个 namespace 分组。
// Name/Label = 显示名（中文目录名）；Slug = 有效 URL key（未分配 slug 时
// 降级=显示名），消费方拼 href 用（D7：URL 用 slug）。
type TreeNamespace struct {
	Name  string     `json:"name"`
	Label string     `json:"label"`
	Slug  string     `json:"slug"`
	Nodes []TreeNode `json:"nodes"`
}

// WikiTreeResult 公开导航树响应（契约包裹层）。
type WikiTreeResult struct {
	Namespaces []TreeNamespace `json:"namespaces"`
}

// ResolvePageByURLPath 按外部 URL path 解析页面（D7 路由语义：URL 用 slug）。
// 解析顺序：
//  1. 直查 path（slug 首段，新 URL）；
//  2. 首段 = 显示名时按 name→urlKey 重建（存量/降级 URL，如中文目录声明
//     slug 前发布的旧链接、或直接访问中文显示名 URL）。
//
// 返回零值实体表示未命中（404）。
func ResolvePageByURLPath(urlPath string) (entity wikiPages.Entity) {
	if urlPath == "" {
		return
	}
	entity = wikiPages.GetByPath(urlPath)
	if entity.Id != 0 {
		return entity
	}
	// 回退：首段按显示名解析 → 重建为 URL key 路径再直查。
	first := NamespaceOf(urlPath)
	if first == "" {
		return
	}
	ns := wikiNamespaces.GetByName(first)
	if ns.Id == 0 {
		return
	}
	rebuilt := namespaceURLKey(&ns) + strings.TrimPrefix(urlPath, first)
	return wikiPages.GetByPath(rebuilt)
}

// BuildTree 构建 wiki 导航树（按 namespace 分组，当前页 active）。
// GitHub SSOT 后内容/标题直接来自 wiki_pages 投影列（不再查修订表）。
// D7 URL key 语义：page.Namespace 列 = URL key（slug，降级=显示名），
// 分组按 URL key；输出 Name/Label 用显示名（中文目录名）。
func BuildTree(activePath string) []TreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []TreeNamespace{}
	}
	allPages := filterPublicPages(wikiPages.ListAll())
	byURLKey := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byURLKey[page.Namespace] = append(byURLKey[page.Namespace], page)
	}

	result := make([]TreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byURLKey[namespaceURLKey(ns)]
		result = append(result, TreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Slug:  namespaceURLKey(ns),
			Nodes: buildTreeNodes(pages, namespaceURLKey(ns), activePath, false),
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

// buildTree 构建导航树；relative=true 时 path 相对 namespace（URL key 前缀）。
func buildTree(activePath string, contractShape bool) []TreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []TreeNamespace{}
	}
	allPages := filterPublicPages(wikiPages.ListAll())
	byURLKey := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byURLKey[page.Namespace] = append(byURLKey[page.Namespace], page)
	}

	result := make([]TreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		urlKey := namespaceURLKey(ns)
		pages := byURLKey[urlKey]
		result = append(result, TreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Slug:  urlKey,
			Nodes: buildTreeNodes(pages, urlKey, activePath, contractShape),
		})
	}
	return result
}

type treeNodeBuilder struct {
	node      TreeNode
	sortOrder int
	children  map[string]*treeNodeBuilder
}

func newDirectoryNode(path, title string, sortOrder int) *treeNodeBuilder {
	return &treeNodeBuilder{
		node:      TreeNode{Kind: WikiTreeNodeDirectory, Path: path, Title: title, Children: []TreeNode{}},
		sortOrder: sortOrder,
		children:  make(map[string]*treeNodeBuilder),
	}
}

// buildTreeNodes projects directory segments from paths. It deliberately does
// not depend on parent_id: a valid repository directory may have no index.md.
func buildTreeNodes(pages []*wikiPages.Entity, urlKey, activePath string, relative bool) []TreeNode {
	root := newDirectoryNode("", "", 0)
	directoryOrder := make(map[string]int)
	for _, page := range pages {
		rel := strings.TrimPrefix(page.Path, urlKey+"/")
		parts := strings.Split(rel, "/")
		for i := 1; i < len(parts); i++ {
			dirPath := strings.Join(parts[:i], "/")
			if current, ok := directoryOrder[dirPath]; !ok || page.SortOrder < current {
				directoryOrder[dirPath] = page.SortOrder
			}
		}
		if len(parts) > 1 && parts[len(parts)-1] == "index" {
			directoryOrder[strings.Join(parts[:len(parts)-1], "/")] = page.SortOrder
		}
	}
	for _, page := range pages {
		rel := strings.TrimPrefix(page.Path, urlKey+"/")
		parts := strings.Split(rel, "/")
		cursor := root
		for i := 0; i < len(parts)-1; i++ {
			dirPath := strings.Join(parts[:i+1], "/")
			child, ok := cursor.children[dirPath]
			if !ok {
				outputPath := dirPath
				if !relative {
					outputPath = urlKey + "/" + dirPath
				}
				child = newDirectoryNode(outputPath, parts[i], directoryOrder[dirPath])
				cursor.children[dirPath] = child
			}
			cursor = child
		}
		path := page.Path
		if relative {
			path = rel
		}
		cursor.children["page:"+rel] = &treeNodeBuilder{
			node: TreeNode{Kind: WikiTreeNodePage, PageId: page.Id, Path: path, Title: page.Title,
				Active: page.Path == activePath, Children: []TreeNode{}},
			sortOrder: page.SortOrder,
			children:  make(map[string]*treeNodeBuilder),
		}
	}
	return flattenTreeChildren(root)
}

func flattenTreeChildren(parent *treeNodeBuilder) []TreeNode {
	items := make([]*treeNodeBuilder, 0, len(parent.children))
	for _, child := range parent.children {
		items = append(items, child)
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && (items[j].sortOrder < items[j-1].sortOrder ||
			(items[j].sortOrder == items[j-1].sortOrder && items[j].node.Path < items[j-1].node.Path)); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	result := make([]TreeNode, 0, len(items))
	for _, item := range items {
		item.node.Children = flattenTreeChildren(item)
		result = append(result, item.node)
	}
	return result
}

// AdminTreeNode 管理端导航树节点。directory 节点没有 GitHub 文件，pageId
// 为零且 sourcePath 为空；page 节点保留 GitHub 外链所需的 sourcePath。
// Path 首段 = URL key（slug，降级=显示名）；SourcePath = 仓库真实路径
// （GitHub 编辑/历史外链拼接用，与 URL 解耦，D7）。
type AdminTreeNode struct {
	Kind       string          `json:"kind"`
	PageId     uint64          `json:"pageId"`
	Path       string          `json:"path"`
	SourcePath string          `json:"sourcePath"`
	Title      string          `json:"title"`
	SortOrder  int             `json:"sortOrder"`
	Children   []AdminTreeNode `json:"children"`
}

// AdminTreeNamespace 管理端导航树中的一个 namespace 分组。
type AdminTreeNamespace struct {
	Name  string          `json:"name"`
	Label string          `json:"label"`
	Nodes []AdminTreeNode `json:"nodes"`
}

// BuildAdminTree 构建管理端导航树（含 sortOrder/sourcePath；path 为完整路径，含 URL key 段）。
func BuildAdminTree() []AdminTreeNamespace {
	namespaces := wikiNamespaces.List()
	if len(namespaces) == 0 {
		return []AdminTreeNamespace{}
	}
	allPages := wikiPages.ListAll()
	byURLKey := make(map[string][]*wikiPages.Entity)
	for _, page := range allPages {
		byURLKey[page.Namespace] = append(byURLKey[page.Namespace], page)
	}
	result := make([]AdminTreeNamespace, 0, len(namespaces))
	for _, ns := range namespaces {
		pages := byURLKey[namespaceURLKey(ns)]
		result = append(result, AdminTreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Nodes: buildAdminTreeNodes(pages, namespaceURLKey(ns)),
		})
	}
	return result
}

type adminTreeNodeBuilder struct {
	node     AdminTreeNode
	children map[string]*adminTreeNodeBuilder
}

func buildAdminTreeNodes(pages []*wikiPages.Entity, urlKey string) []AdminTreeNode {
	root := &adminTreeNodeBuilder{children: make(map[string]*adminTreeNodeBuilder)}
	directoryOrder := make(map[string]int)
	for _, page := range pages {
		rel := strings.TrimPrefix(page.Path, urlKey+"/")
		parts := strings.Split(rel, "/")
		for i := 1; i < len(parts); i++ {
			dirPath := strings.Join(parts[:i], "/")
			if current, ok := directoryOrder[dirPath]; !ok || page.SortOrder < current {
				directoryOrder[dirPath] = page.SortOrder
			}
		}
		if len(parts) > 1 && parts[len(parts)-1] == "index" {
			directoryOrder[strings.Join(parts[:len(parts)-1], "/")] = page.SortOrder
		}
	}
	for _, page := range pages {
		rel := strings.TrimPrefix(page.Path, urlKey+"/")
		parts := strings.Split(rel, "/")
		cursor := root
		for i := 0; i < len(parts)-1; i++ {
			dirRel := strings.Join(parts[:i+1], "/")
			key := "dir:" + dirRel
			child, ok := cursor.children[key]
			if !ok {
				child = &adminTreeNodeBuilder{node: AdminTreeNode{
					Kind: WikiTreeNodeDirectory, Path: urlKey + "/" + dirRel, Title: parts[i],
					SortOrder: directoryOrder[dirRel], Children: []AdminTreeNode{},
				}, children: make(map[string]*adminTreeNodeBuilder)}
				cursor.children[key] = child
			}
			cursor = child
		}
		cursor.children["page:"+rel] = &adminTreeNodeBuilder{node: AdminTreeNode{
			Kind: WikiTreeNodePage, PageId: page.Id, Path: page.Path, SourcePath: page.SourcePath,
			Title: page.Title, SortOrder: page.SortOrder, Children: []AdminTreeNode{},
		}, children: make(map[string]*adminTreeNodeBuilder)}
	}
	return flattenAdminTreeChildren(root)
}

func flattenAdminTreeChildren(parent *adminTreeNodeBuilder) []AdminTreeNode {
	items := make([]*adminTreeNodeBuilder, 0, len(parent.children))
	for _, child := range parent.children {
		items = append(items, child)
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && (items[j].node.SortOrder < items[j-1].node.SortOrder ||
			(items[j].node.SortOrder == items[j-1].node.SortOrder && items[j].node.Path < items[j-1].node.Path)); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	result := make([]AdminTreeNode, 0, len(items))
	for _, item := range items {
		item.node.Children = flattenAdminTreeChildren(item)
		result = append(result, item.node)
	}
	return result
}

// NamespaceSummary 首页 namespace 卡。
type NamespaceSummary struct {
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Description   string    `json:"description"`
	SortOrder     int       `json:"sortOrder"`
	PageCount     int64     `json:"pageCount"`
	UpdatedAt     time.Time `json:"updatedAt"`
	FirstPagePath string    `json:"firstPagePath"`
}

// BuildNamespaceSummaries 返回 namespace 摘要列表（页面数 + 最近更新时间）。
// 分组按 URL key（page.Namespace），输出显示名/URL key 分离（D7）。
func BuildNamespaceSummaries() []NamespaceSummary {
	namespaces := wikiNamespaces.List()
	pages := filterPublicPages(wikiPages.ListAll())
	byURLKey := make(map[string][]*wikiPages.Entity)
	for _, p := range pages {
		byURLKey[p.Namespace] = append(byURLKey[p.Namespace], p)
	}
	summaries := make([]NamespaceSummary, 0, len(namespaces))
	for _, ns := range namespaces {
		nsPages := byURLKey[namespaceURLKey(ns)]
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
			Slug:          ns.SlugOrEmpty(),
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
// GitHub SSOT：无论坛编辑者概念（git 作者信息走 contributors），
// 不再输出 editorId/editorName（历史遗留字段，恒为零值，issue #291）。
type RecentPage struct {
	PageId    uint64 `json:"pageId"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updatedAt"`
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
// D7：Namespace 字段输出显示名（中文目录名），从 wiki_namespaces.name 反查；
// 反查失败（数据异常）时降级输出 URL key。
func LoadPageDetail(page *wikiPages.Entity, topic *topics.Entity) (PageDetail, error) {
	displayName := page.Namespace
	if ns := wikiNamespaces.GetBySlug(page.Namespace); ns.Id != 0 {
		displayName = ns.Name
	}
	detail := PageDetail{
		Id:                  page.Id,
		TopicId:             topic.Id,
		Namespace:           displayName,
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
