package stickerservice

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image/color"
	"slices"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func setupOfficialPackTest(t *testing.T) {
	t.Helper()
	setupMarketTest(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&sticker.OfficialPack{}); err != nil {
		t.Fatal(err)
	}
	clean := func() {
		conn.Where("1 = 1").Delete(&sticker.OfficialPack{})
		conn.Where("name LIKE ?", "opk_%").Delete(&sticker.Entity{})
		InvalidateCache()
	}
	clean()
	t.Cleanup(clean)
}

func officialSticker(t *testing.T, name, pack string, order int, enabled bool) sticker.Entity {
	t.Helper()
	row := sticker.Entity{Name: name, FileName: "stickers/" + name + ".png", IsOfficial: true, Pack: pack, SortOrder: order, IsEnabled: true}
	if err := db.Connect().Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if !enabled {
		if err := db.Connect().Model(&row).Update("is_enabled", false).Error; err != nil {
			t.Fatal(err)
		}
		row.IsEnabled = false
	}
	return row
}

func adminPack(t *testing.T, key string) AdminOfficialPack {
	t.Helper()
	packs, err := AdminOfficialPacks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, pack := range packs {
		if pack.Key == key {
			return pack
		}
	}
	t.Fatalf("pack %q not listed in %+v", key, packs)
	return AdminOfficialPack{}
}

func publicPack(t *testing.T, key string) (OfficialPackItem, bool) {
	t.Helper()
	packs, err := OfficialPackList()
	if err != nil {
		t.Fatal(err)
	}
	for _, pack := range packs {
		if pack.Key == key {
			return pack, true
		}
	}
	return OfficialPackItem{}, false
}

func TestOfficialPacksNameOrderAndCover(t *testing.T) {
	setupOfficialPackTest(t)
	first := officialSticker(t, "opk_first", "wxmeme", 1, true)
	second := officialSticker(t, "opk_second", "wxmeme", 2, true)
	officialSticker(t, "opk_hidden", "wxmeme", 3, false)

	pack := adminPack(t, "wxmeme")
	if pack.Name != "程序员梗" || pack.SortOrder != 30 || pack.StickerCount < 3 || pack.EnabledCount < 2 {
		t.Fatalf("preset defaults = %+v", pack)
	}
	public, ok := publicPack(t, "wxmeme")
	if !ok || public.CoverURL != ResolveURLFor(first) || public.StickerCount < 2 {
		t.Fatalf("public pack = %+v, ok=%v", public, ok)
	}

	key, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Key: "wxmeme", Name: " 程序员 ", SortOrder: -5, CoverStickerID: second.Id})
	if err != nil || key != "wxmeme" {
		t.Fatalf("save preset: key=%q err=%v", key, err)
	}
	public, _ = publicPack(t, "wxmeme")
	if public.Name != "程序员" || public.CoverURL != ResolveURLFor(second) {
		t.Fatalf("edited pack = %+v", public)
	}
	packs, err := OfficialPackList()
	if err != nil || packs[0].Key != "wxmeme" {
		t.Fatalf("a negative order sorts first: %+v, %v", packs, err)
	}

	// The cover must belong to the pack; names are required and bounded.
	other := officialSticker(t, "opk_other", sticker.PackOfficial, 1, true)
	if _, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Key: "wxmeme", Name: "x", CoverStickerID: other.Id}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("foreign cover err=%v", err)
	}
	if _, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Key: "wxmeme", Name: "  "}); !errors.Is(err, ErrPackNameRequired) {
		t.Fatalf("blank name err=%v", err)
	}
	if _, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Key: "nope-missing", Name: "x"}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("unknown key err=%v", err)
	}
	if _, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Key: sticker.PackMarket, Name: "x"}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("market key err=%v", err)
	}
}

