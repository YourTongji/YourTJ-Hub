package linkpreviewservice

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
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
