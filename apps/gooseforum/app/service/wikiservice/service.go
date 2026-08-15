package wikiservice

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topicUserAction"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/moderationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateParams 创建 wiki 页面的入参。
type CreateParams struct {
	Namespace string
	Path      string
	Title     string
	Content   string
	UserId    uint64
}

// CreateResult 创建结果。
type CreateResult struct {
	PageId uint64 `json:"pageId"`
	Path   string `json:"path"`
}

// EditParams 编辑 wiki 页面的入参。
type EditParams struct {
	PageID  uint64
	Title   string
	Content string
	UserId  uint64
	// BaseRevisionNo 编辑基线版本号（前端打开编辑器时的 published_revision_no），
	// 必填：0 直接拒绝 ErrBaseRevisionRequired；非 0 时后端 CAS 比对，过期返回
	// ErrConflict（model-1 编辑锁，客户端无法省略基线静默覆盖他人已发布版本）。
	BaseRevisionNo int
}

// EditResult 编辑结果（契约：status 为 approved 字符串 + 新版本号）。
type EditResult struct {
	RevisionId uint64 `json:"revisionId"`
	Status     string `json:"status"`
	RevisionNo int    `json:"revisionNo"`
}

// ActionResult 通用动作成功响应（契约 {ok:true}）。
type ActionResult struct {
	Ok bool `json:"ok"`
}

// CanManageNamespace 判断用户是否可管理某 namespace（贡献者 / PageManager / Admin）。
func CanManageNamespace(userId uint64, namespace string) bool {
	if userId == 0 {
		return false
	}
	if wikiNamespaceEditors.IsEditor(namespace, userId) {
		return true
	}
	return HasPageManagerPermission(userId)
}

// HasPageManagerPermission 判断用户是否拥有 PageManager（含 Admin）权限。
func HasPageManagerPermission(userId uint64) bool {
	if userId == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(userId)
	if !ok {
		return false
	}
	return permission.CheckRole(roleID, permission.PageManager)
}

// CanEditPage 判断用户是否可编辑某页面（创建者 / namespace 贡献者 / PageManager / Admin）。
func CanEditPage(userId uint64, page *wikiPages.Entity, topic *topics.Entity) bool {
	if userId == 0 {
		return false
	}
	if topic != nil && topic.UserId == userId {
		return true
	}
	return CanManageNamespace(userId, page.Namespace)
}

