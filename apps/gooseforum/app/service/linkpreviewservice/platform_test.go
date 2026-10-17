package linkpreviewservice

import (
	"net/url"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
)

// bilibiliPageFixture 是公开视频页在真实结构上的裁剪：og 元数据、UP 主
// （<meta name="author">）与带统计尾巴的 SEO 描述都在场。
const bilibiliPageFixture = `<!doctype html><html><head>
<meta name="keywords" content="新番,哔哩哔哩">
<meta name="description" content="微博：@瓶子君152点个关注再走啦~, 视频播放量 2019335、弹幕量 12866、点赞数 150086, 视频作者 瓶子君152, 作者简介 各位点个关注再走呗 ，相关视频：1月最疯新番！">
<meta name="author" content="瓶子君152">
<meta property="og:type" content="video.other">
<meta property="og:title" content="四月番毒奶安利！四月看这八部动画就够了！！_哔哩哔哩_bilibili">
<meta property="og:description" content="微博：@瓶子君152点个关注再走啦~, 视频播放量 2019335、弹幕量 12866、点赞数 150086, 视频作者 瓶子君152, 作者简介 各位点个关注再走呗 ，相关视频：1月最疯新番！">
<meta property="og:site_name" content="哔哩哔哩">
<meta property="og:url" content="https://www.bilibili.com/video/BV1B7411m7LV/">
<meta property="og:image" content="https://i2.hdslb.com/bfs/archive/0b83569178ab10ccfdbebd7a7a5f1545c4d76bef.jpg@1200w_630h">
<title>四月番毒奶安利！四月看这八部动画就够了！！_哔哩哔哩_bilibili</title>
</head><body></body></html>`

const appleSongLookupFixture = `{
 "resultCount":1,
 "results": [
{"wrapperType":"track", "kind":"song", "artistId":909253, "collectionId":1440857781, "trackId":1440857786, "artistName":"Jack Johnson", "collectionName":"In Between Dreams (Bonus Track Version)", "trackName":"Better Together", "artworkUrl100":"https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/44/06/fd/4406fdc0-aab5-e300-82ba-3e5fe81a68a7/00602537868858.rgb.jpg/100x100bb.jpg"}
 ]
}`

const appleAlbumLookupFixture = `{
 "resultCount":1,
 "results": [
{"wrapperType":"collection", "collectionType":"Album", "artistName":"Jack Johnson", "collectionName":"In Between Dreams (Bonus Track Version)", "collectionId":1440857781, "artworkUrl100":"https://is1-ssl.mzstatic.com/image/thumb/Music115/v4/44/06/fd/4406fdc0-aab5-e300-82ba-3e5fe81a68a7/00602537868858.rgb.jpg/100x100bb.jpg"}
 ]
}`

const appleEmptyLookupFixture = `{"resultCount":0,"results":[]}`

