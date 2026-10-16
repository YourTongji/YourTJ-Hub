// Package statsservice 实现管理后台「内容贡献统计」的校验与结果组装。
//
// 口径来源：research/issue-1145-design.md §1（R1-R9）。本包只做参数校验、日期序列
// 与结果合并，实际聚合读全部落在 models/forum/contributionstats。
package statsservice

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/category"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/contributionstats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

const (
	// MaxRangeDays 单次统计允许的最大自然日跨度（含首尾）。上限需要管理员可见。
	MaxRangeDays = 366
	// defaultRangeDays 缺省区间长度：与既有 traffic-overview 的「近 7 天」惯例一致。
	defaultRangeDays = 7
	// DefaultPageSize / MaxPageSize / MaxPage 用户排行的分页边界。
	// MaxPage 兜住 (page-1)*pageSize 的溢出与无意义深翻页：超出直接按参数错误拒绝。
	DefaultPageSize = 20
	MaxPageSize     = 100
	MaxPage         = 10000

	dateLayout = "2006-01-02"
)

// Coverage 描述结果的时间口径：切片时区、区间上限，以及是否存在估算时间归属。
type Coverage struct {
	Timezone     string `json:"timezone"`
	MaxRangeDays int    `json:"maxRangeDays"`
	Estimated    bool   `json:"estimated"`
}

// CategorySummaryItem 是板块汇总中的一行；无数据的板块返回 0 而不是缺行。
type CategorySummaryItem struct {
	CategoryId            uint64 `json:"categoryId"`
	Name                  string `json:"name"`
	TopicCount            int64  `json:"topicCount"`
	ReplyCount            int64  `json:"replyCount"`
	ContributorCount      int64  `json:"contributorCount"`
	AgentContributorCount int64  `json:"agentContributorCount"`
}

// CategorySummaryTotals 是全站合计：按内容主键去重，不受多板块重复归属影响。
type CategorySummaryTotals struct {
	TopicCount            int64 `json:"topicCount"`
	ReplyCount            int64 `json:"replyCount"`
	ContributorCount      int64 `json:"contributorCount"`
	AgentContributorCount int64 `json:"agentContributorCount"`
}

// CategorySummaryResult 是板块汇总接口的结果（冻结契约 §2.1）。
type CategorySummaryResult struct {
	Categories []CategorySummaryItem `json:"categories"`
	Totals     CategorySummaryTotals `json:"totals"`
	Coverage   Coverage              `json:"coverage"`
}

// TrendDay 是趋势/明细序列中的一天，Date 为服务器本地日。
type TrendDay struct {
	Date       string `json:"date"`
	TopicCount int64  `json:"topicCount"`
	ReplyCount int64  `json:"replyCount"`
	Estimated  bool   `json:"estimated"`
}

// CategoryTrendResult 是板块趋势接口的结果（冻结契约 §2.2）。
type CategoryTrendResult struct {
	CategoryId uint64     `json:"categoryId"`
	Days       []TrendDay `json:"days"`
	Coverage   Coverage   `json:"coverage"`
}

// UserRankingItem 是用户排行中的一行。
type UserRankingItem struct {
	UserId     uint64 `json:"userId"`
	Username   string `json:"username"`
	Nickname   string `json:"nickname"`
	ActorType  int8   `json:"actorType"`
	TopicCount int64  `json:"topicCount"`
	ReplyCount int64  `json:"replyCount"`
	LikeCount  int64  `json:"likeCount"`
}

// UserRankingResult 是用户排行接口的结果（冻结契约 §2.3）。
type UserRankingResult struct {
	Items    []UserRankingItem `json:"items"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
	Total    int64             `json:"total"`
	Coverage Coverage          `json:"coverage"`
}

// UserDetailIdentity 是被查询账号的公开身份。
type UserDetailIdentity struct {
	UserId    uint64 `json:"userId"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	ActorType int8   `json:"actorType"`
}

// UserDetailSummary 是被查询账号在区间内的贡献汇总。
type UserDetailSummary struct {
	TopicCount int64 `json:"topicCount"`
	ReplyCount int64 `json:"replyCount"`
	LikeCount  int64 `json:"likeCount"`
}

// UserDetailResult 是单用户明细接口的结果（冻结契约 §2.4）。
type UserDetailResult struct {
	User     UserDetailIdentity `json:"user"`
	Summary  UserDetailSummary  `json:"summary"`
	Days     []TrendDay         `json:"days"`
	Coverage Coverage           `json:"coverage"`
}