// Create 创建 wiki 页面：topic + 首楼 post + wiki_pages + revision#1(approved)。
// 创建直接发布（不走待审），触发 TopicPublishedEvent 复用既有 handler。
func Create(params CreateParams) (*CreateResult, error) {
	if params.UserId == 0 {
		return nil, ErrForbidden
	}
	path, ok := ValidatePath(params.Path)
	if !ok {
		return nil, ErrPathInvalid
	}
	// namespace 一律按小写归一（review：大写 Namespace 入参此前会因与
	// path 首段 / 库内小写 namespace 不一致而误报 ErrPathInvalid）。
	namespaceName := strings.ToLower(params.Namespace)
	if NamespaceOf(path) != namespaceName {
		return nil, ErrPathInvalid
	}
	namespace := wikiNamespaces.GetByName(namespaceName)
	if namespace.Id == 0 {
		return nil, ErrNamespaceNotFound
	}
	if !CanManageNamespace(params.UserId, namespaceName) {
		return nil, ErrForbidden
	}
	if wikiPages.PathExists(path, 0) {
		return nil, ErrPathExists
	}
	if len(params.Title) > 512 {
		return nil, ErrTitleTooLong
	}
	if strings.TrimSpace(params.Content) == "" {
		return nil, ErrContentEmpty
	}
	// frontmatter 解析与剥离（issue #258）：wiki 以 GitHub 仓库 Markdown 为唯一
	// 真源，页面可携带 YAML frontmatter 元数据；渲染/摘要/首图/TOC 统一走剥离后
	// 的 body，元数据不进入任何公开派生内容。修订原文保留完整 frontmatter
	//（编辑器往返与后续 content_hash 计算）。
	_, body, err := markdown2html.ParseFrontmatter(params.Content)
	if err != nil {
		return nil, ErrFrontmatterInvalid
	}
	// 仅含 frontmatter（无正文）视为空内容：剥离后没有可渲染/可搜索的正文。
	if strings.TrimSpace(body) == "" {
		return nil, ErrContentEmpty
	}

	// 嵌套路径（>=3 段）时校验父页面存在并记录 parent_id（review P2：
	// 此前嵌套路径直接创建会留下 parent_id=0，树层级断裂）。
	parentID := uint64(0)
	if segments := strings.Split(path, "/"); len(segments) > 2 {
		parentPath := strings.Join(segments[:len(segments)-1], "/")
		parent := wikiPages.GetByPath(parentPath)
		if parent.Id == 0 {
			return nil, ErrPageNotFound
		}
		parentID = parent.Id
	}

	topic := topics.Entity{
		UserId:           params.UserId,
		Title:            params.Title,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
		Excerpt:          markdown2html.ExtractDescription(body, 200),
		FirstImageURL:    markdown2html.ExtractFirstImageURL(body),
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}
	var firstPost posts.Entity
	var page wikiPages.Entity
	var revision wikiPageRevisions.Entity

	topic.WikiSyncedRevisionNo = 1
	firstPost.WikiSyncedRevisionNo = 1
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := topics.CreateTx(tx, &topic); err != nil {
			return err
		}
		firstPost = posts.Entity{
			TopicId:          topic.Id,
			PostNo:           1,
			UserId:           params.UserId,
			Content:          body,
			RenderedHTML:     "",
			RenderedVersion:  markdown2html.GetPostVersion(),
			ProcessStatus:    posts.ProcessStatusNormal,
			VisibilityStatus: posts.VisibilityActive,
			RetentionStatus:  posts.RetentionNormal,
		}
		if err := posts.CreateTx(tx, &firstPost); err != nil {
			return err
		}
		topic.FirstPostId = firstPost.Id
		topic.LastPostId = firstPost.Id
		topic.PostSeq = 1
		if err := topics.SaveTx(tx, &topic); err != nil {
			return err
		}
		page = wikiPages.Entity{
			TopicId:             topic.Id,
			Namespace:           namespaceName,
			Path:                path,
			ParentId:            parentID,
			PublishedRevisionNo: 1,
		}
		if err := wikiPages.CreateTx(tx, &page); err != nil {
			return err
		}
		renderedHTML := markdown2html.PostMarkdownToHTML(body)
		toc, err := encodeTOC(markdown2html.ExtractHeadings(body))
		if err != nil {
			return err
		}
		revision = wikiPageRevisions.Entity{
			PageId:       page.Id,
			RevisionNo:   1,
			Title:        params.Title,
			Content:      params.Content,
			RenderedHTML: renderedHTML,
			Toc:          toc,
			Status:       wikiPageRevisions.StatusApproved,
			EditorId:     params.UserId,
		}
		if err := wikiPageRevisions.CreateTx(tx, &revision); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 提交后副作用：文件引用 + 搜索索引 + 发布事件。
	fileusageservice.ReplaceTopic(topic.Id, params.UserId, body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki create: search index sync failed", "topicId", topic.Id, "error", err)
	}
	eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})

	return &CreateResult{PageId: page.Id, Path: path}, nil
}

