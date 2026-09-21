package linkpreviewservice

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/posts"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
)

type fakeFetcher struct {
	result safefetch.Result
	err    error
	calls  atomic.Int32
	wait   <-chan struct{}
}

func (f *fakeFetcher) Fetch(ctx context.Context, _ string) (safefetch.Result, error) {
	f.calls.Add(1)
	if f.wait != nil {
		select {
		case <-f.wait:
		case <-ctx.Done():
			return safefetch.Result{}, ctx.Err()
		}
	}
	return f.result, f.err
}

func TestResolveExternalMetadata(t *testing.T) {
	finalURL, _ := url.Parse("https://example.com/final")
	fetcher := &fakeFetcher{result: safefetch.Result{
		FinalURL:    finalURL,
		StatusCode:  200,
		ContentType: "text/html",
		Body: []byte(`<!doctype html><html><head>
<meta property="og:title" content="  Example &amp; title  ">
<meta property="og:description" content="A <b>short</b> description">
<meta property="og:site_name" content="Example site">
<meta property="og:image" content="/cover.jpg">
<link rel="icon" href="javascript:alert(1)">
<title>Ignored title</title>
</head></html>`),
	}}
	resolver := New(fetcher, func() []string { return []string{"https://f.yourtj.de"} })

	preview := resolver.Resolve(t.Context(), 0, "https://example.com/article?x=1#part")
	if preview.Status != StatusReady || preview.Kind != KindExternal {
		t.Fatalf("preview status/kind = %q/%q", preview.Status, preview.Kind)
	}
	if preview.Title != "Example & title" || preview.Description != "A short description" {
		t.Fatalf("preview text = %q / %q", preview.Title, preview.Description)
	}
	if preview.ImageURL != "https://example.com/cover.jpg" {
		t.Fatalf("image URL = %q", preview.ImageURL)
	}
	if preview.FaviconURL != "" {
		t.Fatalf("unsafe favicon URL = %q, want empty", preview.FaviconURL)
	}
	if preview.RequestedURL != "https://example.com/article?x=1#part" {
		t.Fatalf("requested URL = %q", preview.RequestedURL)
	}
	if preview.URL != "https://example.com/article?x=1#part" {
		t.Fatalf("navigation URL = %q", preview.URL)
	}
	if preview.DisplayHost != "example.com" || preview.RegistrableDomain != "example.com" {
		t.Fatalf("host fields = %q / %q", preview.DisplayHost, preview.RegistrableDomain)
	}
}

func TestResolveDeduplicatesConcurrentFetchAndCaches(t *testing.T) {
	release := make(chan struct{})
	finalURL, _ := url.Parse("https://example.com/")
	fetcher := &fakeFetcher{
		result: safefetch.Result{
			FinalURL:    finalURL,
			StatusCode:  200,
			ContentType: "text/html",
			Body:        []byte(`<title>Example</title>`),
		},
		wait: release,
	}
	resolver := New(fetcher, nil)

	var wg sync.WaitGroup
	results := make(chan Preview, 2)
	for range 2 {
		wg.Go(func() {
			results <- resolver.Resolve(t.Context(), 0, "https://example.com/")
		})
	}
	for fetcher.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	close(release)
	wg.Wait()
	close(results)
	for result := range results {
		if result.Status != StatusReady {
			t.Fatalf("concurrent status = %q", result.Status)
		}
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("fetch calls = %d, want 1", fetcher.calls.Load())
	}
	if cached := resolver.Resolve(t.Context(), 0, "https://example.com/"); cached.Status != StatusReady {
		t.Fatalf("cached status = %q", cached.Status)
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("fetch calls after cache = %d, want 1", fetcher.calls.Load())
	}
}