// RankingQuery 是用户排行的筛选条件（日期为空时取缺省区间）。
type RankingQuery struct {
	StartDate  string
	EndDate    string
	CategoryId uint64
	SortBy     string
	Page       *int // nil = 缺省（第 1 页）；显式越界值一律拒绝，不静默取默认
	PageSize   *int // nil = 缺省（20）；显式越界值一律拒绝，不夹取
	ActorType  int  // 0 真人 / 1 Agent（users.ActorType*）
}

// DetailQuery 是单用户明细的筛选条件；UserId 与 Username 恰给一个。
type DetailQuery struct {
	StartDate  string
	EndDate    string
	CategoryId uint64
	UserId     uint64
	Username   string
}

// CategorySummary 组装板块汇总（口径 R1/R3/R4/R5/R8）。
func CategorySummary(ctx context.Context, startDate, endDate string) (CategorySummaryResult, error) {
	scope, err := parseScope(startDate, endDate, 0)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	categories := listCategories()
	topicCounts, err := contributionstats.CategoryCounts(ctx, scope, contributionstats.KindTopic)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	replyCounts, err := contributionstats.CategoryCounts(ctx, scope, contributionstats.KindReply)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	human, err := contributionstats.CategoryContributorCounts(ctx, scope, users.ActorTypeHuman)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	bot, err := contributionstats.CategoryContributorCounts(ctx, scope, users.ActorTypeBot)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	totals, err := siteTotals(ctx, scope)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	humanTotal, err := contributionstats.SiteContributorCount(ctx, scope, users.ActorTypeHuman)
	if err != nil {
		return CategorySummaryResult{}, err
	}
	botTotal, err := contributionstats.SiteContributorCount(ctx, scope, users.ActorTypeBot)
	if err != nil {
		return CategorySummaryResult{}, err
	}

	topicByCategory := countByCategory(topicCounts)
	replyByCategory := countByCategory(replyCounts)
	humanByCategory := countByCategory(human)
	botByCategory := countByCategory(bot)

	items := make([]CategorySummaryItem, 0, len(categories))
	for _, entity := range categories {
		items = append(items, CategorySummaryItem{
			CategoryId:            entity.Id,
			Name:                  entity.Name,
			TopicCount:            topicByCategory[entity.Id],
			ReplyCount:            replyByCategory[entity.Id],
			ContributorCount:      humanByCategory[entity.Id],
			AgentContributorCount: botByCategory[entity.Id],
		})
	}
	// 契约与产品文档冻结的排序：主题数降序、板块 ID 升序（零贡献板块自然沉底）。
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TopicCount != items[j].TopicCount {
			return items[i].TopicCount > items[j].TopicCount
		}
		return items[i].CategoryId < items[j].CategoryId
	})
	return CategorySummaryResult{
		Categories: items,
		Totals: CategorySummaryTotals{
			TopicCount:            totals.topic.Count,
			ReplyCount:            totals.reply.Count,
			ContributorCount:      humanTotal,
			AgentContributorCount: botTotal,
		},
		Coverage: coverage(totals.estimated),
	}, nil
}

// CategoryTrend 组装板块趋势（口径 R2/R3/R8）：区间内每一天升序，缺失日为 0；
// 单日含估算来源时该日 estimated=true。
func CategoryTrend(ctx context.Context, categoryId uint64, startDate, endDate string) (CategoryTrendResult, error) {
	if err := requireCategory(ctx, categoryId); err != nil {
		return CategoryTrendResult{}, err
	}
	scope, err := parseScope(startDate, endDate, categoryId)
	if err != nil {
		return CategoryTrendResult{}, err
	}
	topicDays, err := contributionstats.CategoryDailyCounts(ctx, scope, contributionstats.KindTopic)
	if err != nil {
		return CategoryTrendResult{}, err
	}
	replyDays, err := contributionstats.CategoryDailyCounts(ctx, scope, contributionstats.KindReply)
	if err != nil {
		return CategoryTrendResult{}, err
	}
	days, estimated := mergeDays(scope, topicDays, replyDays)
	return CategoryTrendResult{
		CategoryId: categoryId,
		Days:       days,
		Coverage:   coverage(estimated),
	}, nil
}

