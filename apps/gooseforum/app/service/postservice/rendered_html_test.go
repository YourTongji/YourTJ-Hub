package postservice

import (
	"fmt"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
)

func TestRenderStickerMarkerDoesNotTrustImageAlt(t *testing.T) {
	got := renderPostHTML("[:sticker:smile:]\n\n![sticker:smile](/same.png)\n\n![sticker:unknown](/other.png)\n\n<img src=\"/forged.png\" data-gf-sticker=\"smile\">", map[string]string{"smile": "/same.png"})
	if strings.Count(got, `data-gf-sticker="smile"`) != 1 || strings.Contains(got, "/forged.png") {
		t.Fatalf("only the resolved token may have a renderer marker: %s", got)
	}
	if strings.Count(got, `src="/same.png"`) != 2 || !strings.Contains(got, `alt="sticker:unknown"`) {
		t.Fatalf("ordinary image alt or URL changed: %s", got)
	}
}

func TestEnsureRenderedHTMLRebuildsAndSavesStalePost(t *testing.T) {
	post := &posts.Entity{
		Id:              1,
		Content:         "[external](https://example.com)",
		RenderedHTML:    "<p>stale</p>",
		RenderedVersion: markdown2html.GetPostVersion() - 1,
	}
	saved := false

	got, err := ensureRenderedHTML(post, func(entity *posts.Entity) error {
		saved = true
		return nil
	})

	if err != nil || !saved {
		t.Fatalf("ensureRenderedHTML() err = %v, saved = %v", err, saved)
	}
	if strings.Contains(got, "stale") || !strings.Contains(got, `rel="nofollow ugc noopener noreferrer"`) {
		t.Fatalf("ensureRenderedHTML() = %q, want current post rendering", got)
	}
}

func TestEnsureRenderedHTMLReusesCurrentPostWithoutSaving(t *testing.T) {
	post := &posts.Entity{Id: 1, RenderedHTML: "<p>current</p>", RenderedVersion: markdown2html.GetPostVersion()}

	got, err := ensureRenderedHTML(post, func(entity *posts.Entity) error {
		t.Fatal("current post should not be saved")
		return nil
	})

	if err != nil || got != post.RenderedHTML {
		t.Fatalf("ensureRenderedHTML() = %q, %v", got, err)
	}
}

func TestRenderPostHTMLWithMention(t *testing.T) {
	// mention 用户在隔离测试库中创建，验证编排：提取 → 批量解析 → 渲染链接。
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
	user := users.MakeUser("render-target", "secret123", "render-target@example.com")
	if err := users.Create(user); err != nil {
		t.Fatalf("create mention user: %v", err)
	}

	html := RenderPostHTML("hello @render-target and @nobody")
	if !strings.Contains(html, `<a href="/u/`+fmt.Sprint(user.Id)+`">@render-target</a>`) {
		t.Fatalf("RenderPostHTML() = %q, want mention link to /u/%d", html, user.Id)
	}
	if strings.Contains(html, `<a href="/u/`) && strings.Contains(html, "@nobody") && strings.Contains(html, ">@nobody</a>") {
		t.Fatalf("RenderPostHTML() = %q, unknown mention must stay plain", html)
	}

	post := &posts.Entity{Id: 1, Content: "@render-target", RenderedHTML: "<p>@render-target</p>", RenderedVersion: 5}
	saved := false
	rebuilt, err := ensureRenderedHTML(post, func(*posts.Entity) error { saved = true; return nil })
	if err != nil || !saved || !strings.Contains(rebuilt, `href="/u/`) {
		t.Fatalf("pre-mention cache not rebuilt: %s saved=%v err=%v", rebuilt, saved, err)
	}

	plain := RenderPostHTML("no mentions here")
	if strings.Contains(plain, `<a href="/u/`) {
		t.Fatalf("RenderPostHTML() = %q, no mentions must not link", plain)
	}
}

