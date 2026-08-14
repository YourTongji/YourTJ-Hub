package posts

import (
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/pageutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/postRevisions"
	"gorm.io/gorm"
)

func SaveOrCreateById(entity *Entity) int64 {
	if entity.Id == 0 {
		return builder().Create(entity).RowsAffected
	}

	return builder().Save(entity).RowsAffected
}

func SaveNoUpdate(entity *Entity) error {
	return builder().Omit("updated_at").Save(entity).Error
}

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

// CreateTx 事务内创建帖子。
func CreateTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Create(entity).Error
}

func Save(entity *Entity) error {
	return builder().Save(entity).Error
}

// SaveTx 事务内保存帖子。
func SaveTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Save(entity).Error
}

func Get(id uint64) (entity Entity) {
	builder().First(&entity, id)
	return
}

// GetTx 事务内按 id 获取帖子（避免单连接测试库下事务内走全局连接死锁）。
func GetTx(tx *gorm.DB, id uint64) (entity Entity) {
	tx.Table(tableName).First(&entity, id)
	return
}

func GetMaxId() uint64 {
	var entity Entity
	builder().Order(queryopt.Desc("id")).Limit(1).First(&entity)
	return entity.Id
}

func GetByIds(ids []uint64) (entities []*Entity) {
	if len(ids) == 0 {
		return
	}
	builder().Where("id in ?", ids).Find(&entities)
	return
}

func GetMapByIds(ids []uint64) map[uint64]*Entity {
	list := GetByIds(ids)
	result := make(map[uint64]*Entity, len(list))
	for _, item := range list {
		if item != nil {
			result[item.Id] = item
		}
	}
	return result
}

// GetMapByIdsUnscoped 返回含已删除（软删）在内的回复 map。
// 举报审核需要基于快照继续处理已被作者删除的目标，不能因软删过滤而丢失。
func GetMapByIdsUnscoped(ids []uint64) map[uint64]*Entity {
	var list []*Entity
	if len(ids) == 0 {
		return map[uint64]*Entity{}
	}
	builder().Unscoped().Where("id in ?", ids).Find(&list)
	result := make(map[uint64]*Entity, len(list))
	for _, item := range list {
		if item != nil {
			result[item.Id] = item
		}
	}
	return result
}

func UpdateProcessStatus(id uint64, processStatus int8) error {
	return builder().Where(queryopt.Eq("id", id)).Update("process_status", processStatus).Error
}

// ResetPendingReview 作废待审状态：将 process_status 复位为正常。
// 内容被删除后不应继续停留在管理审核队列（PRD R1），避免"已删除+待审"
// 语义叠加导致审核队列出现幽灵项。
func ResetPendingReview(id uint64) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Update("process_status", ProcessStatusNormal).Error
}

// UpdateWikiSyncedContentTx 事务内只更新由 wiki 修订派生的首楼字段
// （正文/渲染/版本/待审状态/水印/更新时间）。不得整行保存事务外读取的
// 帖子——整行 Save 会把并发回复写入的统计字段回写成旧值（review High）。
func UpdateWikiSyncedContentTx(tx *gorm.DB, entity *Entity) error {
	return tx.Table(tableName).Where(queryopt.Eq("id", entity.Id)).
		Updates(map[string]any{
			"content":                 entity.Content,
			"rendered_html":           entity.RenderedHTML,
			"rendered_version":        entity.RenderedVersion,
			"process_status":          entity.ProcessStatus,
			"wiki_synced_revision_no": entity.WikiSyncedRevisionNo,
			"updated_at":              time.Now(),
		}).Error
}

func DeleteEntity(entity *Entity) int64 {
	return builder().Delete(entity).RowsAffected
}

// UnscopedGet 返回含已删除（软删）在内的回复，供恢复/清理/审计使用。
func UnscopedGet(id uint64) (entity Entity) {
	builder().Unscoped().First(&entity, id)
	return
}

// HasChildren 判断回复是否存在未被删除的子回复（reply_to_post_id 指向它）。
func HasChildren(postId uint64) bool {
	var count int64
	builder().
		Where(queryopt.Eq("reply_to_post_id", postId)).
		Where("deleted_at IS NULL").
		Count(&count)
	return count > 0
}