// UserRanking 组装用户排行（口径 R4/R5/R6/R7/R8）。真人/Agent 由 ActorType 选择，
// 不混算；匿名内容与已注销账号不出现；同值按 user_id 升序。
func UserRanking(ctx context.Context, query RankingQuery) (UserRankingResult, error) {
	actorType, err := parseActorType(query.ActorType)
	if err != nil {
		return UserRankingResult{}, err
	}
	page, pageSize, err := parsePaging(query.Page, query.PageSize)
	if err != nil {
		return UserRankingResult{}, err
	}
	sortBy, err := parseSortBy(query.SortBy)
	if err != nil {
		return UserRankingResult{}, err
	}
	scope, err := parseScope(query.StartDate, query.EndDate, query.CategoryId)
	if err != nil {
		return UserRankingResult{}, err
	}
	if query.CategoryId > 0 {
		if err := requireCategory(ctx, query.CategoryId); err != nil {
			return UserRankingResult{}, err
		}
	}
	rows, err := contributionstats.UserRanking(ctx, scope, actorType, sortBy, (page-1)*pageSize, pageSize)
	if err != nil {
		return UserRankingResult{}, err
	}
	// 账号总数与 coverage.estimated 由同一条扫描返回：estimated 只反映「本次排行计入的行」
	// （同主体、同匿名/注销排除口径），且不再为估算标记重复扫一遍同一内容范围。
	totals, err := contributionstats.UserRankingTotal(ctx, scope, actorType)
	if err != nil {
		return UserRankingResult{}, err
	}

	identities := make(map[uint64]*users.EntityComplete)
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserId)
	}
	if len(ids) > 0 {
		identities = users.GetMapByIdsContext(ctx, ids)
	}
	items := make([]UserRankingItem, 0, len(rows))
	for _, row := range rows {
		item := UserRankingItem{
			UserId:     row.UserId,
			ActorType:  actorType,
			TopicCount: row.TopicCount,
			ReplyCount: row.ReplyCount,
			LikeCount:  row.LikeCount,
		}
		if identity, ok := identities[row.UserId]; ok {
			item.Username = identity.Username
			item.Nickname = identity.Nickname
			item.ActorType = identity.ActorType
		}
		// 身份缺失只可能来自「聚合与身份回填之间账号被注销」的竞态（账号过滤器
		// users.ActorAuthorIDsQueryTx 已排除查询时点已注销的账号，软删行也读不到）。
		// 该行已经在同一口径下计入总数，因此保留行本身（昵称留空、actorType 沿用请求侧），
		// 不为此再加查询或丢掉与 total 对不上的行（评审 nit 的显式豁免）。
		items = append(items, item)
	}
	return UserRankingResult{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    totals.Total,
		Coverage: coverage(totals.Estimated),
	}, nil
}

// UserDetail 组装单账号明细（口径 R4/R5/R6/R8）：无贡献时汇总与序列全 0，不报错；
// 账号不存在或已注销返回 user.notFound；未知原因查库失败由调用方映射为统计失败。
func UserDetail(ctx context.Context, query DetailQuery) (UserDetailResult, error) {
	user, err := resolveUser(query.UserId, query.Username)
	if err != nil {
		return UserDetailResult{}, err
	}
	scope, err := parseScope(query.StartDate, query.EndDate, query.CategoryId)
	if err != nil {
		return UserDetailResult{}, err
	}
	if query.CategoryId > 0 {
		if err := requireCategory(ctx, query.CategoryId); err != nil {
			return UserDetailResult{}, err
		}
	}
	total, err := contributionstats.UserContentTotals(ctx, scope, user.Id)
	if err != nil {
		return UserDetailResult{}, err
	}
	topicDays, err := contributionstats.UserDailyCounts(ctx, scope, user.Id, contributionstats.KindTopic)
	if err != nil {
		return UserDetailResult{}, err
	}
	replyDays, err := contributionstats.UserDailyCounts(ctx, scope, user.Id, contributionstats.KindReply)
	if err != nil {
		return UserDetailResult{}, err
	}
	days, estimated := mergeDays(scope, topicDays, replyDays)
	return UserDetailResult{
		User: UserDetailIdentity{
			UserId:    user.Id,
			Username:  user.Username,
			Nickname:  user.Nickname,
			ActorType: user.ActorType,
		},
		Summary: UserDetailSummary{
			TopicCount: total.TopicCount,
			ReplyCount: total.ReplyCount,
			LikeCount:  total.LikeCount,
		},
		Days:     days,
		Coverage: coverage(estimated),
	}, nil
}