func TestStickerReviewCurrentDefinitionsOverridePersistedHTML(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&sticker.Entity{}); err != nil {
		t.Fatal(err)
	}
	row := sticker.Entity{Name: "render_fresh", FileName: "stickers/fresh.png", IsEnabled: true}
	if err := sticker.Save(&row); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Delete(&row) })
	post := &posts.Entity{Id: 9, Content: "[:sticker:render_fresh:]", RenderedHTML: RenderPostHTML("[:sticker:render_fresh:]"), RenderedVersion: markdown2html.GetPostVersion()}
	if !strings.Contains(post.RenderedHTML, "fresh.png") {
		t.Fatal(post.RenderedHTML)
	}
	row.IsEnabled = false
	if err := sticker.Save(&row); err != nil {
		t.Fatal(err)
	}
	got, err := ensureRenderedHTML(post, func(*posts.Entity) error { t.Fatal("dynamic sticker HTML must not be persisted on read"); return nil })
	if post.RenderedHTML != got {
		t.Fatalf("payload readers still see stale entity HTML: %s", post.RenderedHTML)
	}
	if err != nil || strings.Contains(got, "fresh.png") || !strings.Contains(got, "[:sticker:render_fresh:]") {
		t.Fatalf("disabled sticker still served: %s / %v", got, err)
	}
}

// countStickerQueries 注册查询回调统计 stickers 表 SELECT 次数并返回读数函数，
// cleanup 时移除回调。供批量渲染回归测试断言整页只解析一次（issue #706）。
func countStickerQueries(t *testing.T, conn *gorm.DB) func() int {
	t.Helper()
	var count int
	const callbackName = "postservice_test_count_sticker_queries"
	if err := conn.Callback().Query().After("gorm:query").Register(callbackName, func(op *gorm.DB) {
		sql := op.Statement.SQL.String()
		if strings.Contains(sql, "FROM `stickers`") || strings.Contains(sql, `FROM "stickers"`) {
			count++
		}
	}); err != nil {
		t.Fatalf("register sticker query counter: %v", err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove(callbackName) })
	return func() int { return count }
}