func TestResolveCallerCancellationDoesNotPoisonSharedFetch(t *testing.T) {
	release := make(chan struct{})
	finalURL, _ := url.Parse("https://example.com/")
	fetcher := &fakeFetcher{
		result: safefetch.Result{
			FinalURL:    finalURL,
			StatusCode:  200,
			ContentType: "text/html",
			Body:        []byte(`<title>Example</title>`),
		},
		wait: release,
	}
	resolver := New(fetcher, nil)
	firstContext, cancelFirst := context.WithCancel(t.Context())
	first := make(chan Preview, 1)
	second := make(chan Preview, 1)
	go func() { first <- resolver.Resolve(firstContext, 0, "https://example.com/") }()
	for fetcher.calls.Load() == 0 {
		time.Sleep(time.Millisecond)
	}
	go func() { second <- resolver.Resolve(t.Context(), 0, "https://example.com/") }()
	cancelFirst()
	if got := <-first; got.Status != StatusTimeout {
		t.Fatalf("cancelled caller status = %q, want %q", got.Status, StatusTimeout)
	}
	close(release)
	if got := <-second; got.Status != StatusReady {
		t.Fatalf("remaining caller status = %q, want %q", got.Status, StatusReady)
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("fetch calls = %d, want 1", fetcher.calls.Load())
	}
}

func TestParseMetadataUsesSafeFallbacks(t *testing.T) {
	base, _ := url.Parse("https://example.com/articles/one")
	metadata, err := parseMetadata([]byte(`<!doctype html><html><head>
<title> Fallback &amp; title </title>
<meta name="description" content=" Plain &amp; useful ">
<link rel="shortcut icon" href="/favicon.ico">
</head></html>`), "text/html; charset=utf-8", base)
	if err != nil {
		t.Fatalf("parseMetadata() error = %v", err)
	}
	if metadata.title != "Fallback & title" || metadata.description != "Plain & useful" {
		t.Fatalf("metadata text = %q / %q", metadata.title, metadata.description)
	}
	if metadata.faviconURL != "https://example.com/favicon.ico" {
		t.Fatalf("favicon URL = %q", metadata.faviconURL)
	}
}

func TestSafeAssetURLKeepsInternalRelativeAssets(t *testing.T) {
	base, _ := url.Parse("/p/post/42")
	if got := safeAssetURL("/file/img/cover.webp", base); got != "/file/img/cover.webp" {
		t.Fatalf("relative asset URL = %q", got)
	}
	if got := safeAssetURL("javascript:alert(1)", base); got != "" {
		t.Fatalf("unsafe asset URL = %q, want empty", got)
	}
}

func TestResolveMapsStableFailureStatus(t *testing.T) {
	fetcher := &fakeFetcher{err: &safefetch.FetchError{Class: safefetch.ErrorTimeout, Err: errors.New("secret dial detail")}}
	resolver := New(fetcher, nil)
	preview := resolver.Resolve(t.Context(), 0, "https://example.com/")
	if preview.Status != StatusTimeout {
		t.Fatalf("status = %q, want %q", preview.Status, StatusTimeout)
	}
	if preview.Title != "" || preview.Description != "" {
		t.Fatalf("failure leaked metadata: %#v", preview)
	}
}

func TestResolveRejectsInvalidAndUnsupportedWithoutFetching(t *testing.T) {
	fetcher := &fakeFetcher{}
	resolver := New(fetcher, nil)
	if got := resolver.Resolve(t.Context(), 0, "javascript:alert(1)"); got.Status != StatusUnsupported {
		t.Fatalf("unsupported status = %q", got.Status)
	}
	if got := resolver.Resolve(t.Context(), 0, "//example.com"); got.Status != StatusInvalid {
		t.Fatalf("invalid status = %q", got.Status)
	}
	if fetcher.calls.Load() != 0 {
		t.Fatalf("fetch calls = %d, want 0", fetcher.calls.Load())
	}
}

// campusResolver 把 tongji.edu.cn 配置为校园网，用于验证校园网链接既不抓取
// 也能出卡片。名字三级回退：精确主机 → 域名默认名 → 空（客户端出兜底文案）。
func campusResolver(fetcher Fetcher) *Resolver {
	resolver := New(fetcher, nil)
	resolver.campus = func() CampusPolicy {
		return CampusPolicy{
			Domains:     []string{"tongji.edu.cn"},
			DomainNames: map[string]string{"tongji.edu.cn": "同济大学"},
			HostNames:   map[string]string{"1.tongji.edu.cn": "同济大学教学管理系统"},
		}
	}
	return resolver
}