// GetUserDeletedPage 分页返回用户已删除的回复（含软删行与墓碑态行）。
// 使用纯 id 倒序 + id 游标，与 topics 版本保持一致：deleted_at/updated_at 排序
// 在跨数据库时对 NULL 方向不一致（Postgres 默认 NULLS FIRST），且时间并列或
// 墓碑行时间变化会导致按 id 游标翻页漏项。
func GetUserDeletedPage(userId uint64, cursorID uint64, limit int) (entities []Entity) {
	b := builder().Unscoped().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Where(queryopt.Ne("retention_status", RetentionPurged))
	if cursorID != 0 {
		b = b.Where(queryopt.Lt("id", cursorID))
	}
	b.Order(queryopt.Desc("id")).
		Limit(pageutil.BoundPageSize(limit) + 1).
		Find(&entities)
	return
}

// ExpireRecoverable 返回超过恢复窗口仍为 RECOVERABLE 的回复（含软删行与墓碑态行）。
// 墓碑态行无 deleted_at，使用 updated_at 作为删除时刻的近似。
func ExpireRecoverable(before time.Time, limit int) (entities []Entity) {
	builder().Unscoped().
		Where(queryopt.Eq("retention_status", RetentionRecoverable)).
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Where("COALESCE(deleted_at, updated_at) < ?", before).
		Limit(limit).
		Find(&entities)
	return
}

// MarkUserDeleted 将回复标记为用户删除，进入 30 天恢复窗口。
func MarkUserDeleted(id uint64, deletedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": VisibilityUserDeleted,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// MarkUserDeletedKeepVisible 标记回复为用户删除但保留行可见（墓碑态）：
// 用于"存在子回复"的场景，讨论树需要保留该行以维持结构，正文由前端渲染为占位。
// 墓碑态行不置 deleted_at（保持讨论树可见），以 updated_at 作为删除时刻的近似：
// 必须显式写入 updated_at=now（builder() 走 Table()，GORM 的 Updates(map) 不会自动
// 填充 autoUpdateTime），否则 30 天恢复窗口会按最后一次编辑时间判定而立即失效。
func MarkUserDeletedKeepVisible(id uint64, deletedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"updated_at":        time.Now(),
		"visibility_status": VisibilityUserDeleted,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// MarkDeletedKeepVisible 标记回复为删除但保留行可见（墓碑态），visibility 区分用户/管理员来源。
func MarkDeletedKeepVisible(id uint64, visibility string, deletedBy uint64, reason string) error {
	if visibility == "" {
		visibility = VisibilityUserDeleted
	}
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"updated_at":        time.Now(),
		"visibility_status": visibility,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// MarkModeratorRemoved 将回复标记为管理员删除，作者不可自行恢复。
func MarkModeratorRemoved(id uint64, deletedBy uint64, reason string) error {
	return builder().Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
		"deleted_at":        time.Now(),
		"visibility_status": VisibilityModeratorRemoved,
		"retention_status":  RetentionRecoverable,
		"deleted_by":        deletedBy,
		"delete_reason":     reason,
	}).Error
}

// SoftDeleteByTopicId 将某话题下的所有回复软删（级联删除），返回受影响行数。
// visibility 为空时默认 USER_DELETED；管理端级联应传 MODERATOR_REMOVED。
func SoftDeleteByTopicId(topicId uint64, deletedBy uint64, reason string, visibility string) int64 {
	if visibility == "" {
		visibility = VisibilityUserDeleted
	}
	return builder().Unscoped().
		Where(queryopt.Eq("topic_id", topicId)).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_at":        time.Now(),
			"visibility_status": visibility,
			"retention_status":  RetentionRecoverable,
			"deleted_by":        deletedBy,
			"delete_reason":     reason,
		}).RowsAffected
}

