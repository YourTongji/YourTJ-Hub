// Package linkpreviewservice resolves first-party resources and bounded
// third-party HTML into a shared, non-executable preview contract.
package linkpreviewservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	stdhtml "html"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/urlutil"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/topics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/wikiPages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/courseservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/topicaccessservice"
	"github.com/jellydator/ttlcache/v3"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/charset"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

type Kind string

const (
	KindUnknown  Kind = "unknown"
	KindInternal Kind = "internal"
	KindExternal Kind = "external"
)

type Status string

const (
	StatusReady            Status = "ready"
	StatusInvalid          Status = "invalid"
	StatusBlocked          Status = "blocked"
	StatusUnavailable      Status = "unavailable"
	StatusTimeout          Status = "timeout"
	StatusUnsupported      Status = "unsupported"
	StatusPermissionDenied Status = "permission_denied"
)

// Preview is the only metadata shape exposed to clients. It never contains
// remote HTML or backend error strings.
type Preview struct {
	RequestedURL      string    `json:"requestedUrl"`
	Kind              Kind      `json:"kind"`
	Status            Status    `json:"status"`
	URL               string    `json:"url,omitempty"`
	DisplayHost       string    `json:"displayHost,omitempty"`
	RegistrableDomain string    `json:"registrableDomain,omitempty"`
	SiteName          string    `json:"siteName,omitempty"`
	Title             string    `json:"title,omitempty"`
	Description       string    `json:"description,omitempty"`
	ImageURL          string    `json:"imageUrl,omitempty"`
	FaviconURL        string    `json:"faviconUrl,omitempty"`
	FetchedAt         time.Time `json:"fetchedAt,omitzero"`
	// Campus 标记「按部署配置本地渲染、从未抓取」的校园网卡片。此时 Title 可能为
	// 空（配置没给名字），客户端需要用自己的语言补兜底标题与描述。
	Campus bool `json:"campus,omitempty"`
}

type Fetcher interface {
	Fetch(context.Context, string) (safefetch.Result, error)
}

type Resolver struct {
	fetcher Fetcher
	origins func() []string
	campus  func() CampusPolicy
	now     func() time.Time
	cache   *ttlcache.Cache[string, Preview]
	group   singleflight.Group
}

func New(fetcher Fetcher, origins func() []string) *Resolver {
	if fetcher == nil {
		fetcher = safefetch.New(safefetch.Config{})
	}
	if origins == nil {
		origins = func() []string { return nil }
	}
	return &Resolver{
		fetcher: fetcher,
		origins: origins,
		campus:  campusPolicyFromConfig,
		now:     time.Now,
		cache: ttlcache.New[string, Preview](
			ttlcache.WithCapacity[string, Preview](1024),
			ttlcache.WithDisableTouchOnHit[string, Preview](),
		),
	}
}

var defaultResolver = sync.OnceValue(func() *Resolver {
	return New(nil, configuredOrigins)
})

func Default() *Resolver { return defaultResolver() }

func configuredOrigins() []string {
	settings := hotdataserve.GetSiteSettingsConfigCache()
	return []string{
		preferences.GetString("server.url", ""),
		settings.SiteUrl,
	}
}

