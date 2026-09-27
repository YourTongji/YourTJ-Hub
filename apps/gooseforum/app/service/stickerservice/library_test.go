package stickerservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func stickerLibraryDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{}, &fileUsage.Entity{}); err != nil {
		t.Fatal(err)
	}
	return conn
}

func seedLibraryAssets(t *testing.T, conn *gorm.DB, count int) []sticker.Entity {
	t.Helper()
	items := make([]sticker.Entity, count)
	for index := range items {
		items[index] = sticker.Entity{Name: fmt.Sprintf("test_%03d", index), FileName: fmt.Sprintf("stickers/%d.png", index), IsOfficial: true, IsEnabled: true}
	}
	if err := conn.Create(&items).Error; err != nil {
		t.Fatal(err)
	}
	return items
}

func assertLibraryConcurrency(t *testing.T, conn *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	assets := seedLibraryAssets(t, conn, MaxLibraryItems+2)
	for index := 0; index < MaxLibraryItems-1; index++ {
		if err := conn.Create(&sticker.LibraryEntry{UserID: 7, StickerID: assets[index].Id, SortOrder: index}).Error; err != nil {
			t.Fatal(err)
		}
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	start := make(chan struct{})
	for _, asset := range assets[MaxLibraryItems-1 : MaxLibraryItems+1] {
		wg.Go(func() {
			<-start
			_, err := saveToLibrary(conn, ctx, 7, LibrarySaveInput{StickerName: asset.Name})
			errs <- err
		})
	}
	close(start)
	wg.Wait()
	close(errs)
	success, full := 0, 0
	for err := range errs {
		if err == nil {
			success++
		} else if errors.Is(err, ErrLibraryFull) {
			full++
		} else {
			t.Fatalf("concurrent save: %v", err)
		}
	}
	if success != 1 || full != 1 {
		t.Fatalf("quota race: success=%d full=%d", success, full)
	}
	var count int64
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", 7).Count(&count)
	if count != MaxLibraryItems {
		t.Fatalf("quota race count=%d", count)
	}
	// Retrying an existing member at quota remains successful and idempotent.
	if _, err := saveToLibrary(conn, ctx, 7, LibrarySaveInput{StickerName: assets[0].Name}); err != nil {
		t.Fatalf("idempotent collect at capacity: %v", err)
	}
	// Another account has an independent library and can collect the same asset.
	if _, err := saveToLibrary(conn, ctx, 8, LibrarySaveInput{StickerName: assets[0].Name}); err != nil {
		t.Fatal(err)
	}
	// No global label or asset changes on a private rename.
	label := "private label"
	if _, err := saveToLibrary(conn, ctx, 7, LibrarySaveInput{StickerName: assets[0].Name, DisplayName: &label}); err != nil {
		t.Fatal(err)
	}
	var asset sticker.Entity
	conn.First(&asset, assets[0].Id)
	if asset.Name != assets[0].Name || asset.DisplayName != "" {
		t.Fatal("private rename modified shared asset")
	}
}

func TestPersonalStickerLibraryQuotaAndConcurrency(t *testing.T) {
	assertLibraryConcurrency(t, stickerLibraryDB(t))
}

func TestPersonalStickerLibraryQuotaOnPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	// The configured DSN is a disposable test database, as for migration tests.
	if err := conn.Migrator().DropTable(&sticker.LibraryEntry{}, &sticker.LibraryOwner{}, &sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{}); err != nil {
		t.Fatal(err)
	}
	assertLibraryConcurrency(t, conn)
}

func TestPersonalStickerSaveRollbackPreservesMembership(t *testing.T) {
	conn := stickerLibraryDB(t)
	asset := seedLibraryAssets(t, conn, 1)[0]
	if err := conn.Callback().Create().Before("gorm:create").Register("fail_membership", func(tx *gorm.DB) {
		if tx.Statement.Table == "user_stickers" {
			_ = tx.AddError(errors.New("forced membership failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err := saveToLibrary(conn, context.Background(), 4, LibrarySaveInput{StickerName: asset.Name})
	if err == nil {
		t.Fatal("forced failure did not roll back")
	}
	var count int64
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", 4).Count(&count)
	if count != 0 {
		t.Fatal("failed save left a membership")
	}
	conn.Model(&sticker.LibraryOwner{}).Where("user_id = ?", 4).Count(&count)
	if count != 0 {
		t.Fatal("failed save committed owner transaction")
	}
}
