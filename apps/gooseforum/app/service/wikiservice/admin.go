package wikiservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPageRevisions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
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
				newPath, ok := ValidatePath(op.NewPath)
				if !ok {
					return ErrPathInvalid
				}
				if newPath != page.Path && wikiPages.PathExists(newPath, page.Id) {
					return ErrPathExists
				}
				page.Path = newPath
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
	if err := topics.MarkModeratorRemoved(page.TopicId, 0, "wiki page deleted"); err != nil {
		return err
	}
	return wikiPages.Delete(pageId)
}
