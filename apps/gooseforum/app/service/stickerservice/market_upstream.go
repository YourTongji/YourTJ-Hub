package stickerservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/safefetch"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
)

// The sticker market reads the static FluxDO / bug-v3 market index. Only the
// server fetches it: clients never receive an upstream URL, and every image is
// downloaded, validated and stored locally before anyone can use it.
const (
	MarketSourceID        = "fluxdo"
	defaultMarketBaseURL  = "https://s.pwsh.us.kg"
	maxMarketCatalogPages = 50
	maxMarketPacksPerPage = 200
	maxMarketTopics       = 200
	// MaxMarketPackItems bounds one pack's import; larger upstream packs keep
	// their first items in upstream order.
	MaxMarketPackItems = 500
	maxMarketImageSide = 4096
)

// Image hosts used by the default market index. Administrators can replace
// the list with sticker_market.image_hosts; nothing outside it is fetched.
var defaultMarketImageHosts = []string{"cdn.ldstatic.com", "cdn3.ldstatic.com", "idcflare.com", "linux.do"}

var (
	ErrMarketDisabled    = errors.New("sticker market disabled")
	ErrMarketBusy        = errors.New("sticker market busy")
	ErrMarketUnavailable = errors.New("sticker market upstream unavailable")
)

// MarketFetcher is the safefetch subset the market uses.
type MarketFetcher interface {
	FetchJSON(ctx context.Context, rawURL string) (safefetch.Result, error)
	FetchImage(ctx context.Context, rawURL string) (safefetch.Result, error)
}

// SwapMarketFetcher replaces the upstream client and returns a restore
// function. It exists for tests, which must never reach the real market.
func SwapMarketFetcher(fetcher MarketFetcher) func() {
	previous := marketClient
	marketClient = fetcher
	return func() { marketClient = previous }
}

var marketClient MarketFetcher = safefetch.New(safefetch.Config{
	ConnectTimeout: 5 * time.Second,
	TotalTimeout:   20 * time.Second,
	MaxBodyBytes:   filedata.MaxFileSize,
	MaxConcurrency: 4,
	// Image CDNs serve multi-megabyte GIFs; on networks whose IPv6 path
	// connects but stalls, IPv6-first dialing turns every item into a timeout.
	PreferIPv4: true,
})

type marketConfig struct {
	Enabled    bool
	BaseURL    string
	baseHost   string
	ImageHosts map[string]bool
}

func loadMarketConfig() (marketConfig, error) {
	config := marketConfig{Enabled: preferences.GetBool("sticker_market.enabled", true), ImageHosts: map[string]bool{}}
	base, err := url.Parse(strings.TrimRight(preferences.GetString("sticker_market.base_url", defaultMarketBaseURL), "/"))
	if err != nil || base.Scheme != "https" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return config, fmt.Errorf("%w: sticker_market.base_url must be an https origin", ErrMarketUnavailable)
	}
	config.BaseURL = base.String()
	config.baseHost = strings.ToLower(base.Host)
	hosts := preferences.GetStringSlice("sticker_market.image_hosts")
	if len(hosts) == 0 {
		hosts = defaultMarketImageHosts
	}
	for _, host := range hosts {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			config.ImageHosts[host] = true
		}
	}
	return config, nil
}

// allowedImageURL accepts only HTTPS URLs on an allow-listed host and the
// default port; safefetch then rejects private or reserved addresses.
func (c marketConfig) allowedImageURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	return c.ImageHosts[strings.ToLower(u.Hostname())]
}

// marketIcon splits the upstream cover into an allow-listed image URL or a
// short text cover (some packs use one emoji or letter); anything else is dropped.
func (c marketConfig) marketIcon(raw string) (iconURL, iconText string) {
	raw = strings.TrimSpace(raw)
	if strings.Contains(raw, "://") {
		if len(raw) <= 512 && c.allowedImageURL(raw) {
			return raw, ""
		}
		return "", ""
	}
	return "", cleanUpstreamText(raw, 16)
}

type upstreamTopic struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	TotalPages int    `json:"totalPages"`
}

type upstreamGroup struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Topic      string `json:"topic"`
	EmojiCount int    `json:"emojiCount"`
	IsArchived bool   `json:"isArchived"`
}

type upstreamEmoji struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type upstreamGroupDetail struct {
	ID     string          `json:"id"`
	Emojis []upstreamEmoji `json:"emojis"`
}

// Upstream identifiers become URL path segments; keep them to a safe charset.
var upstreamIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// cleanUpstreamText trims, drops control characters and bounds display text.
func cleanUpstreamText(raw string, limit int) string {
	if !utf8.ValidString(raw) {
		return ""
	}
	text := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	runes := []rune(text)
	if len(runes) > limit {
		runes = runes[:limit]
	}
	return strings.TrimSpace(string(runes))
}