type scopeTotals struct {
	topic     contributionstats.ContentTotal
	reply     contributionstats.ContentTotal
	estimated bool
}

func siteTotals(ctx context.Context, scope contributionstats.Scope) (scopeTotals, error) {
	topic, err := contributionstats.SiteContentTotal(ctx, scope, contributionstats.KindTopic)
	if err != nil {
		return scopeTotals{}, err
	}
	reply, err := contributionstats.SiteContentTotal(ctx, scope, contributionstats.KindReply)
	if err != nil {
		return scopeTotals{}, err
	}
	return scopeTotals{topic: topic, reply: reply, estimated: topic.Estimated || reply.Estimated}, nil
}

// mergeDays 把主题/回复两条日序列合并成区间内完整的日期序列（升序），
// 缺失日为 0 且 estimated=false；返回的布尔值是整段区间是否含估算来源。
func mergeDays(scope contributionstats.Scope, topicRows, replyRows []contributionstats.DailyCount) ([]TrendDay, bool) {
	topicCounts := make(map[string]int64)
	replyCounts := make(map[string]int64)
	estimatedDays := make(map[string]bool)
	for _, row := range topicRows {
		topicCounts[row.Day] += row.Count
		estimatedDays[row.Day] = estimatedDays[row.Day] || row.Estimated
	}
	for _, row := range replyRows {
		replyCounts[row.Day] += row.Count
		estimatedDays[row.Day] = estimatedDays[row.Day] || row.Estimated
	}
	days := make([]TrendDay, 0, MaxRangeDays)
	estimated := false
	for day := scope.Start; day.Before(scope.End); day = day.AddDate(0, 0, 1) {
		key := day.Format(dateLayout)
		days = append(days, TrendDay{
			Date:       key,
			TopicCount: topicCounts[key],
			ReplyCount: replyCounts[key],
			Estimated:  estimatedDays[key],
		})
		estimated = estimated || estimatedDays[key]
	}
	return days, estimated
}

func countByCategory(rows []contributionstats.CategoryCount) map[uint64]int64 {
	counts := make(map[uint64]int64, len(rows))
	for _, row := range rows {
		counts[row.CategoryId] = row.Count
	}
	return counts
}

// listCategories 返回全部板块（无数据板块也要出现在汇总里）；
// category.All() 已按 sort、id 升序返回，调用方再按活动量重排。
func listCategories() []*category.Entity {
	return category.All()
}

func requireCategory(ctx context.Context, categoryId uint64) error {
	if categoryId == 0 {
		return invalidParams("categoryId")
	}
	entity, err := category.GetWithContext(ctx, categoryId)
	if err != nil || entity.Id == 0 {
		return invalidParams("categoryId")
	}
	return nil
}

// parseScope 校验日期并换算为本地日边界（左闭右开）。
func parseScope(startDate, endDate string, categoryId uint64) (contributionstats.Scope, error) {
	start, end, err := parseDateRange(startDate, endDate)
	if err != nil {
		return contributionstats.Scope{}, err
	}
	return contributionstats.Scope{Start: start, End: end, CategoryID: categoryId}, nil
}

// parseDateRange 返回 [起始日 00:00, 结束日次日 00:00) 的服务器本地时刻。
// 日期为空取缺省区间（近 7 天到当天）；格式、顺序与跨度都在这里拒绝（口径 R9）。
func parseDateRange(startDate, endDate string) (time.Time, time.Time, error) {
	now := time.Now()
	if startDate == "" {
		startDate = now.AddDate(0, 0, -defaultRangeDays).Format(dateLayout)
	}
	if endDate == "" {
		endDate = now.Format(dateLayout)
	}
	start, err := parseLocalDay(startDate)
	if err != nil {
		return time.Time{}, time.Time{}, invalidParams("startDate")
	}
	end, err := parseLocalDay(endDate)
	if err != nil {
		return time.Time{}, time.Time{}, invalidParams("endDate")
	}
	if end.Before(start) {
		// 区间倒置时归咎结束日（契约 fixtures 冻结为 endDate）。
		return time.Time{}, time.Time{}, invalidParams("endDate")
	}
	// 跨度按自然日含首尾计算；超限时随错误回传上限，界面与文档可直接展示（R9）。
	if days := int(end.Sub(start).Hours()/24) + 1; days > MaxRangeDays {
		return time.Time{}, time.Time{}, component.NewMessageError(
			component.MessageRequestInvalidParams,
			"请求参数验证失败",
			component.MessageParams{"field": "endDate", "maxRangeDays": MaxRangeDays},
		)
	}
	return start, end.AddDate(0, 0, 1), nil
}

