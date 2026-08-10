package fileUsage

import (
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/queryopt"
)

func Create(entity *Entity) error {
	return builder().Create(entity).Error
}

func ReplaceTargetUsages(targetType string, targetId uint64, usageTypes []string, usages []Entity) error {
	if len(usageTypes) == 0 {
		return nil
	}
	db := builder()
	if err := db.
		Where(queryopt.Eq("target_type", targetType)).
		Where(queryopt.Eq("target_id", targetId)).
		Where(queryopt.In("usage_type", usageTypes)).
		Delete(&Entity{}).Error; err != nil {
		return err
	}
	if len(usages) == 0 {
		return nil
	}
	return db.Create(&usages).Error
}

// MarkTargetRecovering 将某内容的附件引用转入受限恢复态（删除后 30 天窗口）。
func MarkTargetRecovering(targetType string, targetId uint64, expiresAt time.Time) error {
	return builder().
		Where(queryopt.Eq("target_type", targetType)).
		Where(queryopt.Eq("target_id", targetId)).
		Where(queryopt.Eq("status", UsageStatusActive)).
		Updates(map[string]any{
			"status":     UsageStatusRecovering,
			"expires_at": expiresAt,
		}).Error
}

// MarkTargetActive 将某内容的附件引用恢复为正常（内容恢复）。
func MarkTargetActive(targetType string, targetId uint64) error {
	return builder().
		Where(queryopt.Eq("target_type", targetType)).
		Where(queryopt.Eq("target_id", targetId)).
		Where(queryopt.Eq("status", UsageStatusRecovering)).
		Updates(map[string]any{
			"status":     UsageStatusActive,
			"expires_at": nil,
		}).Error
}

// MarkTargetPurged 将某内容的附件引用置为已清理（永久删除）。
func MarkTargetPurged(targetType string, targetId uint64) error {
	return builder().
		Where(queryopt.Eq("target_type", targetType)).
		Where(queryopt.Eq("target_id", targetId)).
		Where(queryopt.In("status", []string{UsageStatusActive, UsageStatusRecovering})).
		Updates(map[string]any{
			"status": UsageStatusPurged,
		}).Error
}

// ListByTarget returns all attachment references for one content target.
func ListByTarget(targetType string, targetId uint64) (entities []Entity, err error) {
	err = builder().
		Where(queryopt.Eq("target_type", targetType)).
		Where(queryopt.Eq("target_id", targetId)).
		Find(&entities).Error
	return
}

// HasAnyReferences reports whether the file has at least one usage row.
// This lets public file serving distinguish legacy untracked uploads from a
// tracked file whose content references have all been revoked.
func HasAnyReferences(fileName string) bool {
	var count int64
	builder().Where(queryopt.Eq("file_name", fileName)).Count(&count)
	return count > 0
}

// ListExpiredRecovering 返回超过恢复窗口仍未决的附件引用，供清理任务执行。
func ListExpiredRecovering(before time.Time, limit int) (entities []Entity) {
	builder().
		Where(queryopt.Eq("status", UsageStatusRecovering)).
		Where("expires_at IS NOT NULL AND expires_at < ?", before).
		Limit(limit).
		Find(&entities)
	return
}

// HasLiveReferences 判断文件是否仍被 ACTIVE/RECOVERING 的引用使用。
func HasLiveReferences(fileName string) bool {
	var count int64
	builder().
		Where(queryopt.Eq("file_name", fileName)).
		Where(queryopt.In("status", []string{UsageStatusActive, UsageStatusRecovering})).
		Count(&count)
	return count > 0
}

// HasActiveReferences reports whether a file is referenced by content that is
// currently public. Recovering references stay live for purge coordination but
// must not authorize public downloads.
func HasActiveReferences(fileName string) bool {
	var count int64
	builder().
		Where(queryopt.Eq("file_name", fileName)).
		Where(queryopt.Eq("status", UsageStatusActive)).
		Count(&count)
	return count > 0
}
