package wikiservice

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
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
}

// EditResult 编辑结果（契约：status 为 pending 字符串）。
type EditResult struct {
	RevisionId uint64 `json:"revisionId"`
	Status     string `json:"status"`
}

// ActionResult 通用动作成功响应（契约 {ok:true}）。
type ActionResult struct {
	Ok bool `json:"ok"`
}

// ReviewResult 审核结果（契约：revisionId + 状态字符串）。
type ReviewResult struct {
	RevisionId uint64 `json:"revisionId"`
	Status     string `json:"status"`
}

// ReviewParams 审核 wiki 修订的入参。
type ReviewParams struct {
	RevisionID uint64
	Action     string // "approve" | "reject"
	UserId     uint64
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
	if NamespaceOf(path) != params.Namespace {
		return nil, ErrPathInvalid
	}
	namespace := wikiNamespaces.GetByName(params.Namespace)
	if namespace.Id == 0 {
		return nil, ErrNamespaceNotFound
	}
	if !CanManageNamespace(params.UserId, params.Namespace) {
		return nil, ErrForbidden
	}
	if wikiPages.PathExists(path, 0) {
		return nil, ErrPathExists
	}
	if len(params.Title) > 512 {
		return nil, ErrPathInvalid
	}

	topic := topics.Entity{
		UserId:           params.UserId,
		Title:            params.Title,
		Status:           1,
		ProcessStatus:    topics.ProcessStatusNormal,
		TopicType:        topics.TopicTypeWiki,
		Excerpt:          markdown2html.ExtractDescription(params.Content, 200),
		FirstImageURL:    markdown2html.ExtractFirstImageURL(params.Content),
		VisibilityStatus: topics.VisibilityActive,
		RetentionStatus:  topics.RetentionNormal,
	}
	var firstPost posts.Entity
	var page wikiPages.Entity
	var revision wikiPageRevisions.Entity

	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		if err := topics.CreateTx(tx, &topic); err != nil {
			return err
		}
		firstPost = posts.Entity{
			TopicId:          topic.Id,
			PostNo:           1,
			UserId:           params.UserId,
			Content:          params.Content,
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
			TopicId:   topic.Id,
			Namespace: params.Namespace,
			Path:      path,
		}
		if err := wikiPages.CreateTx(tx, &page); err != nil {
			return err
		}
		renderedHTML := markdown2html.PostMarkdownToHTML(params.Content)
		toc, err := encodeTOC(markdown2html.ExtractHeadings(params.Content))
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
	fileusageservice.ReplaceTopic(topic.Id, params.UserId, params.Content)
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki create: search index sync failed", "topicId", topic.Id, "error", err)
	}
	eventbus.Publish(context.Background(), &eventhandlers.TopicPublishedEvent{Topic: &topic, FirstPost: &firstPost})

	return &CreateResult{PageId: page.Id, Path: path}, nil
}

// Edit 编辑 wiki 页面：生成 pending 修订，置 post#1 为待审（topic 行不动）。
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
		return nil, ErrPathInvalid
	}

	var revision wikiPageRevisions.Entity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 行锁保证 revision_no 单调 + 至多一条 pending 不变式。
		var locked wikiPages.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wiki_pages").Where("id = ?", page.Id).First(&locked).Error; err != nil {
			return err
		}
		// 计算下一个 revision_no。
		revisions := wikiPageRevisions.ListByPageTx(tx, page.Id)
		nextNo := 1
		if len(revisions) > 0 && revisions[0].RevisionNo > 0 {
			nextNo = revisions[0].RevisionNo + 1
		}
		// 旧的 pending 置 superseded。
		if err := wikiPageRevisions.SupersedePendingTx(tx, page.Id); err != nil {
			return err
		}
		revision = wikiPageRevisions.Entity{
			PageId:     page.Id,
			RevisionNo: nextNo,
			Title:      params.Title,
			Content:    params.Content,
			Status:     wikiPageRevisions.StatusPending,
			EditorId:   params.UserId,
		}
		if err := wikiPageRevisions.CreateTx(tx, &revision); err != nil {
			return err
		}
		// 同步首楼 post 为待审内容（topic 行 ProcessStatus 不动，评论保持可用）。
		firstPost := posts.GetTx(tx, topic.FirstPostId)
		if firstPost.Id == 0 {
			return ErrPageNotFound
		}
		firstPost.Content = params.Content
		firstPost.RenderedHTML = ""
		firstPost.RenderedVersion = markdown2html.GetPostVersion()
		firstPost.ProcessStatus = posts.ProcessStatusPending
		return posts.SaveTx(tx, &firstPost)
	})
	if err != nil {
		return nil, err
	}

	// 提交后：搜索文档删除（firstPost pending → isIndexable=false）。
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &posts.Entity{
		TopicId:       topic.Id,
		PostNo:        1,
		Content:       params.Content,
		ProcessStatus: posts.ProcessStatusPending,
	}); err != nil {
		slog.Warn("wiki edit: search index sync failed", "topicId", topic.Id, "error", err)
	}
	return &EditResult{RevisionId: revision.Id, Status: StatusStringPending}, nil
}

