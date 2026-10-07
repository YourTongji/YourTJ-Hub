package stickerservice

import (
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "滑稽", "长_名-字2", strings.Repeat("长", 64)}
	invalid := []string{"", "带 空格", "带:冒号", "带[方括号]", "带]括号", strings.Repeat("长", 65), "带/斜杠"}
	for _, name := range valid {
		if !ValidateName(name) {
			t.Fatalf("ValidateName(%q) = false, want true", name)
		}
	}
	for _, name := range invalid {
		if ValidateName(name) {
			t.Fatalf("ValidateName(%q) = true, want false", name)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"普通名字":    "普通名字",
		"带 空格 名":  "带-空格-名",
		"标点，混排!":  "标点-混排",
		"  trim ": "trim",
		"a:b]c":   "a-b-c",
	}
	for raw, want := range cases {
		if got := SanitizeName(raw); got != want {
			t.Fatalf("SanitizeName(%q) = %q, want %q", raw, got, want)
		}
	}
	if got := SanitizeName(":"); got != "" {
		t.Fatalf("SanitizeName(bare punctuation) = %q, want empty", got)
	}
	long := strings.Repeat("长", 80)
	got := SanitizeName(long)
	if runeCount := len([]rune(got)); runeCount != sticker.MaxNameLen {
		t.Fatalf("SanitizeName long rune count = %d, want %d", runeCount, sticker.MaxNameLen)
	}
}

func TestStemName(t *testing.T) {
	cases := map[string]string{
		"滑稽.png":      "滑稽",
		"dir/内/哈.jpg": "哈",
		"noext":       "noext",
		"多.个.点.gif":   "多.个.点",
		".hidden":     ".hidden",
	}
	for input, want := range cases {
		if got := StemName(input); got != want {
			t.Fatalf("StemName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStickerCacheAndInvalidation(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	InvalidateCache()

	item := sticker.Entity{
		Name:       "cache_test",
		FileName:   "stickers/cache_test.png",
		IsOfficial: true,
		IsEnabled:  true,
		SortOrder:  1,
	}
	if err := conn.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Where("name = ?", "cache_test").Delete(&sticker.Entity{}).Error
		InvalidateCache()
	})

	// 1. First resolve should hit DB and populate cache
	urls, err := ResolveURLs([]string{"cache_test"})
	if err != nil {
		t.Fatalf("ResolveURLs failed: %v", err)
	}
	if len(urls) != 1 || urls["cache_test"] == "" {
		t.Fatalf("expected resolved url, got %v", urls)
	}

	// 2. Modify DB row directly behind the cache's back (disable it via raw table without model hooks)
	if err := conn.Table("stickers").Where("id = ?", item.Id).Update("is_enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	// 3. ResolveURLs should still return cached URL because cache is valid
	cachedURLs, err := ResolveURLs([]string{"cache_test"})
	if err != nil {
		t.Fatalf("ResolveURLs failed: %v", err)
	}
	if cachedURLs["cache_test"] != urls["cache_test"] {
		t.Fatalf("expected cache hit with url %q, got %q", urls["cache_test"], cachedURLs["cache_test"])
	}

	// 4. Invalidate cache - next resolve must see the DB update (disabled)
	InvalidateCache()
	updatedURLs, err := ResolveURLs([]string{"cache_test"})
	if err != nil {
		t.Fatalf("ResolveURLs failed: %v", err)
	}
	if _, ok := updatedURLs["cache_test"]; ok {
		t.Fatalf("expected disabled sticker to not be in resolved urls after invalidation, got %v", updatedURLs)
	}
}
