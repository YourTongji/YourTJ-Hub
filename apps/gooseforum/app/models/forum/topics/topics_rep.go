package topics

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/jsonopt"
	"github.com/leancodebox/GooseForum/app/bundles/pageutil"
	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
	"gorm.io/gorm"
)

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return builder().Create(entity).RowsAffected
	}

	return builder().Save(entity).RowsAffected
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func Delete(entity *Entity) int64 {
	return builder().Delete(entity).RowsAffected
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

func SaveNoUpdate(entity *Entity) error {
	return builder().Omit("updated_at").Save(entity).Error
}

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

func GetSimple(id any) (entity Entity) {
	builder().Where(queryopt.Eq("id", id)).First(&entity)
	return
}

func GetMaxId() uint64 {
	var entity Entity
	builder().Order(queryopt.Desc("id")).Limit(1).First(&entity)
	return entity.Id
}

func QueryById(startId uint64, limit int) (entities []*Entity) {
	builder().Where(queryopt.Gt("id", startId)).Limit(limit).Order(queryopt.Asc("id")).Find(&entities)
	return
}

func GetMapByIds(ids []uint64) map[uint64]Entity {
	var list []Entity
	if len(ids) == 0 {
		return map[uint64]Entity{}
	}
	builder().Where("id in ?", ids).Find(&list)
	result := make(map[uint64]Entity, len(list))
	for _, item := range list {
		result[item.Id] = item
	}
	return result
}

func GetPointerMapByIds(ids []uint64) map[uint64]*Entity {
	valueMap := GetMapByIds(ids)
	result := make(map[uint64]*Entity, len(valueMap))
	for id, item := range valueMap {
		entity := item
		result[id] = &entity
	}
	return result
}

// GetMapByIdsUnscoped 返回含已删除（软删）在内的主题 map。
// 举报审核需要基于快照继续处理已被作者删除的目标，不能因软删过滤而丢失。
func GetMapByIdsUnscoped(ids []uint64) map[uint64]Entity {
	var list []Entity
	if len(ids) == 0 {
		return map[uint64]Entity{}
	}
	builder().Unscoped().Where("id in ?", ids).Find(&list)
	result := make(map[uint64]Entity, len(list))
	for _, item := range list {
		result[item.Id] = item
	}
	return result
}

func GetLatestPublished(limit int) (entities []*Entity, err error) {
	err = builder().
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", 0)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Order(queryopt.Desc("updated_at")).
		Order(queryopt.Desc("id")).
		Limit(limit).
		Find(&entities).Error
	return
}

func GetPublishedAfterID(afterID uint64, limit int) (entities []*Entity, err error) {
	if limit <= 0 {
		return []*Entity{}, nil
	}
	err = builder().
		Where(queryopt.Gt("id", afterID)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", ProcessStatusNormal)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Where("EXISTS (SELECT 1 FROM posts WHERE posts.id = topics.first_post_id AND posts.topic_id = topics.id AND posts.process_status = ? AND posts.deleted_at IS NULL)", ProcessStatusNormal).
		Order(queryopt.Asc("id")).
		Limit(limit).
		Find(&entities).Error
	return
}

// GetPublishedBeforeID 按 id 倒序分页返回已发布主题（游标 id < beforeID）。
// 用于需要"最新优先"的全量遍历（如 llms 导出，超限时保留最新内容而非最旧）。
func GetPublishedBeforeID(beforeID uint64, limit int) (entities []*Entity, err error) {
	if limit <= 0 {
		return []*Entity{}, nil
	}
	err = builder().
		Where(queryopt.Lt("id", beforeID)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", ProcessStatusNormal)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Where("EXISTS (SELECT 1 FROM posts WHERE posts.id = topics.first_post_id AND posts.topic_id = topics.id AND posts.process_status = ? AND posts.deleted_at IS NULL)", ProcessStatusNormal).
		Order(queryopt.Desc("id")).
		Limit(limit).
		Find(&entities).Error
	return
}

func GetPublished(id uint64) (entity Entity, err error) {
	err = builder().
		Where(queryopt.Eq("id", id)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", ProcessStatusNormal)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Where("EXISTS (SELECT 1 FROM posts WHERE posts.id = topics.first_post_id AND posts.topic_id = topics.id AND posts.process_status = ? AND posts.deleted_at IS NULL)", ProcessStatusNormal).
		First(&entity).Error
	return
}

func GetLatestPublishedByUserId(userId uint64, limit int) ([]*Entity, error) {
	var entities []*Entity
	err := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", 0)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Order(queryopt.Desc("updated_at")).
		Order(queryopt.Desc("id")).
		Limit(limit).
		Find(&entities).Error
	return entities, err
}

func GetPublishedByUserBeforeId(userId uint64, beforeId uint64, limit int) ([]*Entity, error) {
	var entities []*Entity
	query := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("process_status", 0)).
		Where(queryopt.Eq("visibility_status", VisibilityActive))
	if beforeId > 0 {
		query = query.Where(queryopt.Lt("id", beforeId))
	}
	err := query.Order(queryopt.Desc("id")).Limit(limit).Find(&entities).Error
	return entities, err
}

func GetDraftsByUserId(userId uint64, limit int) ([]*Entity, error) {
	var entities []*Entity
	err := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("status", 0)).
		Order(queryopt.Desc("updated_at")).
		Order(queryopt.Desc("id")).
		Limit(limit).
		Find(&entities).Error
	return entities, err
}

// GetActiveByUserPage 分页返回本人仍公开（status=1 且 ACTIVE）的话题（PRD R9）。
func GetActiveByUserPage(userId uint64, cursorID uint64, limit int) (entities []Entity) {
	b := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("status", 1)).
		Where(queryopt.Eq("visibility_status", VisibilityActive))
	if cursorID != 0 {
		b = b.Where(queryopt.Lt("id", cursorID))
	}
	b.Order(queryopt.Desc("id")).
		Limit(pageutil.BoundPageSize(limit) + 1).
		Find(&entities)
	return
}

func CantWriteNew(userId uint64, maxCount int64) bool {
	var count int64
	builder().Where(queryopt.Eq("user_id", userId)).Where(queryopt.Gt("created_at", time.Now().Format("2006-01-02"))).Count(&count)
	return count > maxCount
}

type PageQuery struct {
	Page, PageSize int
	Search         string
	UserId         uint64
	FilterStatus   bool
	CategoryId     uint64
	Sort           string
}

type AdminPageQuery struct {
	Page, PageSize int
	Search         string
	UserId         uint64
}

type ModerationPageQuery struct {
	Page, PageSize      int
	FilterProcessStatus bool
	ProcessStatus       int8
	CategoryIDs         []uint64
}

func Page(q PageQuery) struct {
	Page     int
	PageSize int
	HasNext  bool
	Data     []Entity
} {
	var list []Entity
	q.Page = max(q.Page-1, 0)
	q.PageSize = pageutil.BoundPageSize(q.PageSize)
	queryLimit := q.PageSize + 1
	b := builder()
	if q.Search != "" {
		b.Where(queryopt.Like("title", q.Search))
	}
	if q.UserId != 0 {
		b.Where(queryopt.Eq("user_id", q.UserId))
	}
	if q.FilterStatus {
		b.Where(queryopt.Eq("status", 1))
		b.Where(queryopt.Eq("process_status", 0))
		// 公开列表只展示可见性正常的主题；作者删除（USER_DELETED）与管理端
		// 删除（MODERATOR_REMOVED）的话题一律不进首页/分类/Agent 列表，
		// 避免删除后仍公开泄露标题与正文摘录。
		b.Where(queryopt.Eq("visibility_status", VisibilityActive))
	}
	if q.CategoryId != 0 {
		b.Where(
			`EXISTS (SELECT 1 FROM topic_category_index idx WHERE idx.topic_id = topics.id AND idx.category_id = ? AND idx.effective = ?)`,
			q.CategoryId,
			1,
		)
	}
	applyPageSort(b, q.Sort)
	b.Limit(queryLimit).Offset(q.PageSize * q.Page).Find(&list)
	hasNext := len(list) > q.PageSize
	if hasNext {
		list = list[:q.PageSize]
	}
	return struct {
		Page     int
		PageSize int
		HasNext  bool
		Data     []Entity
	}{Page: q.Page + 1, PageSize: q.PageSize, Data: list, HasNext: hasNext}
}

func PageForAdmin(q AdminPageQuery) struct {
	Page     int
	PageSize int
	HasNext  bool
	Data     []Entity
} {
	var list []Entity
	q.Page = max(q.Page-1, 0)
	q.PageSize = pageutil.BoundPageSize(q.PageSize)
	queryLimit := q.PageSize + 1
	b := builder()
	if q.Search != "" {
		b.Where(queryopt.Like("title", q.Search))
	}
	if q.UserId != 0 {
		b.Where(queryopt.Eq("user_id", q.UserId))
	}
	b.Limit(queryLimit).Offset(q.PageSize * q.Page).Order(queryopt.Desc("pin_weight")).Order(queryopt.Desc("updated_at")).Order(queryopt.Desc("id")).Find(&list)
	hasNext := len(list) > q.PageSize
	if hasNext {
		list = list[:q.PageSize]
	}
	return struct {
		Page     int
		PageSize int
		HasNext  bool
		Data     []Entity
	}{Page: q.Page + 1, PageSize: q.PageSize, Data: list, HasNext: hasNext}
}

func PageForModeration(q ModerationPageQuery) struct {
	Page     int
	PageSize int
	Total    int64
	HasNext  bool
	Data     []Entity
} {
	var list []Entity
	q.Page = max(q.Page-1, 0)
	q.PageSize = pageutil.BoundPageSize(q.PageSize)
	queryLimit := q.PageSize + 1
	b := builder().Where(queryopt.Eq("status", 1))
	if q.FilterProcessStatus {
		b.Where(queryopt.Eq("process_status", q.ProcessStatus))
	}
	if len(q.CategoryIDs) > 0 {
		b.Where(
			`EXISTS (SELECT 1 FROM topic_category_index idx WHERE idx.topic_id = topics.id AND idx.category_id IN (?) AND idx.effective = ?)`,
			q.CategoryIDs,
			1,
		)
	}
	b.Limit(queryLimit).Offset(q.PageSize * q.Page).Order(queryopt.Desc("updated_at")).Order(queryopt.Desc("id")).Find(&list)
	hasNext := len(list) > q.PageSize
	if hasNext {
		list = list[:q.PageSize]
	}
	total := int64(q.Page*q.PageSize + len(list))
	if hasNext {
		total++
	}
	return struct {
		Page     int
		PageSize int
		Total    int64
		HasNext  bool
		Data     []Entity
	}{Page: q.Page + 1, PageSize: q.PageSize, Data: list, Total: total, HasNext: hasNext}
}

// PagePendingReview 列出待审（ProcessStatus=2）的主题。
func PagePendingReview(page, pageSize int) struct {
	Page     int
	PageSize int
	Total    int64
	Data     []Entity
} {
	var list []Entity
	page = max(page-1, 0)
	pageSize = pageutil.BoundPageSize(pageSize)
	b := builder().
		Where(queryopt.Eq("process_status", ProcessStatusPending)).
		Order(queryopt.Desc("updated_at")).
		Order(queryopt.Desc("id"))
	var total int64
	b.Session(&gorm.Session{}).Count(&total)
	b.Limit(pageSize).Offset(pageSize * page).Find(&list)
	return struct {
		Page     int
		PageSize int
		Total    int64
		Data     []Entity
	}{Page: page + 1, PageSize: pageSize, Total: total, Data: list}
}

func UpdateProcessStatus(id uint64, processStatus int8) error {
	return builder().Where(queryopt.Eq("id", id)).UpdateColumn("process_status", processStatus).Error
}

func UpdatePinWeight(id uint64, pinWeight int) error {
	return builder().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"pin_weight": pinWeight,
	}).Error
}

