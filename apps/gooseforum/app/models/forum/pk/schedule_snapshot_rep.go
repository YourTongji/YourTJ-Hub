package pk

import (
	"errors"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/queryopt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func scheduleSnapshotBuilder() *gorm.DB {
	return db.Connect().Table(scheduleSnapshotTableName)
}

// ErrScheduleSnapshotNotFound 云端无该用户的快照（GET 语义上等同 data:null）。
var ErrScheduleSnapshotNotFound = gorm.ErrRecordNotFound

// GetScheduleSnapshotByUser 返回用户当前云端快照；无行时返回 ErrScheduleSnapshotNotFound。
func GetScheduleSnapshotByUser(userId uint64) (ScheduleSnapshotEntity, error) {
	var entity ScheduleSnapshotEntity
	err := scheduleSnapshotBuilder().Where(queryopt.Eq("user_id", userId)).First(&entity).Error
	return entity, err
}

// UpsertScheduleSnapshot 按唯一键 user_id 整体替换快照四字段（plans/activePlanId/
// majorSelected/weekView）。created_at 保持首建时间；updated_at 每次保存刷新——
// 服务端权威同步时钟（客户端存为 pk.syncedAt），客户端不写时钟。
//
// 不用 OnConflict.DoUpdates：map 赋值绕过 gorm 字段 serializer（JSON 列会被写成
// 非法驱动值），且冲突路径不会刷新 updated_at。改为事务内读改写；首写并发竞争
// 由 user_id 唯一索引兜底（后到者报唯一冲突，客户端保持 dirty 下次重试即走更新
// 路径，见 issue #537 Blueprint「单笔 pending PUT」语义）。
func UpsertScheduleSnapshot(entity *ScheduleSnapshotEntity) error {
	// 截断到微秒：PG timestamp 列只保留微秒精度，若以 time.Now() 的纳秒值
	// 写响应、落库后被截断，PUT 返回的 updatedAt 与后续 GET 回读必然不同，
	// 客户端 pk.syncedAt 等值比对每次误报冲突（issue #557 review blocker）。
	// 截断后写入值与回读值逐位一致，SQLite/PG 两侧行为相同。
	now := time.Now().Truncate(time.Microsecond)
	return db.Connect().Transaction(func(tx *gorm.DB) error {
		var existing ScheduleSnapshotEntity
		err := tx.Table(scheduleSnapshotTableName).
			Where(queryopt.Eq("user_id", entity.UserId)).
			First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			entity.Id = 0
			entity.CreatedAt = now
			entity.UpdatedAt = now
			return tx.Create(entity).Error
		}
		if err != nil {
			return err
		}
		entity.Id = existing.Id
		entity.CreatedAt = existing.CreatedAt
		entity.UpdatedAt = now
		// Select 强制包含零值字段（整体替换语义）；struct 形式 Updates 走字段
		// serializer，JSON 列编码与 Create 路径一致。
		return tx.Model(&ScheduleSnapshotEntity{}).
			Where(queryopt.Eq("id", existing.Id)).
			Select("plans", "active_plan_id", "major_selected", "week_view", "updated_at").
			Updates(ScheduleSnapshotEntity{
				Plans:         entity.Plans,
				ActivePlanId:  entity.ActivePlanId,
				MajorSelected: entity.MajorSelected,
				WeekView:      entity.WeekView,
				UpdatedAt:     now,
			}).Error
	})
}

// DeleteScheduleSnapshotByUser 删除用户云端快照（账号注销 anonymize/delete 两 mode
// 共用，与 pushDevice.DeleteByUser 同语义；DELETE /api/pk/plans 也复用本函数）。
// 幂等：无匹配行时静默成功。
func DeleteScheduleSnapshotByUser(userId uint64) error {
	return scheduleSnapshotBuilder().Where(queryopt.Eq("user_id", userId)).Delete(&ScheduleSnapshotEntity{}).Error
}

// ErrScheduleSnapshotConflict means another writer changed the observed snapshot.
var ErrScheduleSnapshotConflict = errors.New("schedule snapshot changed")

// CompareAndSwapScheduleSnapshot checks the observed revision in the write itself.
// An empty base creates only if absent; existing snapshots require their exact clock.
func CompareAndSwapScheduleSnapshot(entity *ScheduleSnapshotEntity, base string) error {
	return compareAndSwapScheduleSnapshot(db.Connect(), entity, base)
}

func compareAndSwapScheduleSnapshot(conn *gorm.DB, entity *ScheduleSnapshotEntity, base string) error {
	now := time.Now().UTC().Truncate(time.Microsecond)
	if base == "" {
		entity.CreatedAt, entity.UpdatedAt = now, now
		result := conn.Clauses(clause.OnConflict{DoNothing: true}).Create(entity)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrScheduleSnapshotConflict
		}
		return nil
	}
	revision, err := time.Parse(time.RFC3339Nano, base)
	if err != nil {
		return err
	}
	if !now.After(revision) {
		now = revision.Add(time.Microsecond)
	}
	entity.UpdatedAt = now
	result := conn.Model(&ScheduleSnapshotEntity{}).
		Where("user_id = ? AND updated_at = ?", entity.UserId, revision).
		Select("plans", "active_plan_id", "major_selected", "week_view", "updated_at").Updates(entity)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrScheduleSnapshotConflict
	}
	return nil
}
