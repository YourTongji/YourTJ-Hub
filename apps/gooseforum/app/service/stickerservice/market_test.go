package stickerservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

func marketBool(value bool) *bool { return &value }

// fakeMarket serves fixed upstream responses keyed by absolute URL.
type fakeMarket struct {
	mu     sync.Mutex
	json   map[string]string
	images map[string][]byte
	fail   map[string]error
	calls  map[string]int
	// block holds an image download until the test closes the channel.
	block map[string]chan struct{}
}

func newFakeMarket() *fakeMarket {
	return &fakeMarket{json: map[string]string{}, images: map[string][]byte{}, fail: map[string]error{}, calls: map[string]int{}, block: map[string]chan struct{}{}}
}

func (f *fakeMarket) serve(raw string, body []byte, contentType string) (safefetch.Result, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls[raw]++
	if err := f.fail[raw]; err != nil {
		return safefetch.Result{}, err
	}
	final, _ := url.Parse(raw)
	if body == nil {
		return safefetch.Result{StatusCode: http.StatusNotFound, FinalURL: final}, nil
	}
	return safefetch.Result{StatusCode: http.StatusOK, FinalURL: final, ContentType: contentType, Body: body}, nil
}

func (f *fakeMarket) FetchJSON(_ context.Context, raw string) (safefetch.Result, error) {
	f.mu.Lock()
	body, ok := f.json[raw]
	f.mu.Unlock()
	if !ok {
		return f.serve(raw, nil, "")
	}
	return f.serve(raw, []byte(body), "application/json")
}

func (f *fakeMarket) FetchImage(_ context.Context, raw string) (safefetch.Result, error) {
	f.mu.Lock()
	body, gate := f.images[raw], f.block[raw]
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	return f.serve(raw, body, "image/png")
}

func (f *fakeMarket) callCount(raw string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[raw]
}

