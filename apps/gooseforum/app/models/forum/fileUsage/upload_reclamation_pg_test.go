package fileUsage

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUploadReclamationSerializesWithPublicationPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("upload_reclamation_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Exec("DROP SCHEMA " + schema + " CASCADE"); pool, _ := admin.DB(); _ = pool.Close() })
	if parsed, err := url.Parse(dsn); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		q := parsed.Query()
		q.Set("search_path", schema)
		parsed.RawQuery = q.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
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
	for _, commit := range []bool{true, false} {
		t.Run(fmt.Sprintf("publisher_commit_%v", commit), func(t *testing.T) {
			name := fmt.Sprintf("pg-upload-race-%d-%v", time.Now().UnixNano(), commit)
			owner := Entity{FileName: name, TargetType: TargetUploadOwner, TargetId: 1, UserId: 1, UsageType: UsageUploadOwner}
			if err := conn.Create(&owner).Error; err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { conn.Where("file_name = ?", name).Delete(&Entity{}) })
			tx := conn.Begin()
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			defer tx.Rollback()
			ref := Entity{FileName: name, TargetType: TargetTopic, TargetId: 1, UserId: 1, UsageType: UsageInlineImage}
			if err := tx.Create(&ref).Error; err != nil {
				t.Fatal(err)
			}
			type result struct {
				claimed bool
				err     error
			}
			done := make(chan result, 1)
			go func() { claimed, err := claimOrphanUpload(conn, name); done <- result{claimed, err} }()
			select {
			case result := <-done:
				t.Fatalf("cleanup passed an uncommitted publisher: %+v", result)
			case <-time.After(75 * time.Millisecond):
			}
			if commit {
				err = tx.Commit().Error
			} else {
				err = tx.Rollback().Error
			}
			if err != nil {
				t.Fatal(err)
			}
			select {
			case result := <-done:
				if result.err != nil || result.claimed == commit {
					t.Fatalf("claim after publication: %+v", result)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("cleanup did not resume")
			}
			if !commit {
				ref.Id = 0
				if err := conn.Create(&ref).Error; !errors.Is(err, ErrUploadReclaimed) {
					t.Fatalf("publication admitted after cleanup claim: %v", err)
				}
			}
		})
	}
}
