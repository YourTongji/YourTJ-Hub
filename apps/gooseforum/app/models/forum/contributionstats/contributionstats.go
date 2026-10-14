// Package contributionstats 提供管理后台「内容贡献统计」的只读数据访问。
//
// 统计对象是论坛内容表本身（topics / posts / topic_category_index /
// post_user_action），不是推荐系统的遥测事件：不读推荐原始事件，也不改变任何保留期限。
// 目录内的查询全部是只读聚合，写入路径不经过本包。
package contributionstats

import "time"

// ContentKind 区分统计的内容类型：主题与回复（回复 = post_no > 1，首楼属于主题）。
type ContentKind int

const (
	KindTopic ContentKind = iota
	KindReply
)

// SortKey 是用户排行支持的排序指标（白名单，其余取值在执行前被拒绝）。
type SortKey string

const (
	SortByTopics  SortKey = "topics"
	SortByReplies SortKey = "replies"
	SortByLikes   SortKey = "likes"
)

// Scope 是一次统计的只读口径范围。时间区间为左闭右开 [Start, End)，
// 调用方按服务器本地日换算成边界时刻（Start = 起始日 00:00，End = 结束日次日 00:00）。
type Scope struct {
	Start      time.Time
	End        time.Time
	CategoryID uint64 // 0 = 全站（不按板块过滤）
	UserID     uint64 // 0 = 不限作者
}

// CategoryCount 是某个板块下的一项计数（内容量或去重贡献人数）。
type CategoryCount struct {
	CategoryId uint64 `gorm:"column:category_id"`
	Count      int64  `gorm:"column:count"`
}

// ContentTotal 是内容维度的一行汇总：计数 + 该范围是否含估算时间归属。
type ContentTotal struct {
	Count     int64 `gorm:"column:count"`
	Estimated bool  `gorm:"column:estimated"`
}

// DailyCount 是单日的内容计数；Day 为服务器本地日 "YYYY-MM-DD"。
// Estimated 表示该日计入的内容里是否存在估算时间归属（首次公开时间缺失或为回填估算）。
type DailyCount struct {
	Day       string `gorm:"column:day"`
	Count     int64  `gorm:"column:count"`
	Estimated bool   `gorm:"column:estimated"`
}

// RankingRow 是用户排行中的一行：该账号在范围内贡献的主题、回复与当前有效获赞。
type RankingRow struct {
	UserId     uint64 `gorm:"column:user_id"`
	TopicCount int64  `gorm:"column:topic_count"`
	ReplyCount int64  `gorm:"column:reply_count"`
	LikeCount  int64  `gorm:"column:like_count"`
}

// UserContentTotal 是单个账号的内容汇总。
type UserContentTotal struct {
	TopicCount int64 `gorm:"column:topic_count"`
	ReplyCount int64 `gorm:"column:reply_count"`
	LikeCount  int64 `gorm:"column:like_count"`
}