// SoftDeleteByIDs 按回复 ID 列表软删（级联删除），返回受影响行数。
// 只处理删除瞬间已经收集到的活跃回复，避免删除动作与并发新回复之间的
// TOCTOU 竞态把"读取之后才写入"的回复一并软删并递减其统计。
func SoftDeleteByIDs(ids []uint64, deletedBy uint64, reason string, visibility string) int64 {
	if len(ids) == 0 {
		return 0
	}
	if visibility == "" {
		visibility = VisibilityUserDeleted
	}
	return builder().Unscoped().
		Where(queryopt.In("id", ids)).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"deleted_at":        time.Now(),
			"visibility_status": visibility,
			"retention_status":  RetentionRecoverable,
			"deleted_by":        deletedBy,
			"delete_reason":     reason,
		}).RowsAffected
}

// Restore 恢复回复：清除软删标记并回到正常生命周期。
func Restore(id uint64) error {
	result := builder().Unscoped().Where(queryopt.Eq("id", id)).
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Where(queryopt.Eq("retention_status", RetentionRecoverable)).
		Updates(map[string]any{
			"deleted_at":        gorm.Expr("NULL"),
			"visibility_status": VisibilityActive,
			"retention_status":  RetentionNormal,
			"deleted_by":        0,
			"delete_reason":     "",
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkPurged 标记回复为已永久删除（不再可恢复，仅审计可查）。
// 同时清空正文与版本历史正文，且二者在同一事务内完成：
// 任一步失败整体回滚并返回错误，杜绝"帖子行已清空、版本快照仍留原文"
// 的部分成功状态（版本快照不得绕过删除留存原文）。
func MarkPurged(id uint64) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		result := tx.Table(tableName).Unscoped().Where(queryopt.Eq("id", id)).
			Where(queryopt.Eq("retention_status", RetentionRecoverable)).
			Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
			Updates(map[string]any{
				"deleted_at":       time.Now(),
				"retention_status": RetentionPurged,
				"content":          "",
				"rendered_html":    "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return postRevisions.BlankContentByPostIdTx(tx, id)
	})
}

// MarkPurgedOwned permits the topic owner purge path to erase an ACTIVE reply
// while still requiring ownership and a lifecycle state that has not already
// been purged. Post row update and revision blanking share one transaction.
func MarkPurgedOwned(id uint64, ownerID uint64) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		result := tx.Table(tableName).Unscoped().Where(queryopt.Eq("id", id)).
			Where(queryopt.Eq("user_id", ownerID)).
			Where(queryopt.In("visibility_status", []string{VisibilityActive, VisibilityUserDeleted, VisibilityModeratorRemoved})).
			Where(queryopt.Ne("retention_status", RetentionPurged)).
			Updates(map[string]any{
				"deleted_at":       time.Now(),
				"retention_status": RetentionPurged,
				"content":          "",
				"rendered_html":    "",
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return postRevisions.BlankContentByPostIdTx(tx, id)
	})
}

// MarkPrivacyErased immediately hides and blanks a user's reply.
// Post row update and revision blanking share one transaction.
func MarkPrivacyErased(id uint64, erasedBy uint64, reason string) error {
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := tx.Table(tableName).Unscoped().Where(queryopt.Eq("id", id)).Updates(map[string]any{
			"deleted_at":        time.Now(),
			"visibility_status": VisibilityAccountAnonymized,
			"retention_status":  RetentionPurged,
			"deleted_by":        erasedBy,
			"delete_reason":     reason,
			"content":           "",
			"rendered_html":     "",
		}).Error; err != nil {
			return err
		}
		return postRevisions.BlankContentByPostIdTx(tx, id)
	})
}