func platformFetcher(t *testing.T, jsonBody string) *fakeFetcher {
	t.Helper()
	finalURL, err := url.Parse("https://itunes.apple.com/lookup?id=1")
	if err != nil {
		t.Fatalf("parse final URL: %v", err)
	}
	return &fakeFetcher{
		result: safefetch.Result{
			FinalURL:    mustParseURL(t, "https://www.bilibili.com/video/BV1B7411m7LV/"),
			StatusCode:  200,
			ContentType: "text/html",
			Body:        []byte(bilibiliPageFixture),
		},
		jsonResult: safefetch.Result{
			FinalURL:   finalURL,
			StatusCode: 200,
			// 与线上一致：Apple 的 Lookup 端点是 text/javascript 媒体类型。
			ContentType: "text/javascript",
			Body:        []byte(jsonBody),
		},
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}

func TestRecognizePlatform(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		platform platformKind
		resource resourceKind
		id       string
		storefr  string
		ok       bool
	}{
		{name: "bilibili BV link", raw: "https://www.bilibili.com/video/BV1B7411m7LV", platform: platformBilibili, resource: resourceVideo, id: "BV1B7411m7LV", ok: true},
		{name: "bilibili BV link with part and time", raw: "https://www.bilibili.com/video/BV1B7411m7LV/?p=2&t=30", platform: platformBilibili, resource: resourceVideo, id: "BV1B7411m7LV", ok: true},
		{name: "bilibili av link", raw: "https://m.bilibili.com/video/av170001", platform: platformBilibili, resource: resourceVideo, id: "av170001", ok: true},
		{name: "bilibili bare domain", raw: "https://bilibili.com/video/BV1B7411m7LV", platform: platformBilibili, resource: resourceVideo, id: "BV1B7411m7LV", ok: true},
		{name: "apple song", raw: "https://music.apple.com/cn/song/better-together/1440857786", platform: platformAppleMusic, resource: resourceSong, id: "1440857786", storefr: "cn", ok: true},
		{name: "apple album", raw: "https://music.apple.com/us/album/in-between-dreams/1440857781", platform: platformAppleMusic, resource: resourceAlbum, id: "1440857781", storefr: "us", ok: true},
		{name: "apple album song query wins", raw: "https://music.apple.com/cn/album/better-together/1440857781?i=1440857786&uo=4", platform: platformAppleMusic, resource: resourceSong, id: "1440857786", storefr: "cn", ok: true},
		{name: "apple without storefront", raw: "https://music.apple.com/song/better-together/1440857786", platform: platformAppleMusic, resource: resourceSong, id: "1440857786", ok: true},
		{name: "lookalike bilibili host", raw: "https://www.bilibili.com.evil.example/video/BV1B7411m7LV"},
		{name: "bilibili bangumi is not a video id", raw: "https://www.bilibili.com/bangumi/play/ep123456"},
		{name: "bilibili short BV id", raw: "https://www.bilibili.com/video/BV1B7411m7L"},
		{name: "bilibili extra path segment", raw: "https://www.bilibili.com/video/BV1B7411m7LV/extra"},
		{name: "bilibili live room", raw: "https://live.bilibili.com/12345"},
		{name: "lookalike apple host", raw: "https://music.apple.com.evil.example/cn/song/x/1440857786"},
		{name: "apple playlist", raw: "https://music.apple.com/cn/playlist/road-trip/pl.abc123"},
		{name: "apple non numeric id", raw: "https://music.apple.com/cn/album/better-together/not-an-id"},
		{name: "apple artist page", raw: "https://music.apple.com/cn/artist/jack-johnson/909253"},
	}
	for _, testCase := range cases {
		target, ok := recognizePlatform(mustParseURL(t, testCase.raw))
		if ok != testCase.ok {
			t.Fatalf("%s: recognized = %v, want %v", testCase.name, ok, testCase.ok)
		}
		if !testCase.ok {
			continue
		}
		if target.platform != testCase.platform || target.resource != testCase.resource || target.id != testCase.id || target.storefront != testCase.storefr {
			t.Fatalf("%s: target = %#v", testCase.name, target)
		}
	}
}

func TestResolveBilibiliVideoCard(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	resolver := New(fetcher, nil)
	raw := "https://www.bilibili.com/video/BV1B7411m7LV/?spm_id_from=333.999.0.0&vd_source=abc&p=2&t=30"

	preview := resolver.Resolve(t.Context(), 0, raw)
	if preview.Status != StatusReady {
		t.Fatalf("status = %q", preview.Status)
	}
	if preview.Platform != string(platformBilibili) || preview.ResourceType != string(resourceVideo) {
		t.Fatalf("platform/resource = %q/%q", preview.Platform, preview.ResourceType)
	}
	if preview.Author != "瓶子君152" {
		t.Fatalf("author = %q", preview.Author)
	}
	if preview.Title != "四月番毒奶安利！四月看这八部动画就够了！！" {
		t.Fatalf("title = %q", preview.Title)
	}
	if preview.Description != "微博：@瓶子君152点个关注再走啦~" {
		t.Fatalf("description = %q", preview.Description)
	}
	if !strings.HasPrefix(preview.ImageURL, "https://i2.hdslb.com/") {
		t.Fatalf("cover = %q", preview.ImageURL)
	}
	// 卡片保留作者分享的原始链接（含追踪参数），只有缓存键与抓取地址被规范化。
	if preview.RequestedURL != raw || preview.URL != raw {
		t.Fatalf("requested/navigation URL = %q / %q", preview.RequestedURL, preview.URL)
	}
	// 规范化只删追踪参数：p / t 这类改变内容的参数必须进入实际抓取地址。
	fetched := fetcher.lastURL()
	if strings.Contains(fetched, "spm_id_from") || strings.Contains(fetched, "vd_source") {
		t.Fatalf("tracking params leaked into fetch URL: %q", fetched)
	}
	if !strings.Contains(fetched, "p=2") || !strings.Contains(fetched, "t=30") {
		t.Fatalf("semantic params dropped from fetch URL: %q", fetched)
	}
}

