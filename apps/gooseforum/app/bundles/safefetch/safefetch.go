// Package safefetch performs bounded HTTP fetches without allowing callers to
// bypass URL, DNS, redirect, or response-size policy.
package safefetch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/urlutil"
)

const (
	defaultConnectTimeout = 2 * time.Second
	defaultTotalTimeout   = 3 * time.Second
	defaultMaxRedirects   = 3
	defaultMaxBodyBytes   = 512 << 10
	defaultMaxConcurrency = 16
)

// ErrorClass is a stable category suitable for resolver fallback behavior.
type ErrorClass string

const (
	ErrorInvalid     ErrorClass = "invalid"
	ErrorBlocked     ErrorClass = "blocked"
	ErrorTimeout     ErrorClass = "timeout"
	ErrorRedirect    ErrorClass = "redirect"
	ErrorTooLarge    ErrorClass = "too_large"
	ErrorContentType ErrorClass = "content_type"
	ErrorUnavailable ErrorClass = "unavailable"
)

// FetchError hides low-level network details while preserving error matching.
type FetchError struct {
	Class ErrorClass
	Err   error
}

func (e *FetchError) Error() string { return "safe fetch: " + string(e.Class) }
func (e *FetchError) Unwrap() error { return e.Err }

// IsClass reports whether err contains a FetchError with class.
func IsClass(err error, class ErrorClass) bool {
	fetchErr, ok := errors.AsType[*FetchError](err)
	return ok && fetchErr.Class == class
}

// Resolver is the net.Resolver subset used before every connection.
type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}

// DialContextFunc matches net.Dialer's DialContext method.
type DialContextFunc func(context.Context, string, string) (net.Conn, error)

// Config controls resource budgets and exposes narrow injection points for
// deterministic security tests.
type Config struct {
	Resolver       Resolver
	DialContext    DialContextFunc
	ConnectTimeout time.Duration
	TotalTimeout   time.Duration
	MaxRedirects   int
	MaxBodyBytes   int64
	MaxConcurrency int
	// PreferIPv4 dials validated IPv4 addresses first and falls back to the
	// next validated address when a dial fails. It is for large downloads on
	// networks whose IPv6 path connects but stalls mid-transfer.
	PreferIPv4 bool
}

// Result is the complete, bounded response available to metadata resolvers.
type Result struct {
	RetryAfter  string
	FinalURL    *url.URL
	StatusCode  int
	ContentType string
	Body        []byte
	Duration    time.Duration
}

// Client is safe for concurrent use.
type Client struct {
	httpClient     *http.Client
	resolver       Resolver
	dialContext    DialContextFunc
	maxBodyBytes   int64
	connectTimeout time.Duration
	preferIPv4     bool
	semaphore      chan struct{}
}

// New constructs a bounded client. Zero values select conservative defaults.
func New(config Config) *Client {
	if config.Resolver == nil {
		config.Resolver = net.DefaultResolver
	}
	if config.ConnectTimeout <= 0 {
		config.ConnectTimeout = defaultConnectTimeout
	}
	if config.TotalTimeout <= 0 {
		config.TotalTimeout = defaultTotalTimeout
	}
	if config.MaxRedirects <= 0 {
		config.MaxRedirects = defaultMaxRedirects
	}
	if config.MaxBodyBytes <= 0 {
		config.MaxBodyBytes = defaultMaxBodyBytes
	}
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = defaultMaxConcurrency
	}
	if config.DialContext == nil {
		config.DialContext = (&net.Dialer{Timeout: config.ConnectTimeout}).DialContext
	}

	client := &Client{
		resolver:       config.Resolver,
		dialContext:    config.DialContext,
		maxBodyBytes:   config.MaxBodyBytes,
		connectTimeout: config.ConnectTimeout,
		preferIPv4:     config.PreferIPv4,
		semaphore:      make(chan struct{}, config.MaxConcurrency),
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = client.dialVerified
	transport.MaxConnsPerHost = 2
	transport.MaxResponseHeaderBytes = 64 << 10
	transport.ResponseHeaderTimeout = config.TotalTimeout
	transport.TLSHandshakeTimeout = config.ConnectTimeout
	client.httpClient = &http.Client{
		Transport: transport,
		Timeout:   config.TotalTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= config.MaxRedirects {
				return &FetchError{Class: ErrorRedirect, Err: errors.New("redirect limit exceeded")}
			}
			if urlutil.ClassifyLink(req.URL.String()).Kind != urlutil.LinkExternalHTTP {
				return &FetchError{Class: ErrorBlocked, Err: errors.New("redirect URL rejected")}
			}
			return checkURLPolicy(req.Context(), req.URL)
		},
	}
	return client
}

