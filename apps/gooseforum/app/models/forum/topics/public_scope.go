package topics

import "fmt"

// PublicScopePredicate 返回「当前公开论坛主题」的规范谓词（带表别名）：
// 已发布（status=1）、已过审、未进入删除生命周期、且不是 wiki 页面。
//
// 新增公开列表/导出/统计入口一律以它为准，避免各自复制谓词导致口径漂移
// （历史问题：各入口手写等值条件，出现「列表可见但统计可见」之类的偏差）。
func PublicScopePredicate(alias string) string {
	return fmt.Sprintf(
		"%[1]s.status = 1 AND %[1]s.process_status = %[2]d AND %[1]s.visibility_status = '%[3]s' AND %[1]s.deleted_at IS NULL AND %[1]s.topic_type = %[4]d",
		alias, ProcessStatusNormal, VisibilityActive, TopicTypeForum,
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
