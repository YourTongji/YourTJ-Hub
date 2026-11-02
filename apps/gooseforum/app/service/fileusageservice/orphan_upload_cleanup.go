package fileusageservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
)

// OrphanUploadTTL 上传后始终没有被任何内容引用的附件保留期（issue #1365）。
const OrphanUploadTTL = 7 * 24 * time.Hour

// maxStalePruneRounds 单次运行最多回填几轮候选窗口：陈旧归属行积压很大时只清一批，
// 剩下的留给下一次（任务幂等）。
const maxStalePruneRounds = 5

// CleanupOrphanUploads 回收超过 TTL 仍未被内容引用的上传（issue #1365），返回删除
// 的文件数。只有带 upload_owner 行的文件才是候选：旧数据没有归属记录，读路径仍按
// 公开图片处理，回收会打穿老帖里的图片。
func CleanupOrphanUploads(ctx context.Context, before time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}
	var (
		candidates, existing []string
		cleanupErr           error
	)
	stalePruned := 0
	// 每条返回路径都要留痕：窗口只剩残行时 removed 为 0，cron 侧不会打日志。
	defer func() {
		if stalePruned > 0 {
			slog.Info("retire stale upload owner rows", "count", stalePruned)
		}
	}()
	// 候选窗口可能被「归属行还在、文件行已不存在」的陈旧审计行占满（文件库按快照
	// 重置或回滚后常见）：先将它们标记为 PURGED 并回填窗口，否则真正的过期孤儿永远排不进
	// limit，任务会静默地一直回收不到任何文件。
	for round := 0; round < maxStalePruneRounds; round++ {
		var err error
		candidates, err = fileUsage.ListUnreferencedUploadOwners(limit)
		if err != nil {
			return 0, errors.Join(cleanupErr, err)
		}
		if len(candidates) == 0 {
			return 0, cleanupErr
		}
		existing, err = filedata.ListExistingNames(ctx, candidates)
		if err != nil {
			return 0, errors.Join(cleanupErr, err)
		}
		if len(existing) == len(candidates) {
			break
		}
		pruned, pruneErr := pruneStaleUploadOwners(candidates, existing)
		stalePruned += pruned
		// 一条都没退役时不再回填窗口，避免同一批候选原地重试。
		if pruneErr != nil || pruned == 0 {
			cleanupErr = errors.Join(cleanupErr, pruneErr)
			break
		}
	}
	expired, err := filedata.ListReadyNamesCreatedBefore(ctx, existing, before)
	if err != nil {
		return 0, errors.Join(cleanupErr, err)
	}
	removed := 0
	for _, name := range expired {
		pruned, pruneErr := pruneOrphanUpload(name)
		if pruneErr != nil {
			cleanupErr = errors.Join(cleanupErr, pruneErr)
			continue
		}
		if pruned {
			removed++
		}
	}
	return removed, cleanupErr
}

// pruneStaleUploadOwners 将文件行已不存在的归属记录标记为 PURGED，
// 保留墓碑以拒绝已预解析附件的迟到写入，并让后续候选进入清理窗口。
func pruneStaleUploadOwners(candidates, existing []string) (int, error) {
	present := make(map[string]bool, len(existing))
	for _, name := range existing {
		present[name] = true
	}
	gone := make([]string, 0, len(candidates)-len(existing))
	for _, name := range candidates {
		if !present[name] {
			gone = append(gone, name)
		}
	}
	retired := 0
	for _, name := range gone {
		claimed, err := fileUsage.ClaimOrphanUpload(name)
		if err != nil {
			return retired, err
		}
		if !claimed {
			continue
		}
		if err := fileUsage.FinishOrphanUpload(name); err != nil {
			return retired, err
		}
		retired++
	}
	return retired, nil
}

// Claiming the ownership row and admitting references share a database lock.
// Once RECLAIMING commits, late publishers fail before committing content;
// storage failures keep that state for retry and completed deletes keep a tombstone.
func pruneOrphanUpload(name string) (bool, error) {
	claimed, err := fileUsage.ClaimOrphanUpload(name)
	if err != nil {
		return false, fmt.Errorf("check references for %q: %w", name, err)
	}
	if !claimed {
		return false, nil
	}
	if err := filedata.DeleteByName(name); err != nil {
		return false, fmt.Errorf("delete orphan upload %q: %w", name, err)
	}
	if err := fileUsage.FinishOrphanUpload(name); err != nil {
		return false, fmt.Errorf("finish upload reclamation for %q: %w", name, err)
	}
	return true, nil
}