func marketPNG(t *testing.T, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for x := range 2 {
		for y := range 2 {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const testMarketBase = defaultMarketBaseURL + "/assets/market/"

func setupMarketTest(t *testing.T) *fakeMarket {
	t.Helper()
	conn := db.Connect()
	models := []any{&sticker.Entity{}, &sticker.Alias{}, &sticker.LibraryOwner{}, &sticker.LibraryEntry{}, &sticker.MarketTopic{}, &sticker.MarketPack{}, &sticker.MarketItem{}, &sticker.PackSubscription{}, &fileUsage.Entity{}, &taskQueue.Entity{}}
	if err := conn.AutoMigrate(models...); err != nil {
		t.Fatal(err)
	}
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatal(err)
	}
	clean := func() {
		for _, model := range []any{&sticker.MarketTopic{}, &sticker.MarketPack{}, &sticker.MarketItem{}, &sticker.PackSubscription{}, &sticker.LibraryOwner{}, &sticker.Alias{}} {
			conn.Where("1 = 1").Delete(model)
		}
		conn.Where("pack = ?", sticker.PackMarket).Delete(&sticker.Entity{})
		conn.Where("type = ?", TaskTypeMarketImport).Delete(&taskQueue.Entity{})
		conn.Where("target_type = ?", fileUsage.TargetSticker).Delete(&fileUsage.Entity{})
		db4fileconnect.Connect().Where("name LIKE ?", MarketAssetPrefix+"%").Delete(&filedata.Entity{})
		InvalidateCache()
	}
	clean()
	fake := newFakeMarket()
	restore := SwapMarketFetcher(fake)
	t.Cleanup(func() {
		restore()
		clean()
	})
	return fake
}

func (f *fakeMarket) catalog(groups ...string) {
	f.json[testMarketBase+"index/topics.json"] = `{"topics":[{"id":"all","label":"全部","totalPages":2},{"id":"anime","label":"动画"},{"id":"bad\u0007","label":""}]}`
	half := (len(groups) + 1) / 2
	page := func(items []string) string {
		return `{"groups":[` + strings.Join(items, ",") + `],"totalPages":2}`
	}
	f.json[testMarketBase+"index/page-1.json"] = page(groups[:half])
	f.json[testMarketBase+"index/page-2.json"] = page(groups[half:])
}

func marketGroup(id, name string, count int) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"topic":"anime","emojiCount":%d,"isArchived":false}`, id, name, count)
}

func runQueuedMarketImport(t *testing.T, packID uint64) sticker.MarketPack {
	t.Helper()
	if _, err := QueueMarketImport(context.Background(), packID); err != nil {
		t.Fatalf("queue import: %v", err)
	}
	var task taskQueue.Entity
	if err := db.Connect().Where("type = ? AND status = ?", TaskTypeMarketImport, taskQueue.StatusPending).Order("id DESC").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := RunMarketImportTask(context.Background(), &task); err != nil {
		t.Fatalf("run import: %v", err)
	}
	db.Connect().Model(&task).Update("status", taskQueue.StatusSuccess)
	pack, err := sticker.MarketPackByID(packID)
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func packByUpstream(t *testing.T, upstreamID string) sticker.MarketPack {
	t.Helper()
	var pack sticker.MarketPack
	if err := db.Connect().Where("source = ? AND upstream_id = ?", MarketSourceID, upstreamID).First(&pack).Error; err != nil {
		t.Fatalf("pack %s: %v", upstreamID, err)
	}
	return pack
}

func marketFileCount(t *testing.T) int64 {
	t.Helper()
	var count int64
	db4fileconnect.Connect().Model(&filedata.Entity{}).Where("name LIKE ?", MarketAssetPrefix+"%").Count(&count)
	return count
}

func TestMarketCatalogSyncKeepsLastSnapshotOnFailure(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	fake.catalog(marketGroup("group-1", "Pack One", 3), `{"id":"../escape","name":"x"}`, marketGroup("group-2", "Pack Two", 2))
	result, err := SyncMarketCatalog(ctx)
	if err != nil || result.Packs != 2 || result.Topics != 2 {
		t.Fatalf("sync = %#v, %v", result, err)
	}
	pack := packByUpstream(t, "group-1")
	if pack.Status != sticker.MarketStatusNew || pack.Published || pack.Name != "Pack One" {
		t.Fatalf("new packs stay hidden until imported and published: %#v", pack)
	}
	if page, _ := MarketCatalog("", "", 1, 24); page.Total != 0 {
		t.Fatalf("an unpublished catalog leaked to users: %#v", page)
	}

	// A failed page keeps the previous complete snapshot.
	fake.fail[testMarketBase+"index/page-2.json"] = &safefetch.FetchError{Class: safefetch.ErrorTimeout}
	if _, err := SyncMarketCatalog(ctx); !errors.Is(err, ErrMarketUnavailable) {
		t.Fatalf("failed sync = %v, want ErrMarketUnavailable", err)
	}
	if pack := packByUpstream(t, "group-2"); pack.Missing {
		t.Fatal("a failed sync marked packs missing")
	}

	// A complete catalog without a pack marks it missing; local state survives.
	delete(fake.fail, testMarketBase+"index/page-2.json")
	fake.catalog(marketGroup("group-1", "Pack One renamed", 3))
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	if pack := packByUpstream(t, "group-2"); !pack.Missing {
		t.Fatal("a pack absent from the catalog was not marked missing")
	}
	if pack := packByUpstream(t, "group-1"); pack.Missing || pack.Name != "Pack One renamed" {
		t.Fatalf("metadata refresh = %#v", pack)
	}
}

// Sync keeps the upstream cover for administrators: allow-listed HTTPS images
// or a short text cover; other hosts and plain HTTP are dropped.
func TestMarketCatalogSyncKeepsUpstreamIcons(t *testing.T) {
	fake := setupMarketTest(t)
	icon := func(id, value string) string {
		return fmt.Sprintf(`{"id":%q,"name":%q,"icon":%q,"topic":"anime","emojiCount":1,"isArchived":false}`, id, id, value)
	}
	fake.catalog(
		icon("group-a", "https://cdn3.ldstatic.com/original/a.webp"),
		icon("group-b", " 🌿 "),
		icon("group-c", "https://tracker.example/pixel.gif"),
		icon("group-d", "http://cdn3.ldstatic.com/a.webp"),
	)
	if _, err := SyncMarketCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string][2]string{
		"group-a": {"https://cdn3.ldstatic.com/original/a.webp", ""},
		"group-b": {"", "🌿"},
		"group-c": {"", ""},
		"group-d": {"", ""},
	} {
		pack := packByUpstream(t, id)
		if pack.IconURL != want[0] || pack.IconText != want[1] {
			t.Fatalf("%s icon = (%q, %q), want (%q, %q)", id, pack.IconURL, pack.IconText, want[0], want[1])
		}
	}
	page, err := AdminMarketCatalog("", "", "", "", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, pack := range page.Packs {
		if pack.UpstreamID == "group-a" && pack.IconURL == "" {
			t.Fatal("admin catalog omits the upstream icon")
		}
	}
}

func TestMarketImportDeduplicatesContentAndPublishes(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	red := marketPNG(t, color.RGBA{R: 255, A: 255})
	blue := marketPNG(t, color.RGBA{B: 255, A: 255})
	fake.catalog(marketGroup("group-1", "Pack One", 5), marketGroup("group-2", "Pack Two", 2))
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	fake.json[testMarketBase+"group-1.json"] = `{"id":"group-1","emojis":[
		{"id":"e1","url":"https://cdn3.ldstatic.com/a.png"},
		{"id":"e2","url":"https://cdn3.ldstatic.com/a-copy.png"},
		{"id":"e3","url":"https://evil.example/x.png"},
		{"id":"e4","url":"https://cdn3.ldstatic.com/not-image.png"},
		{"id":"e1","url":"https://cdn3.ldstatic.com/duplicate-id.png"}]}`
	fake.images["https://cdn3.ldstatic.com/a.png"] = red
	fake.images["https://cdn3.ldstatic.com/a-copy.png"] = red
	fake.images["https://cdn3.ldstatic.com/not-image.png"] = []byte("<html>not an image</html>")
	fake.images["https://evil.example/x.png"] = red

	one := packByUpstream(t, "group-1")
	if _, err := QueueMarketImport(ctx, one.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := QueueMarketImport(ctx, one.Id); !errors.Is(err, ErrMarketBusy) {
		t.Fatalf("second queue = %v, want ErrMarketBusy", err)
	}
	db.Connect().Model(&sticker.MarketPack{}).Where("id = ?", one.Id).Update("status", sticker.MarketStatusNew)
	db.Connect().Where("type = ?", TaskTypeMarketImport).Delete(&taskQueue.Entity{})
	one = runQueuedMarketImport(t, one.Id)
	if one.Status != sticker.MarketStatusPartial || one.ItemCount != 2 || one.FailedCount != 2 || one.CoverStickerID == 0 {
		t.Fatalf("partial import summary = %#v", one)
	}
	if fake.callCount("https://evil.example/x.png") != 0 {
		t.Fatal("a host outside the allowlist was fetched")
	}
	items, _ := sticker.MarketItems(one.Id)
	reasons := map[string]string{}
	for _, item := range items {
		reasons[item.UpstreamID] = item.Error
	}
	if reasons["e3"] != "blocked" || reasons["e4"] != "invalid" || reasons["e1"] != "" || reasons["e2"] != "" {
		t.Fatalf("item reasons = %v", reasons)
	}
	if got := marketFileCount(t); got != 1 {
		t.Fatalf("identical bytes from two URLs stored %d objects, want 1", got)
	}

	// A pack cannot be published before it has stickers.
	two := packByUpstream(t, "group-2")
	if _, err := SaveMarketPack(ctx, SaveMarketPackInput{ID: two.Id, Published: true}); !errors.Is(err, ErrMarketPackNotReady) {
		t.Fatalf("publish empty pack = %v", err)
	}
	if _, err := SaveMarketPack(ctx, SaveMarketPackInput{ID: one.Id, Published: true, SortOrder: 3}); err != nil {
		t.Fatal(err)
	}
	page, err := MarketCatalog("", "", 1, 24)
	if err != nil || page.Total != 1 || page.Packs[0].StickerCount != 2 || page.Packs[0].CoverURL == "" || len(page.Topics) != 1 || page.Topics[0].Label != "动画" {
		t.Fatalf("published catalog = %#v, %v", page, err)
	}
	detail, err := MarketPackStickers(one.Id)
	if err != nil || len(detail.Stickers) != 2 {
		t.Fatalf("pack detail = %#v, %v", detail, err)
	}
	first := detail.Stickers[0]
	if !strings.HasPrefix(first.Name, "m_") || first.IsOfficial || first.Pack != sticker.PackMarket || !ValidateName(first.Name) {
		t.Fatalf("market sticker item = %#v", first)
	}
	if detail.Stickers[0].URL != detail.Stickers[1].URL {
		t.Fatal("two stickers with identical bytes should share one stored image")
	}
	if official, _ := EnabledList(); len(official) != 0 {
		for _, item := range official {
			if item.Pack == sticker.PackMarket {
				t.Fatal("market stickers leaked into the official directory")
			}
		}
	}

	// Another pack that reuses an image stores only the new bytes.
	fake.json[testMarketBase+"group-2.json"] = `{"id":"group-2","emojis":[{"id":"f1","url":"https://idcflare.com/z.png"},{"id":"f2","url":"https://idcflare.com/b.png"}]}`
	fake.images["https://idcflare.com/z.png"] = red
	fake.images["https://idcflare.com/b.png"] = blue
	two = runQueuedMarketImport(t, two.Id)
	if two.Status != sticker.MarketStatusReady || two.ItemCount != 2 {
		t.Fatalf("second pack = %#v", two)
	}
	if got := marketFileCount(t); got != 2 {
		t.Fatalf("stored objects = %d, want 2", got)
	}

	// Retrying keeps materialized items immutable and only fetches what failed.
	calls := fake.callCount("https://cdn3.ldstatic.com/a.png")
	fake.images["https://cdn3.ldstatic.com/not-image.png"] = blue
	retried := runQueuedMarketImport(t, one.Id)
	if retried.ItemCount != 3 || retried.FailedCount != 1 || fake.callCount("https://cdn3.ldstatic.com/a.png") != calls {
		t.Fatalf("retry = %#v, calls %d -> %d", retried, calls, fake.callCount("https://cdn3.ldstatic.com/a.png"))
	}
	if !retried.Published {
		t.Fatal("an import changed the publication decision")
	}

	// Subscriptions follow publication; sent tokens outlive both.
	if _, err := SubscribeMarketPack(ctx, 21, one.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := SubscribeMarketPack(ctx, 21, one.Id); err != nil {
		t.Fatalf("repeat subscribe: %v", err)
	}
	if _, err := SubscribeMarketPack(ctx, 21, two.Id); !errors.Is(err, ErrMarketPackUnavailable) {
		t.Fatalf("subscribe unpublished = %v", err)
	}
	mine, err := MyMarketPacks(ctx, 21)
	if err != nil || len(mine) != 1 || mine[0].ID != one.Id {
		t.Fatalf("my packs = %#v, %v", mine, err)
	}
	if _, err := SaveMarketPack(ctx, SaveMarketPackInput{ID: one.Id, Published: false}); err != nil {
		t.Fatal(err)
	}
	if mine, _ := MyMarketPacks(ctx, 21); len(mine) != 0 {
		t.Fatal("a hidden pack stayed in the picker")
	}
	if _, err := MarketPackStickers(one.Id); !errors.Is(err, ErrMarketPackUnavailable) {
		t.Fatalf("hidden pack detail = %v", err)
	}
	if resolved, _ := Resolve([]string{first.Name}); len(resolved) != 1 {
		t.Fatal("hiding a pack broke a sent sticker")
	}
	if urls, _ := ResolveURLs([]string{first.Name}); urls[first.Name] == "" {
		t.Fatal("hiding a pack broke server rendering of a sent sticker")
	}

	// Withdrawing one sticker stops it everywhere, like a disabled official one.
	if _, _, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: first.ID, IsEnabled: marketBool(false)}); err != nil {
		t.Fatal(err)
	}
	if resolved, _ := Resolve([]string{first.Name}); len(resolved) != 0 {
		t.Fatal("a withdrawn market sticker still resolves")
	}
	if _, _, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: first.ID + 100000}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("toggle unknown sticker = %v", err)
	}

	if err := UnsubscribeMarketPack(ctx, 21, one.Id); err != nil {
		t.Fatal(err)
	}
	if rows, _ := sticker.Subscriptions(21); len(rows) != 0 {
		t.Fatal("unsubscribe kept the membership")
	}
}

func TestMarketImportFailureRetries(t *testing.T) {
	fake := setupMarketTest(t)
	fake.catalog(marketGroup("group-9", "Unreachable", 1))
	if _, err := SyncMarketCatalog(context.Background()); err != nil {
		t.Fatal(err)
	}
	pack := packByUpstream(t, "group-9")
	if _, err := QueueMarketImport(context.Background(), pack.Id); err != nil {
		t.Fatal(err)
	}
	var task taskQueue.Entity
	db.Connect().Where("type = ?", TaskTypeMarketImport).First(&task)
	if err := RunMarketImportTask(context.Background(), &task); err == nil {
		t.Fatal("a missing group document must fail the task so the queue retries")
	}
	pack, _ = sticker.MarketPackByID(pack.Id)
	if pack.Status != sticker.MarketStatusFailed || pack.LastError != "status" {
		t.Fatalf("failed import = %#v", pack)
	}
}

// Progress counters move while the run is active, an all-AVIF pack reports its
// shared reason, and a retry starts the failed items over.
func TestMarketImportReportsProgressAndSharedReason(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	fake.catalog(marketGroup("group-7", "Progress", 3))
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	fake.json[testMarketBase+"group-7.json"] = `{"id":"group-7","emojis":[
		{"id":"a","url":"https://cdn3.ldstatic.com/a.avif"},
		{"id":"b","url":"https://cdn3.ldstatic.com/b.avif"},
		{"id":"c","url":"https://cdn3.ldstatic.com/c.avif"}]}`
	avif := append([]byte{0, 0, 0, 0x2c}, []byte("ftypavis\x00\x00\x00\x00avisavif")...)
	for _, name := range []string{"a", "b", "c"} {
		fake.images["https://cdn3.ldstatic.com/"+name+".avif"] = avif
	}
	pack := runQueuedMarketImport(t, packByUpstream(t, "group-7").Id)
	if pack.Status != sticker.MarketStatusFailed || pack.LastError != "unsupported" || pack.FailedCount != 3 || pack.ImportTotal != 3 {
		t.Fatalf("all-AVIF import = %#v, want failed with the shared unsupported reason", pack)
	}

	png := marketPNG(t, color.RGBA{G: 255, A: 255})
	for _, name := range []string{"a", "b", "c"} {
		fake.images["https://cdn3.ldstatic.com/"+name+".avif"] = png
	}
	gate := make(chan struct{})
	fake.block["https://cdn3.ldstatic.com/c.avif"] = gate
	if _, err := QueueMarketImport(ctx, pack.Id); err != nil {
		t.Fatal(err)
	}
	var task taskQueue.Entity
	if err := db.Connect().Where("type = ? AND status = ?", TaskTypeMarketImport, taskQueue.StatusPending).Order("id DESC").First(&task).Error; err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- RunMarketImportTask(ctx, &task) }()
	deadline := time.Now().Add(5 * time.Second)
	for {
		current, err := sticker.MarketPackByID(pack.Id)
		if err != nil {
			t.Fatal(err)
		}
		if current.ItemCount == 2 {
			if current.Status != sticker.MarketStatusImporting || current.FailedCount != 0 || current.ImportTotal != 3 {
				t.Fatalf("mid-run progress = %#v, want 2 of 3 with the earlier failures cleared", current)
			}
			break
		}
		if time.Now().After(deadline) {
			close(gate)
			t.Fatalf("progress never reached 2 of 3: %#v", current)
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(gate)
	if err := <-done; err != nil {
		t.Fatalf("run import: %v", err)
	}
	pack, _ = sticker.MarketPackByID(pack.Id)
	if pack.Status != sticker.MarketStatusReady || pack.ItemCount != 3 || pack.FailedCount != 0 || pack.LastError != "" {
		t.Fatalf("retried import = %#v", pack)
	}
}

func TestMarketSubscriptionLimitAndClosure(t *testing.T) {
	setupMarketTest(t)
	ctx := context.Background()
	packs := make([]sticker.MarketPack, MaxPackSubscriptions+1)
	for index := range packs {
		packs[index] = sticker.MarketPack{Source: MarketSourceID, UpstreamID: fmt.Sprintf("limit-%d", index), Name: "Limit", Status: sticker.MarketStatusReady, Published: true, ItemCount: 1}
	}
	if err := db.Connect().Create(&packs).Error; err != nil {
		t.Fatal(err)
	}
	for _, pack := range packs[:MaxPackSubscriptions] {
		if _, err := SubscribeMarketPack(ctx, 31, pack.Id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SubscribeMarketPack(ctx, 31, packs[MaxPackSubscriptions].Id); !errors.Is(err, ErrMarketSubscriptionFull) {
		t.Fatalf("subscription over limit = %v", err)
	}
	if err := CloseLibrary(ctx, 31); err != nil {
		t.Fatal(err)
	}
	if rows, _ := sticker.Subscriptions(31); len(rows) != 0 {
		t.Fatal("account closure kept pack subscriptions")
	}
	if _, err := SubscribeMarketPack(ctx, 31, packs[0].Id); !errors.Is(err, sticker.ErrLibraryClosed) {
		t.Fatalf("subscribe after closure = %v", err)
	}
}

func TestHiddenMarketSubscriptionsDoNotConsumeLimit(t *testing.T) {
	setupMarketTest(t)
	ctx := context.Background()
	packs := make([]sticker.MarketPack, MaxPackSubscriptions+1)
	for index := range packs {
		packs[index] = sticker.MarketPack{Source: MarketSourceID, UpstreamID: fmt.Sprintf("hidden-limit-%d", index), Name: "Limit", Status: sticker.MarketStatusReady, Published: true, ItemCount: 1}
	}
	if err := db.Connect().Create(&packs).Error; err != nil {
		t.Fatal(err)
	}
	for _, pack := range packs[:MaxPackSubscriptions] {
		if _, err := SubscribeMarketPack(ctx, 32, pack.Id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := SaveMarketPack(ctx, SaveMarketPackInput{ID: packs[0].Id, Published: false}); err != nil {
		t.Fatal(err)
	}
	if _, err := SubscribeMarketPack(ctx, 32, packs[MaxPackSubscriptions].Id); err != nil {
		t.Fatalf("subscription after hiding a subscribed pack = %v", err)
	}
}

func TestStoreMarketAssetConcurrentWritersShareOneObject(t *testing.T) {
	setupMarketTest(t)
	data := marketPNG(t, color.RGBA{G: 255, A: 255})
	var wg sync.WaitGroup
	names := make([]string, 8)
	errs := make([]error, 8)
	for index := range names {
		wg.Go(func() {
			names[index], errs[index] = storeMarketAsset(context.Background(), data, "image/png")
		})
	}
	wg.Wait()
	for index := range names {
		if errs[index] != nil || names[index] != names[0] {
			t.Fatalf("writer %d = %q, %v; want %q", index, names[index], errs[index], names[0])
		}
	}
	if got := marketFileCount(t); got != 1 {
		t.Fatalf("concurrent writers stored %d objects", got)
	}
}

func TestMarketStickerRenameKeepsGeneratedTokenResolvable(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	fake.catalog(marketGroup("group-1", "Pack One", 2))
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	fake.json[testMarketBase+"group-1.json"] = `{"id":"group-1","emojis":[{"id":"e1","url":"https://cdn3.ldstatic.com/a.png"},{"id":"e2","url":"https://cdn3.ldstatic.com/b.png"}]}`
	fake.images["https://cdn3.ldstatic.com/a.png"] = marketPNG(t, color.RGBA{R: 255, A: 255})
	fake.images["https://cdn3.ldstatic.com/b.png"] = marketPNG(t, color.RGBA{G: 255, A: 255})
	pack := runQueuedMarketImport(t, packByUpstream(t, "group-1").Id)
	detail, err := AdminMarketPackItems(pack.Id)
	if err != nil || len(detail.Items) != 2 {
		t.Fatalf("pack items = %#v, %v", detail, err)
	}
	first, second := detail.Items[0], detail.Items[1]

	before, saved, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: first.StickerID, Name: " 开心猫 "})
	if err != nil || before.Name != first.Name || saved.Name != "开心猫" || saved.DisplayName != "" || !saved.IsEnabled {
		t.Fatalf("rename = %#v -> %#v, %v", before, saved, err)
	}
	resolved, _ := Resolve([]string{first.Name, "开心猫"})
	if len(resolved) != 2 || resolved[0].ID != first.StickerID || resolved[1].ID != first.StickerID {
		t.Fatalf("resolved after rename = %#v", resolved)
	}
	if urls, _ := ResolveURLs([]string{first.Name}); urls[first.Name] == "" {
		t.Fatal("server rendering lost the generated token after a rename")
	}
	detail, _ = AdminMarketPackItems(pack.Id)
	if detail.Items[0].Name != "开心猫" || len(detail.Items[0].Aliases) != 1 || detail.Items[0].Aliases[0] != first.Name {
		t.Fatalf("renamed item = %#v", detail.Items[0])
	}

	// Names stay unique across current and previous names of every sticker.
	if _, _, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: second.StickerID, Name: "开心猫", IsEnabled: marketBool(true)}); !errors.Is(err, ErrNameExists) {
		t.Fatalf("rename to a taken name = %v", err)
	}
	if _, _, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: second.StickerID, Name: first.Name, IsEnabled: marketBool(true)}); !errors.Is(err, ErrNameReserved) {
		t.Fatalf("rename to another sticker's previous name = %v", err)
	}
	if _, _, err := SaveMarketSticker(ctx, SaveMarketStickerInput{StickerID: second.StickerID, Name: "bad name!", IsEnabled: marketBool(true)}); !errors.Is(err, ErrNameInvalid) {
		t.Fatalf("rename to an invalid name = %v", err)
	}

	// Re-importing never recreates the generated token of a renamed sticker.
	db.Connect().Where("pack_id = ?", pack.Id).Delete(&sticker.MarketItem{})
	pack = runQueuedMarketImport(t, pack.Id)
	var count int64
	db.Connect().Model(&sticker.Entity{}).Where("pack = ?", sticker.PackMarket).Count(&count)
	if pack.ItemCount != 2 || count != 2 {
		t.Fatalf("re-import after rename: items %d, stickers %d", pack.ItemCount, count)
	}
}

func TestMarketCoverPreviewStoresOneThumbnail(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	fake.catalog(
		`{"id":"group-icon","name":"Icon","topic":"anime","emojiCount":1,"icon":"https://cdn3.ldstatic.com/icon.png"}`,
		marketGroup("group-plain", "Plain", 1),
		marketGroup("group-broken", "Broken", 1),
	)
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	big := image.NewNRGBA(image.Rect(0, 0, 640, 320))
	for i := range big.Pix {
		big.Pix[i] = 200
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, big); err != nil {
		t.Fatal(err)
	}
	fake.images["https://cdn3.ldstatic.com/icon.png"] = encoded.Bytes()
	fake.json[testMarketBase+"group-plain.json"] = `{"id":"group-plain","emojis":[{"id":"p1","url":"https://cdn3.ldstatic.com/first.png"}]}`
	fake.images["https://cdn3.ldstatic.com/first.png"] = marketPNG(t, color.RGBA{B: 255, A: 255})

	withIcon := packByUpstream(t, "group-icon")
	if got := marketPackIconURL(withIcon); got != MarketCoverPreviewPath+"?id="+strconv.FormatUint(withIcon.Id, 10) {
		t.Fatalf("icon url before preview = %q", got)
	}
	name, err := MarketCoverPreview(ctx, withIcon.Id)
	if err != nil || !strings.HasPrefix(name, MarketAssetPrefix) || !strings.HasSuffix(name, ".png") {
		t.Fatalf("cover preview = %q, %v", name, err)
	}
	if again, err := MarketCoverPreview(ctx, withIcon.Id); err != nil || again != name || fake.callCount("https://cdn3.ldstatic.com/icon.png") != 1 {
		t.Fatalf("second preview = %q, %v after %d fetches", again, err, fake.callCount("https://cdn3.ldstatic.com/icon.png"))
	}
	stored, err := filedata.GetFileByName(name)
	if err != nil {
		t.Fatal(err)
	}
	thumb, err := png.DecodeConfig(bytes.NewReader(stored.Data))
	if err != nil || thumb.Width != marketCoverSide || thumb.Height != marketCoverSide/2 {
		t.Fatalf("thumbnail = %#v, %v", thumb, err)
	}
	withIcon = packByUpstream(t, "group-icon")
	if marketPackIconURL(withIcon) != storageservice.PublicAccessPath(name) {
		t.Fatalf("icon url after preview = %q", marketPackIconURL(withIcon))
	}

	// Without an upstream icon the first sticker becomes the cover.
	if name, err := MarketCoverPreview(ctx, packByUpstream(t, "group-plain").Id); err != nil || name == "" {
		t.Fatalf("first-sticker cover = %q, %v", name, err)
	}

	// A failure is remembered instead of refetched on every catalog load.
	broken := packByUpstream(t, "group-broken")
	if _, err := MarketCoverPreview(ctx, broken.Id); err == nil {
		t.Fatal("a pack without any image produced a cover")
	}
	calls := fake.callCount(testMarketBase + "group-broken.json")
	if _, err := MarketCoverPreview(ctx, broken.Id); err == nil || fake.callCount(testMarketBase+"group-broken.json") != calls {
		t.Fatal("a failed cover was fetched again immediately")
	}

	// A changed upstream icon replaces the stored thumbnail on the next sync.
	fake.catalog(
		`{"id":"group-icon","name":"Icon","topic":"anime","emojiCount":1,"icon":"https://cdn3.ldstatic.com/icon-2.png"}`,
		marketGroup("group-plain", "Plain", 1),
		marketGroup("group-broken", "Broken", 1),
	)
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	if pack := packByUpstream(t, "group-icon"); pack.IconFile != "" {
		t.Fatalf("stale thumbnail kept after icon change: %q", pack.IconFile)
	}
	if pack := packByUpstream(t, "group-plain"); pack.IconFile == "" {
		t.Fatal("an unchanged pack lost its thumbnail on sync")
	}
}

func TestMarketCoverPreviewCachesDatabaseUpdateFailure(t *testing.T) {
	fake := setupMarketTest(t)
	ctx := context.Background()
	fake.catalog(`{"id":"group-db-failure","name":"Database failure","topic":"anime","emojiCount":1,"icon":"https://cdn3.ldstatic.com/db-failure.png"}`)
	if _, err := SyncMarketCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	fake.images["https://cdn3.ldstatic.com/db-failure.png"] = marketPNG(t, color.RGBA{B: 255, A: 255})
	pack := packByUpstream(t, "group-db-failure")
	marketCoverMisses.Delete(pack.Id)
	if err := db.Connect().Exec(`CREATE TRIGGER fail_market_cover_update BEFORE UPDATE OF icon_file ON sticker_market_packs BEGIN SELECT RAISE(FAIL, 'blocked'); END`).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		marketCoverMisses.Delete(pack.Id)
		db.Connect().Exec(`DROP TRIGGER IF EXISTS fail_market_cover_update`)
	})
	if _, err := MarketCoverPreview(ctx, pack.Id); err == nil {
		t.Fatal("database update failure was ignored")
	}
	calls := fake.callCount("https://cdn3.ldstatic.com/db-failure.png")
	if _, err := MarketCoverPreview(ctx, pack.Id); err == nil || fake.callCount("https://cdn3.ldstatic.com/db-failure.png") != calls {
		t.Fatal("failed database update was not remembered")
	}
}