func TestResolveBilibiliShortLinkKeepsSharedURL(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	resolver := New(fetcher, nil)

	preview := resolver.Resolve(t.Context(), 0, "https://b23.tv/abcd1234")
	if preview.Platform != string(platformBilibili) || preview.ResourceType != string(resourceVideo) {
		t.Fatalf("short link platform/resource = %q/%q", preview.Platform, preview.ResourceType)
	}
	// 识别发生在重定向之后，但卡片上展示和跳转的仍是作者分享的短链。
	if preview.RequestedURL != "https://b23.tv/abcd1234" || preview.URL != "https://b23.tv/abcd1234" {
		t.Fatalf("requested/navigation URL = %q / %q", preview.RequestedURL, preview.URL)
	}
}

func TestResolveShortLinkLandingOffPlatformStaysGeneric(t *testing.T) {
	// 短链可以指向任何地方：只有落地地址本身是受支持平台时才允许贴平台标签，
	// 否则一张「哔哩哔哩」卡片会把用户带到完全无关的站点。
	fetcher := platformFetcher(t, appleSongLookupFixture)
	fetcher.result.FinalURL = mustParseURL(t, "https://evil.example/redirected")
	fetcher.result.Body = []byte(`<!doctype html><html><head>
<meta property="og:title" content="Landing page_哔哩哔哩_bilibili">
<meta name="author" content="Not an uploader">
</head><body></body></html>`)

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://b23.tv/abcd1234")
	if preview.Status != StatusReady {
		t.Fatalf("status = %q", preview.Status)
	}
	if preview.Platform != "" || preview.ResourceType != "" || preview.Author != "" {
		t.Fatalf("platform fields from unrelated landing page: %#v", preview)
	}
	// 通用卡片原样保留页面自己的文本，不做任何平台专属清理。
	if preview.Title != "Landing page_哔哩哔哩_bilibili" {
		t.Fatalf("title = %q", preview.Title)
	}
}

func TestResolveBilibiliWithoutMetadataStaysGeneric(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	fetcher.result.Body = []byte(`<!doctype html><html><head><meta property="og:site_name" content="哔哩哔哩"></head><body></body></html>`)

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://www.bilibili.com/video/BV1B7411m7LV")
	if preview.Status != StatusReady {
		t.Fatalf("status = %q", preview.Status)
	}
	// 页面没给标题时不贴平台标签：通用卡片 + 原链接，而不是「哔哩哔哩 / 域名」空卡。
	if preview.Platform != "" || preview.ResourceType != "" || preview.Author != "" {
		t.Fatalf("platform fields on metadata-less card: %#v", preview)
	}
	if preview.Title != "哔哩哔哩" {
		t.Fatalf("title = %q", preview.Title)
	}
}

func TestResolveBilibiliDropsCoverOutsidePlatformCdn(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	fetcher.result.Body = []byte(`<!doctype html><html><head>
<meta property="og:title" content="某视频_哔哩哔哩_bilibili">
<meta property="og:image" content="https://evil.example/cover.jpg">
</head><body></body></html>`)

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://www.bilibili.com/video/BV1B7411m7LV")
	if preview.Platform != string(platformBilibili) {
		t.Fatalf("platform = %q", preview.Platform)
	}
	if preview.ImageURL != "" {
		t.Fatalf("cover kept from foreign host: %q", preview.ImageURL)
	}
}

func TestResolveAppleMusicSongCard(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	raw := "https://music.apple.com/cn/album/better-together/1440857781?i=1440857786&uo=4"

	preview := New(fetcher, nil).Resolve(t.Context(), 0, raw)
	if preview.Status != StatusReady {
		t.Fatalf("status = %q", preview.Status)
	}
	if preview.Platform != string(platformAppleMusic) || preview.ResourceType != string(resourceSong) {
		t.Fatalf("platform/resource = %q/%q", preview.Platform, preview.ResourceType)
	}
	if preview.Title != "Better Together" || preview.Author != "Jack Johnson" {
		t.Fatalf("title/author = %q / %q", preview.Title, preview.Author)
	}
	if preview.Description != "In Between Dreams (Bonus Track Version)" {
		t.Fatalf("description = %q", preview.Description)
	}
	if !strings.HasSuffix(preview.ImageURL, "/600x600bb.jpg") {
		t.Fatalf("cover = %q", preview.ImageURL)
	}
	// 结构化端点必须是我们自己拼的固定地址：歌曲 ID 来自 ?i=，地区来自路径。
	if got := fetcher.lastJSONURL(); got != "https://itunes.apple.com/lookup?id=1440857786&country=cn" {
		t.Fatalf("lookup URL = %q", got)
	}
	// 结构化解析成功就不该再去抓原页面。
	if fetcher.calls.Load() != 0 {
		t.Fatalf("page fetches = %d, want 0", fetcher.calls.Load())
	}
	if preview.RequestedURL != raw || preview.URL != raw {
		t.Fatalf("requested/navigation URL = %q / %q", preview.RequestedURL, preview.URL)
	}
}

