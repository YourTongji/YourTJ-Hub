package api

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaceEditors"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiNamespaces"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
)

// ---------- 公开读 ----------

// WikiTree 返回公开导航树（契约包裹形状）。
func WikiTree(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildTreeAPI())
}

// WikiNamespaces 返回 namespace 摘要列表（公开）。
func WikiNamespaces(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildNamespaceSummaries())
}

// WikiHome 返回首页数据（公开）。
func WikiHome(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildHome())
}

// WikiRevisionsReq 修订历史请求。
type WikiRevisionsReq struct {
	PageId uint64 `form:"pageId" validate:"required"`
}

// WikiRevisions 返回某页面全部修订（公开）。
func WikiRevisions(req component.BetterRequest[WikiRevisionsReq]) component.Response {
	page := wikiPagesGet(req.Params.PageId)
	if page.Id == 0 {
		return component.FailResponseCode(component.MessageWikiPageNotFound, nil)
	}
	return component.SuccessResponse(wikiservice.ListRevisions(page.Id))
}

// ---------- 登录写 ----------

// WikiCreatePageReq 创建页面请求。
type WikiCreatePageReq struct {
	Namespace string `json:"namespace" validate:"required"`
	Path      string `json:"path" validate:"required"`
	Title     string `json:"title" validate:"required"`
	Content   string `json:"content"`
}

// WikiCreatePage 创建 wiki 页面（namespace 贡献者/PageManager/Admin）。
func WikiCreatePage(req component.BetterRequest[WikiCreatePageReq]) component.Response {
	result, err := wikiservice.Create(wikiservice.CreateParams{
		Namespace: req.Params.Namespace,
		Path:      req.Params.Path,
		Title:     req.Params.Title,
		Content:   req.Params.Content,
		UserId:    req.UserId,
	})
	if err != nil {
		if errors.Is(err, wikiservice.ErrForbidden) {
			// 契约：非 namespace 贡献者创建 → wiki.namespace.notFound 语义。
			return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
		}
		if errors.Is(err, wikiservice.ErrPathInvalid) {
			// 契约：非法 path → common.request.invalidParams。
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		}
		return wikiErrorResponse(err)
	}
	return component.SuccessResponse(result)
}

// WikiEditPageReq 编辑页面请求。
type WikiEditPageReq struct {
	PageId  uint64 `uri:"pageId" json:"-" validate:"required"`
	Title   string `json:"title" validate:"required"`
	Content string `json:"content"`
}

// WikiEditPage 编辑 wiki 页面（创建者/贡献者/PageManager/Admin）→ pending 修订。
func WikiEditPage(req component.BetterRequest[WikiEditPageReq]) component.Response {
	result, err := wikiservice.Edit(wikiservice.EditParams{
		PageID:  req.Params.PageId,
		Title:   req.Params.Title,
		Content: req.Params.Content,
		UserId:  req.UserId,
	})
	if err != nil {
		if errors.Is(err, wikiservice.ErrForbidden) {
			// 契约：非编辑权限 → wiki.namespace.notFound 语义。
			return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
		}
		return wikiErrorResponse(err)
	}
	return component.SuccessResponse(result)
}

// WikiReviewReq 审核请求。
type WikiReviewReq struct {
	RevisionId uint64 `uri:"revisionId" json:"-" validate:"required"`
	Action     string `json:"action" validate:"required,oneof=approve reject"`
}

// WikiReview 审核修订（PageManager/Admin）。
func WikiReview(req component.BetterRequest[WikiReviewReq]) component.Response {
	result, err := wikiservice.Review(wikiservice.ReviewParams{
		RevisionID: req.Params.RevisionId,
		Action:     req.Params.Action,
		UserId:     req.UserId,
	})
	if err != nil {
		if errors.Is(err, wikiservice.ErrForbidden) {
			// 契约：无 PageManager → permission.denied。
			return component.FailResponseCode(component.MessagePermissionDenied, nil)
		}
		return wikiErrorResponse(err)
	}
	return component.SuccessResponse(result)
}

// ---------- admin ----------

// WikiCreateNamespaceReq 创建 namespace 请求。
type WikiCreateNamespaceReq struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