func (r *Resolver) Resolve(ctx context.Context, viewerID uint64, rawURL string) Preview {
	classified := urlutil.ClassifyLink(rawURL, r.origins()...)
	base := Preview{RequestedURL: rawURL, Kind: KindUnknown, Status: StatusInvalid}
	switch classified.Kind {
	case urlutil.LinkRelativeInternal, urlutil.LinkAbsoluteInternal:
		base.Kind = KindInternal
		base.URL = rawURL
		preview := r.resolveInternal(ctx, viewerID, classified.URL, base)
		r.log(preview, rawURL, "bypass", internalProvider(classified.URL), 0)
		return preview
	case urlutil.LinkUnsupportedScheme:
		base.Status = StatusUnsupported
		r.log(base, rawURL, "bypass", "classification", 0)
		return base
	case urlutil.LinkInvalid:
		r.log(base, rawURL, "bypass", "classification", 0)
		return base
	case urlutil.LinkExternalHTTP:
		base.Kind = KindExternal
		base.URL = rawURL
		base.DisplayHost = classified.URL.Hostname()
		base.RegistrableDomain = registrableDomain(base.DisplayHost)
		// 校园网资源在进入缓存/单飞/抓取之前就分流掉：这条路径不发任何请求。
		if title, isCampus := r.campus().Card(base.DisplayHost); isCampus {
			preview := r.campusCard(base, title)
			r.log(preview, rawURL, "bypass", "campus", 0)
			return preview
		}
	default:
		return base
	}

	keyURL := *classified.URL
	keyURL.Fragment = ""
	key := keyURL.String()
	if item := r.cache.Get(key); item != nil {
		preview := item.Value()
		preview.RequestedURL = rawURL
		preview.URL = rawURL
		r.log(preview, key, "hit", "external", 0)
		return preview
	}

	result := r.group.DoChan(key, func() (any, error) {
		if item := r.cache.Get(key); item != nil {
			preview := item.Value()
			r.log(preview, key, "hit", "external", 0)
			return preview, nil
		}
		started := time.Now()
		// A singleflight call is shared by unrelated HTTP requests. Let one
		// caller stop waiting without cancelling the bounded fetch for every
		// other waiter or poisoning the negative cache with its cancellation.
		preview := r.resolveExternal(context.WithoutCancel(ctx), keyURL, base)
		r.cache.Set(key, preview, ttlFor(preview.Status))
		r.log(preview, key, "miss", "external", time.Since(started))
		return preview, nil
	})
	select {
	case <-ctx.Done():
		base.Status = StatusTimeout
		return base
	case resolved := <-result:
		if resolved.Err != nil {
			base.Status = StatusUnavailable
			return base
		}
		preview := resolved.Val.(Preview)
		preview.RequestedURL = rawURL
		preview.URL = rawURL
		return preview
	}
}

// campusCard 用本地派生字段渲染校园网链接。刻意不进缓存、不走单飞：它既不发
// 请求也不读库，重复计算没有成本，省掉缓存反而避免了「配置改了卡片不更新」。
func (r *Resolver) campusCard(base Preview, title string) Preview {
	// SiteName 沿用 resolveExternal 的约定回退到 host：来源行因此显示域名，
	// 底部 host 行按「与来源行相同则省略」的既有规则自动收敛，不会重复打印。
	base.SiteName = base.DisplayHost
	base.Status = StatusReady
	base.Campus = true
	// Title 可能为空（配置没给名字）：兜底文案由客户端按自身语言渲染，服务端
	// 不留任何中文字面量。Description 同理，恒由客户端补。
	base.Title = title
	base.Description = ""
	return base
}

func (r *Resolver) resolveExternal(ctx context.Context, target url.URL, base Preview) Preview {
	result, err := r.fetcher.Fetch(ctx, target.String())
	if err != nil {
		base.Status = statusForFetchError(err)
		return base
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		if result.StatusCode == http.StatusNotFound || result.StatusCode == http.StatusGone {
			base.Status = StatusUnsupported
		} else {
			base.Status = StatusUnavailable
		}
		return base
	}
	metadata, err := parseMetadata(result.Body, result.ContentType, result.FinalURL)
	if err != nil {
		base.Status = StatusUnsupported
		return base
	}
	base.Status = StatusReady
	base.SiteName = firstNonEmpty(metadata.siteName, base.DisplayHost)
	base.Title = firstNonEmpty(metadata.title, base.SiteName)
	base.Description = metadata.description
	base.ImageURL = metadata.imageURL
	base.FaviconURL = metadata.faviconURL
	base.FetchedAt = r.now().UTC()
	return base
}

