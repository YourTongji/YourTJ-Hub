package stickerservice

import (
	"errors"
	"fmt"
	"image/color"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestUploadedMarketCapacityConcurrentPostgreSQL(t *testing.T) {
	dsn := os.Getenv("YOURTJ_TEST_PG_URL")
	if dsn == "" {
		t.Skip("YOURTJ_TEST_PG_URL not set")
	}
	setupMarketTest(t) // Own disposable image store, no external market requests.
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminSQL.Close() })
	schema := fmt.Sprintf("sticker_capacity_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error })
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " search_path=" + schema
	}
	conn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, err := conn.DB()
	if err != nil {
		t.Fatal(err)
	}
	pool.SetMaxOpenConns(6)
	t.Cleanup(func() { _ = pool.Close() })
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.Alias{}, &sticker.MarketTopic{}, &sticker.MarketPack{}, &sticker.MarketItem{}, &fileUsage.Entity{}); err != nil {
		t.Fatal(err)
	}
	pack := sticker.MarketPack{Source: MarketSourceUpload, UpstreamID: "capacity", Name: "Concurrent uploads"}
	if err := conn.Create(&pack).Error; err != nil {
		t.Fatal(err)
	}
	firstName, err := storeMarketImage(t.Context(), marketPNG(t, color.RGBA{R: 255, A: 255}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := commitUploadedMarketItem(conn, pack.Id, "first", firstName); err != nil {
		t.Fatal(err)
	}
	var first sticker.MarketItem
	if err := conn.Where("pack_id = ?", pack.Id).First(&first).Error; err != nil {
		t.Fatal(err)
	}
	rows := make([]sticker.MarketItem, 0, MaxMarketPackItems-2)
	for i := 1; i < MaxMarketPackItems-1; i++ {
		rows = append(rows, sticker.MarketItem{PackID: pack.Id, UpstreamID: fmt.Sprintf("seed-%d", i), StickerID: first.StickerID, SortOrder: i})
	}
	if err := conn.CreateInBatches(rows, 100).Error; err != nil {
		t.Fatal(err)
	}
	const workers = 5
	names := make([]string, workers)
	for i := range names {
		names[i], err = storeMarketImage(t.Context(), marketPNG(t, color.RGBA{B: uint8(100 + i), A: 255}))
		if err != nil {
			t.Fatal(err)
		}
	}
	start := make(chan struct{})
	results := make(chan error, workers)
	var group sync.WaitGroup
	for i := range names {
		group.Go(func() {
			<-start
			_, err := commitUploadedMarketItem(conn.WithContext(t.Context()), pack.Id, fmt.Sprintf("item-%d", i), names[i])
			results <- err
		})
	}
	close(start)
	group.Wait()
	close(results)
	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if !errors.Is(err, errMarketPackFull) {
			t.Fatal(err)
		}
	}
	order, count, err := sticker.NextMarketItemOrder(conn, pack.Id)
	if err != nil || accepted != 1 || count != MaxMarketPackItems || order != MaxMarketPackItems {
		t.Fatalf("accepted=%d count=%d next=%d err=%v", accepted, count, order, err)
	}
	skipped, err := commitUploadedMarketItem(conn, pack.Id, "retry", firstName)
	if err != nil || !skipped {
		t.Fatalf("full-pack retry: skipped=%v err=%v", skipped, err)
	}
}