func TestResolveAppleMusicAlbumCard(t *testing.T) {
	fetcher := platformFetcher(t, appleAlbumLookupFixture)
	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://music.apple.com/us/album/in-between-dreams/1440857781")

	if preview.Platform != string(platformAppleMusic) || preview.ResourceType != string(resourceAlbum) {
		t.Fatalf("platform/resource = %q/%q", preview.Platform, preview.ResourceType)
	}
	if preview.Title != "In Between Dreams (Bonus Track Version)" || preview.Author != "Jack Johnson" {
		t.Fatalf("title/author = %q / %q", preview.Title, preview.Author)
	}
	if preview.Description != "" {
		t.Fatalf("album description = %q, want empty", preview.Description)
	}
	if got := fetcher.lastJSONURL(); got != "https://itunes.apple.com/lookup?id=1440857781&country=us" {
		t.Fatalf("lookup URL = %q", got)
	}
}

func TestResolveAppleMusicLookupFailureFallsBackToGenericCard(t *testing.T) {
	cases := []struct {
		name string
		body string
		err  error
	}{
		{name: "no result in this storefront", body: appleEmptyLookupFixture},
		{name: "malformed json", body: `{"resultCount":`},
		{name: "transport error", err: &safefetch.FetchError{Class: safefetch.ErrorTimeout}},
	}
	for _, testCase := range cases {
		fetcher := platformFetcher(t, appleEmptyLookupFixture)
		fetcher.jsonResult.Body = []byte(testCase.body)
		fetcher.jsonErr = testCase.err
		// 通用降级路径抓的是 music.apple.com 原页面，带 og 元数据。
		fetcher.result.FinalURL = mustParseURL(t, "https://music.apple.com/cn/song/better-together/1440857786")
		fetcher.result.Body = []byte(`<!doctype html><html><head>
<meta property="og:title" content="Apple Music 上Jack Johnson的歌曲《Better Together》">
<meta property="og:image" content="https://is1-ssl.mzstatic.com/image/thumb/x/1200x630wp-60.jpg">
</head><body></body></html>`)

		preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786")
		if preview.Status != StatusReady {
			t.Fatalf("%s: status = %q", testCase.name, preview.Status)
		}
		// 平台与资源类型只来自严格识别的地址，因此降级卡片仍然标注平台；
		// 名称退回页面 og，艺术家缺失就省略。
		if preview.Platform != string(platformAppleMusic) || preview.ResourceType != string(resourceSong) {
			t.Fatalf("%s: platform/resource = %q/%q", testCase.name, preview.Platform, preview.ResourceType)
		}
		if preview.Title != "Apple Music 上Jack Johnson的歌曲《Better Together》" || preview.Author != "" {
			t.Fatalf("%s: title/author = %q / %q", testCase.name, preview.Title, preview.Author)
		}
		if fetcher.calls.Load() != 1 {
			t.Fatalf("%s: page fetches = %d, want 1", testCase.name, fetcher.calls.Load())
		}
	}
}

func TestResolveAppleMusicRejectsMismatchedRecordType(t *testing.T) {
	fetcher := platformFetcher(t, appleAlbumLookupFixture)
	// 链接声明是歌曲，Lookup 却只回了专辑：不得把专辑当成歌曲。
	fetcher.result.Body = []byte(`<!doctype html><html><head></head><body></body></html>`)

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786")
	if preview.Platform != "" || preview.ResourceType != "" {
		t.Fatalf("platform fields from mismatched record: %#v", preview)
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("page fetches = %d, want 1", fetcher.calls.Load())
	}
}

