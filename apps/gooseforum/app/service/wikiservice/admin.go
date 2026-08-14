package wikiservice

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/notificationservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/searchservice"
	"gorm.io/gorm"
)

// EditorView 贡献者视图（契约：userId/username/avatarUrl）。
type EditorView struct {
	UserId    uint64 `json:"userId"`
	Username  string `json:"username"`
	AvatarUrl string `json:"avatarUrl"`
}

// SetEditors 整表替换某 namespace 的贡献者列表（PageManager/Admin）。
func SetEditors(namespace string, userIds []uint64, addedBy uint64) error {
	return dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		return wikiNamespaceEditors.SetEditorsTx(tx, namespace, userIds, addedBy)
	})
}

// TreeOp 树批量操作项。
type TreeOp struct {
	Op         string `json:"op"` // move | rename | sort | delete
	PageId     uint64 `json:"pageId"`
	ParentPath string `json:"parentPath,omitempty"`
	NewPath    string `json:"newPath,omitempty"`
	NewTitle   string `json:"newTitle,omitempty"`
	SortOrder  int    `json:"sortOrder,omitempty"`
}

// ApplyTreeOps 批量应用树操作（move/rename/sort/delete），PageManager/Admin。
// 整个批次包裹在一个事务内：任一操作失败整体回滚，满足契约"any failure aborts the batch"。
// 事务提交后再执行 best-effort 副作用（附件切换/通知预览清空/搜索索引重建/删除事件），
// 与 service.go Create 的提交后副作用模式保持一致。
func ApplyTreeOps(ops []TreeOp, userId uint64) error {
	if userId == 0 || !HasPageManagerPermission(userId) {
		return ErrForbidden
	}
	var deletedPages []wikiPages.Entity
	var renamedTopics []topics.Entity
	err := dbconnect.Connect().Transaction(func(tx *gorm.DB) error {
		for _, op := range ops {
			page := wikiPages.GetTx(tx, op.PageId)
			if page.Id == 0 {
				return ErrPageNotFound
			}
			switch op.Op {
			case "move":
				if op.ParentPath != "" {
					parent := getPageByPathTx(tx, op.ParentPath)
					if parent.Id == 0 {
						return ErrPageNotFound
					}
					// 禁止跨 namespace 移动：父路径首段必须与页面所属 namespace 一致
					//（与 rename 的 NamespaceOf 守卫一致，review B2）。
					if NamespaceOf(op.ParentPath) != page.Namespace {
						return ErrPathInvalid
					}
					// 禁止自引用/子孙循环：向自身或后代移动会形成 parent_id 环，
					// 使环上所有页面 CountChildren>=1 而永不可删（review B2）。
					// 256 步上限兜底：修复前遗留的脏数据若已存在环，不能拖死批次接口。
					for a, hops := parent, 0; a.Id != 0; a = wikiPages.GetTx(tx, a.ParentId) {
						if a.Id == page.Id {
							return ErrPathInvalid
						}
						hops++
						if hops > 256 {
							return ErrPathInvalid
						}
					}
					page.ParentId = parent.Id
				} else {
					page.ParentId = 0
				}
				if err := wikiPages.SaveTx(tx, &page); err != nil {
					return err
				}
			case "rename":
				if op.NewPath != "" {
					newPath := op.NewPath
					// 兼容管理端传 namespace 相对路径：自动补全 namespace 前缀（review B2，
					// 此前 title-only rename 会因 ValidatePath 缺 namespace 段而失败）。
					if !strings.HasPrefix(newPath, page.Namespace+"/") {
						newPath = page.Namespace + "/" + newPath
					}
					normalized, ok := ValidatePath(newPath)
					if !ok {
						return ErrPathInvalid
					}
					// 禁止跨 namespace 重命名：path 首段必须保持页面所属 namespace（review B2）。
					if NamespaceOf(normalized) != page.Namespace {
						return ErrPathInvalid
					}
					if normalized != page.Path && pathExistsTx(tx, normalized, page.Id) {
						return ErrPathExists
					}
					if normalized != page.Path {
						// 级联更新后代页面的 path 前缀，保持 path 层级与 parent_id 层级一致。
						if err := renameChildPathsTx(tx, page.Path, normalized); err != nil {
							return err
						}
					}
					page.Path = normalized
				}
				if op.NewTitle != "" {
					topic, err := updatePageTitleTx(tx, page.Id, op.NewTitle)
					if err != nil {
						return err
					}
					if topic.Id != 0 {
						renamedTopics = append(renamedTopics, topic)
					}
				}
				if err := wikiPages.SaveTx(tx, &page); err != nil {
					return err
				}
			case "sort":
				page.SortOrder = op.SortOrder
				if err := wikiPages.SaveTx(tx, &page); err != nil {
					return err
				}
			case "delete":
				if countChildrenTx(tx, page.Id) > 0 {
					return ErrPageHasChildren
				}
				deleted, err := deletePageTreeTx(tx, page.Id)
				if err != nil {
					return err
				}
				if deleted.Id != 0 {
					deletedPages = append(deletedPages, deleted)
				}
			default:
				return ErrPathInvalid
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 提交后 best-effort 副作用。
	for _, page := range deletedPages {
		deletePageSideEffects(page)
	}
	for _, topic := range renamedTopics {
		firstPost := posts.Get(topic.FirstPostId)
		if firstPost.Id == 0 {
			continue
		}
		if _, err := searchservice.BuildSingleTopicSearchDocument(&topic, &firstPost); err != nil {
			slog.Warn("wiki rename: search index sync failed", "topicId", topic.Id, "error", err)
		}
	}
	return nil
}

// updatePageTitleTx 事务内更新页面最新 approved 修订的标题，并同步论坛 topics.title
// （review B2：此前 rename 只改修订行，topics.title 与搜索索引都停留在旧值；
// 与 Review approve 的 topics.SaveTx 同步一致）。Excerpt/FirstImageURL 由内容派生，
// rename 只改标题，不应触碰。返回被同步的 topic（供提交后重建搜索文档）；无 approved
// 修订时返回零值。
func updatePageTitleTx(tx *gorm.DB, pageId uint64, title string) (topics.Entity, error) {
	rev := wikiPageRevisions.GetLatestApprovedTx(tx, pageId)
	if rev.Id == 0 {
		return topics.Entity{}, nil
	}
	if err := tx.Table("wiki_page_revisions").
		Where("id = ?", rev.Id).
		Update("title", title).Error; err != nil {
		return topics.Entity{}, err
	}
	page := wikiPages.GetTx(tx, rev.PageId)
	if page.Id == 0 {
		return topics.Entity{}, nil
	}
	topic := topics.GetTx(tx, page.TopicId)
	if topic.Id == 0 {
		return topics.Entity{}, nil
	}
	topic.Title = title
	if err := topics.SaveTx(tx, &topic); err != nil {
		return topics.Entity{}, err
	}
	return topic, nil
}

// deletePageTreeTx 事务内删除页面及其 topic 的 DB 写入（复用论坛删除生命周期：
// 管理员删除不可自行恢复）。只做 DB 变更；best-effort 副作用（附件/通知/搜索/事件）
// 由调用方在事务提交后通过 deletePageSideEffects 执行，避免半删状态残留（review B2）。
func deletePageTreeTx(tx *gorm.DB, pageId uint64) (wikiPages.Entity, error) {
	page := wikiPages.GetTx(tx, pageId)
	if page.Id == 0 {
		return page, nil
	}
	// 作废待审状态：被删 wiki 页面不应停留在论坛审核队列（review N1）。
	if err := resetPendingReviewTx(tx, page.TopicId); err != nil {
		return page, err
	}
	if err := markModeratorRemovedTx(tx, page.TopicId, 0, "wiki page deleted"); err != nil {
		return page, err
	}
	// 清理该页全部修订，避免 pending 修订残留进 wiki 审核队列且 Review
	// 返回 ErrPageNotFound 的幽灵项（review P2）。
	if err := deleteRevisionsByPageTx(tx, pageId); err != nil {
		return page, err
	}
	// 删除立即生效：附件转入受限恢复态 + 通知预览置空 + 搜索索引清理（review N3），
	// 全部留到事务提交后统一执行。
	if err := deletePageTx(tx, pageId); err != nil {
		return page, err
	}
	return page, nil
}

// deletePageSideEffects 删除提交后的 best-effort 副作用。
func deletePageSideEffects(page wikiPages.Entity) {
	fileusageservice.HardenTargetFiles(
		fileusageservice.TargetRef{TargetType: fileUsage.TargetTopic, TargetID: page.TopicId},
		time.Now().Add(30*24*time.Hour),
	)
	notificationservice.NullifyContentPreviews(page.TopicId, 0)
	eventbus.Publish(context.Background(), &eventhandlers.ContentDeletedEvent{
		ContentType:  string(contentDeleteEventTypeTopic),
		TopicId:      page.TopicId,
		DeletedBy:    0,
		DeleteReason: "wiki page deleted",
	})
}

// 以下为 ApplyTreeOps/deletePageTree 在事务内使用的 Tx 变体（本地实现，避免修改
// 各 repo 文件）：单连接测试库下事务内必须走 tx 句柄，否则全局连接会死锁。

// getPageByPathTx 事务内按完整 path 查询页面。
func getPageByPathTx(tx *gorm.DB, path string) (entity wikiPages.Entity) {
	tx.Table("wiki_pages").Where("path = ?", path).First(&entity)
	return
}

// pathExistsTx 事务内判断 path 是否已被占用（排除指定 id）。
func pathExistsTx(tx *gorm.DB, path string, excludeID uint64) bool {
	var count int64
	q := tx.Table("wiki_pages").Where("path = ?", path)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

// countChildrenTx 事务内统计某页面的直接子页面数。
func countChildrenTx(tx *gorm.DB, parentID uint64) int64 {
	var count int64
	tx.Table("wiki_pages").Where("parent_id = ?", parentID).Count(&count)
	return count
}

// deletePageTx 事务内物理删除页面行（wiki_pages 无软删）。
func deletePageTx(tx *gorm.DB, id uint64) error {
	return tx.Table("wiki_pages").Where("id = ?", id).Delete(&wikiPages.Entity{}).Error
}

// deleteRevisionsByPageTx 事务内删除某页面的全部修订。
func deleteRevisionsByPageTx(tx *gorm.DB, pageID uint64) error {
	return tx.Table("wiki_page_revisions").Where("page_id = ?", pageID).Delete(&wikiPageRevisions.Entity{}).Error
}

// resetPendingReviewTx 事务内将话题 process_status 复位为正常（作废待审）。
func resetPendingReviewTx(tx *gorm.DB, id uint64) error {
	return tx.Table("topics").Unscoped().Where("id = ?", id).UpdateColumn("process_status", topics.ProcessStatusNormal).Error
}

// markModeratorRemovedTx 事务内将话题标记为管理员删除（作者不可自行恢复）。
func markModeratorRemovedTx(tx *gorm.DB, id uint64, deletedBy uint64, reason string) error {
	return tx.Table("topics").Unscoped().Where("id = ?", id).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": topics.VisibilityModeratorRemoved,
		"retention_status":  topics.RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// renameChildPathsTx 重命名页面路径时级联更新其后代页面的 path 前缀，
// 保持 path 层级与 parent_id 层级一致（一次前缀替换覆盖全部层级）。
func renameChildPathsTx(tx *gorm.DB, oldPath, newPath string) error {
	prefix := oldPath + "/"
	var children []wikiPages.Entity
	if err := tx.Table("wiki_pages").
		Where("path LIKE ?", prefix+"%").
		Find(&children).Error; err != nil {
		return err
	}
	for i := range children {
		children[i].Path = newPath + "/" + strings.TrimPrefix(children[i].Path, prefix)
		if err := wikiPages.SaveTx(tx, &children[i]); err != nil {
			return err
		}
	}
	return nil
}

// contentDeleteEventTypeTopic 与 contentdeleteservice.ContentTypeTopic 同值，
// 避免 wikiservice → contentdeleteservice 的包级依赖（review N3 事件广播）。
const contentDeleteEventTypeTopic = "topic"

// DeleteNamespace 删除 namespace 及其贡献者记录（PageManager/Admin；存在页面时 409）。
func DeleteNamespace(name string) error {
	if err := wikiNamespaceEditors.DeleteByNamespace(name); err != nil {
		slog.Error("wiki delete namespace editors failed", "namespace", name, "error", err)
	}
	return wikiNamespaces.DeleteByName(name)
}
