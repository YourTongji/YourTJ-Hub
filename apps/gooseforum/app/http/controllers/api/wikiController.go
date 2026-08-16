package api

import (
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
)

// ---------- 公开读 ----------

// WikiTree 返回公开导航树（契约包裹形状）。
func WikiTree(req component.BetterRequest[component.Null]) component.Response {
	data, err := wikiservice.BuildTreeAPI()
	if err != nil {
		return component.BuildResponse(http.StatusInternalServerError, component.FailData())
	}
	return component.SuccessResponse(data)
}

// WikiNamespaces 返回 namespace 摘要列表（公开）。
func WikiNamespaces(req component.BetterRequest[component.Null]) component.Response {
	data, err := wikiservice.BuildNamespaceSummaries()
	if err != nil {
		return component.BuildResponse(http.StatusInternalServerError, component.FailData())
	}
	return component.SuccessResponse(data)
}

// WikiHome 返回首页数据（公开）。
func WikiHome(req component.BetterRequest[component.Null]) component.Response {
	data, err := wikiservice.BuildHome()
	if err != nil {
		return component.BuildResponse(http.StatusInternalServerError, component.FailData())
	}
	return component.SuccessResponse(data)
}

// ---------- admin：只读页面树（PageManager/Admin） ----------

// WikiAdminTree 返回管理端导航树（PageManager/Admin）。
// GitHub SSOT：命名空间与页面结构由仓库同步决定，管理端只读。
func WikiAdminTree(req component.BetterRequest[component.Null]) component.Response {
	data, err := wikiservice.BuildAdminTree()
	if err != nil {
		return component.BuildResponse(http.StatusInternalServerError, component.FailData())
	}
	return component.SuccessResponse(data)
}
