package stickerservice

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func TestUploadedMarketPackLifecycle(t *testing.T) {
	setupMarketTest(t)
	ctx := context.Background()
	if _, err := CreateMarketPack(ctx, CreateMarketPackInput{Name: "  "}); !errors.Is(err, ErrPackNameRequired) {
		t.Fatalf("blank name err=%v", err)
	}
	if _, err := CreateMarketPack(ctx, CreateMarketPackInput{Name: "x", Topic: "missing"}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("unknown topic err=%v", err)
	}
	pack, err := CreateMarketPack(ctx, CreateMarketPackInput{Name: "自制表情", TopicLabel: "原创"})
	if err != nil || pack.Source != MarketSourceUpload || pack.Published || pack.Topic == "" {
		t.Fatalf("create: %+v err=%v", pack, err)
	}
	again, err := CreateMarketPack(ctx, CreateMarketPackInput{Name: "第二包", TopicLabel: "原创"})
	if err != nil || again.Topic != pack.Topic {
		t.Fatalf("a topic label is reused: %+v err=%v", again, err)
	}

	red, blue := marketPNG(t, color.RGBA{R: 255, A: 255}), marketPNG(t, color.RGBA{B: 255, A: 255})
	result, err := UploadMarketPackFile(ctx, pack.Id, "开心.png", red)
	if err != nil || result.Imported != 1 {
		t.Fatalf("upload image: %+v err=%v", result, err)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, data := range map[string][]byte{"a/again.png": red, "b/blue.png": blue, "broken.png": []byte("not an image")} {
		entry, _ := writer.Create(name)
		_, _ = entry.Write(data)
	}
	_ = writer.Close()
	result, err = UploadMarketPackFile(ctx, pack.Id, "more.zip", archive.Bytes())
	if err != nil || result.Imported != 1 || result.Skipped != 1 || len(result.Failed) != 1 || result.Failed[0].Reason != "invalidImage" {
		t.Fatalf("upload zip: %+v err=%v", result, err)
	}

	stored, _ := sticker.MarketPackByID(pack.Id)
	if stored.ItemCount != 2 || stored.Status != sticker.MarketStatusReady || stored.CoverStickerID == 0 {
		t.Fatalf("pack summary = %+v", stored)
	}
	detail, err := AdminMarketPackItems(pack.Id)
	if err != nil || len(detail.Items) != 2 || detail.Items[0].Name == "" || detail.Pack.Source != MarketSourceUpload || detail.Pack.IconURL != "" {
		t.Fatalf("admin detail = %+v err=%v", detail, err)
	}
	var first sticker.Entity
	db.Connect().First(&first, detail.Items[0].StickerID)
	if first.DisplayName != "开心" || first.Pack != sticker.PackMarket || first.IsOfficial {
		t.Fatalf("uploaded sticker = %+v", first)
	}

	// Uploaded packs never queue an upstream import or a cover fetch.
	if _, err := QueueMarketImport(ctx, pack.Id); !errors.Is(err, ErrMarketPackNotFound) {
		t.Fatalf("queue import err=%v", err)
	}
	if _, err := MarketCoverPreview(ctx, pack.Id); err == nil {
		t.Fatal("uploaded packs have no upstream cover")
	}
	if _, err := UploadMarketPackFile(ctx, 999999, "a.png", red); !errors.Is(err, ErrMarketPackNotFound) {
		t.Fatalf("unknown pack err=%v", err)
	}

	name, topic := "改名", ""
	saved, err := SaveMarketPack(ctx, SaveMarketPackInput{ID: pack.Id, Published: true, SortOrder: 3, Name: &name, Topic: &topic})
	if err != nil || saved.Name != "改名" || saved.Topic != "" || !saved.Published {
		t.Fatalf("save: %+v err=%v", saved, err)
	}
	page, err := MarketCatalog("", "", 1, 20)
	if err != nil || len(page.Packs) != 1 || page.Packs[0].ID != pack.Id || page.Packs[0].CoverURL == "" {
		t.Fatalf("public catalog = %+v err=%v", page, err)
	}
	admin, err := AdminMarketCatalog(MarketSourceUpload, "", "", "", 1, 20)
	if err != nil || admin.Total != 2 {
		t.Fatalf("source filter = %+v err=%v", admin, err)
	}
	if _, err := SubscribeMarketPack(ctx, 42, pack.Id); err != nil {
		t.Fatalf("members subscribe to uploaded packs like synced ones: %v", err)
	}
}

func TestUploadedMarketAVIFFilenames(t *testing.T) {
	setupMarketTest(t)
	for _, archive := range []bool{false, true} {
		pack, err := CreateMarketPack(t.Context(), CreateMarketPackInput{Name: "AVIF"})
		if err != nil {
			t.Fatal(err)
		}
		data := readMarketFixture(t, "market-still.avif")
		name := "still.avif"
		if archive {
			var buffer bytes.Buffer
			writer := zip.NewWriter(&buffer)
			entry, _ := writer.Create("nested/still.avif")
			_, _ = entry.Write(data)
			_ = writer.Close()
			data, name = buffer.Bytes(), "pack.zip"
		}
		result, err := UploadMarketPackFile(t.Context(), pack.Id, name, data)
		if err != nil || result.Imported != 1 || len(result.Failed) != 0 {
			t.Fatalf("archive=%v result=%+v err=%v", archive, result, err)
		}
	}
}

func seedUploadedPackCapacity(t *testing.T, count int) sticker.MarketPack {
	t.Helper()
	pack, err := CreateMarketPack(t.Context(), CreateMarketPackInput{Name: "Capacity"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := UploadMarketPackFile(t.Context(), pack.Id, "first.png", marketPNG(t, color.RGBA{R: 255, A: 255}))
	if err != nil || result.Imported != 1 {
		t.Fatalf("seed: %+v %v", result, err)
	}
	var first sticker.MarketItem
	if err := db.Connect().Where("pack_id = ?", pack.Id).First(&first).Error; err != nil {
		t.Fatal(err)
	}
	rows := make([]sticker.MarketItem, 0, count-1)
	for i := 1; i < count; i++ {
		rows = append(rows, sticker.MarketItem{PackID: pack.Id, UpstreamID: fmt.Sprintf("seed-%d", i), StickerID: first.StickerID, SortOrder: i})
	}
	if len(rows) > 0 {
		if err := db.Connect().CreateInBatches(rows, 100).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := finishMarketImport(t.Context(), pack.Id, ""); err != nil {
		t.Fatal(err)
	}
	return pack
}

func TestUploadedFullPackDuplicateRemainsIdempotent(t *testing.T) {
	setupMarketTest(t)
	pack := seedUploadedPackCapacity(t, MaxMarketPackItems)
	result, err := UploadMarketPackFile(t.Context(), pack.Id, "retry.png", marketPNG(t, color.RGBA{R: 255, A: 255}))
	if err != nil || result.Skipped != 1 || result.Imported != 0 || len(result.Failed) != 0 {
		t.Fatalf("full-pack retry: %+v %v", result, err)
	}
}

func TestUploadedItemCommitRechecksPackCapacity(t *testing.T) {
	setupMarketTest(t)
	pack := seedUploadedPackCapacity(t, MaxMarketPackItems-1)
	// Two upload workers may already have observed this same count/order. The
	// commit boundary must recheck it even when their preparation was concurrent.
	for i := 0; i < 2; i++ {
		_, err := addUploadedMarketItem(t.Context(), pack, fmt.Sprintf("new-%d.png", i), marketPNG(t, color.RGBA{B: uint8(100 + i), A: 255}))
		if i == 0 && err != nil {
			t.Fatal(err)
		}
		if i == 1 && !errors.Is(err, errMarketPackFull) {
			t.Fatalf("stale capacity accepted a second item: %v", err)
		}
	}
	_, count, err := sticker.NextMarketItemOrder(db.Connect(), pack.Id)
	if err != nil || count != MaxMarketPackItems {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
