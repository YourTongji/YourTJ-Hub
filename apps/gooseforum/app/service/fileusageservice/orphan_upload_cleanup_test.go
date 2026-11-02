package fileusageservice

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"gorm.io/gorm"
)

const filedataEntityTable = "file_data"

// Interleave a real reference insert after the last main-DB check but before
// the first file-DB deletion. Either publication or reclamation may win; a
// successfully admitted reference must never point at deleted bytes.
func TestPruneOrphanUploadFencesReferenceAtDeletion(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-publish-at-delete.webp"
	seedOrphanCandidate(t, 9800300901, name, time.Now().Add(-8*24*time.Hour))
	var publicationErr error
	attempted := false
	callbacks := fileDB.Connect().Callback().Delete()
	const hook = "test:publish_before_orphan_delete"
	if err := callbacks.Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		if attempted {
			return
		}
		attempted = true
		publicationErr = fileUsage.Create(&fileUsage.Entity{
			FileName: name, TargetType: fileUsage.TargetTopic, TargetId: 9800300902,
			UsageType: fileUsage.UsageInlineImage, UserId: 9800300901,
		})
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = callbacks.Remove(hook) })
	_, err := pruneOrphanUpload(name)
	if err != nil {
		t.Fatal(err)
	}
	if !attempted {
		t.Fatal("publication interleaving was not reached")
	}
	if publicationErr == nil && filedata.GetByName(name).Id == 0 {
		t.Fatal("accepted a content reference but permanently deleted its image")
	}
}

func TestReclaimedUploadRejectsPreparedCover(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-prepared-cover.webp"
	const owner = 9800300903
	seedOrphanCandidate(t, owner, name, time.Now().Add(-8*24*time.Hour))
	ref, err := ResolveCover(owner, "/file/img/"+name)
	if err != nil || !ref.Owned {
		t.Fatalf("resolve: %#v %v", ref, err)
	}
	if _, err := pruneOrphanUpload(name); err != nil {
		t.Fatal(err)
	}
	err = db.Connect().Transaction(func(tx *gorm.DB) error {
		return ReplaceCoverTx(tx, owner, ref)
	})
	if err == nil {
		t.Fatal("committed a stale cover reference after reclamation")
	}
}

// seedOrphanCandidate 造一个「已上传、带归属行、创建时间可回溯」的文件。
func seedOrphanCandidate(t *testing.T, userID uint64, name string, createdAt time.Time) {
	t.Helper()
	seedOwnedFile(t, userID, name)
	// created_at 是 <-:create 字段，GORM 更新会忽略它，测试里用原始 SQL 回溯。
	if err := fileDB.Connect().Exec("UPDATE "+filedataEntityTable+" SET created_at = ? WHERE name = ?", createdAt, name).Error; err != nil {
		t.Fatalf("backdate %s: %v", name, err)
	}
	if err := AddUploadOwner(userID, name); err != nil {
		t.Fatalf("add upload owner %s: %v", name, err)
	}
	t.Cleanup(func() {
		db.Connect().Where("file_name = ?", name).Delete(&fileUsage.Entity{})
	})
}

func cleanupTestUsageRows(t *testing.T) {
	t.Helper()
	if err := db.Connect().AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
}

// seedStaleUploadOwner 造一条只存在于主库、对应文件行已经不存在的归属行，并把它回溯到
// 候选窗口最前面，模拟文件库被快照重置后的残行。
func seedStaleUploadOwner(t *testing.T, userID uint64, name string, createdAt time.Time) {
	t.Helper()
	if err := AddUploadOwner(userID, name); err != nil {
		t.Fatalf("add stale upload owner %s: %v", name, err)
	}
	if err := db.Connect().Exec("UPDATE file_usages SET created_at = ? WHERE file_name = ?", createdAt, name).Error; err != nil {
		t.Fatalf("backdate stale owner %s: %v", name, err)
	}
	t.Cleanup(func() {
		db.Connect().Where("file_name = ?", name).Delete(&fileUsage.Entity{})
	})
}

func orphanUsageCount(t *testing.T, fileName string) int64 {
	t.Helper()
	var count int64
	if err := db.Connect().Model(&fileUsage.Entity{}).Where("file_name = ?", fileName).Count(&count).Error; err != nil {
		t.Fatalf("count usages %s: %v", fileName, err)
	}
	return count
}

