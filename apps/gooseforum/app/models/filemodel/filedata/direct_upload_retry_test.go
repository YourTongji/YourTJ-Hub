package filedata

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestCompletedDirectUploadLookupSurvivesSourceCleanup(t *testing.T) {
	setupFileDataTestDB(t)
	ctx := context.Background()
	source, err := CreateFileMetadata(ctx, 7, "retry/source.png", "image/png", 4)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := CreateFileCopyMetadata(ctx, 7, "retry/copy.png", "image/png", 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PublishRenamedFileMetadata(ctx, source.Name, copy.Name); err != nil {
		t.Fatal(err)
	}
	adapter := directUploadMetadataStore{}
	assertCompleted := func() {
		t.Helper()
		for _, name := range []string{source.Name, copy.Name} {
			got, err := adapter.GetFileMetadataByName(ctx, name)
			if err != nil || got == nil || got.Name != copy.Name || got.UserId != source.UserId {
				t.Fatalf("completed lookup %q = %+v, %v", name, got, err)
			}
		}
		if _, err := GetFileMetadataByNameContext(ctx, source.Name); err == nil {
			t.Fatal("ordinary ready lookup resolved mutable source")
		}
	}
	assertCompleted()
	if err := DeleteByNameContext(ctx, source.Name); err != nil {
		t.Fatal(err)
	}
	assertCompleted()
	if err := DeleteByNameContext(ctx, copy.Name); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.GetFileMetadataByName(ctx, source.Name); err == nil {
		t.Fatal("deleted final object still resolves")
	}
}

func TestFailedDirectUploadPublicationHasNoCompletedResult(t *testing.T) {
	setupFileDataTestDB(t)
	ctx := context.Background()
	source, err := CreateFileMetadata(ctx, 7, "retry/source.png", "image/png", 4)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := CreateFileCopyMetadata(ctx, 8, "retry/foreign.png", "image/png", 4)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PublishRenamedFileMetadata(ctx, source.Name, copy.Name); err == nil {
		t.Fatal("foreign copy publication succeeded")
	}
	adapter := directUploadMetadataStore{}
	for _, name := range []string{source.Name, copy.Name} {
		if _, err := adapter.GetFileMetadataByName(ctx, name); err == nil {
			t.Fatalf("failed publication created a result for %q", name)
		}
	}
}

// File metadata uses SQLite even when the main forum database is PostgreSQL.
func TestDirectUploadSourceMigrationPreservesLegacyFiles(t *testing.T) {
	conn, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := conn.AutoMigrate(&Entity{}); err != nil {
		t.Fatal(err)
	}
	// Remove only the new column/index to reconstruct the previous schema.
	if err := conn.Migrator().DropIndex(&Entity{}, "idx_file_data_direct_upload_source"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Migrator().DropColumn(&Entity{}, "DirectUploadSourceName"); err != nil {
		t.Fatal(err)
	}
	legacy := Entity{Name: "legacy/file.png", Type: "image/png", Data: []byte("old"), Size: 3, UserId: 7, StorageStatus: StorageStatusReady}
	if err := conn.Omit("DirectUploadSourceName").Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := conn.AutoMigrate(&Entity{}); err != nil {
			t.Fatal(err)
		}
	}
	var got Entity
	if err := conn.First(&got, legacy.Id).Error; err != nil {
		t.Fatal(err)
	}
	if got.Name != legacy.Name || got.UserId != legacy.UserId || got.Size != 3 || got.StorageStatus != StorageStatusReady || string(got.Data) != "old" || got.DirectUploadSourceName != "" {
		t.Fatalf("migration changed legacy metadata: %+v", got)
	}
	if !conn.Migrator().HasIndex(&Entity{}, "idx_file_data_direct_upload_source") {
		t.Fatal("completion lookup index missing")
	}
}
