package wikiservice

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
)

// TreePage 导航树中的一个节点（页面或目录）。
// 目录节点：PageId=0（无对应页面行），Path=目录路径，Title=目录名，Active=false，
// 仅作分组（可折叠）；页面节点：PageId>0，Active 仅精确匹配当前页。
// `<目录>/index` 页面（叶子，且无 `<目录>.md` 页面时）提升为目录节点本身：
// PageId/Title/排序/active 取自该页（URL 为目录路径，路由解析到 index 页）；
// `<目录>.md` 页面同时存在时优先作为目录代表页（可点击，带子级）。
// Children 缺省表示叶子节点。层级语义与仓库目录层级一致（issue #289），
// 同级按 frontmatter order 排序（目录节点取子级最小 order，目录名决胜）。
type TreePage struct {
	PageId   uint64      `json:"pageId"`
	Path     string      `json:"path"`
	Title    string      `json:"title"`
	Active   bool        `json:"active"`
	Children []*TreePage `json:"children,omitempty"`
}

// TreeNamespace 导航树中的一个 namespace 分组。
// Name/Label = 显示名（中文目录名）；Slug = 有效 URL key（未分配 slug 时
// 降级=显示名），消费方拼 href 用（D7：URL 用 slug）。
type TreeNamespace struct {
	Name  string     `json:"name"`
	Label string     `json:"label"`
	Slug  string     `json:"slug"`
	Pages []TreePage `json:"pages"`
}

// WikiTreeResult 公开导航树响应（契约包裹层）。
type WikiTreeResult struct {
	Namespaces []TreeNamespace `json:"namespaces"`
}

// ResolvePageByURLPath 按外部 URL path 解析页面（D7 路由语义：URL 用 slug）。
// 解析顺序：
//  1. 直查 path（slug 首段，新 URL）；
//  2. 目录路径 → `<目录>/index` 页面（导航树目录节点提升语义，issue #289）；
//  3. 首段 = 显示名时按 name→urlKey 重建（存量/降级 URL，如中文目录声明
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
	// 目录路径 → index 页：导航树把 `<目录>/index` 提升为目录节点（URL 为目录
	// 路径），路由解析回 index 页（issue #289）。
	if entity = wikiPages.GetByPath(urlPath + "/index"); entity.Id != 0 {
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

// BuildTree 构建 wiki 导航树（按 namespace 分组，当前页 active，完整路径）。
// GitHub SSOT 后内容/标题直接来自 wiki_pages 投影列（不再查修订表）。
// D7 URL key 语义：page.Namespace 列 = URL key（slug，降级=显示名），
// 分组按 URL key；输出 Name/Label 用显示名（中文目录名）。
// 目录层级 = 仓库目录层级（issue #289）：子目录为嵌套节点，`<目录>/index`
// 页面提升为目录节点本身（可点击），同级按 order 排序。
func BuildTree(activePath string) []TreeNamespace {
	return assembleTree(activePath, false)
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
	return WikiTreeResult{Namespaces: assembleTree("", true)}
}

// assembleTree 组装导航树；contractShape=true 时 path 为 namespace 内相对路径
// （公开 API 契约形状），false 时 path 为完整路径（SSR 左栏）。
func assembleTree(activePath string, contractShape bool) []TreeNamespace {
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
		raw := buildRawTree(byURLKey[urlKey], urlKey)
		items := make([]*TreePage, 0, len(raw))
		for _, rt := range raw {
			items = append(items, rawToTreePage(rt, activePath))
		}
		if !contractShape {
			prefixTreePaths(items, urlKey)
		}
		pages := make([]TreePage, 0, len(items))
		for _, item := range items {
			pages = append(pages, *item)
		}
		result = append(result, TreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Slug:  urlKey,
			Pages: pages,
		})
	}
	return result
}

// ---------- 层级构建（issue #289：目录层级 = 仓库路径层级） ----------