func (r *Resolver) resolveInternal(ctx context.Context, viewerID uint64, target *url.URL, base Preview) Preview {
	path := "/"
	if target != nil && target.Path != "" {
		path = target.Path
	}
	base.DisplayHost = internalDisplayHost(r.origins())
	base.RegistrableDomain = registrableDomain(base.DisplayHost)
	base.SiteName = firstNonEmpty(hotdataserve.GetSiteSettingsConfigCache().SiteName, "YourTJ")
	base.FetchedAt = r.now().UTC()

	segments := strings.Split(strings.Trim(path, "/"), "/")
	isPostPath := len(segments) >= 3 && segments[0] == "p" && segments[1] == "post"
	isTopicPath := len(segments) >= 2 && segments[0] == "topics"
	if isPostPath || isTopicPath {
		idIndex := 2
		if segments[0] == "topics" {
			idIndex = 1
		}
		id, err := strconv.ParseUint(segments[idIndex], 10, 64)
		if err != nil {
			base.Status = StatusInvalid
			return base
		}
		topic := topics.GetSimple(id)
		if topic.Id == 0 {
			topic = topics.UnscopedGet(id)
		}
		if topic.Id == 0 || !topicaccessservice.CanView(&topic, viewerID) {
			base.Status = StatusPermissionDenied
			return base
		}
		base.Status = StatusReady
		base.Title = cleanText(topic.Title, 200)
		base.Description = cleanText(topic.Excerpt, 400)
		base.ImageURL = safeAssetURL(topic.FirstImageURL, target)
		return base
	}

	if len(segments) == 2 && segments[0] == "courses" {
		id, err := strconv.ParseUint(segments[1], 10, 64)
		if err != nil {
			base.Status = StatusInvalid
			return base
		}
		course, err := courseservice.GetCourseDetail(id)
		if err != nil {
			base.Status = hiddenStatus(err)
			return base
		}
		base.Status = StatusReady
		base.Title = cleanText(course.Name, 200)
		base.Description = cleanText(strings.Join(compactStrings(course.TeacherName, course.Department, course.PrimaryCode), " · "), 400)
		return base
	}

	if len(segments) >= 2 && segments[0] == "wiki" {
		wikiPath, err := url.PathUnescape(strings.Join(segments[1:], "/"))
		if err != nil {
			base.Status = StatusInvalid
			return base
		}
		page := wikiPages.GetByPath(wikiPath)
		if page.Id == 0 {
			base.Status = StatusPermissionDenied
			return base
		}
		topic := topics.GetSimple(page.TopicId)
		if topic.Id == 0 {
			topic = topics.UnscopedGet(page.TopicId)
		}
		if topic.Id == 0 || !topicaccessservice.CanView(&topic, viewerID) {
			base.Status = StatusPermissionDenied
			return base
		}
		base.Status = StatusReady
		base.Title = cleanText(page.Title, 200)
		base.Description = cleanText(stripMarkdown(page.Content), 400)
		return base
	}

	if len(segments) >= 2 && segments[0] == "u" {
		id, err := strconv.ParseUint(segments[1], 10, 64)
		if err != nil {
			base.Status = StatusInvalid
			return base
		}
		user, err := users.GetWithContext(ctx, id)
		if err != nil {
			base.Status = hiddenStatus(err)
			return base
		}
		base.Status = StatusReady
		base.Title = cleanText(firstNonEmpty(user.Nickname, user.Username), 200)
		base.Description = cleanText(user.Bio, 400)
		base.ImageURL = user.GetWebAvatarUrl()
		return base
	}

	base.Status = StatusUnsupported
	return base
}

func hiddenStatus(err error) Status {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, courseservice.ErrCourseNotFound) {
		return StatusPermissionDenied
	}
	return StatusUnavailable
}

type metadata struct {
	title       string
	description string
	siteName    string
	imageURL    string
	faviconURL  string
}

