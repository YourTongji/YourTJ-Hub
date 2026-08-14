package wikiservice

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/eventbus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/eventhandlers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/notificationservice"
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
func ApplyTreeOps(ops []TreeOp, userId uint64) error {
	if userId == 0 || !HasPageManagerPermission(userId) {
		return ErrForbidden
	}
	for _, op := range ops {
		page := wikiPages.Get(op.PageId)
		if page.Id == 0 {
			return ErrPageNotFound
		}
		switch op.Op {
		case "move":
			if op.ParentPath != "" {
				parent := wikiPages.GetByPath(op.ParentPath)
				if parent.Id == 0 {
					return ErrPageNotFound
				}
				page.ParentId = parent.Id
			} else {
				page.ParentId = 0
			}
			if err := wikiPages.Save(&page); err != nil {
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
				if normalized != page.Path && wikiPages.PathExists(normalized, page.Id) {
					return ErrPathExists
				}
				page.Path = normalized
			}
			if op.NewTitle != "" {
				if err := updatePageTitle(page.Id, op.NewTitle); err != nil {
					return err
				}
			}
			if err := wikiPages.Save(&page); err != nil {
				return err
			}
		case "sort":
			page.SortOrder = op.SortOrder
			if err := wikiPages.Save(&page); err != nil {
				return err
			}
		case "delete":
			if wikiPages.CountChildren(page.Id) > 0 {
				return ErrPageHasChildren
			}
			if err := deletePageTree(page.Id); err != nil {
				return err
			}
		default:
			return ErrPathInvalid
		}
	}
	return nil
}

// updatePageTitle 更新页面最新 approved 修订的标题（重命名）。
func updatePageTitle(pageId uint64, title string) error {
	rev := wikiPageRevisions.GetLatestApproved(pageId)
	if rev.Id == 0 {
		return nil
	}
	return dbconnect.Connect().Table("wiki_page_revisions").
		Where("id = ?", rev.Id).
		Update("title", title).Error
}

// deletePageTree 删除页面及其 topic（复用论坛删除生命周期：管理员删除不可自行恢复）。
func deletePageTree(pageId uint64) error {
	page := wikiPages.Get(pageId)
	if page.Id == 0 {
		return nil
	}
	// 作废待审状态：被删 wiki 页面不应停留在论坛审核队列（review N1）。
	if err := topics.ResetPendingReview(page.TopicId); err != nil {
		return err
	}
	if err := topics.MarkModeratorRemoved(page.TopicId, 0, "wiki page deleted"); err != nil {
		return err
	}
	// 清理该页全部修订，避免 pending 修订残留进 wiki 审核队列且 Review
	// 返回 ErrPageNotFound 的幽灵项（review P2）。
	if err := wikiPageRevisions.DeleteByPage(pageId); err != nil {
		return err
	}
	// 删除立即生效：附件转入受限恢复态 + 通知预览置空 + 搜索索引清理（review N3）。
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
	return wikiPages.Delete(pageId)
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
