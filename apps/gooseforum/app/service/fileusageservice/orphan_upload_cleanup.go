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
			slog.Info("prune stale upload owner rows", "count", stalePruned)
		}
	}()
	// 候选窗口可能被「归属行还在、文件行已不存在」的陈旧审计行占满（文件库按快照
	// 重置或回滚后常见）：先批量删掉它们并回填窗口，否则真正的过期孤儿永远排不进
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
		// 一条都没删掉时不再回填窗口，避免同一批候选原地重试。
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

// pruneStaleUploadOwners 批量删除文件行已不存在的归属行：字节已不在，只留审计残行；
// 返回删除的条数（一条语句，limit ≤ 500 不会碰到 SQLite 的参数上限）。
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
	if err := fileUsage.DeleteUploadOwnerRows(gone); err != nil {
		return 0, fmt.Errorf("delete %d stale upload owner rows: %w", len(gone), err)
	}
	return len(gone), nil
}

// pruneOrphanUpload 复核引用后删除一个孤儿上传，复核发现有内容引用时保留并返回 false。
// 复核查询失败时同样保留（删除不可逆，不能把错误读成「无引用」）。
// shortcut: 复核与删除之间仍有竞态，TTL 边界上的并发发布会丢图，不做加锁。
func pruneOrphanUpload(name string) (bool, error) {
	referenced, err := fileUsage.HasAnyReferencesErr(name)
	if err != nil {
		return false, fmt.Errorf("check references for %q: %w", name, err)
	}
	if referenced {
		return false, nil
	}
	if err := filedata.DeleteByName(name); err != nil {
		return false, fmt.Errorf("delete orphan upload %q: %w", name, err)
	}
	if err := fileUsage.DeleteUploadOwnerRows([]string{name}); err != nil {
		return false, fmt.Errorf("delete upload owner rows for %q: %w", name, err)
	}
	return true, nil
}