// treeTrie 页面路径 trie：按相对路径段（去 namespace 前缀）组织页面。
type treeTrie struct {
	children map[string]*treeTrie
	page     *wikiPages.Entity // 完整路径恰好等于本节点路径的页面（如 <dir>.md）
}

func newTreeTrie() *treeTrie {
	return &treeTrie{children: map[string]*treeTrie{}}
}

func (t *treeTrie) insert(relPath string, page *wikiPages.Entity) {
	node := t
	for _, seg := range strings.Split(relPath, "/") {
		next, ok := node.children[seg]
		if !ok {
			next = newTreeTrie()
			node.children[seg] = next
		}
		node = next
	}
	node.page = page
}

// rawTreeNode 层级构建的中间表示。
// page：目录代表页（`<dir>.md` 页面优先，否则 `<dir>/index` 页面提升）；nil = 纯目录节点。
// dirPath：相对路径；sortKey：排序键（代表页 order，纯目录取子级最小 order）。
type rawTreeNode struct {
	page     *wikiPages.Entity
	dirPath  string
	children []*rawTreeNode
	sortKey  int
}

// buildRawTree 把某 namespace 的页面列表构建为嵌套层级（相对路径）。
func buildRawTree(pages []*wikiPages.Entity, urlKey string) []*rawTreeNode {
	trie := newTreeTrie()
	for _, p := range pages {
		trie.insert(strings.TrimPrefix(p.Path, urlKey+"/"), p)
	}
	var out []*rawTreeNode
	for name, child := range trie.children {
		if rt := rawNodeOf(child, name, urlKey, true); rt != nil {
			out = append(out, rt)
		}
	}
	sortRawNodes(out)
	return out
}

// rawNodeOf 把一个 trie 节点转为 rawTreeNode（递归子节点 + index 提升）。
// promoteIndex=true 时，若本目录存在叶子 `index` 页面且无 `dir.md` 页面，
// 该 index 页面提升为本目录代表页，不再单列子项。
func rawNodeOf(node *treeTrie, dirPath, urlKey string, promoteIndex bool) *rawTreeNode {
	rep := node.page
	absorbedIndex := false
	if rep == nil && promoteIndex {
		if idx, ok := node.children["index"]; ok && idx.page != nil && len(idx.children) == 0 {
			rep = idx.page
			absorbedIndex = true
		}
	}
	var kids []*rawTreeNode
	for name, child := range node.children {
		if absorbedIndex && name == "index" {
			continue
		}
		childPath := name
		if dirPath != "" {
			childPath = dirPath + "/" + name
		}
		if rt := rawNodeOf(child, childPath, urlKey, true); rt != nil {
			kids = append(kids, rt)
		}
	}
	if rep == nil && len(kids) == 0 {
		return nil
	}
	sortRawNodes(kids)
	rt := &rawTreeNode{children: kids}
	if rep != nil {
		rt.page = rep
		if absorbedIndex {
			// index 页面提升为目录节点：路径用目录路径，标题/排序/可点击性取自页面。
			rt.dirPath = dirPath
		} else {
			rt.dirPath = strings.TrimPrefix(rep.Path, urlKey+"/")
		}
		rt.sortKey = rep.SortOrder
	} else {
		rt.dirPath = dirPath
		rt.sortKey = minRawSortKey(kids)
	}
	return rt
}

func sortRawNodes(nodes []*rawTreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].sortKey != nodes[j].sortKey {
			return nodes[i].sortKey < nodes[j].sortKey
		}
		return nodes[i].dirPath < nodes[j].dirPath
	})
}

func minRawSortKey(nodes []*rawTreeNode) int {
	if len(nodes) == 0 {
		return 0
	}
	min := nodes[0].sortKey
	for _, n := range nodes[1:] {
		if n.sortKey < min {
			min = n.sortKey
		}
	}
	return min
}

func lastSegment(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}