type urlPolicyKey struct{}

// WithURLPolicy restricts a document request and every redirect before sending.
// It supplements, and never replaces, the client's URL and public-IP checks.
func WithURLPolicy(ctx context.Context, allowed func(string) bool) context.Context {
	return context.WithValue(ctx, urlPolicyKey{}, allowed)
}

func checkURLPolicy(ctx context.Context, target *url.URL) error {
	if allowed, ok := ctx.Value(urlPolicyKey{}).(func(string) bool); ok && !allowed(target.String()) {
		return &FetchError{Class: ErrorBlocked, Err: errors.New("request URL rejected by caller policy")}
	}
	return nil
}

// documentPolicy 描述一次抓取允许请求和接受哪种有界文档。
type documentPolicy struct {
	accept     string
	mediaTypes []string
}

var (
	htmlDocument = documentPolicy{accept: "text/html, application/xhtml+xml", mediaTypes: []string{"text/html", "application/xhtml+xml"}}
	// text/javascript 不是笔误：Apple 的固定 Lookup 端点就是用这个旧媒体类型
	// 返回 JSON 的（实测 content-type: text/javascript; charset=utf-8）。放行它
	// 不会放宽任何边界——响应体仍必须通过调用方的 JSON 解析才能变成字段，畸形或
	// HTML 内容在解析那一步就被丢弃。
	jsonDocument = documentPolicy{accept: "application/json", mediaTypes: []string{"application/json", "text/javascript"}}
	// 图片响应头只是第一道筛选：调用方仍须按字节判定真实类型（imagepolicy）。
	// image/avif 只有贴纸市场导入会转码（PNG/GIF）后入库，其它调用方仍按字节拒绝。
	imageDocument = documentPolicy{accept: "image/webp, image/png, image/gif, image/jpeg, image/avif", mediaTypes: []string{"image/webp", "image/png", "image/gif", "image/jpeg", "image/avif"}}
)

func (p documentPolicy) allows(mediaType string) bool {
	for _, allowed := range p.mediaTypes {
		if mediaType == allowed {
			return true
		}
	}
	return false
}

// Fetch retrieves one HTML response within the configured limits.
func (c *Client) Fetch(ctx context.Context, rawURL string) (Result, error) {
	return c.fetchDocument(ctx, rawURL, htmlDocument)
}

// FetchJSON retrieves one bounded JSON response. It exists for the fixed,
// code-owned metadata endpoints of link-preview platform adapters: callers must
// never pass a user- or page-supplied URL through it. Every other policy —
// URL/DNS/redirect validation, timeouts, response size, concurrency — is
// identical to Fetch, so a JSON endpoint cannot widen the fetch surface.
func (c *Client) FetchJSON(ctx context.Context, rawURL string) (Result, error) {
	return c.fetchDocument(ctx, rawURL, jsonDocument)
}

// FetchImage retrieves one bounded image response under the same URL, DNS,
// redirect, timeout, size and concurrency policy as Fetch. It exists for
// server-side materialization of administrator-approved assets; the declared
// type only filters responses, so callers must still validate the bytes.
func (c *Client) FetchImage(ctx context.Context, rawURL string) (Result, error) {
	return c.fetchDocument(ctx, rawURL, imageDocument)
}

