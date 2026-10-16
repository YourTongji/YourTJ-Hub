package contributionstats

import (
	"context"
	"fmt"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

const (
	topicsTable = "topics"
	postsTable  = "posts"
	indexTable  = "topic_category_index"
	actionTable = "post_user_action"
)

// topicScopePredicate 复用 topics 归属方的规范公开谓词 + 首楼可见性半连接：
// 主题必须「当前公开」（已发布、已过审、未删除、非 wiki）且首楼仍可见（issue #492），
// 与公开列表/排行榜同一口径，不在本包复制等值条件。status + process_status 等值也让
// first_public_at 缺失时的 created_at 支路能走
// idx_topics_list_new(status, process_status, created_at DESC, id DESC)。
var topicScopePredicate = topics.PublicScopePredicate("t") + " AND " + topics.FirstPostVisiblePredicate("t")

// 内容范围谓词（口径 R1）。写成固定 SQL 字面量以保持聚合语句可读；
// 修改内容可见性口径时必须同步本组谓词与 topics 的规范谓词。
const (
	// replyScopePredicate：楼层自身公开未删除且不是首楼（回复 = post_no > 1）。
	replyScopePredicate = "p.deleted_at IS NULL AND p.visibility_status = 'ACTIVE' AND p.process_status = 0 AND p.post_no > 1"

	// 时间归属谓词（口径 R2）：有效时间 = 首次公开时间优先、缺失回退创建时间。
	// 拆成两个可索引分支（等价于 COALESCE 落在区间内），参数顺序均为 [start,end,start,end)。
	topicRangePredicate = "((t.first_public_at IS NOT NULL AND t.first_public_at >= ? AND t.first_public_at < ?) OR (t.first_public_at IS NULL AND t.created_at >= ? AND t.created_at < ?))"
	replyRangePredicate = "((p.first_public_at IS NOT NULL AND p.first_public_at >= ? AND p.first_public_at < ?) OR (p.first_public_at IS NULL AND p.created_at >= ? AND p.created_at < ?))"

	// 按用户归因时必须排除匿名行（口径 R4）：匿名只掩码展示，user_id 仍是真实作者。
	topicAuthorPredicate = "t.persona_uid = ''"
	replyAuthorPredicate = "p.persona_uid = '' AND p.is_anonymous = false"

	topicEffectiveExpr  = "COALESCE(t.first_public_at, t.created_at)"
	replyEffectiveExpr  = "COALESCE(p.first_public_at, p.created_at)"
	topicEstimatedExpr  = "(t.first_public_at IS NULL OR t.first_public_estimated)"
	replyEstimatedExpr  = "(p.first_public_at IS NULL OR p.first_public_estimated)"
	rankingOuterSelect  = "SELECT author_totals.user_id AS user_id, SUM(author_totals.topic_count) AS topic_count, SUM(author_totals.reply_count) AS reply_count, SUM(author_totals.like_count) AS like_count FROM (? UNION ALL ?) AS author_totals GROUP BY author_totals.user_id"
	rankingTotalSelect  = "SELECT COUNT(DISTINCT authors.user_id) AS total, COALESCE(MAX(authors.estimated), 0) AS estimated FROM (? UNION ALL ?) AS authors"
	userTotalSelect     = "SELECT COALESCE(SUM(author_totals.topic_count), 0) AS topic_count, COALESCE(SUM(author_totals.reply_count), 0) AS reply_count, COALESCE(SUM(author_totals.like_count), 0) AS like_count FROM (? UNION ALL ?) AS author_totals"
	contributorCountSQL = "SELECT COUNT(DISTINCT authors.user_id) AS total FROM (? UNION ALL ?) AS authors"
	contributorGroupSQL = "SELECT authors.category_id AS category_id, COUNT(DISTINCT authors.user_id) AS count FROM (? UNION ALL ?) AS authors GROUP BY authors.category_id ORDER BY authors.category_id ASC"
	categoryColumns     = "tci.category_id AS category_id, "
	categoryIndexJoin   = "JOIN topic_category_index AS tci ON tci.topic_id = t.id AND tci.effective = 1"
	replyLikeJoin       = "LEFT JOIN post_user_action AS pua ON pua.post_id = p.id AND pua.liked_at IS NOT NULL"
	topicRankingSelect  = "t.user_id AS user_id, COUNT(*) AS topic_count, 0 AS reply_count, COALESCE(SUM(t.like_count), 0) AS like_count"
	replyRankingSelect  = "p.user_id AS user_id, 0 AS topic_count, COUNT(DISTINCT p.id) AS reply_count, COUNT(pua.id) AS like_count"
	topicRankingGroupBy = "t.user_id"
	replyRankingGroupBy = "p.user_id"
	rankingOrderTopics  = "topic_count DESC, author_totals.user_id ASC"
	rankingOrderReplies = "reply_count DESC, author_totals.user_id ASC"
	rankingOrderLikes   = "like_count DESC, author_totals.user_id ASC"
)

// CategoryCounts 返回区间内每个板块的内容量（口径 R3）：多板块主题在每个所属板块各计一次。
func CategoryCounts(ctx context.Context, scope Scope, kind ContentKind) ([]CategoryCount, error) {
	return categoryCounts(db.ConnectContext(ctx), scope, kind)
}

// SiteContentTotal 返回全站内容量：不 JOIN 板块索引，因此多板块主题只计一次。
// 同时返回区间内是否存在估算时间归属的行，供 coverage 使用（口径 R8）。
func SiteContentTotal(ctx context.Context, scope Scope, kind ContentKind) (ContentTotal, error) {
	return siteContentTotal(db.ConnectContext(ctx), scope, kind)
}

// CategoryDailyCounts 返回每个板块逐日的内容量（升序），供板块趋势使用。
// 内容口径含匿名内容；只返回有内容的日期，缺失日由调用方补 0。
func CategoryDailyCounts(ctx context.Context, scope Scope, kind ContentKind) ([]DailyCount, error) {
	return categoryDailyCounts(db.ConnectContext(ctx), scope, kind)
}

// UserDailyCounts 返回某账号逐日的内容量（升序），按用户归因（排除该账号的匿名内容）。
func UserDailyCounts(ctx context.Context, scope Scope, userID uint64, kind ContentKind) ([]DailyCount, error) {
	return userDailyCounts(db.ConnectContext(ctx), scope, userID, kind)
}

// CategoryContributorCounts 返回区间内每个板块的去重贡献人数（口径 R4）：
// 主题作者与回复作者取并集后按账号去重，排除匿名行与已注销账号；只统计指定主体类型。
func CategoryContributorCounts(ctx context.Context, scope Scope, actorType int8) ([]CategoryCount, error) {
	return categoryContributorCounts(db.ConnectContext(ctx), scope, actorType)
}

// SiteContributorCount 返回全站去重贡献人数（口径 R4）：主题作者与回复作者并集去重，
// 排除匿名行与已注销账号；只统计指定主体类型。
func SiteContributorCount(ctx context.Context, scope Scope, actorType int8) (int64, error) {
	return siteContributorCount(db.ConnectContext(ctx), scope, actorType)
}

// UserRanking 返回区间内的用户贡献排行（只含指定主体类型；匿名与已注销账号不出现）。
// 排序稳定：同值按 user_id 升序。
func UserRanking(ctx context.Context, scope Scope, actorType int8, sort SortKey, offset, limit int) ([]RankingRow, error) {
	return userRanking(db.ConnectContext(ctx), scope, actorType, sort, offset, limit)
}

// UserRankingTotal 返回同一筛选下的账号总数（分页依据）与该范围内是否有估算时间归属。
// 两者在同一条 UNION 扫描里产出，因此 estimated 与 total 描述的是同一批计入的行。
func UserRankingTotal(ctx context.Context, scope Scope, actorType int8) (RankingTotal, error) {
	return userRankingTotal(db.ConnectContext(ctx), scope, actorType)
}

// UserContentTotals 返回单个账号在区间内贡献的内容量与当前有效获赞（口径 R5）。
func UserContentTotals(ctx context.Context, scope Scope, userID uint64) (UserContentTotal, error) {
	return userContentTotals(db.ConnectContext(ctx), scope, userID)
}

// 以下查询实现都接受显式连接：默认路径由上面的导出函数注入全局连接，
// PostgreSQL 方言测试可直接传入真实 PG 连接复用同一份 SQL。

func categoryCounts(conn *gorm.DB, scope Scope, kind ContentKind) ([]CategoryCount, error) {
	query := contentQuery(conn, scope, kind).Joins(categoryIndexJoin)
	if scope.CategoryID > 0 {
		query = query.Where("tci.category_id = ?", scope.CategoryID)
	}
	var rows []CategoryCount
	err := query.
		Select("tci.category_id AS category_id, COUNT(*) AS count").
		Group("tci.category_id").
		Order("tci.category_id ASC").
		Scan(&rows).Error
	return rows, err
}

func siteContentTotal(conn *gorm.DB, scope Scope, kind ContentKind) (ContentTotal, error) {
	var row ContentTotal
	err := contentQuery(conn, scope, kind).
		Select("COUNT(*) AS count, " + estimatedFlagSelect(estimatedExpr(kind))).
		Scan(&row).Error
	return row, err
}

func categoryDailyCounts(conn *gorm.DB, scope Scope, kind ContentKind) ([]DailyCount, error) {
	// 板块筛选走 contentQuery 的 IN 子查询，不再 JOIN 索引表：
	// 逐日分组时 JOIN 会让跨板块主题在未给板块筛选时重复计数。
	return scanDailyCounts(contentQuery(conn, scope, kind), kind)
}

func userDailyCounts(conn *gorm.DB, scope Scope, userID uint64, kind ContentKind) ([]DailyCount, error) {
	scope.UserID = userID
	return scanDailyCounts(authorQuery(conn, scope, nil, kind), kind)
}

func categoryContributorCounts(conn *gorm.DB, scope Scope, actorType int8) ([]CategoryCount, error) {
	// 指定板块时两条作者子查询都必须按该板块收窄，否则会带出其他板块的分组行。
	// 收窄只走索引表 JOIN 的 tci.category_id 等值：这里清掉 scope 的板块谓词，
	// 免得再叠一层等价的 `t.id IN (子查询)`（评审 nit：双重过滤冗余）。
	joinedScope := scope
	joinedScope.CategoryID = 0
	topicAuthors := authorQuery(conn, joinedScope, &actorType, KindTopic).
		Joins(categoryIndexJoin).
		Select(categoryColumns + "t.user_id AS user_id")
	replyAuthors := authorQuery(conn, joinedScope, &actorType, KindReply).
		Joins(categoryIndexJoin).
		Select(categoryColumns + "p.user_id AS user_id")
	if scope.CategoryID > 0 {
		topicAuthors = topicAuthors.Where("tci.category_id = ?", scope.CategoryID)
		replyAuthors = replyAuthors.Where("tci.category_id = ?", scope.CategoryID)
	}
	var rows []CategoryCount
	err := conn.Raw(contributorGroupSQL, topicAuthors, replyAuthors).Scan(&rows).Error
	return rows, err
}

func siteContributorCount(conn *gorm.DB, scope Scope, actorType int8) (int64, error) {
	topicAuthors := authorQuery(conn, scope, &actorType, KindTopic).Select("t.user_id AS user_id")
	replyAuthors := authorQuery(conn, scope, &actorType, KindReply).Select("p.user_id AS user_id")
	var row struct {
		Total int64 `gorm:"column:total"`
	}
	err := conn.Raw(contributorCountSQL, topicAuthors, replyAuthors).Scan(&row).Error
	return row.Total, err
}

func userRanking(conn *gorm.DB, scope Scope, actorType int8, sort SortKey, offset, limit int) ([]RankingRow, error) {
	var rows []RankingRow
	err := conn.
		Raw(rankingOuterSelect+" ORDER BY "+rankingOrder(sort)+" LIMIT ? OFFSET ?",
			topicRankingQuery(conn, scope, &actorType),
			replyRankingQuery(conn, scope, &actorType),
			limit, offset).
		Scan(&rows).Error
	return rows, err
}

func userRankingTotal(conn *gorm.DB, scope Scope, actorType int8) (RankingTotal, error) {
	var row RankingTotal
	err := conn.
		Raw(rankingTotalSelect,
			topicRankingQuery(conn, scope, &actorType),
			replyRankingQuery(conn, scope, &actorType)).
		Scan(&row).Error
	return row, err
}

func userContentTotals(conn *gorm.DB, scope Scope, userID uint64) (UserContentTotal, error) {
	scope.UserID = userID
	var row UserContentTotal
	err := conn.
		Raw(userTotalSelect,
			topicRankingQuery(conn, scope, nil),
			replyRankingQuery(conn, scope, nil)).
		Scan(&row).Error
	return row, err
}

// contentQuery 组装内容范围查询：主题或回复，别名固定（t / p）。
func contentQuery(conn *gorm.DB, scope Scope, kind ContentKind) *gorm.DB {
	switch kind {
	case KindReply:
		query := conn.
			Table(postsTable+" AS p").
			Joins("JOIN "+topicsTable+" AS t ON t.id = p.topic_id").
			Where(topicScopePredicate).
			Where(replyScopePredicate).
			Where(replyRangePredicate, rangeArgs(scope)...)
		if scope.UserID > 0 {
			query = query.Where("p.user_id = ?", scope.UserID)
		}
		return applyCategoryScope(conn, query, scope)
	default:
		query := conn.
			Table(topicsTable+" AS t").
			Where(topicScopePredicate).
			Where(topicRangePredicate, rangeArgs(scope)...)
		if scope.UserID > 0 {
			query = query.Where("t.user_id = ?", scope.UserID)
		}
		return applyCategoryScope(conn, query, scope)
	}
}

// authorQuery 在内容范围之上叠加按用户归因的谓词（口径 R4）：匿名行排除，
// actorType 非 nil 时只保留该主体类型且未注销的账号。
func authorQuery(conn *gorm.DB, scope Scope, actorType *int8, kind ContentKind) *gorm.DB {
	query := contentQuery(conn, scope, kind)
	switch kind {
	case KindReply:
		query = query.Where(replyAuthorPredicate)
		if actorType != nil {
			query = query.Where("p.user_id IN (?)", users.ActorAuthorIDsQueryTx(conn, *actorType))
		}
	default:
		query = query.Where(topicAuthorPredicate)
		if actorType != nil {
			query = query.Where("t.user_id IN (?)", users.ActorAuthorIDsQueryTx(conn, *actorType))
		}
	}
	return query
}

// topicRankingQuery 按作者聚合主题数与主题获赞（R5：获赞取 topics.like_count 之和）。
// 每个作者顺带产出估算标记，供总数扫描在同一个 UNION 里取 MAX（口径 R8）。
func topicRankingQuery(conn *gorm.DB, scope Scope, actorType *int8) *gorm.DB {
	return authorQuery(conn, scope, actorType, KindTopic).
		Select(topicRankingSelect + ", " + estimatedFlagSelect(topicEstimatedExpr)).
		Group(topicRankingGroupBy)
}

// replyRankingQuery 按作者聚合回复数与回复获赞（R5：只统计 liked_at 非空的当前有效点赞，
// 取消点赞只置空该列不删行）。
func replyRankingQuery(conn *gorm.DB, scope Scope, actorType *int8) *gorm.DB {
	return authorQuery(conn, scope, actorType, KindReply).
		Joins(replyLikeJoin).
		Select(replyRankingSelect + ", " + estimatedFlagSelect(replyEstimatedExpr)).
		Group(replyRankingGroupBy)
}

func applyCategoryScope(conn *gorm.DB, query *gorm.DB, scope Scope) *gorm.DB {
	if scope.CategoryID == 0 {
		return query
	}
	return query.Where("t.id IN (?)", categoryTopicIDsQuery(conn, scope.CategoryID))
}

func categoryTopicIDsQuery(conn *gorm.DB, categoryID uint64) *gorm.DB {
	return conn.
		Table(indexTable).
		Select("topic_id").
		Where("effective = 1").
		Where("category_id = ?", categoryID)
}

func scanDailyCounts(query *gorm.DB, kind ContentKind) ([]DailyCount, error) {
	dayExpr := localDayExpr(query, effectiveExpr(kind))
	var rows []DailyCount
	err := query.
		Select(dayExpr + " AS day, COUNT(*) AS count, " + estimatedFlagSelect(estimatedExpr(kind))).
		Group(dayExpr).
		Order("day ASC").
		Scan(&rows).Error
	return rows, err
}

// localDayExpr 把时间表达式截断为「服务器本地日」字符串（'YYYY-MM-DD'），
// 与 dailyStats 写入侧（time.Now().Format("2006-01-02")）同口径。
//
// 两种方言都按「绝对时刻 + 本地时区偏移后取 UTC 日历日」换算，语义与
// time.Time.In(time.Local) 一致；偏移量取查询时刻的 time.Local 偏移，
// 只在无夏令时的服务器时区（部署约束，见 docs/operations/deployment.md 的
// 「服务器时区：必须无夏令时」）与逐日换算完全一致，测试兜底见
// TestDeploymentTimezoneHasNoDST。SQLite 用 strftime 的秒偏移修饰符、
// PostgreSQL 用 make_interval，避免依赖数据库会话时区，也避免 BOOL_OR 这类方言专有聚合。
func localDayExpr(conn *gorm.DB, effective string) string {
	_, offset := time.Now().In(time.Local).Zone()
	// gorm 内嵌 Dialector 接口，conn.Name() 即方言名（postgres / sqlite）。
	if conn.Name() == "postgres" {
		return fmt.Sprintf("to_char((%s + make_interval(secs => %d)) AT TIME ZONE 'UTC', 'YYYY-MM-DD')", effective, offset)
	}
	return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, '%+d seconds')", effective, offset)
}

// estimatedFlagSelect 返回「该范围是否含估算时间归属」的聚合片段（口径 R8）。
// 站点汇总、逐日分桶与排行子查询共用同一段 SQL 文本，避免同一标记出现多种写法。
func estimatedFlagSelect(estimatedExpr string) string {
	return "MAX(CASE WHEN " + estimatedExpr + " THEN 1 ELSE 0 END) AS estimated"
}

func estimatedExpr(kind ContentKind) string {
	if kind == KindReply {
		return replyEstimatedExpr
	}
	return topicEstimatedExpr
}

func effectiveExpr(kind ContentKind) string {
	if kind == KindReply {
		return replyEffectiveExpr
	}
	return topicEffectiveExpr
}

func rangeArgs(scope Scope) []any {
	return []any{scope.Start, scope.End, scope.Start, scope.End}
}

// rankingOrder 返回白名单内的排序片段；未知取值回退主题数（服务层已拒绝非法值）。
func rankingOrder(sort SortKey) string {
	switch sort {
	case SortByReplies:
		return rankingOrderReplies
	case SortByLikes:
		return rankingOrderLikes
	default:
		return rankingOrderTopics
	}
}