func TestResolveAppleMusicRejectsForeignLookupFinalHost(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	fetcher.jsonResult.FinalURL = mustParseURL(t, "https://evil.example/lookup")
	fetcher.result.Body = []byte(`<!doctype html><html><head></head><body></body></html>`)

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786")
	if preview.Platform != "" {
		t.Fatalf("foreign lookup host accepted: %#v", preview)
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("page fetches = %d, want 1", fetcher.calls.Load())
	}
}

func TestResolveAppleMusicDropsArtworkOutsidePlatformCdn(t *testing.T) {
	fetcher := platformFetcher(t, strings.ReplaceAll(appleSongLookupFixture, "https://is1-ssl.mzstatic.com", "https://evil.example"))

	preview := New(fetcher, nil).Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786")
	if preview.Platform != string(platformAppleMusic) {
		t.Fatalf("platform = %q", preview.Platform)
	}
	if preview.ImageURL != "" {
		t.Fatalf("cover kept from foreign host: %q", preview.ImageURL)
	}
}

func TestCacheKeyURLCanonicalizesOnlySupportedPlatformLinks(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "bilibili drops tracking and trailing slash",
			raw:  "https://www.bilibili.com/video/BV1B7411m7LV/?spm_id_from=333.999.0.0&vd_source=abc",
			want: "https://www.bilibili.com/video/BV1B7411m7LV",
		},
		{
			name: "bilibili keeps part and time",
			raw:  "https://www.bilibili.com/video/BV1B7411m7LV?p=2&t=30&spm_id_from=333.9",
			want: "https://www.bilibili.com/video/BV1B7411m7LV?p=2&t=30",
		},
		{
			name: "host case is normalized",
			raw:  "https://WWW.Bilibili.com/video/BV1B7411m7LV",
			want: "https://www.bilibili.com/video/BV1B7411m7LV",
		},
		{
			name: "apple keeps song id and drops share tokens",
			raw:  "https://music.apple.com/cn/album/better-together/1440857781?i=1440857786&uo=4&at=1010lbf1",
			want: "https://music.apple.com/cn/album/better-together/1440857781?i=1440857786",
		},
		{
			name: "unsupported links are untouched",
			raw:  "https://example.com/x/?spm_id_from=1",
			want: "https://example.com/x/?spm_id_from=1",
		},
	}
	for _, testCase := range cases {
		key := cacheKeyURL(*mustParseURL(t, testCase.raw))
		if got := key.String(); got != testCase.want {
			t.Fatalf("%s: key = %q, want %q", testCase.name, got, testCase.want)
		}
	}
}

func TestResolveSharesCacheAcrossSharingVariantsOnly(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	resolver := New(fetcher, nil)

	base := "https://www.bilibili.com/video/BV1B7411m7LV"
	for _, raw := range []string{base + "/?spm_id_from=333.1", base + "?vd_source=abc", base} {
		if preview := resolver.Resolve(t.Context(), 0, raw); preview.Status != StatusReady {
			t.Fatalf("status = %q for %q", preview.Status, raw)
		}
	}
	if fetcher.calls.Load() != 1 {
		t.Fatalf("fetch calls = %d, want 1 shared across sharing variants", fetcher.calls.Load())
	}
	// 分 P 改变用户看到的内容，必须各自成键，不能被合并。
	if preview := resolver.Resolve(t.Context(), 0, base+"?p=3"); preview.Status != StatusReady {
		t.Fatalf("part status = %q", preview.Status)
	}
	if fetcher.calls.Load() != 2 {
		t.Fatalf("fetch calls = %d, want 2 after a different part", fetcher.calls.Load())
	}
}

func TestResolveAppleMusicKeyedByStorefrontAndResource(t *testing.T) {
	fetcher := platformFetcher(t, appleSongLookupFixture)
	resolver := New(fetcher, nil)

	resolver.Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786?uo=4")
	resolver.Resolve(t.Context(), 0, "https://music.apple.com/cn/song/better-together/1440857786?at=1010lbf1")
	if fetcher.jsonCalls.Load() != 1 {
		t.Fatalf("lookup calls = %d, want 1 for the same region and resource", fetcher.jsonCalls.Load())
	}
	resolver.Resolve(t.Context(), 0, "https://music.apple.com/us/song/better-together/1440857786")
	resolver.Resolve(t.Context(), 0, "https://music.apple.com/cn/album/better-together/1440857781")
	if fetcher.jsonCalls.Load() != 3 {
		t.Fatalf("lookup calls = %d, want 3 across regions and resources", fetcher.jsonCalls.Load())
	}
}