func (c *Client) fetchDocument(ctx context.Context, rawURL string, policy documentPolicy) (Result, error) {
	classified := urlutil.ClassifyLink(rawURL)
	if classified.Kind != urlutil.LinkExternalHTTP {
		return Result{}, &FetchError{Class: ErrorInvalid, Err: errors.New("URL rejected")}
	}
	target := *classified.URL
	target.Fragment = ""
	if err := checkURLPolicy(ctx, &target); err != nil {
		return Result{}, err
	}

	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return Result{}, classifyError(ctx.Err())
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return Result{}, &FetchError{Class: ErrorInvalid, Err: err}
	}
	request.Header.Set("Accept", policy.accept)
	request.Header.Set("User-Agent", "YourTJ-LinkPreview/1.0")

	started := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, classifyError(err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.ContentLength > c.maxBodyBytes {
		return Result{}, &FetchError{Class: ErrorTooLarge, Err: errors.New("content length exceeds limit")}
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || !policy.allows(mediaType) {
		return Result{}, &FetchError{Class: ErrorContentType, Err: errors.New("response type not allowed")}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, c.maxBodyBytes+1))
	if err != nil {
		return Result{}, classifyError(err)
	}
	if int64(len(body)) > c.maxBodyBytes {
		return Result{}, &FetchError{Class: ErrorTooLarge, Err: errors.New("decompressed body exceeds limit")}
	}
	return Result{
		FinalURL:    response.Request.URL,
		StatusCode:  response.StatusCode,
		ContentType: mediaType,
		Body:        body,
		Duration:    time.Since(started),
	}, nil
}

func (c *Client) dialVerified(ctx context.Context, network, address string) (net.Conn, error) {
	return c.dialVerifiedWithPolicy(ctx, network, address, false, isPublic)
}

// dialVerifiedWithPolicy resolves again for every connection and pins the
// accepted address, so a receiver cannot rebind DNS between check and dial.
func (c *Client) dialVerifiedWithPolicy(ctx context.Context, network, address string, rejectMapped bool, accept func(netip.Addr) bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, &FetchError{Class: ErrorBlocked, Err: err}
	}
	addresses, err := c.resolveAccepted(ctx, strings.Trim(host, "[]"), rejectMapped, accept)
	if err != nil {
		return nil, err
	}
	if !c.preferIPv4 {
		return c.dialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
	}
	// Every candidate passed the same policy check, so falling back never
	// reaches an address the caller did not accept.
	addresses = slices.Clone(addresses)
	slices.SortStableFunc(addresses, func(a, b netip.Addr) int {
		switch a4, b4 := a.Unmap().Is4(), b.Unmap().Is4(); {
		case a4 == b4:
			return 0
		case a4:
			return -1
		default:
			return 1
		}
	})
	var dialErr error
	for _, address := range addresses {
		conn, err := c.dialContext(ctx, network, net.JoinHostPort(address.String(), port))
		if err == nil {
			return conn, nil
		}
		dialErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return nil, dialErr
}

// resolveAccepted is shared by configuration checks and each fresh connection.
func (c *Client) resolveAccepted(ctx context.Context, host string, rejectMapped bool, accept func(netip.Addr) bool) ([]netip.Addr, error) {
	addresses := []netip.Addr{}
	if literal, err := netip.ParseAddr(host); err == nil {
		addresses = append(addresses, literal)
	} else {
		dnsCtx, cancel := context.WithTimeout(ctx, c.connectTimeout)
		defer cancel()
		addresses, err = c.resolver.LookupNetIP(dnsCtx, "ip", host)
		if err != nil {
			return nil, &FetchError{Class: ErrorUnavailable, Err: err}
		}
	}
	if len(addresses) == 0 {
		return nil, &FetchError{Class: ErrorUnavailable, Err: errors.New("DNS returned no addresses")}
	}
	for _, address := range addresses {
		if (rejectMapped && address.Is4In6()) || !accept(address.Unmap()) {
			return nil, &FetchError{Class: ErrorBlocked, Err: errors.New("rejected target address")}
		}
	}
	return addresses, nil
}