// marketErrorReason maps a failure to a stable reason stored for administrators.
// It never contains upstream URLs, addresses or response bodies.
func marketErrorReason(err error) string {
	switch {
	case err == nil:
		return ""
	case safefetch.IsClass(err, safefetch.ErrorTimeout):
		return "timeout"
	case safefetch.IsClass(err, safefetch.ErrorBlocked), safefetch.IsClass(err, safefetch.ErrorRedirect), errors.Is(err, errMarketHostBlocked):
		return "blocked"
	case safefetch.IsClass(err, safefetch.ErrorTooLarge), errors.Is(err, errMarketImageTooLarge):
		return "tooLarge"
	case safefetch.IsClass(err, safefetch.ErrorContentType), errors.Is(err, errMarketUnsupportedImage):
		return "unsupported"
	case errors.Is(err, errMarketInvalidImage), errors.Is(err, errMarketInvalidData):
		return "invalid"
	case errors.Is(err, errMarketStatus):
		return "status"
	case errors.Is(err, errMarketStore):
		return "storage"
	default:
		return "unavailable"
	}
}

var (
	errMarketHostBlocked      = errors.New("market host not allowed")
	errMarketStatus           = errors.New("market upstream returned a non-200 status")
	errMarketInvalidData      = errors.New("market upstream returned invalid data")
	errMarketInvalidImage     = errors.New("market image is not a usable image")
	errMarketUnsupportedImage = errors.New("market image format is not supported")
	errMarketImageTooLarge    = errors.New("market image exceeds limits")
	errMarketStore            = errors.New("market image could not be stored")
)

func fetchMarketJSON(ctx context.Context, config marketConfig, path string, target any) error {
	result, err := marketClient.FetchJSON(ctx, config.BaseURL+path)
	if err != nil {
		return err
	}
	if result.StatusCode != http.StatusOK {
		return errMarketStatus
	}
	// Redirects stay on the configured index host.
	if result.FinalURL != nil && (result.FinalURL.Scheme != "https" || !strings.EqualFold(result.FinalURL.Host, config.baseHost)) {
		return errMarketHostBlocked
	}
	if err := json.Unmarshal(result.Body, target); err != nil {
		return fmt.Errorf("%w: %w", errMarketInvalidData, err)
	}
	return nil
}

func fetchMarketTopics(ctx context.Context, config marketConfig) ([]upstreamTopic, error) {
	var payload struct {
		Topics []upstreamTopic `json:"topics"`
	}
	if err := fetchMarketJSON(ctx, config, "/assets/market/index/topics.json", &payload); err != nil {
		return nil, err
	}
	if len(payload.Topics) == 0 || len(payload.Topics) > maxMarketTopics {
		return nil, errMarketInvalidData
	}
	return payload.Topics, nil
}

func fetchMarketPage(ctx context.Context, config marketConfig, page int) ([]upstreamGroup, int, error) {
	var payload struct {
		Groups     []upstreamGroup `json:"groups"`
		TotalPages int             `json:"totalPages"`
	}
	if err := fetchMarketJSON(ctx, config, fmt.Sprintf("/assets/market/index/page-%d.json", page), &payload); err != nil {
		return nil, 0, err
	}
	if len(payload.Groups) > maxMarketPacksPerPage {
		return nil, 0, errMarketInvalidData
	}
	return payload.Groups, payload.TotalPages, nil
}

// marketGroupPath mirrors the upstream file layout: ids already carry the
// "group-" prefix in the current index; older ids do not.
func marketGroupPath(upstreamID string) (string, error) {
	if !upstreamIDRe.MatchString(upstreamID) {
		return "", errMarketInvalidData
	}
	if !strings.HasPrefix(upstreamID, "group-") {
		upstreamID = "group-" + upstreamID
	}
	return "/assets/market/" + upstreamID + ".json", nil
}

func fetchMarketGroup(ctx context.Context, config marketConfig, upstreamID string) ([]upstreamEmoji, error) {
	path, err := marketGroupPath(upstreamID)
	if err != nil {
		return nil, err
	}
	var payload upstreamGroupDetail
	if err := fetchMarketJSON(ctx, config, path, &payload); err != nil {
		return nil, err
	}
	emojis := make([]upstreamEmoji, 0, min(len(payload.Emojis), MaxMarketPackItems))
	seen := make(map[string]bool, len(payload.Emojis))
	for _, emoji := range payload.Emojis {
		if len(emojis) == MaxMarketPackItems {
			break
		}
		emoji.ID = strings.TrimSpace(emoji.ID)
		if emoji.ID == "" || len(emoji.ID) > 128 || !utf8.ValidString(emoji.ID) || seen[emoji.ID] {
			continue
		}
		seen[emoji.ID] = true
		emojis = append(emojis, emoji)
	}
	return emojis, nil
}
