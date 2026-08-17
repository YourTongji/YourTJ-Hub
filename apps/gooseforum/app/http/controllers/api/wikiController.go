package api

import (
	"log/slog"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/wikiservice"
)

// ---------- 公开读 ----------

// WikiTree 返回公开导航树（契约包裹形状）。
func WikiTree(req component.BetterRequest[component.Null]) component.Response {
	tree, err := wikiservice.BuildTreeAPI()
	if err != nil {
		return wikiReadFailure("build wiki tree", err)
	}
	return component.SuccessResponse(tree)
}

// WikiNamespaces 返回 namespace 摘要列表（公开）。
func WikiNamespaces(req component.BetterRequest[component.Null]) component.Response {
	summaries, err := wikiservice.BuildNamespaceSummaries()
	if err != nil {
		return wikiReadFailure("build wiki namespace summaries", err)
	}
	return component.SuccessResponse(summaries)
}

// WikiHome 返回首页数据（公开）。
func WikiHome(req component.BetterRequest[component.Null]) component.Response {
	home, err := wikiservice.BuildHome()
	if err != nil {
		return wikiReadFailure("build wiki home", err)
	}
	return component.SuccessResponse(home)
}

// ---------- admin：只读页面树（PageManager/Admin） ----------

// WikiAdminTree 返回管理端导航树（PageManager/Admin）。
// GitHub SSOT：命名空间与页面结构由仓库同步决定，管理端只读。
func WikiAdminTree(req component.BetterRequest[component.Null]) component.Response {
	tree, err := wikiservice.BuildAdminTree()
	if err != nil {
		return wikiReadFailure("build admin wiki tree", err)
	}
	return component.SuccessResponse(tree)
}

// wikiReadFailure 把 wiki 读路径的 DB 故障映射为 HTTP 500（契约已声明 500 响应）。
// 数据库故障必须区别于空 wiki（issue #287）；不向客户端泄漏原始错误。
func wikiReadFailure(what string, err error) component.Response {
	slog.Error("wiki read failed", "op", what, "error", err)
	return component.BuildResponse(http.StatusInternalServerError,
		component.FailDataCode(component.MessageWikiReadFailed, nil))
}