func TestCleanupOrphanUploadsRemovesExpiredOrphan(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-expired.webp"
	seedOrphanCandidate(t, 9_800_300_001, name, time.Now().Add(-8*24*time.Hour))
	// 图片变体是独立 file_data 行，回收原图时必须一并删除。
	variantName := name + "-small.webp"
	if _, err := filedata.SaveFile(9_800_300_001, variantName, "image/webp", []byte("variant")); err != nil {
		t.Fatalf("seed variant: %v", err)
	}
	if err := fileDB.Connect().Model(&filedata.Entity{}).Where("name = ?", variantName).UpdateColumn("parent_name", name).Error; err != nil {
		t.Fatalf("link variant: %v", err)
	}
	t.Cleanup(func() { fileDB.Connect().Where("name = ?", variantName).Delete(&filedata.Entity{}) })

	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 200)
	if err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("CleanupOrphanUploads() removed = %d, want 1", removed)
	}
	if filedata.GetByName(name).Id != 0 {
		t.Fatal("expired orphan file_data row survived cleanup")
	}
	if filedata.GetByName(variantName).Id != 0 {
		t.Fatal("expired orphan image variant survived cleanup")
	}
	if count := orphanUsageCount(t, name); count != 1 {
		t.Fatalf("expired orphan kept %d rows, want one ownership tombstone", count)
	}
	if err := fileUsage.CheckOwnedUploadAvailable(9_800_300_001, name); !errors.Is(err, fileUsage.ErrUploadReclaimed) {
		t.Fatalf("reclaimed attachment can be reused: %v", err)
	}
}

func TestCleanupOrphanUploadsKeepsFreshOrphan(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-fresh.webp"
	seedOrphanCandidate(t, 9_800_300_002, name, time.Now())

	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 200)
	if err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("CleanupOrphanUploads() removed = %d, want 0", removed)
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("upload inside the TTL was reclaimed")
	}
}

func TestCleanupOrphanUploadsKeepsReferencedUpload(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-referenced.webp"
	seedOrphanCandidate(t, 9_800_300_003, name, time.Now().Add(-8*24*time.Hour))
	if err := fileUsage.Create(&fileUsage.Entity{
		FileName: name, TargetType: fileUsage.TargetTopic, TargetId: 9_800_300_004,
		UsageType: fileUsage.UsageInlineImage, UserId: 9_800_300_003, Status: fileUsage.UsageStatusActive,
	}); err != nil {
		t.Fatalf("create inline_image usage: %v", err)
	}

	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 200)
	if err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("CleanupOrphanUploads() removed = %d, want 0", removed)
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("content-referenced upload was reclaimed")
	}
	if count := orphanUsageCount(t, name); count != 2 {
		t.Fatalf("usage rows = %d, want 2 (owner + inline reference)", count)
	}
}

func TestCleanupOrphanUploadsKeepsLegacyFileWithoutOwnerRow(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-legacy.webp"
	seedOwnedFile(t, 0, name)
	if err := fileDB.Connect().Exec("UPDATE "+filedataEntityTable+" SET created_at = ? WHERE name = ?", time.Now().Add(-30*24*time.Hour), name).Error; err != nil {
		t.Fatalf("backdate %s: %v", name, err)
	}

	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 200)
	if err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if removed != 0 {
		t.Fatalf("CleanupOrphanUploads() removed = %d, want 0", removed)
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("legacy file without upload_owner row was reclaimed")
	}
}

// TestPruneOrphanUploadKeepsFileReferencedAfterSelection 覆盖选择与删除之间的复核：
// 候选集按「无内容引用」选出，删除前若已被内容引用就必须保留。
func TestPruneOrphanUploadKeepsFileReferencedAfterSelection(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-raced.webp"
	seedOrphanCandidate(t, 9_800_300_005, name, time.Now().Add(-8*24*time.Hour))
	// 先确认它确实被选进候选窗口，再插入引用，才能覆盖「选择之后、删除之前」的复核。
	candidates, err := fileUsage.ListUnreferencedUploadOwners(200)
	if err != nil {
		t.Fatalf("ListUnreferencedUploadOwners() error = %v", err)
	}
	if !slices.Contains(candidates, name) {
		t.Fatalf("orphan %s was not selected as a candidate: %v", name, candidates)
	}
	if err := fileUsage.Create(&fileUsage.Entity{
		FileName: name, TargetType: fileUsage.TargetTopic, TargetId: 9_800_300_006,
		UsageType: fileUsage.UsageInlineImage, UserId: 9_800_300_005, Status: fileUsage.UsageStatusPending,
	}); err != nil {
		t.Fatalf("create pending usage: %v", err)
	}

	pruned, err := pruneOrphanUpload(name)
	if err != nil {
		t.Fatalf("pruneOrphanUpload() error = %v", err)
	}
	if pruned {
		t.Fatal("pruneOrphanUpload() reclaimed a file referenced after selection")
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("file_data row was deleted despite the re-check")
	}
}

