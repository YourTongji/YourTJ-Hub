package fileusageservice

import (
	"slices"
	"strings"
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

// TestGalleryImageURLsKeepsOnlyInSiteCandidates is the issue #1315 counting
// unit: only URLs that resolve to an in-site file name can ever become a
// gallery entry, so external images must not count against the gallery cap.
func TestGalleryImageURLsKeepsOnlyInSiteCandidates(t *testing.T) {
	got := GalleryImageURLs([]string{
		" https://example.com/a.png ",            // external: dropped
		"  /file/img/2026/09/gallery-a.webp  ",   // in-site, trimmed and kept
		"https://cdn.example.com/file/img/a.png", // in-site absolute: kept
		"/file/img/2026/09/gallery-a.webp",       // same file again: dropped
		"",                                       // empty: dropped
		"/file/img/" + strings.Repeat("x", MaxImageURLLength), // oversized: dropped
	})
	want := []string{"/file/img/2026/09/gallery-a.webp", "https://cdn.example.com/file/img/a.png"}
	if !slices.Equal(got, want) {
		t.Fatalf("GalleryImageURLs = %v, want %v", got, want)
	}
}

// TestGalleryAndFilterSplitOnOversizedNameLength covers the two size guards
// separately: a URL within MaxImageURLLength whose resolved file name still
// exceeds the file_usages.file_name varchar(512) column stays a gallery
// candidate but must not become a file reference.
func TestGalleryAndFilterSplitOnOversizedNameLength(t *testing.T) {
	url := "/file/img/" + strings.Repeat("x", 517)
	if len(url) > MaxImageURLLength {
		t.Fatalf("test URL is %d bytes, exceeds MaxImageURLLength %d", len(url), MaxImageURLLength)
	}
	if got := GalleryImageURLs([]string{url}); !slices.Equal(got, []string{url}) {
		t.Fatalf("GalleryImageURLs = %v, want [%s]", got, url)
	}
	if got := FilterOwnedImageURLs(101, []string{url}); len(got) != 0 {
		t.Fatalf("FilterOwnedImageURLs = %v, want empty", got)
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
		[]string{"/file/img/2026/09/b4-topic-own.webp"}, false)

	if !fileUsage.HasActiveReferences("2026/09/b4-topic-own.webp") {
		t.Fatal("own file should be referenced by the topic")
	}
	if fileUsage.HasActiveReferences("2026/09/b4-topic-foreign.webp") {
		t.Fatal("foreign file must not be pinned as an ACTIVE content reference")
	}
}