// Edit 编辑 wiki 页面：写即发布（追加一条 approved 修订 + CAS 前移版本指针）。
// 模型 1（编辑锁）以乐观锁 CAS 实现：前端打开编辑器时记录 baseRevisionNo，
// 提交时后端执行
//
//	UPDATE wiki_pages SET published_revision_no = base+1
//	WHERE id = ? AND published_revision_no = base
//
// 影响 0 行 = 页面已被他人更新 → 409；影响 1 行 = 独占提交，同事务插入修订。
// 零内容丢失、无应用层锁、数据库原子。敏感词写时拦截，命中直接拒绝发布。
func Edit(params EditParams) (*EditResult, error) {
	if params.UserId == 0 {
		return nil, ErrForbidden
	}
	page := wikiPages.Get(params.PageID)
	if page.Id == 0 {
		return nil, ErrPageNotFound
	}
	topic := topics.Get(page.TopicId)
	if !CanEditPage(params.UserId, &page, &topic) {
		return nil, ErrForbidden
	}
	if len(params.Title) > 512 {
		return nil, ErrTitleTooLong
	}
	if strings.TrimSpace(params.Content) == "" {
		return nil, ErrContentEmpty
	}
	// frontmatter 解析与剥离（issue #258）：见 Create 注释；编辑同样只把剥离后
	// 的 body 写入物化视图/派生字段，修订原文保留完整 frontmatter。
	_, body, err := markdown2html.ParseFrontmatter(params.Content)
	if err != nil {
		return nil, ErrFrontmatterInvalid
	}
	// 仅含 frontmatter（无正文）视为空内容。
	if strings.TrimSpace(body) == "" {
		return nil, ErrContentEmpty
	}
	// CAS 基线必填：0 = 客户端省略 baseRevisionNo 绕过乐观锁（review Medium：
	// 基于陈旧基线提交会静默覆盖他人已发布的较新版本）。
	if params.BaseRevisionNo <= 0 {
		return nil, ErrBaseRevisionRequired
	}
	// 写时敏感词拦截：写即发布无审核兜底，命中直接拒绝（review 决策）。
	if hit, word := moderationservice.CheckContentAllowed(params.Title + "\n" + params.Content); hit {
		moderationservice.SensitiveContentBlocked(params.UserId, "wiki", params.PageID, word, markdown2html.ExtractDescription(body, 200))
		return nil, ErrSensitiveBlocked
	}

	var revision wikiPageRevisions.Entity
	var firstPost posts.Entity
	err = dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 行锁 page 行：同一页面并发编辑串行化（模型 1 编辑锁）。
		var locked wikiPages.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wiki_pages").Where("id = ?", page.Id).First(&locked).Error; err != nil {
			return err
		}
		// CAS：编辑基线版本号必须等于当前发布指针。
		if params.BaseRevisionNo != 0 && locked.PublishedRevisionNo != params.BaseRevisionNo {
			return ErrConflict
		}
		nextNo := locked.PublishedRevisionNo + 1
		renderedHTML := markdown2html.PostMarkdownToHTML(body)
		toc, err := encodeTOC(markdown2html.ExtractHeadings(body))
		if err != nil {
			return err
		}
		revision = wikiPageRevisions.Entity{
			PageId:       page.Id,
			RevisionNo:   nextNo,
			Title:        params.Title,
			Content:      params.Content,
			RenderedHTML: renderedHTML,
			Toc:          toc,
			Status:       wikiPageRevisions.StatusApproved, // 写即发布
			EditorId:     params.UserId,
		}
		if err := wikiPageRevisions.CreateTx(tx, &revision); err != nil {
			return err
		}
		// 版本指针前移（唯一跨表写点）。
		if err := tx.Table("wiki_pages").Where("id = ?", page.Id).
			Update("published_revision_no", nextNo).Error; err != nil {
			return err
		}
		// 物化视图同步（同一事务内，水印 = 新版本号）。
		firstPost = posts.GetTx(tx, topic.FirstPostId)
		if firstPost.Id == 0 {
			return ErrPageNotFound
		}
		firstPost.Content = body
		firstPost.RenderedHTML = renderedHTML
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		firstPost.ProcessStatus = posts.ProcessStatusNormal
		firstPost.WikiSyncedRevisionNo = nextNo
		if err := posts.UpdateWikiSyncedContentTx(tx, &firstPost); err != nil {
			return err
		}
		topic.Title = params.Title
		topic.Excerpt = markdown2html.ExtractDescription(body, 200)
		topic.FirstImageURL = markdown2html.ExtractFirstImageURL(body)
		topic.WikiSyncedRevisionNo = nextNo
		return topics.UpdateWikiSyncedMetaTx(tx, &topic)
	})
	if err != nil {
		return nil, err
	}

	// 提交后副作用：文件引用 + 搜索 + 通知（节流）。
	fileusageservice.ReplaceTopic(topic.Id, params.UserId, body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki edit: search index sync failed", "topicId", topic.Id, "error", err)
	}
	notifyWatchersThrottled(topic.Id, page.Path, topic.Title, params.UserId)
	return &EditResult{RevisionId: revision.Id, Status: StatusStringApproved, RevisionNo: revision.RevisionNo}, nil
}