func parseMetadata(body []byte, contentType string, baseURL *url.URL) (metadata, error) {
	reader, err := charset.NewReader(bytes.NewReader(body), contentType)
	if err != nil {
		return metadata{}, err
	}
	document, err := xhtml.Parse(reader)
	if err != nil {
		return metadata{}, err
	}
	var result metadata
	var fallbackTitle string
	var fallbackDescription string
	var visit func(*xhtml.Node)
	visit = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode {
			switch strings.ToLower(node.Data) {
			case "title":
				fallbackTitle = cleanText(nodeText(node), 200)
			case "meta":
				name := strings.ToLower(firstNonEmpty(attribute(node, "property"), attribute(node, "name")))
				content := attribute(node, "content")
				switch name {
				case "og:title":
					result.title = cleanText(content, 200)
				case "og:description":
					result.description = cleanText(content, 400)
				case "description":
					fallbackDescription = cleanText(content, 400)
				case "og:site_name":
					result.siteName = cleanText(content, 100)
				case "og:image", "og:image:url", "twitter:image":
					if result.imageURL == "" {
						result.imageURL = safeAssetURL(content, baseURL)
					}
				}
			case "link":
				if result.faviconURL == "" && hasRel(node, "icon") {
					result.faviconURL = safeAssetURL(attribute(node, "href"), baseURL)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(document)
	result.title = firstNonEmpty(result.title, fallbackTitle)
	result.description = firstNonEmpty(result.description, fallbackDescription)
	return result, nil
}

func attribute(node *xhtml.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func hasRel(node *xhtml.Node, target string) bool {
	for value := range strings.FieldsSeq(strings.ToLower(attribute(node, "rel"))) {
		if value == target || value == "shortcut" && target == "icon" {
			return true
		}
	}
	return false
}

func nodeText(node *xhtml.Node) string {
	var builder strings.Builder
	var visit func(*xhtml.Node)
	visit = func(current *xhtml.Node) {
		if current.Type == xhtml.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(node)
	return builder.String()
}

func cleanText(value string, maxRunes int) string {
	value = stdhtml.UnescapeString(value)
	tokenizer := xhtml.NewTokenizer(strings.NewReader(value))
	var text strings.Builder
	for {
		switch tokenizer.Next() {
		case xhtml.ErrorToken:
			value = text.String()
			goto cleaned
		case xhtml.TextToken:
			text.Write(tokenizer.Text())
		}
	}

cleaned:
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && !unicode.IsSpace(r) {
			return -1
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

func safeAssetURL(raw string, baseURL *url.URL) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || baseURL == nil {
		return ""
	}
	reference, err := url.Parse(raw)
	if err != nil || reference.User != nil {
		return ""
	}
	resolved := baseURL.ResolveReference(reference)
	kind := urlutil.ClassifyLink(resolved.String()).Kind
	if kind != urlutil.LinkExternalHTTP && kind != urlutil.LinkRelativeInternal {
		return ""
	}
	return resolved.String()
}

func stripMarkdown(value string) string {
	replacer := strings.NewReplacer("#", " ", "*", " ", "_", " ", "`", " ", ">", " ", "[", " ", "]", " ", "(", " ", ")", " ")
	return replacer.Replace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, strings.TrimSpace(value))
		}
	}
	return result
}

func registrableDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	if domain, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return domain
	}
	return host
}

func internalDisplayHost(origins []string) string {
	for _, origin := range origins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err == nil && parsed.Hostname() != "" {
			return parsed.Hostname()
		}
	}
	return ""
}

func internalProvider(target *url.URL) string {
	if target == nil {
		return "internal"
	}
	segments := strings.Split(strings.Trim(target.Path, "/"), "/")
	if len(segments) == 0 {
		return "internal"
	}
	switch segments[0] {
	case "p", "topics":
		return "topic"
	case "courses":
		return "course"
	case "wiki":
		return "wiki"
	case "u":
		return "user"
	default:
		return "internal"
	}
}

func statusForFetchError(err error) Status {
	switch {
	case safefetch.IsClass(err, safefetch.ErrorBlocked):
		return StatusBlocked
	case safefetch.IsClass(err, safefetch.ErrorTimeout):
		return StatusTimeout
	case safefetch.IsClass(err, safefetch.ErrorInvalid):
		return StatusInvalid
	case safefetch.IsClass(err, safefetch.ErrorContentType):
		return StatusUnsupported
	default:
		return StatusUnavailable
	}
}

func ttlFor(status Status) time.Duration {
	switch status {
	case StatusReady:
		return 24 * time.Hour
	case StatusUnsupported, StatusPermissionDenied:
		return time.Hour
	case StatusBlocked, StatusInvalid:
		return 24 * time.Hour
	default:
		return 10 * time.Minute
	}
}

func (r *Resolver) log(preview Preview, key, cache, provider string, latency time.Duration) {
	hash := sha256.Sum256([]byte(key))
	slog.Info("link_preview_resolved",
		"kind", preview.Kind,
		"status", preview.Status,
		"provider", provider,
		"host", preview.DisplayHost,
		"urlHash", hex.EncodeToString(hash[:8]),
		"cache", cache,
		"latencyMs", latency.Milliseconds(),
	)
}
