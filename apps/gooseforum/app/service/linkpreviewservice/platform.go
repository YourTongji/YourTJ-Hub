package linkpreviewservice

import (
	"net/url"
	"regexp"
	"strings"
)

// platform.go 是「严格识别平台资源」这一层。识别结果只用来补卡片字段——平台、
// 资源类型、作者——不改变抓取策略：平台页面仍然走同一个 safefetch，仍然逐跳校验
// 公网地址、限制时长与响应体。
//
// 识别本身不放宽任何边界。只有主机、路径形状和资源 ID 三者同时对得上才算命中，
// 命中的 ID 只用于查询平台自己的固定端点与日志，绝不参与拼接请求地址。
type platformKind string

const (
	platformBilibili   platformKind = "bilibili"
	platformAppleMusic platformKind = "apple_music"
)

type resourceKind string

const (
	resourceVideo resourceKind = "video"
	resourceSong  resourceKind = "song"
	resourceAlbum resourceKind = "album"
)

// platformTarget 是一次严格识别出的平台资源。
type platformTarget struct {
	platform platformKind
	resource resourceKind
	// id 是平台自己的资源号（B 站 BV/av 号、Apple 的 trackId/collectionId）。
	id string
	// storefront 是 Apple Music 链接里的国家/地区代码。同一首歌在不同地区的
	// 目录里是不同资源，所以它必须同时进入 Lookup 请求和缓存键。
	storefront string
}

// recognizePlatform 按主机分派识别。短链（b23.tv）不在这里：它要先经过 safefetch
// 的合法跳转，拿到落地的真实地址再识别，这也是识别放在抓取之后的原因。
func recognizePlatform(target *url.URL) (platformTarget, bool) {
	if target == nil {
		return platformTarget{}, false
	}
	switch normalizePlatformHost(target.Hostname()) {
	case "bilibili.com", "www.bilibili.com", "m.bilibili.com":
		return recognizeBilibiliVideo(target)
	case "music.apple.com":
		return recognizeAppleMusic(target)
	}
	return platformTarget{}, false
}

func normalizePlatformHost(host string) string {
	return strings.ToLower(strings.TrimSuffix(host, "."))
}

// bilibiliVideoIDPattern 只认官方地址里的规范写法：BV + 10 位 base58 字符（大小写
// 敏感，不能规范化），或 av + 十进制号。ID 不参与请求，所以宁可严格。
var bilibiliVideoIDPattern = regexp.MustCompile(`^(?:BV[0-9A-Za-z]{10}|av[0-9]{1,15})$`)

func recognizeBilibiliVideo(target *url.URL) (platformTarget, bool) {
	segments := pathSegments(target.Path)
	if len(segments) != 2 || segments[0] != "video" || !bilibiliVideoIDPattern.MatchString(segments[1]) {
		return platformTarget{}, false
	}
	return platformTarget{platform: platformBilibili, resource: resourceVideo, id: segments[1]}, true
}

var (
	appleStorefrontPattern = regexp.MustCompile(`^[A-Za-z]{2}$`)
	appleResourceIDPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
)

// recognizeAppleMusic 识别 music.apple.com/{storefront}/{song|album}/{slug}/{id}。
// storefront 可以缺失（此时不向 Lookup 传 country，用它的默认地区）。
func recognizeAppleMusic(target *url.URL) (platformTarget, bool) {
	segments := pathSegments(target.Path)
	storefront := ""
	switch len(segments) {
	case 4:
		if !appleStorefrontPattern.MatchString(segments[0]) {
			return platformTarget{}, false
		}
		storefront = strings.ToLower(segments[0])
		segments = segments[1:]
	case 3:
	default:
		return platformTarget{}, false
	}
	resource, ok := map[string]resourceKind{"song": resourceSong, "album": resourceAlbum}[segments[0]]
	if !ok || !appleResourceIDPattern.MatchString(segments[2]) {
		return platformTarget{}, false
	}
	// 专辑链接用 ?i= 指向其中一首歌：歌曲 ID 优先，否则整张专辑会被当成一首歌。
	if resource == resourceAlbum {
		if songID := target.Query().Get("i"); appleResourceIDPattern.MatchString(songID) {
			return platformTarget{platform: platformAppleMusic, resource: resourceSong, id: songID, storefront: storefront}, true
		}
	}
	return platformTarget{platform: platformAppleMusic, resource: resource, id: segments[2], storefront: storefront}, true
}