// Review 审核修订：approve 发布（渲染快照 + 通知 watcher）；reject 回滚上一 approved。
func Review(params ReviewParams) (*ReviewResult, error) {
	if params.UserId == 0 || !HasPageManagerPermission(params.UserId) {
		return nil, ErrForbidden
	}

	var page wikiPages.Entity
	var topic topics.Entity
	var firstPost posts.Entity
	action := params.Action

	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		// 行锁 revision 行：仅 pending 可审，重复审核 409。
		var revision wikiPageRevisions.Entity
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Table("wiki_page_revisions").Where("id = ?", params.RevisionID).First(&revision).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRevisionNotFound
			}
			return err
		}
		if revision.Status != wikiPageRevisions.StatusPending {
			return ErrRevisionNotPending
		}
		page = wikiPages.GetTx(tx, revision.PageId)
		if page.Id == 0 {
			return ErrPageNotFound
		}
		topic = topics.GetTx(tx, page.TopicId)
		firstPost = posts.GetTx(tx, topic.FirstPostId)

		switch action {
		case "approve":
			renderedHTML := markdown2html.PostMarkdownToHTML(revision.Content)
			toc, err := encodeTOC(markdown2html.ExtractHeadings(revision.Content))
			if err != nil {
				return err
			}
			if err := tx.Table("wiki_page_revisions").Where("id = ?", revision.Id).Updates(map[string]any{
				"status":        wikiPageRevisions.StatusApproved,
				"rendered_html": renderedHTML,
				"toc":           toc,
				"reviewed_by":   params.UserId,
				"reviewed_at":   time.Now(),
			}).Error; err != nil {
				return err
			}
			// 首楼 post 同步为已发布内容 + topic 标题同步。
			firstPost.Content = revision.Content
			firstPost.RenderedHTML = renderedHTML
			firstPost.RenderedVersion = markdown2html.GetPostVersion()
			firstPost.ProcessStatus = posts.ProcessStatusNormal
			if err := posts.SaveTx(tx, &firstPost); err != nil {
				return err
			}
			topic.Title = revision.Title
			return topics.SaveTx(tx, &topic)
		case "reject":
			if err := tx.Table("wiki_page_revisions").Where("id = ?", revision.Id).Updates(map[string]any{
				"status":      wikiPageRevisions.StatusRejected,
				"reviewed_by": params.UserId,
				"reviewed_at": time.Now(),
			}).Error; err != nil {
				return err
			}
			// 回滚为上一 approved 内容。
			prev := wikiPageRevisions.GetLatestApprovedTx(tx, page.Id)
			content := ""
			title := topic.Title
			if prev.Id != 0 {
				content = prev.Content
				title = prev.Title
			}
			firstPost.Content = content
			firstPost.RenderedHTML = ""
			firstPost.RenderedVersion = markdown2html.GetPostVersion()
			firstPost.ProcessStatus = posts.ProcessStatusNormal
			if err := posts.SaveTx(tx, &firstPost); err != nil {
				return err
			}
			topic.Title = title
			return topics.SaveTx(tx, &topic)
		default:
			return ErrPathInvalid
		}
	})
	if err != nil {
		return nil, err
	}

	// 提交后副作用。
	if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
		slog.Warn("wiki review: search index sync failed", "topicId", topic.Id, "error", err)
	}
	status := StatusStringRejected
	if action == "approve" {
		status = StatusStringApproved
		notifyWatchers(topic.Id, page.Path, topic.Title, firstPost.UserId)
	}
	return &ReviewResult{RevisionId: params.RevisionID, Status: status}, nil
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
				slog.Warn("wiki review: notify watchers failed", "topicId", topicId, "error", err)
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
