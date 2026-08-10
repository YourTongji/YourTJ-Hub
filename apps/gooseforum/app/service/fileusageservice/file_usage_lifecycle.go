package fileusageservice

import (
	"log/slog"
	"time"

	"github.com/leancodebox/GooseForum/app/models/filemodel/filedata"
	"github.com/leancodebox/GooseForum/app/models/forum/fileUsage"
)

// TargetRef 标识一个内容目标（话题/回复），用于附件生命周期管理。
type TargetRef struct {
	TargetType string
	TargetID   uint64
}

// HardenTargetFiles 内容删除时把附件引用转入受限恢复态（30 天窗口）。
func HardenTargetFiles(ref TargetRef, expiresAt time.Time) {
	if err := fileUsage.MarkTargetRecovering(ref.TargetType, ref.TargetID, expiresAt); err != nil {
		slog.Error("mark file usages recovering failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
	}
}

// RecoverTargetFiles 内容恢复时把附件引用恢复为正常可见。
func RecoverTargetFiles(ref TargetRef) {
	if err := fileUsage.MarkTargetActive(ref.TargetType, ref.TargetID); err != nil {
		slog.Error("mark file usages active failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
	}
}

// HasAnyReferences reports whether a filename is tracked by the content
// attachment lifecycle.
func HasAnyReferences(fileName string) bool {
	return fileUsage.HasAnyReferences(fileName)
}

// HasLiveReferences reports whether a tracked filename still has a visible
// content reference.
func HasLiveReferences(fileName string) bool {
	return fileUsage.HasLiveReferences(fileName)
}

// PurgeTargetFiles 永久删除内容时清理附件：引用置 PURGED，无其他引用的文件本体删除。
func PurgeTargetFiles(ref TargetRef) {
	usages, err := fileUsage.ListByTarget(ref.TargetType, ref.TargetID)
	if err != nil {
		slog.Error("list file usages for purge failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
		return
	}
	if err := fileUsage.MarkTargetPurged(ref.TargetType, ref.TargetID); err != nil {
		slog.Error("mark file usages purged failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
		return
	}
	for _, usage := range usages {
		if !hasLiveReferences(usage.FileName) {
			if err := filedata.DeleteByName(usage.FileName); err != nil {
				slog.Error("delete purged file failed", "fileName", usage.FileName, "err", err)
			}
		}
	}
}

// ExpireRecoveringFiles 供 retention scheduler 调用：清理超过恢复窗口的附件引用并删除文件本体。
func ExpireRecoveringFiles(limit int) {
	if limit <= 0 {
		limit = 200
	}
	before := time.Now()
	expired := fileUsage.ListExpiredRecovering(before, limit)
	for _, usage := range expired {
		if err := fileUsage.MarkTargetPurged(usage.TargetType, usage.TargetId); err != nil {
			slog.Error("expire file usage failed", "targetType", usage.TargetType, "targetId", usage.TargetId, "err", err)
			continue
		}
		if !hasLiveReferences(usage.FileName) {
			if err := filedata.DeleteByName(usage.FileName); err != nil {
				slog.Error("delete orphan file failed", "fileName", usage.FileName, "err", err)
			}
		}
	}
}

// hasLiveReferences 判断文件是否仍被 ACTIVE/RECOVERING 的引用使用。
func hasLiveReferences(fileName string) bool {
	return fileUsage.HasLiveReferences(fileName)
}