// RollbackParams 回滚 wiki 页面的入参。
type RollbackParams struct {
	PageID uint64
	// ToRevisionNo 回滚目标版本号：该版本之后的修订全部永久删除（不可撤销），
	// 指针回到该版本，物化视图重同步。版本号自然无空洞（下次编辑 = N+1）。
	ToRevisionNo int
	UserId       uint64
}

// Rollback 管理员回滚 wiki 页面（唯一的管理员写路径）。
// 不可撤销：ToRevisionNo 之后的修订硬删（DELETE），版本历史截断。
func Rollback(params RollbackParams) error {
	if params.UserId == 0 || !HasPageManagerPermission(params.UserId) {
		return ErrForbidden
	}
	page := wikiPages.Get(params.PageID)
	if page.Id == 0 {
		return ErrPageNotFound
	}
	var target wikiPageRevisions.Entity
	var topic topics.Entity
	var firstPost posts.Entity
	body := ""
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 行锁 page 行：与 Edit 的 CAS/指针前移串行化（review High：回滚与并发
		// 编辑交错会丢失已提交的编辑，或让指针指向被硬删的版本）。
		var locked wikiPages.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wiki_pages").Where("id = ?", page.Id).First(&locked).Error; err != nil {
			return err
		}
		// 锁内重校验 target（review Medium：双回滚竞争下 target 可能已被另一
		// 管理员物理删除，或 ToRevisionNo 超过当前指针——旧代码在锁外读取
		// target，锁内硬删后可让指针指向已删修订、版本历史空洞）。
		if params.ToRevisionNo > locked.PublishedRevisionNo {
			return ErrRevisionNotFound
		}
		target = wikiPageRevisions.GetByPageAndRevisionNoTx(tx, page.Id, params.ToRevisionNo)
		if target.Id == 0 {
			return ErrRevisionNotFound
		}
		// 硬删目标之后全部修订（不可撤销，永久丢弃）：模型含 DeletedAt 后普通
		// Delete 会变成软删，回滚语义要求物理删除（Unscoped），否则页面恢复时
		// 被回滚的版本会复活。
		if err := tx.Table("wiki_page_revisions").Unscoped().
			Where("page_id = ?", page.Id).
			Where("revision_no > ?", params.ToRevisionNo).
			Delete(&wikiPageRevisions.Entity{}).Error; err != nil {
			return err
		}
		// 版本指针回退。
		if err := tx.Table("wiki_pages").Where("id = ?", page.Id).
			Update("published_revision_no", params.ToRevisionNo).Error; err != nil {
			return err
		}
		// 物化视图重同步为回滚目标版本。
		topic = topics.GetTx(tx, page.TopicId)
		if topic.Id == 0 {
			return ErrPageNotFound
		}
		firstPost = posts.GetTx(tx, topic.FirstPostId)
		if firstPost.Id == 0 {
			return ErrPageNotFound
		}
		// 回滚目标修订可能携带 frontmatter（issue #258）：物化视图与派生字段
		// 统一走剥离后的 body，与 Create/Edit 一致。回滚对象是历史已存储修订，
		// 其中可能含本特性落地前的旧内容（--- 开头但并非合法 frontmatter），
		// 因此用宽松剥离（块存在即剥离，解析失败不阻断回滚），避免管理员无法
		// 回滚到旧版本。
		_, body = markdown2html.SplitFrontmatter(target.Content)
		firstPost.Content = body
		firstPost.RenderedHTML = target.RenderedHTML
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		firstPost.ProcessStatus = posts.ProcessStatusNormal
		firstPost.WikiSyncedRevisionNo = params.ToRevisionNo
		if err := posts.UpdateWikiSyncedContentTx(tx, &firstPost); err != nil {
			return err
		}
		topic.Title = target.Title
		topic.Excerpt = markdown2html.ExtractDescription(body, 200)
		topic.FirstImageURL = markdown2html.ExtractFirstImageURL(body)
		topic.WikiSyncedRevisionNo = params.ToRevisionNo
		return topics.UpdateWikiSyncedMetaTx(tx, &topic)
	})
	if err != nil {
		return err
	}
	// 提交后副作用：文件引用 + 搜索 + 通知 watcher。
	fileusageservice.ReplaceTopic(topic.Id, target.EditorId, body)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki rollback: search index sync failed", "topicId", topic.Id, "error", err)
	}
	notifyWatchersThrottled(topic.Id, page.Path, topic.Title, params.UserId)
	return nil
}