func TestResolveRendersCampusCardWithoutFetching(t *testing.T) {
	fetcher := &fakeFetcher{}
	preview := campusResolver(fetcher).Resolve(t.Context(), 0, "https://1.tongji.edu.cn/personal#/index")

	if preview.Status != StatusReady || preview.Kind != KindExternal {
		t.Fatalf("status/kind = %q/%q", preview.Status, preview.Kind)
	}
	if !preview.Campus {
		t.Fatalf("campus marker is not set: %#v", preview)
	}
	if preview.Title != "同济大学教学管理系统" {
		t.Fatalf("title = %q", preview.Title)
	}
	// 描述不再由服务端硬编码，留给客户端按语言渲染。
	if preview.Description != "" {
		t.Fatalf("description = %q, want empty", preview.Description)
	}
	if preview.DisplayHost != "1.tongji.edu.cn" || preview.SiteName != "1.tongji.edu.cn" {
		t.Fatalf("host fields = %q / %q", preview.DisplayHost, preview.SiteName)
	}
	if preview.RegistrableDomain != "tongji.edu.cn" {
		t.Fatalf("registrable domain = %q", preview.RegistrableDomain)
	}
	if preview.URL != "https://1.tongji.edu.cn/personal#/index" {
		t.Fatalf("navigation URL = %q", preview.URL)
	}
	// 卡片必须是纯本地派生，不得夹带任何抓取产物。
	if preview.ImageURL != "" || preview.FaviconURL != "" || !preview.FetchedAt.IsZero() {
		t.Fatalf("campus card carries fetched data: %#v", preview)
	}
	if fetcher.calls.Load() != 0 {
		t.Fatalf("campus link fetched %d times, want 0", fetcher.calls.Load())
	}
}

func TestResolveCampusCardNameFallbackChain(t *testing.T) {
	cases := []struct {
		name  string
		raw   string
		title string
	}{
		{"host name wins", "https://1.tongji.edu.cn/x", "同济大学教学管理系统"},
		{"domain name covers unlisted subdomain", "https://yunpan.tongji.edu.cn/anyshare/link/AA7B?title=x", "同济大学"},
		{"bare domain also matches", "https://tongji.edu.cn/", "同济大学"},
	}
	for _, testCase := range cases {
		fetcher := &fakeFetcher{}
		preview := campusResolver(fetcher).Resolve(t.Context(), 0, testCase.raw)
		if preview.Status != StatusReady || !preview.Campus {
			t.Fatalf("%s: status/campus = %q/%v", testCase.name, preview.Status, preview.Campus)
		}
		if preview.Title != testCase.title {
			t.Fatalf("%s: title = %q, want %q", testCase.name, preview.Title, testCase.title)
		}
		if fetcher.calls.Load() != 0 {
			t.Fatalf("%s: fetch calls = %d, want 0", testCase.name, fetcher.calls.Load())
		}
	}

	// 两级都没配名字时 title 留空，由客户端出兜底文案；服务端不得再返回任何
	// 硬编码中文。
	nameless := New(&fakeFetcher{}, nil)
	nameless.campus = func() CampusPolicy { return CampusPolicy{Domains: []string{"tongji.edu.cn"}} }
	preview := nameless.Resolve(t.Context(), 0, "https://agent.tongji.edu.cn/chat")
	if preview.Status != StatusReady || !preview.Campus {
		t.Fatalf("nameless campus card: status/campus = %q/%v", preview.Status, preview.Campus)
	}
	if preview.Title != "" || preview.Description != "" {
		t.Fatalf("server hardcoded fallback copy: title=%q description=%q", preview.Title, preview.Description)
	}
}

func TestResolveKeepsLookalikeHostsOffCampus(t *testing.T) {
	for _, raw := range []string{
		"https://tongji.edu.cn.evil.com/portal",
		"https://not-tongji.edu.cn/portal",
		"https://xtongji.edu.cn/portal",
	} {
		// 走到外链分支即说明没被误判成校园网。让 fetcher 直接失败，断言点只有
		// 「不得是 ready 的校园卡片」与「确实尝试过抓取」。
		fetcher := &fakeFetcher{err: &safefetch.FetchError{Class: safefetch.ErrorBlocked, Err: errors.New("blocked")}}
		preview := campusResolver(fetcher).Resolve(t.Context(), 0, raw)
		if preview.Status == StatusReady {
			t.Fatalf("%s rendered as ready: %#v", raw, preview)
		}
		if preview.Title != "" {
			t.Fatalf("%s leaked campus title %q", raw, preview.Title)
		}
		if fetcher.calls.Load() != 1 {
			t.Fatalf("%s fetch calls = %d, want 1", raw, fetcher.calls.Load())
		}
	}
}