// parseLocalDay 按服务器本地时区解析自然日；只接受严格的 YYYY-MM-DD。
func parseLocalDay(value string) (time.Time, error) {
	if len(value) != len(dateLayout) {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}
	parsed, err := time.ParseInLocation(dateLayout, value, time.Local)
	if err != nil || parsed.Format(dateLayout) != value {
		return time.Time{}, fmt.Errorf("invalid date %q", value)
	}
	return parsed, nil
}

// parsePaging 解析分页：缺省使用默认值，显式给出的越界值一律拒绝（不夹取）。
func parsePaging(page, pageSize *int) (int, int, error) {
	resolvedPage, resolvedSize := 1, DefaultPageSize
	if page != nil {
		if *page < 1 || *page > MaxPage {
			return 0, 0, invalidParams("page")
		}
		resolvedPage = *page
	}
	if pageSize != nil {
		if *pageSize < 1 || *pageSize > MaxPageSize {
			return 0, 0, invalidParams("pageSize")
		}
		resolvedSize = *pageSize
	}
	return resolvedPage, resolvedSize, nil
}

func parseActorType(actorType int) (int8, error) {
	switch actorType {
	case int(users.ActorTypeHuman):
		return users.ActorTypeHuman, nil
	case int(users.ActorTypeBot):
		return users.ActorTypeBot, nil
	default:
		return 0, invalidParams("actorType")
	}
}

func parseSortBy(sortBy string) (contributionstats.SortKey, error) {
	switch contributionstats.SortKey(strings.TrimSpace(sortBy)) {
	case "":
		return contributionstats.SortByTopics, nil
	case contributionstats.SortByTopics:
		return contributionstats.SortByTopics, nil
	case contributionstats.SortByReplies:
		return contributionstats.SortByReplies, nil
	case contributionstats.SortByLikes:
		return contributionstats.SortByLikes, nil
	default:
		return "", invalidParams("sortBy")
	}
}

// resolveUser 解析 userId/username（恰给一个）并拒绝已注销账号（口径 R6）。
func resolveUser(userId uint64, username string) (users.EntityComplete, error) {
	username = strings.TrimSpace(username)
	if userId == 0 && username == "" {
		return users.EntityComplete{}, invalidParams("userId")
	}
	if userId > 0 && username != "" {
		return users.EntityComplete{}, invalidParams("username")
	}
	if userId > 0 {
		entity, err := users.Get(userId)
		if err != nil || entity.Id == 0 || entity.DeletedAt.Valid {
			return users.EntityComplete{}, userNotFound()
		}
		return entity, nil
	}
	entity, err := users.GetByUsername(username)
	if err != nil || entity.Id == 0 || entity.DeletedAt.Valid {
		return users.EntityComplete{}, userNotFound()
	}
	return entity, nil
}

func coverage(estimated bool) Coverage {
	return Coverage{
		Timezone:     timezoneName(),
		MaxRangeDays: MaxRangeDays,
		Estimated:    estimated,
	}
}

// timezoneName 返回日切使用的服务器本地时区，格式「缩写 (UTC±HH:MM)」，
// 例如 CST (UTC+08:00)。TZ 未设置时 time.Local.String() 只有 "Local"，
// 但 Zone() 仍给出系统时区库的缩写与偏移，因此以 Zone() 为准。
func timezoneName() string {
	abbreviation, offset := time.Now().In(time.Local).Zone()
	if abbreviation == "" || abbreviation == "Local" {
		abbreviation = "Local"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s (UTC%s%02d:%02d)", abbreviation, sign, offset/3600, (offset%3600)/60)
}

func invalidParams(field string) error {
	return component.NewMessageError(
		component.MessageRequestInvalidParams,
		"请求参数验证失败",
		component.MessageParams{"field": field},
	)
}

func userNotFound() error {
	return component.NewMessageError(component.MessageUserNotFound, "用户不存在", nil)
}
