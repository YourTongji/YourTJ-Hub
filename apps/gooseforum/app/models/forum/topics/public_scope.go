package topics

import "fmt"

// PublicScopePredicate 返回「当前公开论坛主题」的规范谓词（带表别名）：
// 已发布（status=1）、已过审、未进入删除生命周期、且不是 wiki 页面。
//
// 新增公开列表/统计入口一律以它为准，避免各自复制谓词导致口径漂移
// （历史问题：各入口手写等值条件，出现「列表可见但统计可见」之类的偏差）。
// 只有单篇 Markdown 导出用 ExportScopePredicate（issue #1166 含 wiki）。
func PublicScopePredicate(alias string) string {
	return publishedBasePredicate(alias) + fmt.Sprintf(" AND %s.topic_type = %d", alias, TopicTypeForum)
}

// ExportScopePredicate 返回「可导出的单篇主题」谓词（带表别名）：与 PublicScopePredicate
// 相同的发布/审核/可见/未删除条件，主题类型按白名单放行（论坛话题 + wiki 页面），
// 新增的第三种类型默认不导出。
//
// issue #1166 决定恢复 wiki 页面的单篇 Markdown 导出（#1160 让单篇导出复用公开谓词后
// 误把 wiki 页面变成 404）；列表、索引/全文导出与内容统计口径仍以 PublicScopePredicate
// 为准，继续排除 wiki。
func ExportScopePredicate(alias string) string {
	return publishedBasePredicate(alias) + fmt.Sprintf(" AND %s.topic_type IN (%d, %d)", alias, TopicTypeForum, TopicTypeWiki)
}

// publishedBasePredicate 是两条公开谓词共用的发布/审核/可见/删除生命周期条件（带表别名）。
func publishedBasePredicate(alias string) string {
	return fmt.Sprintf(
		"%[1]s.status = 1 AND %[1]s.process_status = %[2]d AND %[1]s.visibility_status = '%[3]s' AND %[1]s.deleted_at IS NULL",
		alias, ProcessStatusNormal, VisibilityActive,
	)
}

// FirstPostVisiblePredicate 返回首楼可见性半连接条件（带表别名）：首楼已过审、
// 未软删、未进入删除生命周期。首楼被删除/擦除后主题无正文，继续公开展示会产生
// 「有标题无正文」的孤儿条目（issue #492），因此所有公开列表/导出入口都应附加本条件。
func FirstPostVisiblePredicate(alias string) string {
	return fmt.Sprintf(
		"(SELECT posts.process_status FROM posts WHERE posts.id = %[1]s.first_post_id AND posts.topic_id = %[1]s.id AND posts.deleted_at IS NULL AND posts.visibility_status = '%[2]s') = %[3]d",
		alias, VisibilityActive, ProcessStatusNormal,
	)
}
