package filedata

import (
	"context"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
)

func setupFileDataTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatalf("migrate file data: %v", err)
	}
	conn.Where("1 = 1").Delete(&Entity{})
}

func TestFileResourcePageListsFilesByIDRangeWithoutContent(t *testing.T) {
	setupFileDataTestDB(t)

	text, err := SaveFile(1, "docs/readme.txt", "text/plain", []byte("text"))
	if err != nil {
		t.Fatalf("save text file: %v", err)
	}
	first, err := SaveFile(2, "images/old.png", "image/png", []byte("old"))
	if err != nil {
		t.Fatalf("save first image: %v", err)
	}
	second, err := SaveFile(3, "images/new.webp", "image/webp", []byte("new-image"))
	if err != nil {
		t.Fatalf("save second image: %v", err)
	}

	page := FileResourcePage(1, 2)
	if page.MaxId != int64(second.Id) {
		t.Fatalf("maxId = %d, want %d", page.MaxId, second.Id)
	}
	if len(page.List) != 2 {
		t.Fatalf("len = %d, want 2", len(page.List))
	}
	if page.List[0].Id != second.Id || page.List[1].Id != first.Id {
		t.Fatalf("order = [%d,%d], want [%d,%d]", page.List[0].Id, page.List[1].Id, second.Id, first.Id)
	}

	next := FileResourcePage(2, 2)
	if len(next.List) != 1 {
		t.Fatalf("next len = %d, want 1", len(next.List))
	}
	if next.List[0].Id != text.Id || next.List[0].Type != "text/plain" {
		t.Fatalf("next row = id %d type %q, want text file id %d", next.List[0].Id, next.List[0].Type, text.Id)
	}
	if page.List[0].Size != int64(len("new-image")) {
		t.Fatalf("size = %d, want %d", page.List[0].Size, len("new-image"))
	}
	if page.List[0].Data != nil {
		t.Fatal("image resource list loaded blob content")
	}
	if page.List[0].URL != "/file/img/images/new.webp" {
		t.Fatalf("url = %q, want image access path", page.List[0].URL)
	}
}

// TestCountFilesUpTo verifies the cumulative-count helper used to derive the
// task-level migration progress from the persisted cursor (id <= cursor).
func TestCountFilesUpTo(t *testing.T) {
	setupFileDataTestDB(t)

	first, err := SaveFile(1, "migrate/a.png", "image/png", []byte("a"))
	if err != nil {
		t.Fatalf("save a.png: %v", err)
	}
	second, err := SaveFile(2, "migrate/b.png", "image/png", []byte("b"))
	if err != nil {
		t.Fatalf("save b.png: %v", err)
	}
	third, err := SaveFile(3, "migrate/c.png", "image/png", []byte("c"))
	if err != nil {
		t.Fatalf("save c.png: %v", err)
	}

	if got := CountFilesUpTo(0); got != 0 {
		t.Fatalf("CountFilesUpTo(0) = %d, want 0", got)
	}
	if got := CountFilesUpTo(first.Id); got != 1 {
		t.Fatalf("CountFilesUpTo(%d) = %d, want 1", first.Id, got)
	}
	if got := CountFilesUpTo(second.Id); got != 2 {
		t.Fatalf("CountFilesUpTo(%d) = %d, want 2", second.Id, got)
	}
	if got := CountFilesUpTo(third.Id); got != 3 {
		t.Fatalf("CountFilesUpTo(%d) = %d, want 3", third.Id, got)
	}
}

func TestCreateFileMetadataAndMarkReady(t *testing.T) {
	setupFileDataTestDB(t)
	ctx := context.Background()

	entity, err := CreateFileMetadata(ctx, 7, "2026/09/01/direct.png", "image/png", 1024)
	if err != nil {
		t.Fatalf("CreateFileMetadata() error = %v", err)
	}
	if entity.StorageStatus != StorageStatusPending {
		t.Fatalf("storage status = %q, want pending", entity.StorageStatus)
	}
	if entity.Size != 1024 || entity.UserId != 7 {
		t.Fatalf("metadata = %+v, want size 1024 user 7", entity)
	}

	pending, err := GetPendingFileMetadataByNameContext(ctx, entity.Name)
	if err != nil {
		t.Fatalf("GetPendingFileMetadataByNameContext() error = %v", err)
	}
	if pending.Id != entity.Id {
		t.Fatalf("pending id = %d, want %d", pending.Id, entity.Id)
	}

	// Pending rows must not be served as ready files.
	if _, err := GetFileByName(entity.Name); err == nil {
		t.Fatal("GetFileByName() on pending row error = nil, want not found")
	}

	ready, err := MarkFileReady(ctx, entity.Name)
	if err != nil {
		t.Fatalf("MarkFileReady() error = %v", err)
	}
	if ready.StorageStatus != StorageStatusReady {
		t.Fatalf("storage status = %q, want ready", ready.StorageStatus)
	}
	if _, err := GetPendingFileMetadataByNameContext(ctx, entity.Name); err == nil {
		t.Fatal("GetPendingFileMetadataByNameContext() after ready error = nil, want not found")
	}
	if _, err := GetFileMetadataByNameContext(ctx, entity.Name); err != nil {
		t.Fatalf("GetFileMetadataByNameContext() after ready error = %v", err)
	}
}

func TestCreateFileMetadataRejectsDuplicateName(t *testing.T) {
	setupFileDataTestDB(t)
	ctx := context.Background()

	if _, err := CreateFileMetadata(ctx, 1, "2026/09/01/dup.png", "image/png", 10); err != nil {
		t.Fatalf("first CreateFileMetadata() error = %v", err)
	}
	if _, err := CreateFileMetadata(ctx, 2, "2026/09/01/dup.png", "image/png", 10); err == nil {
		t.Fatal("duplicate CreateFileMetadata() error = nil, want error")
	}
}

func TestListPendingFilesBefore(t *testing.T) {
	setupFileDataTestDB(t)
	ctx := context.Background()

	old, err := CreateFileMetadata(ctx, 1, "2026/08/01/old.png", "image/png", 10)
	if err != nil {
		t.Fatalf("CreateFileMetadata(old) error = %v", err)
	}
	// Backdate the old row past the cutoff.
	if err := db.Connect().Exec("UPDATE file_data SET created_at = ? WHERE name = ?", time.Now().Add(-3*time.Hour), old.Name).Error; err != nil {
		t.Fatalf("backdate old row: %v", err)
	}
	if _, err := CreateFileMetadata(ctx, 1, "2026/09/01/fresh.png", "image/png", 10); err != nil {
		t.Fatalf("CreateFileMetadata(fresh) error = %v", err)
	}

	items, err := ListPendingFilesBefore(ctx, time.Now().Add(-2*time.Hour), 100)
	if err != nil {
		t.Fatalf("ListPendingFilesBefore() error = %v", err)
	}
	if len(items) != 1 || items[0].Name != old.Name {
		t.Fatalf("ListPendingFilesBefore() = %+v, want only the old pending row", items)
	}
}