// CheckWebhookTarget checks URL policy and every DNS answer without making an
// HTTP request. Actual sends resolve again and pin their validated dial address.
func (c *Client) CheckWebhookTarget(ctx context.Context, rawURL string) error {
	if err := ValidateWebhookURL(rawURL); err != nil {
		return err
	}
	target, _ := url.Parse(rawURL)
	_, err := c.resolveAccepted(ctx, target.Hostname(), true, isPublic)
	return err
}

// CheckNotifyTarget checks the notification URL policy and every DNS answer
// without making an HTTP request. Actual sends resolve again and pin their
// validated dial address.
func (c *Client) CheckNotifyTarget(ctx context.Context, rawURL string) error {
	if err := ValidateNotifyURL(rawURL); err != nil {
		return err
	}
	target, err := url.Parse(rawURL)
	if err != nil {
		return &FetchError{Class: ErrorInvalid, Err: err}
	}
	_, err = c.resolveAccepted(ctx, target.Hostname(), true, isNotifyTarget)
	return err
}

// DoNotify sends a caller-built notification request to a target accepted by
// ValidateNotifyURL and returns the live response; the caller closes the body
// and owns response interpretation. Every connection re-resolves and pins the
// accepted address, proxies are off, connections are not pooled and redirects
// are not followed. Request headers stay exactly as the caller built them:
// they are set by this codebase (X-Goose-* and channel credentials).
func (c *Client) DoNotify(ctx context.Context, req *http.Request, timeout time.Duration) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, &FetchError{Class: ErrorInvalid}
	}
	if err := ValidateNotifyURL(req.URL.String()); err != nil {
		return nil, err
	}
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return nil, classifyError(ctx.Err())
	}
	base := c.httpClient.Transport.(*http.Transport)
	transport := base.Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return c.dialVerifiedWithPolicy(ctx, network, address, true, isNotifyTarget)
	}
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, classifyError(err)
	}
	return response, nil
}

var reservedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b:1::/48"),
	netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
}

func isPublic(address netip.Addr) bool { return !address.IsPrivate() && isNotifyTarget(address) }

// isNotifyTarget accepts intranet notification receivers: RFC1918/ULA private
// space and any port are legitimate for an enterprise webhook. Everything the
// server itself can reach locally (loopback, link-local, multicast,
// unspecified) and the reserved non-routable prefixes stay blocked.
//
// shortcut: 私网目标一律放行（企业内网通知是合法场景）；若通知配置将来向多租户或
// 非站点管理员开放，需改为显式的“允许内网目标”开关或端点白名单。
func isNotifyTarget(address netip.Addr) bool {
	if !address.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range reservedPrefixes {
		if prefix.Contains(address) {
			return false
		}
	}
	return true
}

func classifyError(err error) error {
	if fetchErr, ok := errors.AsType[*FetchError](err); ok {
		return fetchErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &FetchError{Class: ErrorTimeout, Err: err}
	}
	if networkErr, ok := errors.AsType[net.Error](err); ok && networkErr.Timeout() {
		return &FetchError{Class: ErrorTimeout, Err: err}
	}
	return &FetchError{Class: ErrorUnavailable, Err: err}
}

// ValidateWebhookURL is shared by save, test and every outbound attempt.
func ValidateWebhookURL(raw string) error {
	// Validate the exact representation that net/http will send. Link rendering
	// may decode HTML entities; webhook configuration is an API URL, not markup.
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return &FetchError{Class: ErrorInvalid}
	}
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") || strings.Contains(hostname, "%") {
		return &FetchError{Class: ErrorBlocked}
	}
	if _, err := netip.ParseAddr(hostname); err != nil {
		if numericHost(hostname) {
			return &FetchError{Class: ErrorBlocked}
		}
	}
	if u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Port() != "" && u.Port() != "443") {
		return &FetchError{Class: ErrorBlocked}
	}
	if ip, err := netip.ParseAddr(u.Hostname()); err == nil && (ip.Is4In6() || !isPublic(ip)) {
		return &FetchError{Class: ErrorBlocked}
	}
	return nil
}