func WikiCreateNamespace(req component.BetterRequest[WikiCreateNamespaceReq]) component.Response {
	name := strings.ToLower(strings.TrimSpace(req.Params.Name))
	if !wikiservice.ValidateNamespace(name) {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if wikiNamespaces.Exists(name) {
		return component.FailResponseCode(component.MessageWikiNamespaceNameConflict, nil)
	}
	entity := &wikiNamespaces.Entity{
		Name:        name,
		Description: req.Params.Description,
	}
	if err := wikiNamespaces.Create(entity); err != nil {
		slog.Error("wiki create namespace failed", "name", name, "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiUpdateNamespaceReq 更新 namespace 请求。
type WikiUpdateNamespaceReq struct {
	Name        string `uri:"name" json:"-" validate:"required"`
	Description string `json:"description"`
}

// WikiUpdateNamespace 更新 namespace 描述（PageManager/Admin）。
func WikiUpdateNamespace(req component.BetterRequest[WikiUpdateNamespaceReq]) component.Response {
	entity := wikiNamespaces.GetByName(req.Params.Name)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	}
	entity.Description = req.Params.Description
	if err := wikiNamespaces.Save(&entity); err != nil {
		slog.Error("wiki update namespace failed", "name", req.Params.Name, "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiDeleteNamespaceReq 删除 namespace 请求。
type WikiDeleteNamespaceReq struct {
	Name string `uri:"name" json:"-" validate:"required"`
}

// WikiDeleteNamespace 删除 namespace（PageManager/Admin；存在页面时 409）。
func WikiDeleteNamespace(req component.BetterRequest[WikiDeleteNamespaceReq]) component.Response {
	entity := wikiNamespaces.GetByName(req.Params.Name)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	}
	if len(wikiPagesList(req.Params.Name)) > 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceHasPages, nil)
	}
	if err := wikiNamespaces.DeleteByName(req.Params.Name); err != nil {
		slog.Error("wiki delete namespace failed", "name", req.Params.Name, "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiNamespaceEditorsReq 贡献者请求。
type WikiNamespaceEditorsReq struct {
	Name string `uri:"name" json:"-" validate:"required"`
}

// WikiNamespaceEditors 返回某 namespace 贡献者列表（PageManager/Admin）。
func WikiNamespaceEditors(req component.BetterRequest[WikiNamespaceEditorsReq]) component.Response {
	entity := wikiNamespaces.GetByName(req.Params.Name)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	}
	editors := wikiNamespaceEditors.ListByNamespace(req.Params.Name)
	userIDs := make([]uint64, 0, len(editors))
	for _, e := range editors {
		userIDs = append(userIDs, e.UserId)
	}
	userMap := users.GetMapByIds(userIDs)
	result := make([]wikiservice.EditorView, 0, len(editors))
	for _, e := range editors {
		view := wikiservice.EditorView{UserId: e.UserId}
		if u, ok := userMap[e.UserId]; ok && u != nil {
			view.Username = u.Username
			view.AvatarUrl = u.GetWebAvatarUrl()
		}
		result = append(result, view)
	}
	return component.SuccessResponse(result)
}

// WikiSetEditorsReq 整表设置贡献者请求。
type WikiSetEditorsReq struct {
	Name    string   `uri:"name" json:"-" validate:"required"`
	UserIds []uint64 `json:"userIds"`
}

// WikiSetEditors 整表设置某 namespace 贡献者（PageManager/Admin）。
func WikiSetEditors(req component.BetterRequest[WikiSetEditorsReq]) component.Response {
	entity := wikiNamespaces.GetByName(req.Params.Name)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	}
	if err := wikiservice.SetEditors(req.Params.Name, req.Params.UserIds, req.UserId); err != nil {
		slog.Error("wiki set editors failed", "name", req.Params.Name, "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiAdminTree 返回管理端导航树（PageManager/Admin）。
func WikiAdminTree(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildAdminTree())
}

// WikiAdminTreeOpReq 树批量操作请求。
type WikiAdminTreeOpReq struct {
	Ops []wikiservice.TreeOp `json:"ops"`
}

// WikiAdminTreeOps 批量树操作：move/rename/sort/delete（PageManager/Admin）。
func WikiAdminTreeOps(req component.BetterRequest[WikiAdminTreeOpReq]) component.Response {
	if err := wikiservice.ApplyTreeOps(req.Params.Ops, req.UserId); err != nil {
		if errors.Is(err, wikiservice.ErrPathInvalid) {
			// 契约：非法树操作参数 → common.request.invalidParams。
			return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
		}
		return wikiErrorResponse(err)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiAdminRevisionsReq 审核队列请求。
type WikiAdminRevisionsReq struct {
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

// WikiAdminRevisions 返回审核队列（PageManager/Admin；status 必填且须为契约枚举）。
func WikiAdminRevisions(req component.BetterRequest[WikiAdminRevisionsReq]) component.Response {
	status, ok := wikiservice.ParseRevisionStatus(req.Params.Status)
	if !ok {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	list := wikiservice.ListAdminRevisions(status, req.Params.Page, req.Params.PageSize)
	return component.SuccessResponse(list)
}

// wikiErrorResponse 映射 wikiservice 哨兵错误到稳定 messageCode（契约对齐）。
func wikiErrorResponse(err error) component.Response {
	switch {
	case errors.Is(err, wikiservice.ErrNamespaceNotFound):
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	case errors.Is(err, wikiservice.ErrNamespaceExists):
		return component.FailResponseCode(component.MessageWikiNamespaceNameConflict, nil)
	case errors.Is(err, wikiservice.ErrNamespaceHasPages):
		return component.FailResponseCode(component.MessageWikiNamespaceHasPages, nil)
	case errors.Is(err, wikiservice.ErrPathInvalid):
		return component.FailResponseCode(component.MessageWikiPathInvalid, nil)
	case errors.Is(err, wikiservice.ErrPathExists):
		return component.FailResponseCode(component.MessageWikiPathConflict, nil)
	case errors.Is(err, wikiservice.ErrPageNotFound):
		return component.FailResponseCode(component.MessageWikiPageNotFound, nil)
	case errors.Is(err, wikiservice.ErrForbidden):
		return component.FailResponseCode(component.MessageWikiForbidden, nil)
	case errors.Is(err, wikiservice.ErrRevisionNotFound):
		return component.FailResponseCode(component.MessageWikiRevisionNotFound, nil)
	case errors.Is(err, wikiservice.ErrRevisionNotPending):
		return component.FailResponseCode(component.MessageWikiRevisionNotPending, nil)
	case errors.Is(err, wikiservice.ErrPageHasChildren):
		return component.FailResponseCode(component.MessageWikiPageHasChildren, nil)
	case errors.Is(err, wikiservice.ErrNamespaceNameInvalid):
		return component.FailResponseCode(component.MessageWikiNamespaceNameInvalid, nil)
	default:
		slog.Error("wiki operation failed", "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
}

// wikiPagesGet 按 id 取 wiki 页面。
func wikiPagesGet(id uint64) (page wikiPages.Entity) {
	return wikiPages.Get(id)
}

// wikiPagesList 返回某 namespace 的页面列表。
func wikiPagesList(namespace string) []*wikiPages.Entity {
	return wikiPages.ListByNamespace(namespace)
}
