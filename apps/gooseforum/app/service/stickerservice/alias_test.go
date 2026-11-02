package stickerservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
)

func aliasNames(t *testing.T, id uint64) []string {
	t.Helper()
	names, err := sticker.AliasNamesBySticker([]uint64{id})
	if err != nil {
		t.Fatal(err)
	}
	got := names[id]
	slices.Sort(got)
	return got
}

// Issue #1428: posts, excerpts and messages store [:sticker:name:] text, so a
// rename must keep every previous name resolving to the same sticker.
func TestRenamedStickerKeepsPreviousNamesResolvable(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&sticker.Entity{}, &sticker.Alias{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{}, &fileUsage.Entity{}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	row := sticker.Entity{Name: "rename_old", FileName: "stickers/rename.png", IsOfficial: true, IsEnabled: true}
	other := sticker.Entity{Name: "rename_other", FileName: "stickers/rename-other.png", IsOfficial: true, IsEnabled: true}
	if err := conn.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&other).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = conn.Where("sticker_id IN ?", []uint64{row.Id, other.Id}).Delete(&sticker.Alias{}).Error
		_ = conn.Where("id IN ?", []uint64{row.Id, other.Id}).Delete(&sticker.Entity{}).Error
		_ = conn.Where("user_id = ?", 9).Delete(&sticker.LibraryEntry{}).Error
		InvalidateCache()
	})
	InvalidateCache()
	before, err := ResolveURLs([]string{"rename_old"})
	if err != nil || before["rename_old"] == "" {
		t.Fatalf("resolve before rename = %v, %v", before, err)
	}
	want := before["rename_old"]

	if err := Save(ctx, 1, SaveInput{Id: row.Id, Name: "rename_new", IsEnabled: true}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	urls, err := ResolveURLs([]string{"rename_old", "rename_new"})
	if err != nil || urls["rename_old"] != want || urls["rename_new"] != want {
		t.Fatalf("resolve after rename = %v, %v; want both %q", urls, err, want)
	}
	items, err := Resolve([]string{"rename_old"})
	if err != nil || len(items) != 1 || items[0].Name != "rename_old" || items[0].ID != row.Id || items[0].URL != want {
		t.Fatalf("client resolve keeps the requested token: %#v, %v", items, err)
	}

	// A second rename keeps both earlier names pointing directly at the sticker.
	if err := Save(ctx, 1, SaveInput{Id: row.Id, Name: "rename_third", IsEnabled: true}); err != nil {
		t.Fatalf("second rename: %v", err)
	}
	if got := aliasNames(t, row.Id); !slices.Equal(got, []string{"rename_new", "rename_old"}) {
		t.Fatalf("aliases = %v", got)
	}
	urls, _ = ResolveURLs([]string{"rename_old", "rename_new", "rename_third"})
	if len(urls) != 3 {
		t.Fatalf("every name resolves after a rename chain: %v", urls)
	}

	// Another sticker cannot take a name that existing content still uses,
	// and a new sticker cannot either.
	if err := Save(ctx, 1, SaveInput{Id: other.Id, Name: "rename_old", IsEnabled: true}); !errors.Is(err, ErrNameReserved) {
		t.Fatalf("rename onto a held previous name: %v, want ErrNameReserved", err)
	}
	if err := Save(ctx, 1, SaveInput{Name: "rename_new", FileName: "stickers/new.png", IsEnabled: true}); !errors.Is(err, ErrNameReserved) {
		t.Fatalf("create with a held previous name: %v, want ErrNameReserved", err)
	}

	// Taking a previous name back releases that alias only.
	if err := Save(ctx, 1, SaveInput{Id: row.Id, Name: "rename_old", IsEnabled: true}); err != nil {
		t.Fatalf("rename back: %v", err)
	}
	if got := aliasNames(t, row.Id); !slices.Equal(got, []string{"rename_new", "rename_third"}) {
		t.Fatalf("aliases after rename back = %v", got)
	}

	// Collecting or removing by any previous name targets the same sticker.
	item, err := SaveToLibrary(ctx, 9, LibrarySaveInput{StickerName: "rename_third"})
	if err != nil || item.ID != row.Id || item.Name != "rename_old" {
		t.Fatalf("collect by previous name = %#v, %v", item, err)
	}
	if err := RemoveFromLibrary(ctx, 9, "rename_new"); err != nil {
		t.Fatal(err)
	}
	var count int64
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", 9).Count(&count)
	if count != 0 {
		t.Fatalf("remove by previous name left %d memberships", count)
	}

	// A disabled sticker stays unavailable under every name.
	if err := Save(ctx, 1, SaveInput{Id: row.Id, Name: "rename_old", IsEnabled: false}); err != nil {
		t.Fatal(err)
	}
	if urls, _ := ResolveURLs([]string{"rename_old", "rename_new"}); len(urls) != 0 {
		t.Fatalf("disabled sticker resolved: %v", urls)
	}
	if items, _ := Resolve([]string{"rename_new"}); len(items) != 0 {
		t.Fatalf("disabled sticker resolved by previous name: %#v", items)
	}

	// Permanent deletion ends the identity, previous names included.
	if err := Delete(ctx, row.Id); err != nil {
		t.Fatal(err)
	}
	if got := aliasNames(t, row.Id); len(got) != 0 {
		t.Fatalf("deleted sticker kept aliases %v", got)
	}
	if err := Save(ctx, 1, SaveInput{Id: other.Id, Name: "rename_new", IsEnabled: true}); err != nil {
		t.Fatalf("a released name is available again: %v", err)
	}
}