// numericHost reports hostnames that some resolvers read as an IPv4 literal
// (127.1, 0177.0.0.1, 0x7f.1, 2130706433). They must be rejected at the URL
// layer instead of being handed to DNS.
func numericHost(hostname string) bool {
	parts := strings.Split(hostname, ".")
	if len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		base := 10
		if strings.HasPrefix(part, "0x") {
			base = 16
			part = strings.TrimPrefix(part, "0x")
		}
		if _, err := strconv.ParseUint(part, base, 32); err != nil {
			return false
		}
	}
	return true
}

// ValidateNotifyURL is the URL-level half of the notification target policy:
// http or https on any port, including RFC1918/ULA intranet receivers, but no
// loopback, link-local, multicast, unspecified, reserved, IPv4-mapped literal,
// userinfo, fragment or localhost host name.
func ValidateNotifyURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return &FetchError{Class: ErrorInvalid}
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.Fragment != "" {
		return &FetchError{Class: ErrorBlocked}
	}
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if hostname == "" || hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.Contains(hostname, "%") {
		return &FetchError{Class: ErrorBlocked}
	}
	if ip, err := netip.ParseAddr(hostname); err == nil {
		if ip.Is4In6() || !isNotifyTarget(ip) {
			return &FetchError{Class: ErrorBlocked}
		}
		return nil
	}
	if numericHost(hostname) {
		return &FetchError{Class: ErrorBlocked}
	}
	return nil
}

// forwardedHeaderNames is the exact set of caller-supplied headers PostJSON is
// allowed to put on the wire. The request is built from scratch, proxies are
// disabled and redirects are refused, so no ambient credential can reach a
// receiver; a header still has to be named here to be forwarded. Authorization
// carries the optional per-Agent static credential (stored values are validated
// as visible ASCII before being encrypted, so they cannot inject header lines);
// the Webhook-* names carry the Standard Webhooks signature.
var forwardedHeaderNames = []string{
	"Webhook-Id",
	"Webhook-Timestamp",
	"Webhook-Signature",
	"Webhook-Attempt-Id",
	"Webhook-Delivery-Id",
	"Authorization",
}

// PostJSON only sends bounded JSON to public HTTPS/443; no redirect, proxy,
// ambient credentials or pooled connection can bypass a fresh DNS check.
func (c *Client) PostJSON(ctx context.Context, rawURL string, body []byte, headers http.Header) (Result, error) {
	if err := ValidateWebhookURL(rawURL); err != nil {
		return Result{}, err
	}
	if len(body) > 64<<10 {
		return Result{}, &FetchError{Class: ErrorTooLarge}
	}
	select {
	case c.semaphore <- struct{}{}:
		defer func() { <-c.semaphore }()
	case <-ctx.Done():
		return Result{}, classifyError(ctx.Err())
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, bytes.NewReader(body))
	if err != nil {
		return Result{}, &FetchError{Class: ErrorInvalid, Err: err}
	}
	for _, name := range forwardedHeaderNames {
		if v := headers.Get(name); v != "" {
			req.Header.Set(name, v)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "YourTJ-AgentWebhook/1.0")
	base := c.httpClient.Transport.(*http.Transport)
	transport := base.Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return c.dialVerifiedWithPolicy(ctx, network, address, true, isPublic)
	}
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: c.httpClient.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	started := time.Now()
	response, err := client.Do(req)
	if err != nil {
		return Result{}, classifyError(err)
	}
	defer func() { _ = response.Body.Close() }()
	// Webhook acceptance/retry is determined by the received status and headers;
	// an oversized or interrupted diagnostic body must not erase that outcome.
	result := Result{StatusCode: response.StatusCode, RetryAfter: response.Header.Get("Retry-After")}
	if response.ContentLength > c.maxBodyBytes {
		result.Duration = time.Since(started)
		return result, &FetchError{Class: ErrorTooLarge}
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, c.maxBodyBytes+1))
	result.Duration = time.Since(started)
	if err != nil {
		return result, classifyError(err)
	}
	if int64(len(responseBody)) > c.maxBodyBytes {
		return result, &FetchError{Class: ErrorTooLarge}
	}
	return result, nil
}