// MarkPurgedByTopicID 将某话题下已删除的回复置为已永久删除（话题永久删除级联）。
// 只处理已进入删除生命周期的回复（非 ACTIVE），其他用户仍 ACTIVE 的回复
// 属于他人内容，不得被话题作者/自动过期的级联永久删除（PRD Out of Scope）。
// 帖子行更新与版本清空在同一事务内，任一步失败整体回滚并返回错误。
func MarkPurgedByTopicID(topicID uint64) (int64, error) {
	var rows int64
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		var targets []uint64
		if err := tx.Table(tableName).Unscoped().
			Where(queryopt.Eq("topic_id", topicID)).
			Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
			Where(queryopt.Eq("retention_status", RetentionRecoverable)).
			Select("id").
			Scan(&targets).Error; err != nil {
			return err
		}
		result := tx.Table(tableName).Unscoped().
			Where(queryopt.Eq("topic_id", topicID)).
			Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
			Where(queryopt.Eq("retention_status", RetentionRecoverable)).
			Updates(map[string]any{
				"deleted_at":       time.Now(),
				"retention_status": RetentionPurged,
				"content":          "",
				"rendered_html":    "",
			})
		if result.Error != nil {
			return result.Error
		}
		rows = result.RowsAffected
		return postRevisions.BlankContentByPostIdsTx(tx, targets)
	})
	return rows, err
}

// ListUnscopedByTopicID 返回某话题下全部回复（含已软删行），用于级联恢复/统计。
func ListUnscopedByTopicID(topicID uint64, list *[]*Entity) error {
	return builder().Unscoped().
		Where(queryopt.Eq("topic_id", topicID)).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Find(list).Error
}

// ListByTopicID 返回某话题下未删除的回复。
func ListByTopicID(topicID uint64, list *[]*Entity) error {
	return builder().
		Where(queryopt.Eq("topic_id", topicID)).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Find(list).Error
}

// GetActiveByUserPage 分页返回本人仍公开（ACTIVE、post_no>1）的回复（PRD R9）。
func GetActiveByUserPage(userId uint64, cursorID uint64, limit int) (entities []Entity) {
	b := builder().
		Where(queryopt.Eq("user_id", userId)).
		Where(queryopt.Gt("post_no", 1)).
		Where(queryopt.Eq("visibility_status", VisibilityActive))
	if cursorID != 0 {
		b = b.Where(queryopt.Lt("id", cursorID))
	}
	b.Order(queryopt.Desc("id")).
		Limit(pageutil.BoundPageSize(limit) + 1).
		Find(&entities)
	return
}

// RestoreDeletedByTopicID 恢复某话题下所有被级联软删的回复。
func RestoreDeletedByTopicID(topicID uint64) int64 {
	return builder().Unscoped().
		Where(queryopt.Eq("topic_id", topicID)).
		Where("deleted_at IS NOT NULL").
		Where(queryopt.In("visibility_status", []string{VisibilityUserDeleted, VisibilityModeratorRemoved})).
		Updates(map[string]any{
			"deleted_at":        gorm.Expr("NULL"),
			"visibility_status": VisibilityActive,
			"retention_status":  RetentionNormal,
			"deleted_by":        0,
			"delete_reason":     "",
		}).RowsAffected
}

// RestoreCascadeDeletedByTopicID restores only user-deleted rows changed by
// the current topic deletion operation. Moderator removals and independently
// deleted replies remain untouched.
func RestoreCascadeDeletedByTopicID(topicID uint64, deletedBy uint64, deleteReason string) int64 {
	return RestoreCascadeDeletedByTopicIDWithVisibility(topicID, deletedBy, deleteReason, VisibilityUserDeleted)
}

// RestoreCascadeDeletedByTopicIDWithVisibility 按指定删除来源恢复话题级联删除的回复。
// 只恢复"本次删除操作"标记的行（deleted_by + delete_reason 精确匹配），
// 独立删除的回复与管理端删除不随作者恢复；管理端恢复话题时传 MODERATOR_REMOVED。
func RestoreCascadeDeletedByTopicIDWithVisibility(topicID uint64, deletedBy uint64, deleteReason string, visibility string) int64 {
	return builder().Unscoped().
		Where(queryopt.Eq("topic_id", topicID)).
		Where("deleted_at IS NOT NULL").
		Where(queryopt.Eq("visibility_status", visibility)).
		Where(queryopt.Eq("retention_status", RetentionRecoverable)).
		Where(queryopt.Eq("deleted_by", deletedBy)).
		Where(queryopt.Eq("delete_reason", deleteReason)).
		Updates(map[string]any{
			"deleted_at":        gorm.Expr("NULL"),
			"visibility_status": VisibilityActive,
			"retention_status":  RetentionNormal,
			"deleted_by":        0,
			"delete_reason":     "",
		}).RowsAffected
}

