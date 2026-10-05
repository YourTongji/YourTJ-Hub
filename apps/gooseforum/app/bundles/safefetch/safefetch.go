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
			return nil
		},
	}
	return client
}

// Fetch retrieves one HTML response within the configured limits.
func (c *Client) Fetch(ctx context.Context, rawURL string) (Result, error) {
	classified := urlutil.ClassifyLink(rawURL)
	if classified.Kind != urlutil.LinkExternalHTTP {
		return Result{}, &FetchError{Class: ErrorInvalid, Err: errors.New("URL rejected")}
	}
	target := *classified.URL
	target.Fragment = ""

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
	request.Header.Set("Accept", "text/html, application/xhtml+xml")
	request.Header.Set("User-Agent", "YourTJ-LinkPreview/1.0")

	started := time.Now()
	response, err := c.httpClient.Do(request)
	if err != nil {
		return Result{}, classifyError(err)
	}
	defer response.Body.Close()

	if response.ContentLength > c.maxBodyBytes {
		return Result{}, &FetchError{Class: ErrorTooLarge, Err: errors.New("content length exceeds limit")}
	}
	mediaType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || (mediaType != "text/html" && mediaType != "application/xhtml+xml") {
		return Result{}, &FetchError{Class: ErrorContentType, Err: errors.New("response is not HTML")}
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
	return c.dialVerifiedWithPolicy(ctx, network, address, false)
}
func (c *Client) dialVerifiedWithPolicy(ctx context.Context, network, address string, rejectMapped bool) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, &FetchError{Class: ErrorBlocked, Err: err}
	}
	addresses, err := c.resolvePublic(ctx, strings.Trim(host, "[]"), rejectMapped)
	if err != nil {
		return nil, err
	}
	return c.dialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}

// resolvePublic is shared by configuration checks and each fresh connection.
func (c *Client) resolvePublic(ctx context.Context, host string, rejectMapped bool) ([]netip.Addr, error) {
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
		if (rejectMapped && address.Is4In6()) || !isPublic(address.Unmap()) {
			return nil, &FetchError{Class: ErrorBlocked, Err: errors.New("non-public target")}
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
	_, err := c.resolvePublic(ctx, target.Hostname(), true)
	return err
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

func isPublic(address netip.Addr) bool {
	if !address.IsValid() || !address.IsGlobalUnicast() || address.IsPrivate() || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsUnspecified() {
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
	classified := urlutil.ClassifyLink(raw)
	if classified.Kind != urlutil.LinkExternalHTTP || classified.URL == nil {
		return &FetchError{Class: ErrorInvalid}
	}
	u := classified.URL
	hostname := strings.ToLower(strings.TrimSuffix(u.Hostname(), "."))
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") || strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") || strings.Contains(hostname, "%") {
		return &FetchError{Class: ErrorBlocked}
	}
	if _, err := netip.ParseAddr(hostname); err != nil {
		numeric := true
		parts := strings.Split(hostname, ".")
		if len(parts) > 4 {
			numeric = false
		}
		for _, part := range parts {
			base := 10
			if strings.HasPrefix(part, "0x") {
				base = 16
				part = strings.TrimPrefix(part, "0x")
			}
			if _, err := strconv.ParseUint(part, base, 32); err != nil {
				numeric = false
			}
		}
		if numeric {
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
	for _, name := range []string{"Webhook-Id", "Webhook-Timestamp", "Webhook-Signature", "Webhook-Attempt-Id", "Webhook-Delivery-Id"} {
		if v := headers.Get(name); v != "" {
			req.Header.Set(name, v)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "YourTJ-AgentWebhook/1.0")
	base := c.httpClient.Transport.(*http.Transport)
	transport := base.Clone()
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		return c.dialVerifiedWithPolicy(ctx, network, address, true)
	}
	transport.DisableKeepAlives = true
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: c.httpClient.Timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	started := time.Now()
	response, err := client.Do(req)
	if err != nil {
		return Result{}, classifyError(err)
	}
	defer response.Body.Close()
	if response.ContentLength > c.maxBodyBytes {
		return Result{}, &FetchError{Class: ErrorTooLarge}
	}
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, c.maxBodyBytes+1))
	if err != nil {
		return Result{}, classifyError(err)
	}
	if int64(len(responseBody)) > c.maxBodyBytes {
		return Result{}, &FetchError{Class: ErrorTooLarge}
	}
	return Result{StatusCode: response.StatusCode, Duration: time.Since(started), RetryAfter: response.Header.Get("Retry-After")}, nil
}
