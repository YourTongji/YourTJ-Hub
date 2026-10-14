package api

import (
	"errors"
	"log/slog"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/statsservice"
)

// 管理后台「内容贡献统计」接口（冻结契约 §2，全部 POST + Admin 权限）。
// 校验口径、日期序列与聚合读在 statsservice / contributionstats。

// AdminStatsCategorySummaryRequest 板块汇总请求：日期为空时取近 7 天到当天。
type AdminStatsCategorySummaryRequest struct {
	StartDate string `json:"startDate"` // YYYY-MM-DD
	EndDate   string `json:"endDate"`   // YYYY-MM-DD
}

// AdminStatsCategorySummaryResult 板块汇总结果。
type AdminStatsCategorySummaryResult = statsservice.CategorySummaryResult

// AdminStatsCategorySummary 返回各板块主题/回复/贡献人数与全站合计。
func AdminStatsCategorySummary(req component.BetterRequest[AdminStatsCategorySummaryRequest]) component.Response {
	result, err := statsservice.CategorySummary(requestContext(req.GinContext), req.Params.StartDate, req.Params.EndDate)
	if err != nil {
		return adminStatsFailure(err)
	}
	return component.SuccessResponse(result)
}

// AdminStatsCategoryTrendRequest 板块趋势请求：categoryId 必填。
type AdminStatsCategoryTrendRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	CategoryId uint64 `json:"categoryId"`
}

// AdminStatsCategoryTrendResult 板块趋势结果。
type AdminStatsCategoryTrendResult = statsservice.CategoryTrendResult

// AdminStatsCategoryTrend 返回指定板块逐日的主题/回复数与估算标记。
func AdminStatsCategoryTrend(req component.BetterRequest[AdminStatsCategoryTrendRequest]) component.Response {
	result, err := statsservice.CategoryTrend(requestContext(req.GinContext), req.Params.CategoryId, req.Params.StartDate, req.Params.EndDate)
	if err != nil {
		return adminStatsFailure(err)
	}
	return component.SuccessResponse(result)
}

// AdminStatsUserRankingRequest 用户排行请求：categoryId=0/省略为全站；
// actorType=0 真人、1 Agent（默认真人），两者不混算。
type AdminStatsUserRankingRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	CategoryId uint64 `json:"categoryId"`
	SortBy     string `json:"sortBy"`   // topics | replies | likes，默认 topics
	Page       *int   `json:"page"`     // 从 1 开始；省略=1，显式越界拒绝
	PageSize   *int   `json:"pageSize"` // 1..100；省略=20，显式越界拒绝
	ActorType  int    `json:"actorType"`
}

// AdminStatsUserRankingResult 用户排行结果。
type AdminStatsUserRankingResult = statsservice.UserRankingResult

// AdminStatsUserRanking 返回按主题数/回复数/获赞排序的账号排行（分页）。
func AdminStatsUserRanking(req component.BetterRequest[AdminStatsUserRankingRequest]) component.Response {
	result, err := statsservice.UserRanking(requestContext(req.GinContext), statsservice.RankingQuery{
		StartDate:  req.Params.StartDate,
		EndDate:    req.Params.EndDate,
		CategoryId: req.Params.CategoryId,
		SortBy:     req.Params.SortBy,
		Page:       req.Params.Page,
		PageSize:   req.Params.PageSize,
		ActorType:  req.Params.ActorType,
	})
	if err != nil {
		return adminStatsFailure(err)
	}
	return component.SuccessResponse(result)
}

// AdminStatsUserDetailRequest 单用户明细请求：userId 与 username 恰给一个。
type AdminStatsUserDetailRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	CategoryId uint64 `json:"categoryId"`
	UserId     uint64 `json:"userId"`
	Username   string `json:"username"`
}

// AdminStatsUserDetailResult 单用户明细结果。
type AdminStatsUserDetailResult = statsservice.UserDetailResult

// AdminStatsUserDetail 返回单个账号的贡献汇总与逐日序列；无贡献时全为 0。
func AdminStatsUserDetail(req component.BetterRequest[AdminStatsUserDetailRequest]) component.Response {
	result, err := statsservice.UserDetail(requestContext(req.GinContext), statsservice.DetailQuery{
		StartDate:  req.Params.StartDate,
		EndDate:    req.Params.EndDate,
		CategoryId: req.Params.CategoryId,
		UserId:     req.Params.UserId,
		Username:   req.Params.Username,
	})
	if err != nil {
		return adminStatsFailure(err)
	}
	return component.SuccessResponse(result)
}

// adminStatsFailure 把统计错误映射为稳定错误码：校验失败与用户不存在透传，
// 其余查询失败统一为 admin.stats.fetchFailed——不把失败伪装成 0。
func adminStatsFailure(err error) component.Response {
	var messageErr component.MessageError
	if errors.As(err, &messageErr) {
		return component.FailResponseCode(messageErr.Code, messageErr.Params)
	}
	slog.Error("admin stats query failed", "error", err)
	return component.FailResponseCode(component.MessageAdminStatsFetchFailed, nil)
}
