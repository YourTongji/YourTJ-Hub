package fileusageservice

import (
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
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

// PromotePendingTargetFiles 待审内容获批后让其附件引用转为 ACTIVE（公开可读，issue #975）。
func PromotePendingTargetFiles(ref TargetRef) {
	if err := fileUsage.MarkTargetPendingActive(ref.TargetType, ref.TargetID); err != nil {
		slog.Error("promote pending file usages failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
	}
}

// PromotePendingTopicFiles 话题转为正常可见（审核通过/解封）时，让话题与首楼
// 的待审附件引用转 ACTIVE。幂等，非 PENDING 行不受影响。
func PromotePendingTopicFiles(topicID, firstPostID uint64) {
	PromotePendingTargetFiles(TargetRef{TargetType: fileUsage.TargetTopic, TargetID: topicID})
	if firstPostID > 0 {
		PromotePendingPostFiles(firstPostID)
	}
}

// PromotePendingPostFiles 回复转为正常可见时让其待审附件引用转 ACTIVE。
func PromotePendingPostFiles(postID uint64) {
	PromotePendingTargetFiles(TargetRef{TargetType: fileUsage.TargetPost, TargetID: postID})
}

// HasPendingReferences reports whether a filename is referenced by content
// awaiting moderation (issue #975).
func HasPendingReferences(fileName string) bool {
	return fileUsage.HasPendingReferences(fileName)
}

// HasAnyReferences reports whether a filename is tracked by the content
// attachment lifecycle.
func HasAnyReferences(fileName string) bool {
	return fileUsage.HasAnyReferences(fileName)
}

// HasActiveReferences reports whether a tracked filename is referenced by
// content that is currently public. RECOVERING/PURGED references must not
// authorize public downloads.
func HasActiveReferences(fileName string) bool {
	return fileUsage.HasActiveReferences(fileName)
}

// RetireTargetFiles 内容永久删除/过期时退役附件引用：引用置 PURGED，附件
// 字节保留在存储中（删除终态数据保留，MADR-0021 / issue #555）。PURGED 引用
// 不构成公开下载授权（fileController 按 ACTIVE 引用判权）；取证视图当前
// 仅返回文本正文，附件字节的取证回显是后续增强——留存不等于现有读路径可访问。
func RetireTargetFiles(ref TargetRef) {
	if err := fileUsage.MarkTargetPurged(ref.TargetType, ref.TargetID); err != nil {
		slog.Error("retire file usages failed", "targetType", ref.TargetType, "targetId", ref.TargetID, "err", err)
	}
}

// ExpireRecoveringFiles 供 retention scheduler 调用：将超过恢复窗口的附件
// 引用置为 PURGED。引用退役即切断公开下载授权；附件本体保留在存储中
// （MADR-0021），不再随窗口过期被物理删除。
func ExpireRecoveringFiles(limit int) {
	if limit <= 0 {
		limit = 200
	}
	before := time.Now()
	expired := fileUsage.ListExpiredRecovering(before, limit)
	for _, usage := range expired {
		if err := fileUsage.MarkTargetPurged(usage.TargetType, usage.TargetId); err != nil {
			slog.Error("expire file usage failed", "targetType", usage.TargetType, "targetId", usage.TargetId, "err", err)
		}
	}
}