func TestCampusPolicyCardMatching(t *testing.T) {
	// 走 normalizeCampusNames 构造，顺带验证配置里的大小写与空格被归一化。
	policy := CampusPolicy{
		Domains:     []string{" tongji.edu.cn ", "TONGJI.EDU.CN."},
		DomainNames: normalizeCampusNames(map[string]string{" Tongji.Edu.CN ": " 同济大学 "}),
		HostNames: normalizeCampusNames(map[string]string{
			"1.tongji.edu.cn":     "同济大学教学管理系统",
			"blank.tongji.edu.cn": "   ",
		}),
	}
	cases := []struct {
		host  string
		title string
		ok    bool
	}{
		{"1.tongji.edu.cn", "同济大学教学管理系统", true},
		{"1.Tongji.Edu.CN.", "同济大学教学管理系统", true},
		{"a.b.tongji.edu.cn", "同济大学", true},
		{"tongji.edu.cn", "同济大学", true},
		// 空白主机名被丢弃后落到域名级默认名，而不是渲染成空卡。
		{"blank.tongji.edu.cn", "同济大学", true},
		{"tongji.edu.cn.evil.com", "", false},
		{"not-tongji.edu.cn", "", false},
		{"", "", false},
		{"   ", "", false},
	}
	for _, testCase := range cases {
		title, ok := policy.Card(testCase.host)
		if ok != testCase.ok || title != testCase.title {
			t.Fatalf("Card(%q) = %q/%v, want %q/%v", testCase.host, title, ok, testCase.title, testCase.ok)
		}
	}

	// 未配置域名默认名时命中但名为空：交给客户端出兜底文案，宁可空也不猜。
	bare := CampusPolicy{Domains: []string{"tongji.edu.cn"}}
	if title, ok := bare.Card("a.b.tongji.edu.cn"); !ok || title != "" {
		t.Fatalf("Card(a.b.tongji.edu.cn) = %q/%v, want \"\"/true", title, ok)
	}
	// 未配置任何域名时永远不命中，即默认关闭。
	if title, ok := (CampusPolicy{}).Card("1.tongji.edu.cn"); ok {
		t.Fatalf("empty policy matched with title %q", title)
	}
}

func TestResolvePrefersConfiguredOriginOverCampus(t *testing.T) {
	// 站点自身域名同时落在校园域下时必须先判为站内链接：站内要读一方数据，
	// 不能被降级成校园卡片。路径取一个未支持的形态，避免单元测试触碰数据库。
	fetcher := &fakeFetcher{}
	resolver := New(fetcher, func() []string { return []string{"https://forum.tongji.edu.cn"} })
	resolver.campus = func() CampusPolicy { return CampusPolicy{Domains: []string{"tongji.edu.cn"}} }

	preview := resolver.Resolve(t.Context(), 0, "https://forum.tongji.edu.cn/unknown")
	if preview.Kind != KindInternal {
		t.Fatalf("kind = %q, want %q", preview.Kind, KindInternal)
	}
	if preview.Title != "" {
		t.Fatalf("internal link rendered a campus title %q", preview.Title)
	}
	if fetcher.calls.Load() != 0 {
		t.Fatalf("fetch calls = %d, want 0", fetcher.calls.Load())
	}
}

// --- 站内链接（resolveInternal）：resolve API 的隐私声明是「私密、隐藏、无权限
// 资源不会通过 preview API 泄露标题、摘要或作者」，这里把声明钉到测试上。
// go test 下 dbconnect.Connect() 自动落到内存 sqlite（与 contentdeleteservice
// 等服务层测试同一基建），种子数据使用专属 ID 段避免用例间串扰。 ---

func setupInternalPreviewDB(t *testing.T) {
	t.Helper()
	if err := dbconnect.Connect().AutoMigrate(
		&users.EntityComplete{},
		&topics.Entity{},
		&posts.Entity{},
		&wikiPages.Entity{},
		&rolePermissionRs.Entity{},
	); err != nil {
		t.Fatalf("migrate link preview tables: %v", err)
	}
}

func seedPreviewTopic(t *testing.T, topic topics.Entity) {
	t.Helper()
	if err := dbconnect.Connect().Create(&topic).Error; err != nil {
		t.Fatalf("seed topic %d: %v", topic.Id, err)
	}
}

