package badgeservice

import (
	"net/url"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/badges"
)

const (
	CodeFirstPost      = "first_post"
	CodeFirstComment   = "first_comment"
	CodeFirstLikeGiven = "first_like_given"
	CodeFirstFollower  = "first_follower"
	CodeWriter10       = "writer_10"
	CodeCommenter50    = "commenter_50"
	CodeLiked10        = "liked_10"
	CodePopular100     = "popular_100"
	CodeSocial10       = "social_10"
	CodeEarlyMember    = "early_member"
	CodeContributor    = "contributor"
	CodeModerator      = "moderator"
	CodeSponsor        = "sponsor"
	CodeKing           = "king"
	CodeRobot          = "robot"
)

const (
	LevelBronze  = "bronze"
	LevelSilver  = "silver"
	LevelGold    = "gold"
	LevelSpecial = "special"
)

// systemDefinitions 返回内置徽章集合。系统徽章 SVG 已按统一光学尺寸归一化
// （墨迹半径 10/24，缩放时描边按 1/scale 补偿以保持屏幕粗细不变）；新增
// 系统徽章须沿用同一尺寸。改动 static/badges 下任何 SVG 都要递增
// badgeAssetVersion。约定与被否决方案见
// docs/decisions/0054-badge-artwork-optical-size.md。
func systemDefinitions() []Badge {
	return []Badge{
		{Code: CodeFirstPost, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "初次发帖", Description: "发布了第一篇主题", IconType: badges.IconTypeAsset, IconURL: "/static/badges/first-post.svg", Color: "blue", Level: LevelBronze, IsEnabled: true, IsWearable: false, SortOrder: 10},
		{Code: CodeFirstComment, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "初次评论", Description: "留下了第一条评论或回复", IconType: badges.IconTypeAsset, IconURL: "/static/badges/first-comment.svg", Color: "teal", Level: LevelBronze, IsEnabled: true, IsWearable: false, SortOrder: 20},
		{Code: CodeFirstLikeGiven, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "友善点赞", Description: "第一次为他人的内容点赞", IconType: badges.IconTypeAsset, IconURL: "/static/badges/first-like-given.svg", Color: "rose", Level: LevelBronze, IsEnabled: true, IsWearable: false, SortOrder: 30},
		{Code: CodeFirstFollower, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "被看见了", Description: "获得了第一位粉丝", IconType: badges.IconTypeAsset, IconURL: "/static/badges/first-follower.svg", Color: "violet", Level: LevelBronze, IsEnabled: true, IsWearable: false, SortOrder: 40},
		{Code: CodeWriter10, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "持续创作", Description: "累计发布 10 篇主题", IconType: badges.IconTypeAsset, IconURL: "/static/badges/writer-10.svg", Color: "sky", Level: LevelSilver, IsEnabled: true, IsWearable: true, SortOrder: 50},
		{Code: CodeCommenter50, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "热心讨论", Description: "累计发布 50 条评论或回复", IconType: badges.IconTypeAsset, IconURL: "/static/badges/commenter-50.svg", Color: "emerald", Level: LevelSilver, IsEnabled: true, IsWearable: true, SortOrder: 60},
		{Code: CodeLiked10, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "受到认可", Description: "累计获得 10 个赞", IconType: badges.IconTypeAsset, IconURL: "/static/badges/liked-10.svg", Color: "amber", Level: LevelSilver, IsEnabled: true, IsWearable: true, SortOrder: 70},
		{Code: CodePopular100, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "社区之光", Description: "累计获得 100 个赞", IconType: badges.IconTypeAsset, IconURL: "/static/badges/popular-100.svg", Color: "orange", Level: LevelGold, IsEnabled: true, IsWearable: true, SortOrder: 80},
		{Code: CodeSocial10, Type: badges.TypeSystem, GrantMode: badges.GrantModeAuto, Name: "小有名气", Description: "累计获得 10 位粉丝", IconType: badges.IconTypeAsset, IconURL: "/static/badges/social-10.svg", Color: "purple", Level: LevelSilver, IsEnabled: true, IsWearable: true, SortOrder: 90},
		{Code: CodeEarlyMember, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "早期成员", Description: "社区早期加入者", IconType: badges.IconTypeAsset, IconURL: "/static/badges/early-member.svg", Color: "cyan", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 100},
		{Code: CodeContributor, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "贡献者", Description: "为社区建设做出贡献", IconType: badges.IconTypeAsset, IconURL: "/static/badges/contributor.svg", Color: "fuchsia", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 110},
		{Code: CodeModerator, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "社区维护者", Description: "协助维护社区秩序", IconType: badges.IconTypeAsset, IconURL: "/static/badges/moderator.svg", Color: "emerald", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 120},
		{Code: CodeSponsor, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "赞助者", Description: "支持社区持续运行", IconType: badges.IconTypeAsset, IconURL: "/static/badges/sponsor.svg", Color: "yellow", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 130},
		{Code: CodeKing, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "King", Description: "社区之王", IconType: badges.IconTypeAsset, IconURL: "/static/badges/king.svg", Color: "amber", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 140},
		{Code: CodeRobot, Type: badges.TypeSystem, GrantMode: badges.GrantModeManual, Name: "机器人", Description: "你就是机器人！", IconType: badges.IconTypeAsset, IconURL: "/static/badges/robot.svg", Color: "slate", Level: LevelSpecial, IsEnabled: true, IsWearable: true, SortOrder: 150},
	}
}

// badgeAssetVersion 是内置徽章图形的版本号。/static/* 在生产环境带约 210 天的
// 公共缓存且不做重验证，图形改了而 URL 不变，老用户就会一直看到旧图形。
// 改动 static/badges 下任何 SVG 都必须递增本常量，否则
// TestBadgeAssetsHashMatches 会失败。
const badgeAssetVersion = "2"

// badgeAssetHash 是 static/badges 下全部 SVG（按文件名排序、内容归一化掉 \r）
// 的 SHA-256。它是 badgeAssetVersion 的机械守卫：图形变了而版本号没递增时
// 测试失败，错误信息里会打印新哈希；确认要改图形时，递增 badgeAssetVersion
// 并把本常量更新为新哈希。
const badgeAssetHash = "3ae463f76b5f36b5ee83b6211e5910c7628ff06e4c36eaca7433fef0aa11a263"

const badgeAssetPrefix = "/static/badges/"

// versionedBadgeIconURL 给内置徽章图形 URL 附上当前版本号；已带的旧版本号
// （例如管理端编辑后回存进覆盖记录的 URL）会被替换，其它查询参数与 fragment
// 原样保留；非内置图形 URL 原样返回。
func versionedBadgeIconURL(raw string) string {
	if !strings.HasPrefix(raw, badgeAssetPrefix) {
		return raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		// 解析失败时保守处理：丢弃 query 后附加版本号。
		path, _, _ := strings.Cut(raw, "?")
		return path + "?v=" + badgeAssetVersion
	}
	q := u.Query()
	q.Set("v", badgeAssetVersion)
	u.RawQuery = q.Encode()
	return u.String()
}
