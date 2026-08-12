package moderationLog

import "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

type CursorPageQuery struct {
	Cursor, PageSize uint64
	ScopeCategoryIDs []uint64
}

func CursorPage(q CursorPageQuery) []Entity {
	var list []Entity
	if q.PageSize < 1 {
		q.PageSize = 20
	}
	b := builder()
	if q.Cursor > 0 {
		b.Where(queryopt.Lt("id", q.Cursor))
	}
	if len(q.ScopeCategoryIDs) > 0 {
		b.Where(`(
			(subject_type = ? AND EXISTS (
				SELECT 1 FROM topic_category_index idx
				WHERE idx.topic_id = moderation_logs.subject_id
				AND idx.category_id IN (?)
				AND idx.effective = ?
			)) OR
			(subject_type = ? AND EXISTS (
				SELECT 1 FROM posts scoped_posts
				JOIN topic_category_index idx ON idx.topic_id = scoped_posts.topic_id
				WHERE scoped_posts.id = moderation_logs.subject_id
				AND idx.category_id IN (?)
				AND idx.effective = ?
			)) OR
			(subject_type = ? AND EXISTS (
				SELECT 1 FROM reports scoped_reports
				JOIN topic_category_index idx ON idx.topic_id = scoped_reports.topic_id
				WHERE scoped_reports.id = moderation_logs.subject_id
				AND idx.category_id IN (?)
				AND idx.effective = ?
			))
		)`, SubjectTopic, q.ScopeCategoryIDs, 1, SubjectPost, q.ScopeCategoryIDs, 1, SubjectReport, q.ScopeCategoryIDs, 1)
	}
	b.Limit(int(q.PageSize)).Order(queryopt.Desc("id")).Find(&list)
	return list
}