func IncrementLike(entity Entity) int64 {
	return builder().Exec("UPDATE topics SET like_count = like_count + 1 WHERE id = ?", entity.Id).RowsAffected
}

func DecrementLike(entity Entity) int64 {
	return builder().Exec("UPDATE topics SET like_count = like_count - 1 WHERE id = ?", entity.Id).RowsAffected
}

func IncrementViews(counts map[uint64]uint64) error {
	for topicID, count := range counts {
		if topicID == 0 || count == 0 {
			continue
		}
		if err := builder().Exec("UPDATE topics SET view_count = view_count + ? WHERE id = ?", count, topicID).Error; err != nil {
			return err
		}
	}
	return nil
}

func IncrementPostFast(topicId uint64, posters []Poster, lastPostID uint64, lastPostedAt time.Time) error {
	return builder().Where("id = ?", topicId).Updates(map[string]any{
		"post_count":  gorm.Expr("post_count + 1"),
		"reply_count": gorm.Expr("reply_count + 1"),
		"posters":     jsonopt.Encode(posters),
		"last_post_id": gorm.Expr(
			"CASE WHEN last_posted_at IS NULL OR last_posted_at < ? OR (last_posted_at = ? AND last_post_id < ?) THEN ? ELSE last_post_id END",
			lastPostedAt, lastPostedAt, lastPostID, lastPostID,
		),
		"last_posted_at": gorm.Expr(
			"CASE WHEN last_posted_at IS NULL OR last_posted_at < ? THEN ? ELSE last_posted_at END",
			lastPostedAt, lastPostedAt,
		),
		"updated_at": time.Now(),
	}).Error
}