// applyPlatformOverlay 在通用 HTML 卡片之上补平台字段。页面没给出标题时不补：与其
// 显示一张「平台 + 域名」的空卡片，不如保持通用卡片，客户端再退到原链接。
func applyPlatformOverlay(preview Preview, target platformTarget, meta metadata) Preview {
	if meta.title == "" {
		return preview
	}
	preview.Platform = string(target.platform)
	preview.ResourceType = string(target.resource)
	preview.Author = meta.author
	switch target.platform {
	case platformBilibili:
		preview.Title = bilibiliTitle(preview.Title)
		preview.Description = bilibiliDescription(preview.Description)
		if !assetHostAllowed(preview.ImageURL, "hdslb.com") {
			preview.ImageURL = ""
		}
	case platformAppleMusic:
		if !assetHostAllowed(preview.ImageURL, "mzstatic.com") {
			preview.ImageURL = ""
		}
	}
	return preview
}

// bilibiliTitleSuffixes 是 B 站 og:title / <title> 统一追加的站点后缀。来源行已经
// 写明平台，标题里再带一遍「_哔哩哔哩_bilibili」只是噪音。
var bilibiliTitleSuffixes = []string{"_哔哩哔哩_bilibili", "_哔哩哔哩"}

func bilibiliTitle(title string) string {
	for _, suffix := range bilibiliTitleSuffixes {
		trimmed := strings.TrimSpace(strings.TrimSuffix(title, suffix))
		if trimmed == title {
			continue
		}
		if trimmed != "" {
			return trimmed
		}
		return title
	}
	return title
}

// bilibiliDescription 砍掉 B 站 SEO 描述的统计尾巴（「视频播放量 …、弹幕量 …、
// 相关视频：…」）：这些字段属于平台统计而不是资源本身。找不到切口就原样保留，
// 不做模糊猜测。
func bilibiliDescription(description string) string {
	index := strings.Index(description, "视频播放量")
	if index < 0 {
		return description
	}
	return strings.TrimRight(strings.TrimSpace(description[:index]), " ,，、")
}

// assetHostAllowed 只接受平台自己 CDN 域下的 https 封面。平台卡片是在断言「这是
// 该资源的封面」，因此封面主机也必须属于该平台，不沿用通用卡片对任意主机的放行。
func assetHostAllowed(rawURL, domain string) bool {
	if rawURL == "" {
		return false
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Host == "" {
		return false
	}
	host := normalizePlatformHost(parsed.Hostname())
	return host == domain || strings.HasSuffix(host, "."+domain)
}

// trackingParams 是平台分享链里的追踪参数白名单。删掉它们不改变资源身份，只让同一
// 资源的不同分享链命中同一个缓存；p / t / i 这类会改变用户看到的内容的参数一律保留，
// 不做笼统清空 query。
var trackingParams = map[string]bool{
	"spm_id_from": true, "vd_source": true, "from_source": true, "from_spmid": true,
	"share_source": true, "share_medium": true, "bbid": true, "ts": true,
	"seid": true, "buvid": true,
	"uo": true, "itscg": true, "itsct": true, "at": true, "ct": true,
}

// cacheKeyURL 为受支持平台地址生成缓存键：小写主机、去掉白名单追踪参数和末尾斜杠。
// 只认平台地址：其它链接原样返回，既有缓存键语义不变。
func cacheKeyURL(target url.URL) url.URL {
	if _, ok := recognizePlatform(&target); !ok {
		return target
	}
	target.Host = strings.ToLower(target.Host)
	query := target.Query()
	for name := range trackingParams {
		query.Del(name)
	}
	target.RawQuery = query.Encode()
	if target.Path != "/" {
		target.Path = strings.TrimSuffix(target.Path, "/")
	}
	return target
}

func pathSegments(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}
