package fileusageservice

import (
	"errors"
	"testing"
	"time"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"gorm.io/gorm"
)

func TestFileNameFromURL(t *testing.T) {
	tests := map[string]string{
		"/file/img/2026/06/a.webp":              "2026/06/a.webp",
		"https://example.com/file/img/a/b.webp": "a/b.webp",
		"avatars/1/avatar.webp":                 "avatars/1/avatar.webp",
		"/static/pic/default-avatar.webp":       "",
		"https://example.com/static/a.webp":     "",
		"../secret.webp":                        "",
	}
	for input, want := range tests {
		if got := fileNameFromURL(input); got != want {
			t.Fatalf("fileNameFromURL(%q) = %q, want %q", input, got, want)
		}
	}
	for input, want := range map[string]string{
		"https://cdn.example.com/media/2026/a.jpg":       "2026/a.jpg",
		"https://cdn.example.com.evil/media/2026/a.jpg":  "",
		"https://cdn.example.com/media/../private/a.jpg": "",
		"https://cdn.example.com/other/2026/a.jpg":       "",
		"https://other.example.com/media/2026/a.jpg":     "",
	} {
		if got := fileNameFromPublicURL(input, "https://cdn.example.com/media/"); got != want {
			t.Errorf("fileNameFromPublicURL(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestUploadOwnerDoesNotKeepFileLiveAfterContentDelete is the regression for
// review B2: the upload_owner row is an ownership-audit record, not a content
// reference. It must not keep a file publicly downloadable after its content
// is deleted. Without the usage_type filter, the always-ACTIVE upload_owner
// row made HasActiveReferences return true forever.
func TestUploadOwnerDoesNotKeepFileLiveAfterContentDelete(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	const fileName = "2026/09/upload-owner-regression.png"
	t.Cleanup(func() {
		conn.Unscoped().Where("file_name = ?", fileName).Delete(&fileUsage.Entity{})
	})

	// A completed direct upload carries an upload_owner ownership row; the same
	// file is then referenced inline by a topic (ACTIVE content reference).
	if err := fileUsage.Create(&fileUsage.Entity{
		FileName: fileName, TargetType: fileUsage.TargetUploadOwner, TargetId: 9_800_200_001,
		UsageType: fileUsage.UsageUploadOwner, UserId: 9_800_200_001, Status: fileUsage.UsageStatusActive,
	}); err != nil {
		t.Fatalf("create upload_owner usage: %v", err)
	}
	if err := fileUsage.Create(&fileUsage.Entity{
		FileName: fileName, TargetType: fileUsage.TargetTopic, TargetId: 9_800_200_002,
		UsageType: fileUsage.UsageInlineImage, UserId: 9_800_200_001, Status: fileUsage.UsageStatusActive,
	}); err != nil {
		t.Fatalf("create inline_image usage: %v", err)
	}

	// While the content is live the file stays downloadable.
	if !fileUsage.HasActiveReferences(fileName) {
		t.Fatal("file should be downloadable while content reference is ACTIVE")
	}

	// Content is deleted: its inline reference enters the 30-day RECOVERING
	// window. The upload_owner row stays ACTIVE but must not authorize reads.
	HardenTargetFiles(TargetRef{TargetType: fileUsage.TargetTopic, TargetID: 9_800_200_002}, time.Now().Add(30*24*time.Hour))
	if fileUsage.HasActiveReferences(fileName) {
		t.Fatal("file still publicly downloadable after content delete (upload_owner must not keep it live)")
	}
	if !fileUsage.HasAnyReferences(fileName) {
		t.Fatal("file should still be tracked during the recovering window")
	}

	// Permanent purge (MADR-0021): content reference is PURGED, which revokes
	// the public download grant; the file bytes stay in storage for forensics
	// (data-retention final state) instead of being physically deleted.
	RetireTargetFiles(TargetRef{TargetType: fileUsage.TargetTopic, TargetID: 9_800_200_002})
	if fileUsage.HasActiveReferences(fileName) {
		t.Fatal("file still publicly downloadable after purge (purged reference must not authorize reads)")
	}
}

// TestReplaceTargetUsagesKeepsOldRowsWhenInsertFails 是 P1 事务回归：删除旧引用与插入
// 新引用必须同生共死，否则一次插入失败会让活着的帖子没有任何内联图片引用，孤儿清理
// 随后会不可逆地删掉它的图。
func TestReplaceTargetUsagesKeepsOldRowsWhenInsertFails(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	seedOwnedFile(t, 101, "2026/09/tx-old.webp")
	seedOwnedFile(t, 101, "2026/09/tx-new.webp")
	const topicID = 9_800_200_901
	t.Cleanup(func() {
		conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetTopic, topicID).Delete(&fileUsage.Entity{})
	})
	RegisterTopicInlineImagesOwned(topicID, 101, "![](/file/img/2026/09/tx-old.webp)", nil, false)
	if !fileUsage.HasActiveReferences("2026/09/tx-old.webp") {
		t.Fatal("initial reference was not registered")
	}

	const callback = "test:fail_file_usage_insert"
	if err := conn.Callback().Create().Before("gorm:create").Register(callback, func(tx *gorm.DB) {
		if tx.Statement.Table == "file_usages" {
			_ = tx.AddError(errors.New("test: insert failed"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove(callback) })

	RegisterTopicInlineImagesOwned(topicID, 101, "![](/file/img/2026/09/tx-new.webp)", nil, false)
	if !fileUsage.HasActiveReferences("2026/09/tx-old.webp") {
		t.Fatal("failed insert deleted the previous reference (delete and insert are not transactional)")
	}
}

// coverRows 读取目标用户的封面引用行。
func coverRows(t *testing.T, conn *gorm.DB, userID uint64) []fileUsage.Entity {
	t.Helper()
	var rows []fileUsage.Entity
	if err := conn.Where("target_type = ? AND target_id = ? AND usage_type = ?",
		fileUsage.TargetUser, userID, fileUsage.UsageCover).Find(&rows).Error; err != nil {
		t.Fatalf("load cover rows: %v", err)
	}
	return rows
}

// replaceCover 按生产顺序替换封面：先在事务外解析（存储配置/文件库读），再在事务里
// 只写引用行。
func replaceCover(t *testing.T, conn *gorm.DB, userID uint64, coverURL string, ownerIDs ...uint64) error {
	t.Helper()
	ref, err := ResolveCover(userID, coverURL, ownerIDs...)
	if err != nil {
		return err
	}
	return conn.Transaction(func(tx *gorm.DB) error {
		return ReplaceCoverTx(tx, userID, ref)
	})
}

// TestReplaceCoverAcceptsManagerUpload 覆盖代管路径：机器人资料由管理员编辑，封面由
// 操作者上传，引用必须记在目标用户名下并放行操作者的归属。
func TestReplaceCoverAcceptsManagerUpload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	const (
		botID     = 9_800_200_911
		managerID = 9_800_200_912
		fileName  = "2026/09/agent-cover.webp"
	)
	seedOwnedFile(t, managerID, fileName)
	t.Cleanup(func() {
		conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetUser, botID).Delete(&fileUsage.Entity{})
	})

	if err := replaceCover(t, conn, botID, "/file/img/"+fileName, managerID); err != nil {
		t.Fatalf("ReplaceCoverTx() error = %v", err)
	}
	rows := coverRows(t, conn, botID)
	if len(rows) != 1 || rows[0].FileName != fileName || rows[0].Status != fileUsage.UsageStatusActive {
		t.Fatalf("cover rows = %+v, want one ACTIVE %s on the target user", rows, fileName)
	}

	// 第三方上传的文件不能借封面固定成公开引用。
	const foreignFile = "2026/09/foreign-cover.webp"
	seedOwnedFile(t, 9_800_200_913, foreignFile)
	if err := replaceCover(t, conn, botID, "/file/img/"+foreignFile, managerID); err != nil {
		t.Fatalf("ReplaceCoverTx(foreign) error = %v", err)
	}
	if rows := coverRows(t, conn, botID); len(rows) != 0 {
		t.Fatalf("foreign cover rows = %+v, want none", rows)
	}
}

// TestReplaceCoverKeepsUnchangedCoverAcrossManagers 是代管回归：封面由管理员 A 上传，
// 管理员 B 保存同样的封面 URL（或只改别的字段）时，A 的引用不能被吊销。
func TestReplaceCoverKeepsUnchangedCoverAcrossManagers(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	const (
		botID     = 9_800_200_921
		managerA  = 9_800_200_922
		managerB  = 9_800_200_923
		fileName  = "2026/09/manager-a-cover.webp"
		nextCover = "2026/09/manager-a-next-cover.webp"
	)
	seedOwnedFile(t, managerA, fileName)
	seedOwnedFile(t, managerA, nextCover)
	t.Cleanup(func() {
		conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetUser, botID).Delete(&fileUsage.Entity{})
	})

	if err := replaceCover(t, conn, botID, "/file/img/"+fileName, managerA); err != nil {
		t.Fatalf("ReplaceCoverTx(manager A) error = %v", err)
	}
	// 管理员 B 再保存同一张封面：引用保留，不做归属校验。
	if err := replaceCover(t, conn, botID, "/file/img/"+fileName, managerB); err != nil {
		t.Fatalf("ReplaceCoverTx(manager B, unchanged) error = %v", err)
	}
	rows := coverRows(t, conn, botID)
	if len(rows) != 1 || rows[0].FileName != fileName || rows[0].Status != fileUsage.UsageStatusActive {
		t.Fatalf("unchanged cover rows = %+v, want the retained %s row", rows, fileName)
	}

	// 真正换封面：旧行必须被替换（旧封面重回收）。
	if err := replaceCover(t, conn, botID, "/file/img/"+nextCover, managerA); err != nil {
		t.Fatalf("ReplaceCoverTx(new cover) error = %v", err)
	}
	rows = coverRows(t, conn, botID)
	if len(rows) != 1 || rows[0].FileName != nextCover {
		t.Fatalf("replaced cover rows = %+v, want only %s", rows, nextCover)
	}
}

// TestResolveCoverFailsClosedOnLookupError 是 P1 失败关闭回归：文件库查询出错时不能把
// 归属读成「没有」，否则封面引用会被吊销、URL 却已经改成新值。
func TestResolveCoverFailsClosedOnLookupError(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	const (
		userID   = 9_800_200_931
		fileName = "2026/09/failclosed-cover.webp"
	)
	seedOwnedFile(t, userID, fileName)
	t.Cleanup(func() {
		conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetUser, userID).Delete(&fileUsage.Entity{})
	})
	if err := replaceCover(t, conn, userID, "/file/img/"+fileName); err != nil {
		t.Fatalf("ReplaceCoverTx() error = %v", err)
	}

	fileConn := fileDB.Connect()
	if err := fileConn.Migrator().RenameTable("file_data", "hotfix_file_data_unavailable"); err != nil {
		t.Fatalf("hide file_data: %v", err)
	}
	defer func() {
		if err := fileConn.Migrator().RenameTable("hotfix_file_data_unavailable", "file_data"); err != nil {
			t.Fatalf("restore file_data: %v", err)
		}
	}()
	if _, err := ResolveCover(userID, "/file/img/2026/09/other-cover.webp"); err == nil {
		t.Fatal("ResolveCover() error = nil, want the failed ownership lookup")
	}
	if rows := coverRows(t, conn, userID); len(rows) != 1 || rows[0].FileName != fileName {
		t.Fatalf("cover rows after failed lookup = %+v, want the retained %s row", rows, fileName)
	}
}