// TestCleanupOrphanUploadsPrunesStaleOwnerRowsToReachOrphan 是静默饿死的回归：文件库被
// 快照重置后，文件行已不存在的陈旧归属行会占满候选窗口，limit 之下的过期孤儿永远排不
// 进来。清理任务必须先删掉这些残行再回填窗口。
func TestCleanupOrphanUploadsPrunesStaleOwnerRowsToReachOrphan(t *testing.T) {
	cleanupTestUsageRows(t)
	staleNames := []string{"2026/08/stale-a.webp", "2026/08/stale-b.webp"}
	for i, stale := range staleNames {
		seedStaleUploadOwner(t, uint64(9_800_300_201+i), stale, time.Now().Add(-9*24*time.Hour))
	}
	const name = "2026/09/orphan-behind-stale.webp"
	seedOrphanCandidate(t, 9_800_300_203, name, time.Now().Add(-8*24*time.Hour))

	// limit=2 与陈旧行数相同：不先清残行的话窗口里只有残行。
	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 2)
	if err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if removed != 1 {
		t.Fatalf("CleanupOrphanUploads() removed = %d, want 1", removed)
	}
	if filedata.GetByName(name).Id != 0 {
		t.Fatal("expired orphan behind stale owner rows was not reclaimed")
	}
	for _, stale := range staleNames {
		if count := orphanUsageCount(t, stale); count != 1 {
			t.Fatalf("stale owner rows for %s = %d, want one tombstone", stale, count)
		}
	}
}

func TestOrphanDeleteFailureKeepsClaimForRetry(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-delete-retry.webp"
	seedOrphanCandidate(t, 9800300910, name, time.Now().Add(-8*24*time.Hour))
	callbacks := fileDB.Connect().Callback().Delete()
	const hook = "test:orphan_storage_failure"
	if err := callbacks.Before("gorm:delete").Register(hook, func(tx *gorm.DB) {
		_ = tx.AddError(errors.New("storage unavailable"))
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = callbacks.Remove(hook) })
	if _, err := pruneOrphanUpload(name); err == nil {
		t.Fatal("delete error was ignored")
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("failed deletion removed bytes")
	}
	if err := fileUsage.CheckOwnedUploadAvailable(9800300910, name); !errors.Is(err, fileUsage.ErrUploadReclaimed) {
		t.Fatalf("failed deletion reopened admission: %v", err)
	}
	if err := callbacks.Remove(hook); err != nil {
		t.Fatal(err)
	}
	removed, err := CleanupOrphanUploads(context.Background(), time.Now().Add(-OrphanUploadTTL), 200)
	if err != nil || removed != 1 {
		t.Fatalf("retry: removed=%d err=%v", removed, err)
	}
	if filedata.GetByName(name).Id != 0 {
		t.Fatal("retry did not remove bytes")
	}
}

// TestPruneOrphanUploadKeepsFileWhenLookupFails 是 P1 失败关闭回归：复核查询出错时必须
// 保留文件，不能把错误当成「没有引用」后删掉不可逆的字节。
func TestPruneOrphanUploadKeepsFileWhenLookupFails(t *testing.T) {
	cleanupTestUsageRows(t)
	const name = "2026/09/orphan-lookup-fails.webp"
	seedOrphanCandidate(t, 9_800_300_009, name, time.Now().Add(-8*24*time.Hour))
	conn := db.Connect()
	if err := conn.Migrator().RenameTable("file_usages", "hotfix_file_usages_unavailable"); err != nil {
		t.Fatalf("hide file_usages: %v", err)
	}
	defer func() {
		if err := conn.Migrator().RenameTable("hotfix_file_usages_unavailable", "file_usages"); err != nil {
			t.Fatalf("restore file_usages: %v", err)
		}
	}()

	pruned, err := pruneOrphanUpload(name)
	if err == nil {
		t.Fatal("pruneOrphanUpload() error = nil, want the failed reference lookup")
	}
	if pruned {
		t.Fatal("pruneOrphanUpload() reported a reclaim despite the lookup failure")
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatal("file was deleted after a failed reference lookup")
	}
}
