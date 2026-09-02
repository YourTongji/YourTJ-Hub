package fileusageservice

import (
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
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
}

// TestUploadOwnerDoesNotKeepFileLiveAfterContentDelete is the regression for
// review B2: the upload_owner row is an ownership-audit record, not a content
// reference. It must not (a) keep a file publicly downloadable after its
// content is deleted, nor (b) prevent physical cleanup. Without the
// usage_type filter, the always-ACTIVE upload_owner row made
// HasActiveReferences/HasLiveReferences return true forever.
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
	if !fileUsage.HasLiveReferences(fileName) {
		t.Fatal("recovering reference should keep the file from physical cleanup during the window")
	}

	// Permanent purge: content reference is PURGED; the upload_owner row must
	// not block physical deletion.
	PurgeTargetFiles(TargetRef{TargetType: fileUsage.TargetTopic, TargetID: 9_800_200_002})
	if fileUsage.HasLiveReferences(fileName) {
		t.Fatal("file should be physically cleanable after purge (upload_owner must not block deletion)")
	}
}