func GetFirstPageByTopicId(topicId uint64) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Limit(20).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return
}

func GetByTopicPostNoAsc(topicId uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Limit(limit).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return
}

func GetNormalByTopicPostNoAfter(topicID uint64, afterPostNo uint64, limit int) (entities []*Entity, err error) {
	if limit <= 0 {
		return []*Entity{}, nil
	}
	err = builder().
		Where(queryopt.Eq("topic_id", topicID)).
		Where(queryopt.Eq("process_status", ProcessStatusNormal)).
		Where(queryopt.Eq("visibility_status", VisibilityActive)).
		Where(queryopt.Gt("post_no", afterPostNo)).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Limit(limit).
		Find(&entities).Error
	return
}

func GetByTopicPostNoDesc(topicId uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Limit(limit).
		Order(queryopt.Desc("post_no")).
		Order(queryopt.Desc("id")).
		Find(&entities)
	reversePosts(entities)
	return
}

func GetByTopicPostNoAfter(topicId uint64, postNo uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Gt("post_no", postNo)).
		Limit(limit).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return
}

func GetByTopicPostNoBefore(topicId uint64, postNo uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Lt("post_no", postNo)).
		Limit(limit).
		Order(queryopt.Desc("post_no")).
		Order(queryopt.Desc("id")).
		Find(&entities)
	reversePosts(entities)
	return
}

func GetByTopicPostNoAtOrAfter(topicId uint64, postNo uint64) (entity Entity, ok bool) {
	err := builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Ge("post_no", postNo)).
		Order(queryopt.Asc("post_no")).
		Order(queryopt.Asc("id")).
		First(&entity).Error
	return entity, err == nil
}

func GetByTopicPostNoAtOrBefore(topicId uint64, postNo uint64) (entity Entity, ok bool) {
	err := builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Le("post_no", postNo)).
		Order(queryopt.Desc("post_no")).
		Order(queryopt.Desc("id")).
		First(&entity).Error
	return entity, err == nil
}

func GetLastByTopicID(topicID uint64) (entity Entity, ok bool) {
	err := builder().
		Where(queryopt.Eq("topic_id", topicID)).
		Order(queryopt.Desc("post_no")).
		Order(queryopt.Desc("id")).
		First(&entity).Error
	return entity, err == nil
}

func GetMaxPostNoByTopicId(topicId uint64) uint64 {
	var entity Entity
	err := builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Order(queryopt.Desc("post_no")).
		Order(queryopt.Desc("id")).
		Limit(1).
		First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0
	}
	return entity.PostNo
}

func GetByTopicIdAfter(topicId uint64, id uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Gt("id", id)).
		Limit(limit).
		Order(queryopt.Asc("id")).
		Find(&entities)
	return
}

func GetByTopicIdBefore(topicId uint64, id uint64, limit int) (entities []*Entity) {
	builder().
		Where(queryopt.Eq("topic_id", topicId)).
		Where(queryopt.Lt("id", id)).
		Limit(limit).
		Order(queryopt.Desc("id")).
		Find(&entities)
	reversePosts(entities)
	return
}

func reversePosts(entities []*Entity) {
	for i, j := 0, len(entities)-1; i < j; i, j = i+1, j-1 {
		entities[i], entities[j] = entities[j], entities[i]
	}
}

// PagePendingReview 列出待审（ProcessStatus=2）的回复。
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
		// wiki 首楼由 wiki 修订审核队列管理，不进入论坛审核（review N1，
		// 避免绕过 wiki 修订流程直接审核/拒绝）；wiki 分站评论（post_no>1）仍走论坛审核队列。
		// 字面量 0 == topics.TopicTypeForum（论坛话题）。不能 import topics：
		// topics 的测试包已 import posts，反向导入会构成测试编译环。
		Where("(topic_id IN (SELECT id FROM topics WHERE topic_type = ?) OR post_no > ?)", 0, 1).
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