func DecrementPostFast(topicId uint64, posters []Poster, lastPostID uint64, lastPostedAt time.Time) error {
	return builder().Where("id = ?", topicId).Updates(map[string]any{
		"post_count":     gorm.Expr("CASE WHEN post_count > 0 THEN post_count - 1 ELSE 0 END"),
		"reply_count":    gorm.Expr("CASE WHEN reply_count > 0 THEN reply_count - 1 ELSE 0 END"),
		"posters":        jsonopt.Encode(posters),
		"last_post_id":   lastPostID,
		"last_posted_at": lastPostedAt,
	}).Error
}

// ReplacePostStats writes the exact derived post counters for a topic.
// Recovery must use this instead of increment/decrement helpers because those
// helpers intentionally model a single state transition, not a full rebuild.
func ReplacePostStats(topicID uint64, postCount uint64, replyCount uint64, posters []Poster, lastPostID uint64, lastPostedAt time.Time) error {
	return builder().Where("id = ?", topicID).Updates(map[string]any{
		"post_count":     postCount,
		"reply_count":    replyCount,
		"posters":        jsonopt.Encode(posters),
		"last_post_id":   lastPostID,
		"last_posted_at": lastPostedAt,
	}).Error
}

func ReservePostSequence(topicId uint64) (uint64, error) {
	result := builder().
		Where("id = ?", topicId).
		Update("post_seq", gorm.Expr("post_seq + 1"))
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return 0, gorm.ErrRecordNotFound
	}

	var postSeq uint64
	err := builder().
		Select("post_seq").
		Where("id = ?", topicId).
		Scan(&postSeq).Error
	return postSeq, err
}