// TestEnsureRenderedHTMLBatchResolvesStickersOncePerPayload 回归 issue #706：
// 整批实体无论含多少贴纸帖都只做一次贴纸解析；启用贴纸展开、停用贴纸保持
// token 原样；贴纸帖读时渲染不落库，普通帖照常重渲染并落库。
func TestEnsureRenderedHTMLBatchResolvesStickersOncePerPayload(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&posts.Entity{}, &sticker.Entity{}); err != nil {
		t.Fatalf("migrate batch render tables: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	conn.Where("1 = 1").Delete(&sticker.Entity{})
	enabled := sticker.Entity{Name: "batch_ok", FileName: "stickers/batch_ok.png", IsEnabled: true}
	disabled := sticker.Entity{Name: "batch_off", FileName: "stickers/batch_off.png", IsEnabled: false}
	for _, row := range []*sticker.Entity{&enabled, &disabled} {
		if err := sticker.Save(row); err != nil {
			t.Fatalf("seed sticker row: %v", err)
		}
	}
	// GORM 写库跳过布尔零值（列 default:true 兜底），停用行需显式落库。
	if err := conn.Model(&sticker.Entity{}).Where("name = ?", disabled.Name).Update("is_enabled", false).Error; err != nil {
		t.Fatalf("disable batch_off row: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("1 = 1").Delete(&sticker.Entity{})
		conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	})

	stickerPost := posts.Entity{Id: 22, TopicId: 3, PostNo: 1, Content: "hi [:sticker:batch_ok:] and [:sticker:batch_off:]"}
	if err := conn.Create(&stickerPost).Error; err != nil {
		t.Fatalf("seed sticker post: %v", err)
	}
	seeded := posts.Entity{Id: 21, TopicId: 2, PostNo: 1, Content: "plain *text*", RenderedHTML: "<p>stale</p>", RenderedVersion: 1}
	if err := conn.Create(&seeded).Error; err != nil {
		t.Fatalf("seed plain post: %v", err)
	}
	secondStickerPost := &posts.Entity{Id: 23, Content: "[:sticker:batch_ok:] again"}
	parentPost := &posts.Entity{Id: 24, Content: "[:sticker:batch_ok:]"}
	noIdPost := &posts.Entity{Content: "[:sticker:batch_ok:]"}

	stickerSelects := countStickerQueries(t, conn)
	EnsureRenderedHTMLBatch([]*posts.Entity{nil, &stickerPost, secondStickerPost, parentPost, noIdPost, &seeded})

	if got := stickerSelects(); got != 1 {
		t.Fatalf("sticker resolves for one 6-entity payload = %d, want 1", got)
	}
	for _, post := range []*posts.Entity{&stickerPost, secondStickerPost, parentPost} {
		if !strings.Contains(post.RenderedHTML, "batch_ok.png") {
			t.Fatalf("post %d rendered html = %q, want expanded batch_ok url", post.Id, post.RenderedHTML)
		}
		if post.RenderedVersion != markdown2html.GetPostVersion() {
			t.Fatalf("post %d rendered version = %d, want current", post.Id, post.RenderedVersion)
		}
	}
	if !strings.Contains(stickerPost.RenderedHTML, "[:sticker:batch_off:]") {
		t.Fatalf("disabled sticker expanded: %q", stickerPost.RenderedHTML)
	}
	if noIdPost.RenderedHTML != "" || noIdPost.RenderedVersion != 0 {
		t.Fatalf("id-less entity touched: %+v", noIdPost)
	}
	if !strings.Contains(seeded.RenderedHTML, "<em>text</em>") {
		t.Fatalf("plain post not re-rendered in place: %q", seeded.RenderedHTML)
	}

	var storedSticker posts.Entity
	if err := conn.First(&storedSticker, stickerPost.Id).Error; err != nil {
		t.Fatalf("reload sticker post: %v", err)
	}
	if storedSticker.RenderedHTML != "" {
		t.Fatalf("sticker html persisted, want read-time only: %q", storedSticker.RenderedHTML)
	}
	var storedPlain posts.Entity
	if err := conn.First(&storedPlain, seeded.Id).Error; err != nil {
		t.Fatalf("reload plain post: %v", err)
	}
	if storedPlain.RenderedHTML != seeded.RenderedHTML || storedPlain.RenderedVersion != markdown2html.GetPostVersion() {
		t.Fatalf("plain post not persisted: %q v%d", storedPlain.RenderedHTML, storedPlain.RenderedVersion)
	}
}

// TestEnsureRenderedHTMLBatchWithoutTokensFallsBackPerEntity 回归：无贴纸 token
// 时批量入口退化为逐实体 EnsureRenderedHTML——过期缓存重渲染并落库，版本缓存
// 命中的实体保持原样不再写库。
func TestEnsureRenderedHTMLBatchWithoutTokensFallsBackPerEntity(t *testing.T) {
	conn := db.Connect()
	if err := conn.AutoMigrate(&posts.Entity{}); err != nil {
		t.Fatalf("migrate posts table: %v", err)
	}
	conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{})
	t.Cleanup(func() { conn.Unscoped().Where("1 = 1").Delete(&posts.Entity{}) })

	stale := posts.Entity{Id: 31, TopicId: 5, PostNo: 1, Content: "plain *text*", RenderedHTML: "<p>stale</p>", RenderedVersion: 1}
	current := posts.Entity{Id: 32, TopicId: 6, PostNo: 1, Content: "plain *text*", RenderedHTML: "<p>current-marker</p>", RenderedVersion: markdown2html.GetPostVersion()}
	for _, post := range []*posts.Entity{&stale, &current} {
		if err := conn.Create(post).Error; err != nil {
			t.Fatalf("seed post %d: %v", post.Id, err)
		}
	}

	EnsureRenderedHTMLBatch([]*posts.Entity{nil, &stale, &current})

	if !strings.Contains(stale.RenderedHTML, "<em>text</em>") || stale.RenderedVersion != markdown2html.GetPostVersion() {
		t.Fatalf("stale post not refreshed in place: %q v%d", stale.RenderedHTML, stale.RenderedVersion)
	}
	if current.RenderedHTML != "<p>current-marker</p>" {
		t.Fatalf("cache-hit post re-rendered: %q", current.RenderedHTML)
	}

	var storedStale posts.Entity
	if err := conn.First(&storedStale, stale.Id).Error; err != nil {
		t.Fatalf("reload stale post: %v", err)
	}
	if storedStale.RenderedHTML != stale.RenderedHTML {
		t.Fatalf("stale post render not persisted: %q", storedStale.RenderedHTML)
	}
	var storedCurrent posts.Entity
	if err := conn.First(&storedCurrent, current.Id).Error; err != nil {
		t.Fatalf("reload current post: %v", err)
	}
	if storedCurrent.RenderedHTML != "<p>current-marker</p>" {
		t.Fatalf("cache-hit post was rewritten: %q", storedCurrent.RenderedHTML)
	}
}