// rawToTreePage 公开导航树节点转换（相对路径；Active 仅精确匹配当前页）。
func rawToTreePage(rt *rawTreeNode, activePath string) *TreePage {
	tp := &TreePage{
		PageId: pageIDOf(rt),
		Path:   rt.dirPath,
		Title:  titleOf(rt),
		Active: rt.page != nil && rt.page.Path == activePath,
	}
	for _, c := range rt.children {
		tp.Children = append(tp.Children, rawToTreePage(c, activePath))
	}
	if len(tp.Children) == 0 {
		tp.Children = nil
	}
	return tp
}

func pageIDOf(rt *rawTreeNode) uint64 {
	if rt.page != nil {
		return rt.page.Id
	}
	return 0
}

func titleOf(rt *rawTreeNode) string {
	if rt.page != nil {
		return rt.page.Title
	}
	return lastSegment(rt.dirPath)
}

// prefixTreePaths 给相对路径补上 namespace URL key 前缀（SSR/管理端完整路径）。
func prefixTreePaths(items []*TreePage, prefix string) {
	for _, item := range items {
		item.Path = prefix + "/" + item.Path
		prefixTreePaths(item.Children, prefix)
	}
}

// AdminTreePage 管理端导航树中的一个节点（页面或目录）。
// Path 首段 = URL key（slug，降级=显示名）；SourcePath = 仓库真实路径
// （GitHub 编辑/历史外链拼接用，与 URL 解耦，D7）。
// 目录节点：PageId=0，Title=目录名，SortOrder=子级最小 order；`<目录>/index`
// 页面提升为目录节点（与公开树一致的层级语义，issue #289）。
type AdminTreePage struct {
	PageId     uint64           `json:"pageId"`
	Path       string           `json:"path"`
	SourcePath string           `json:"sourcePath"`
	Title      string           `json:"title"`
	SortOrder  int              `json:"sortOrder"`
	Children   []*AdminTreePage `json:"children,omitempty"`
}

// AdminTreeNamespace 管理端导航树中的一个 namespace 分组。
type AdminTreeNamespace struct {
	Name  string          `json:"name"`
	Label string          `json:"label"`
	Pages []AdminTreePage `json:"pages"`
}

// BuildAdminTree 构建管理端导航树（含 sortOrder/sourcePath；path 为完整路径，含 URL key 段）。
// 目录层级 = 仓库目录层级（issue #289）。
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
		urlKey := namespaceURLKey(ns)
		raw := buildRawTree(byURLKey[urlKey], urlKey)
		items := make([]*AdminTreePage, 0, len(raw))
		for _, rt := range raw {
			items = append(items, rawToAdminTreePage(rt))
		}
		prefixAdminTreePaths(items, urlKey)
		pages := make([]AdminTreePage, 0, len(items))
		for _, item := range items {
			pages = append(pages, *item)
		}
		result = append(result, AdminTreeNamespace{
			Name:  ns.Name,
			Label: ns.Name,
			Pages: pages,
		})
	}
	return result
}

// rawToAdminTreePage 管理端树节点转换（相对路径；PageId=0 表示纯目录节点）。
func rawToAdminTreePage(rt *rawTreeNode) *AdminTreePage {
	atp := &AdminTreePage{
		PageId:    pageIDOf(rt),
		Path:      rt.dirPath,
		Title:     titleOf(rt),
		SortOrder: rt.sortKey,
	}
	if rt.page != nil {
		atp.SourcePath = rt.page.SourcePath
	}
	for _, c := range rt.children {
		atp.Children = append(atp.Children, rawToAdminTreePage(c))
	}
	if len(atp.Children) == 0 {
		atp.Children = nil
	}
	return atp
}

// prefixAdminTreePaths 给管理端树相对路径补上 namespace URL key 前缀。
func prefixAdminTreePaths(items []*AdminTreePage, prefix string) {
	for _, item := range items {
		item.Path = prefix + "/" + item.Path
		prefixAdminTreePaths(item.Children, prefix)
	}
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