func applyPageSort(b *gorm.DB, sort string) {
	switch sort {
	case "hot":
		b.Order(queryopt.Desc("reply_count")).Order(queryopt.Desc("id"))
	case "popular":
		b.Order(queryopt.Desc("view_count")).Order(queryopt.Desc("id"))
	case "new":
		b.Order(queryopt.Desc("created_at")).Order(queryopt.Desc("id"))
	default:
		b.Order(queryopt.Desc("pin_weight")).Order(queryopt.Desc("updated_at")).Order(queryopt.Desc("id"))
	}
}

// UnscopedGet 返回含已删除（软删）在内的主题，供恢复/清理/审计使用。
func UnscopedGet(id uint64) (entity Entity) {
	builder().Unscoped().First(&entity, id)
	return
}

// GetUserDeletedPage 分页返回用户自己删除的话题。
// 使用 id 倒序与 cursorID 对齐，保证游标分页不会因 deleted_at/updated_at 并列或墓碑行时间变化而漏项。
func GetUserDeletedPage(userId uint64, cursorID uint64, limit int) (entities []Entity) {
	b := builder().Unscoped().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Eq("visibility_status", VisibilityUserDeleted)).
		Where(queryopt.Ne("retention_status", RetentionPurged))
	if cursorID != 0 {
		b = b.Where(queryopt.Lt("id", cursorID))
	}
	b.Order(queryopt.Desc("id")).
		Limit(pageutil.BoundPageSize(limit) + 1).
		Find(&entities)
	return
}

// ExpireRecoverable 返回超过恢复窗口仍为 RECOVERABLE 的主题（含软删行），
// 供 retention scheduler 将其置为 PURGED 并执行清理。
func ExpireRecoverable(before time.Time, limit int) (entities []Entity) {
	builder().Unscoped().
		Where(queryopt.Eq("retention_status", RetentionRecoverable)).
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Where("COALESCE(deleted_at, updated_at) < ?", before).
		Limit(limit).
		Find(&entities)
	return
}

// MarkDeleted 将主题标记为用户删除，进入 30 天恢复窗口。
func MarkUserDeleted(id uint64, deletedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": VisibilityUserDeleted,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// MarkModeratorRemoved 将主题标记为管理员删除，作者不可自行恢复。
func MarkModeratorRemoved(id uint64, deletedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": VisibilityModeratorRemoved,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// Restore 恢复主题：清除软删标记并回到正常生命周期。
func Restore(id uint64) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        gorm.Expr("NULL"),
		"visibility_status": VisibilityActive,
		"retention_status":  RetentionNormal,
		"deleted_by":        0,
		"delete_reason":     "",
	}).Error
}

// MarkPurged 标记主题为已永久删除（不再可恢复，仅审计可查）。
func MarkPurged(id uint64) error {
	result := builder().Unscoped().
		Where(queryopt.Eq("id", id)).
		Where(queryopt.Eq("retention_status", RetentionRecoverable)).
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Updates(map[string]any{
			"deleted_at":       time.Now(),
			"retention_status": RetentionPurged,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkPrivacyErased immediately hides a user's content and makes it unrecoverable.
// The visibility state remains distinct from moderator removal so governance
// records can distinguish privacy erasure from moderation action.
func MarkPrivacyErased(id uint64, erasedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": VisibilityAccountAnonymized,
		"retention_status":  RetentionPurged,
		"deleted_by":        erasedBy,
		"delete_reason":     reason,
	}).Error
}