func TestResolveInternalTopicVisibility(t *testing.T) {
	setupInternalPreviewDB(t)
	resolver := New(&fakeFetcher{}, nil)
	published := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	deletedAt := gorm.DeletedAt{Time: published, Valid: true}

	// 公开话题：匿名可出卡，字段来自一方数据；相对首图按目标页原样保留。
	seedPreviewTopic(t, topics.Entity{
		Id: 7301, UserId: 7310, Title: "Public topic", Excerpt: "Public excerpt",
		FirstImageURL: "/file/img/cover.webp", Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive,
		RetentionStatus: topics.RetentionNormal, CreatedAt: published, UpdatedAt: published,
	})
	public := resolver.Resolve(t.Context(), 0, "/topics/7301")
	if public.Status != StatusReady || public.Title != "Public topic" || public.Description != "Public excerpt" {
		t.Fatalf("public topic preview = %#v", public)
	}
	if public.ImageURL != "/file/img/cover.webp" {
		t.Fatalf("public topic image = %q", public.ImageURL)
	}

	// 作者草稿：作者本人之外一律拒绝，包括匿名。
	seedPreviewTopic(t, topics.Entity{
		Id: 7302, UserId: 7311, Title: "Draft topic", Status: 0,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive,
		RetentionStatus: topics.RetentionNormal, CreatedAt: published, UpdatedAt: published,
	})
	if got := resolver.Resolve(t.Context(), 0, "/topics/7302"); got.Status != StatusPermissionDenied || got.Title != "" {
		t.Fatalf("draft leaked to anonymous: %#v", got)
	}
	if got := resolver.Resolve(t.Context(), 7311, "/topics/7302"); got.Status != StatusReady || got.Title != "Draft topic" {
		t.Fatalf("draft owner preview = %#v", got)
	}

	// 作者删除且无回复：内容只在作者侧可见。
	seedPreviewTopic(t, topics.Entity{
		Id: 7303, UserId: 7312, Title: "Deleted topic", Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus: topics.RetentionNormal, DeletedAt: deletedAt, CreatedAt: published, UpdatedAt: published,
	})
	if got := resolver.Resolve(t.Context(), 0, "/topics/7303"); got.Status != StatusPermissionDenied || got.Title != "" {
		t.Fatalf("user-deleted topic leaked to anonymous: %#v", got)
	}
	if got := resolver.Resolve(t.Context(), 7312, "/topics/7303"); got.Status != StatusReady {
		t.Fatalf("user-deleted topic owner preview = %#v", got)
	}

	// 作者删除但仍有正常回复：讨论上下文保持可读（与详情页行为一致）。
	seedPreviewTopic(t, topics.Entity{
		Id: 7304, UserId: 7313, Title: "Deleted with replies", Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus: topics.RetentionNormal, DeletedAt: deletedAt, CreatedAt: published, UpdatedAt: published,
	})
	if err := dbconnect.Connect().Create(&posts.Entity{Id: 73404, TopicId: 7304, PostNo: 2}).Error; err != nil {
		t.Fatalf("seed reply: %v", err)
	}
	if got := resolver.Resolve(t.Context(), 0, "/topics/7304"); got.Status != StatusReady {
		t.Fatalf("deleted topic with replies should stay readable: %#v", got)
	}

	// 待审/封禁话题：版主与 TopicsManager 通道之外一律拒绝，匿名不得拿到标题。
	seedPreviewTopic(t, topics.Entity{
		Id: 7305, UserId: 7310, Title: "Blocked topic", Status: 1,
		ProcessStatus: topics.ProcessStatusBlocked, VisibilityStatus: topics.VisibilityActive,
		RetentionStatus: topics.RetentionNormal, CreatedAt: published, UpdatedAt: published,
	})
	if got := resolver.Resolve(t.Context(), 0, "/topics/7305"); got.Status != StatusPermissionDenied || got.Title != "" {
		t.Fatalf("processed topic leaked to anonymous: %#v", got)
	}

	// 不存在的话题与不可见同形，不泄露存在性。
	if got := resolver.Resolve(t.Context(), 0, "/topics/73999"); got.Status != StatusPermissionDenied {
		t.Fatalf("missing topic status = %q", got.Status)
	}
}

