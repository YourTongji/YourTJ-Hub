package fileusageservice

import (
	"slices"
	"testing"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
)

// seedOwnedFile creates a ready file_data row owned by userID, cleaning up any
// previous row with the same name (SaveFile rejects duplicates) and removing
// the row after the test.
func seedOwnedFile(t *testing.T, userID uint64, name string) {
	t.Helper()
	conn := fileDB.Connect()
	if err := conn.AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	conn.Where("name = ?", name).Delete(&filedata.Entity{})
	t.Cleanup(func() {
		conn.Where("name = ?", name).Delete(&filedata.Entity{})
	})
	if _, err := filedata.SaveFile(userID, name, "image/webp", []byte("img")); err != nil {
		t.Fatalf("seed file %s: %v", name, err)
	}
}

// TestFilterOwnedImageURLsOwnership is the B4 regression: explicit gallery
// images must be restricted to files the writer actually uploaded. Registering
// a foreign or unknown /file/img/ URL as an ACTIVE inline reference would pin
// another user's file and keep it publicly served after its owner's
// delete/privacy purge (GC skips files with live references).
func TestFilterOwnedImageURLsOwnership(t *testing.T) {
	seedOwnedFile(t, 101, "2026/09/b4-owner-a1.webp")
	seedOwnedFile(t, 101, "2026/09/b4-owner-a2.png")
	seedOwnedFile(t, 202, "2026/09/b4-owner-b1.webp")

	got := FilterOwnedImageURLs(101, []string{
		"/file/img/2026/09/b4-owner-a1.webp",
		"/file/img/2026/09/b4-owner-b1.webp",        // another user's file: must be dropped
		"2026/09/b4-owner-a2.png",                   // bare relative path, owned: kept
		"/file/img/2026/09/b4-ghost.webp",           // unknown file: dropped
		"https://cdn.example.com/file/img/x/y.webp", // foreign absolute URL (unresolvable row): dropped
	})
	want := []string{"/file/img/2026/09/b4-owner-a1.webp", "2026/09/b4-owner-a2.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("FilterOwnedImageURLs = %v, want %v", got, want)
	}
}

// TestRegisterTopicInlineImagesOwnedSkipsForeignFiles is the B4 usage-row
// regression: a topic whose markdown embeds another user's file must not end
// up with an ACTIVE inline_image reference for that file.
func TestRegisterTopicInlineImagesOwnedSkipsForeignFiles(t *testing.T) {
	seedOwnedFile(t, 101, "2026/09/b4-topic-own.webp")
	seedOwnedFile(t, 202, "2026/09/b4-topic-foreign.webp")

	conn := db.Connect()
	if err := conn.AutoMigrate(&fileUsage.Entity{}); err != nil {
		t.Fatalf("migrate file_usages: %v", err)
	}
	const topicID = 9_800_200_777
	t.Cleanup(func() {
		conn.Where("target_type = ? AND target_id = ?", fileUsage.TargetTopic, topicID).Delete(&fileUsage.Entity{})
	})

	RegisterTopicInlineImagesOwned(topicID, 101,
		"own ![](/file/img/2026/09/b4-topic-own.webp) foreign ![](/file/img/2026/09/b4-topic-foreign.webp)",
		[]string{"/file/img/2026/09/b4-topic-own.webp"})

	if !fileUsage.HasActiveReferences("2026/09/b4-topic-own.webp") {
		t.Fatal("own file should be referenced by the topic")
	}
	if fileUsage.HasActiveReferences("2026/09/b4-topic-foreign.webp") {
		t.Fatal("foreign file must not be pinned as an ACTIVE content reference")
	}
}