func TestOfficialPackCreateMoveBatchAndDelete(t *testing.T) {
	setupOfficialPackTest(t)
	a := officialSticker(t, "opk_a", sticker.PackOfficial, 1, true)
	b := officialSticker(t, "opk_b", sticker.PackOfficial, 2, true)
	c := officialSticker(t, "opk_c", sticker.PackOfficial, 3, true)

	key, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Name: "新包", SortOrder: 5})
	if err != nil || !validPackKey(key) {
		t.Fatalf("create: key=%q err=%v", key, err)
	}
	if pack := adminPack(t, key); pack.Name != "新包" || pack.StickerCount != 0 {
		t.Fatalf("an empty created pack is listed for administrators: %+v", pack)
	}
	if _, ok := publicPack(t, key); ok {
		t.Fatal("an empty pack must not reach pickers")
	}

	changed, err := BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id, b.Id, b.Id, 0}, Action: BatchMove, Pack: key})
	if err != nil || changed != 2 {
		t.Fatalf("move: changed=%d err=%v", changed, err)
	}
	if public, ok := publicPack(t, key); !ok || public.StickerCount != 2 || public.CoverURL != ResolveURLFor(a) {
		t.Fatalf("moved pack = %+v ok=%v", public, ok)
	}
	if changed, err = BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id}, Action: BatchDisable}); err != nil || changed != 1 {
		t.Fatalf("disable: changed=%d err=%v", changed, err)
	}
	if public, _ := publicPack(t, key); public.StickerCount != 1 || public.CoverURL != ResolveURLFor(b) {
		t.Fatalf("a disabled sticker leaves the picker and the cover: %+v", public)
	}
	if changed, err = BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id, b.Id}, Action: BatchEnable}); err != nil || changed != 1 {
		t.Fatalf("enable skips enabled rows: changed=%d err=%v", changed, err)
	}
	if _, err := BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id}, Action: BatchMove, Pack: "personal"}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("moving into a non-official pack err=%v", err)
	}
	if _, err := BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id}, Action: "rename"}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("unknown action err=%v", err)
	}
	if _, err := BatchStickers(context.Background(), BatchInput{}); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("empty selection err=%v", err)
	}

	// Saving one sticker can move it as well.
	if err := Save(context.Background(), 0, SaveInput{Id: c.Id, Name: c.Name, Pack: key, SortOrder: 9, IsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	if row, _ := sticker.GetById(c.Id); row.Pack != key {
		t.Fatalf("saved pack = %q", row.Pack)
	}

	if _, err := DeleteOfficialPack(context.Background(), sticker.PackOfficial, key); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("the default pack cannot be removed: %v", err)
	}
	moved, err := DeleteOfficialPack(context.Background(), key, "")
	if err != nil || moved != 3 {
		t.Fatalf("delete: moved=%d err=%v", moved, err)
	}
	for _, id := range []uint64{a.Id, b.Id, c.Id} {
		if row, _ := sticker.GetById(id); row.Pack != sticker.PackOfficial {
			t.Fatalf("sticker %d stayed in %q", id, row.Pack)
		}
	}
	packs, _ := AdminOfficialPacks(context.Background())
	if slices.ContainsFunc(packs, func(p AdminOfficialPack) bool { return p.Key == key }) {
		t.Fatal("deleted pack still listed")
	}

	if changed, err = BatchStickers(context.Background(), BatchInput{IDs: []uint64{a.Id, b.Id}, Action: BatchDelete}); err != nil || changed != 2 {
		t.Fatalf("delete stickers: changed=%d err=%v", changed, err)
	}
	if _, err := sticker.GetById(a.Id); err == nil {
		t.Fatal("batch delete kept the row")
	}
}

func TestDeletePresetPackPreservesStickersAndUserMembership(t *testing.T) {
	setupOfficialPackTest(t)
	row := officialSticker(t, "opk_preset", "emojipackage", 1, true)
	entry := sticker.LibraryEntry{UserID: 991510, StickerID: row.Id, DisplayName: "收藏表情", SortOrder: 1}
	if err := db.Connect().Create(&entry).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Connect().Where("user_id = ?", entry.UserID).Delete(&sticker.LibraryEntry{})
	})

	moved, err := DeleteOfficialPack(context.Background(), "emojipackage", "")
	if err != nil || moved != 1 {
		t.Fatalf("delete preset pack: moved=%d err=%v", moved, err)
	}
	var saved sticker.Entity
	if err := db.Connect().First(&saved, row.Id).Error; err != nil || saved.Name != row.Name || saved.FileName != row.FileName || saved.Pack != sticker.PackOfficial {
		t.Fatalf("sticker after pack deletion = %+v err=%v", saved, err)
	}
	var savedEntry sticker.LibraryEntry
	if err := db.Connect().Where("user_id = ? AND sticker_id = ?", entry.UserID, row.Id).First(&savedEntry).Error; err != nil || savedEntry.DisplayName != entry.DisplayName {
		t.Fatalf("user membership after pack deletion = %+v err=%v", savedEntry, err)
	}
}

func TestImportOfficialIntoPackAcceptsImageOrZip(t *testing.T) {
	setupOfficialPackTest(t)
	key, err := SaveOfficialPack(context.Background(), SaveOfficialPackInput{Name: "导入"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Connect().Where("pack = ?", key).Delete(&sticker.Entity{}) })
	image := marketPNG(t, color.RGBA{R: 200, A: 255})
	result, err := ImportOfficial(context.Background(), 0, key, "opk_one.png", image)
	if err != nil || result.Imported != 1 {
		t.Fatalf("single image: %+v err=%v", result, err)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for _, name := range []string{"opk_two.png", "opk_three.png", "notes.txt"} {
		entry, _ := writer.Create(name)
		_, _ = entry.Write(image)
	}
	_ = writer.Close()
	result, err = ImportOfficial(context.Background(), 0, key, "more.zip", archive.Bytes())
	if err != nil || result.Imported != 2 || result.Skipped != 1 {
		t.Fatalf("zip: %+v err=%v", result, err)
	}
	var rows []sticker.Entity
	db.Connect().Where("pack = ?", key).Order("sort_order ASC").Find(&rows)
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Name)
	}
	if !slices.Equal(names, []string{"opk_one", "opk_two", "opk_three"}) || rows[0].SortOrder >= rows[1].SortOrder {
		t.Fatalf("imports append in order: %v", rows)
	}
	if _, err := ImportOfficial(context.Background(), 0, key, "notes.txt", []byte("plain text")); !errors.Is(err, ErrInvalidZip) {
		t.Fatalf("non-image err=%v", err)
	}
	if _, err := ImportOfficial(context.Background(), 0, "missing-pack", "a.png", image); !errors.Is(err, ErrPackInvalid) {
		t.Fatalf("unknown pack err=%v", err)
	}
}