func TestResolveInternalProcessedTopicVisibleToTopicsManager(t *testing.T) {
	setupInternalPreviewDB(t)
	published := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	seedPreviewTopic(t, topics.Entity{
		Id: 7603, UserId: 7310, Title: "Blocked topic", Status: 1,
		ProcessStatus: topics.ProcessStatusBlocked, VisibilityStatus: topics.VisibilityActive,
		RetentionStatus: topics.RetentionNormal, CreatedAt: published, UpdatedAt: published,
	})
	// TopicsManager 通道：用户角色 + 角色权限关系都落库后，管理者可以预览待审
	// 话题（与详情路径同权）。角色/用户 ID 取专属值，避开包级缓存的跨用例污染。
	if err := dbconnect.Connect().Create(&users.EntityComplete{Id: 7601, Username: "preview-topics-manager", RoleId: 7602}).Error; err != nil {
		t.Fatalf("seed manager: %v", err)
	}
	if err := dbconnect.Connect().Create(&rolePermissionRs.Entity{Id: 76021, RoleId: 7602, PermissionId: uint64(permission.TopicsManager), Effective: 1}).Error; err != nil {
		t.Fatalf("seed role permission: %v", err)
	}
	resolver := New(&fakeFetcher{}, nil)
	if got := resolver.Resolve(t.Context(), 7601, "/topics/7603"); got.Status != StatusReady || got.Title != "Blocked topic" {
		t.Fatalf("topics manager preview = %#v", got)
	}
}

func TestResolveInternalWikiAndUserProfile(t *testing.T) {
	setupInternalPreviewDB(t)
	resolver := New(&fakeFetcher{}, nil)
	published := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	deletedAt := gorm.DeletedAt{Time: published, Valid: true}

	// 公开 wiki 页出卡：标题/描述来自页面快照，Markdown 记号被剥成纯文本。
	seedPreviewTopic(t, topics.Entity{
		Id: 7401, UserId: 7310, Title: "Wiki topic", Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityActive,
		RetentionStatus: topics.RetentionNormal, CreatedAt: published, UpdatedAt: published,
	})
	if err := dbconnect.Connect().Create(&wikiPages.Entity{Id: 7411, TopicId: 7401, Path: "start", Title: "Wiki Start", Content: "# Heading\nBody text"}).Error; err != nil {
		t.Fatalf("seed wiki page: %v", err)
	}
	wiki := resolver.Resolve(t.Context(), 0, "/wiki/start")
	if wiki.Status != StatusReady || wiki.Title != "Wiki Start" || wiki.Description != "Heading Body text" {
		t.Fatalf("wiki preview = %#v", wiki)
	}

	// 其话题不可见的 wiki 页同样不可见，不泄露页面标题。
	seedPreviewTopic(t, topics.Entity{
		Id: 7402, UserId: 7310, Title: "Secret wiki topic", Status: 1,
		ProcessStatus: topics.ProcessStatusNormal, VisibilityStatus: topics.VisibilityUserDeleted,
		RetentionStatus: topics.RetentionNormal, DeletedAt: deletedAt, CreatedAt: published, UpdatedAt: published,
	})
	if err := dbconnect.Connect().Create(&wikiPages.Entity{Id: 7412, TopicId: 7402, Path: "secret", Title: "Secret page"}).Error; err != nil {
		t.Fatalf("seed secret wiki page: %v", err)
	}
	if got := resolver.Resolve(t.Context(), 0, "/wiki/secret"); got.Status != StatusPermissionDenied || got.Title != "" {
		t.Fatalf("hidden wiki page leaked: %#v", got)
	}

	// 用户主页出卡：昵称/简介来自公开资料；不存在的用户与不可见资源同形。
	if err := dbconnect.Connect().Create(&users.EntityComplete{Id: 7501, Username: "alice", Nickname: "Alice", Bio: "Hello bio"}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	profile := resolver.Resolve(t.Context(), 0, "/u/7501")
	if profile.Status != StatusReady || profile.Title != "Alice" || profile.Description != "Hello bio" {
		t.Fatalf("user profile preview = %#v", profile)
	}
	if got := resolver.Resolve(t.Context(), 0, "/u/7599"); got.Status != StatusPermissionDenied {
		t.Fatalf("missing user status = %q", got.Status)
	}
}
