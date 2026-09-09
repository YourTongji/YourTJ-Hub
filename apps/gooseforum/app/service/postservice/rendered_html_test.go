package postservice

import (
	"fmt"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

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