// DiffRevision 返回两版本修订的 markdown 原文（管理端 diff 视图数据源）。
// 前端以 jsdiff 渲染 side-by-side / inline diff。
type DiffRevision struct {
	From *RevisionDiffSide `json:"from"`
	To   *RevisionDiffSide `json:"to"`
}

// RevisionDiffSide 单侧版本快照。
type RevisionDiffSide struct {
	RevisionNo int       `json:"revisionNo"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	EditorId   uint64    `json:"editorId"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Diff 返回页面两个版本的内容差异（from/to 均可为 0 = 空版本，用于对比创建前）。
func Diff(pageID uint64, fromNo, toNo int) (*DiffRevision, error) {
	page := wikiPages.Get(pageID)
	if page.Id == 0 {
		return nil, ErrPageNotFound
	}
	result := &DiffRevision{}
	if fromNo > 0 {
		from := wikiPageRevisions.GetByPageAndRevisionNo(pageID, fromNo)
		if from.Id == 0 {
			return nil, ErrRevisionNotFound
		}
		result.From = &RevisionDiffSide{
			RevisionNo: from.RevisionNo,
			Title:      from.Title,
			Content:    from.Content,
			EditorId:   from.EditorId,
			CreatedAt:  from.CreatedAt,
		}
	}
	if toNo > 0 {
		to := wikiPageRevisions.GetByPageAndRevisionNo(pageID, toNo)
		if to.Id == 0 {
			return nil, ErrRevisionNotFound
		}
		result.To = &RevisionDiffSide{
			RevisionNo: to.RevisionNo,
			Title:      to.Title,
			Content:    to.Content,
			EditorId:   to.EditorId,
			CreatedAt:  to.CreatedAt,
		}
	}
	return result, nil
}

// wikiNotifyThrottleWindow 同页面 wiki_updated 通知的节流窗口（review 决策：
// 写即发布后每次编辑都是发布，watcher 会收到大量通知；窗口内只发首条）。
const wikiNotifyThrottleWindow = 10 * time.Minute

// notifyWatchersThrottled 节流后通知 watcher：窗口内已有通知则跳过本次。
func notifyWatchersThrottled(topicId uint64, pagePath string, title string, editorId uint64) {
	latest := eventNotification.GetLatestByTopicAndType(topicId, eventNotification.EventTypeWikiUpdated)
	if latest.Id != 0 && time.Since(latest.CreatedAt) < wikiNotifyThrottleWindow {
		return
	}
	notifyWatchers(topicId, pagePath, title, editorId)
}

// notifyWatchers 给全部 watcher 发送 wiki_updated 通知。
func notifyWatchers(topicId uint64, pagePath string, title string, editorId uint64) {
	after := uint64(0)
	for {
		watchers := topicUserAction.ListActiveWatchUserIDsAfter(topicId, after, nil, 500)
		if len(watchers) == 0 {
			return
		}
		notifications := make([]*eventNotification.Entity, 0, len(watchers))
		for _, userId := range watchers {
			if userId == editorId {
				continue
			}
			notifications = append(notifications, &eventNotification.Entity{
				UserId:    userId,
				EventType: eventNotification.EventTypeWikiUpdated,
				TopicID:   topicId,
				Payload: eventNotification.NotificationPayload{
					Title:       title,
					Content:     title,
					TemplateKey: eventNotification.TemplateWikiUpdated,
					TemplateParams: eventNotification.NotificationTemplateParams{
						Preview: title,
					},
					ActorId:    editorId,
					TopicId:    topicId,
					TopicTitle: title,
					Extra: eventNotification.Extra{
						ProfileURL: "/wiki/" + pagePath,
					},
				},
			})
		}
		if len(notifications) > 0 {
			if err := eventNotification.CreateBatch(notifications, 100); err != nil {
				slog.Warn("wiki: notify watchers failed", "topicId", topicId, "error", err)
			} else {
				for _, userId := range watchers {
					if userId != editorId {
						unreadservice.Invalidate(userId)
					}
				}
			}
		}
		after = watchers[len(watchers)-1]
		if len(watchers) < 500 {
			return
		}
	}
}

func encodeTOC(items []markdown2html.HeadingItem) (string, error) {
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
