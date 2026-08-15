package api

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/forum"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
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

// ---------- admin：namespace 管理（PageManager/Admin） ----------

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
	entity := wikiNamespaces.GetByName(strings.ToLower(req.Params.Name))
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
	entity := wikiNamespaces.GetByName(strings.ToLower(req.Params.Name))
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceNotFound, nil)
	}
	if len(wikiPagesList(req.Params.Name)) > 0 {
		return component.FailResponseCode(component.MessageWikiNamespaceHasPages, nil)
	}
	if err := wikiservice.DeleteNamespace(req.Params.Name); err != nil {
		slog.Error("wiki delete namespace failed", "name", req.Params.Name, "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
	return component.SuccessResponse(wikiservice.ActionResult{Ok: true})
}

// WikiAdminTree 返回管理端导航树（PageManager/Admin）。
func WikiAdminTree(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(wikiservice.BuildAdminTree())
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
	case errors.Is(err, wikiservice.ErrPageNotFound):
		return component.FailResponseCode(component.MessageWikiPageNotFound, nil)
	case errors.Is(err, wikiservice.ErrForbidden):
		return component.FailResponseCode(component.MessageWikiForbidden, nil)
	default:
		slog.Error("wiki operation failed", "error", err)
		return component.FailResponseCode(component.MessageWikiSaveFailed, nil)
	}
}

// wikiPagesList 返回某 namespace 的页面列表。
func wikiPagesList(namespace string) []*wikiPages.Entity {
	return wikiPages.ListByNamespace(namespace)
}

// 保留 forum 引用（topic 可见性谓词在 forum 包）。
var _ = forum.CanViewTopicSimple
var _ = topics.TopicTypeWiki
