// Package agentwebhookservice delivers Agent inbox wakeups over HTTPS with
// strict SSRF defenses: sender-time endpoint validation, resolve-all
// A/AAAA with reject-any non-public address, DNS-pinned dialing, no
// redirects, a fixed small JSON payload, and sanitized errors that never
// contain tokens, full URLs, headers, or bodies.
package agentwebhookservice

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/leancodebox/GooseForum/app/models/forum/agentInbox"
	"github.com/leancodebox/GooseForum/app/models/forum/agents"
	"github.com/leancodebox/GooseForum/app/models/forum/posts"
	"github.com/leancodebox/GooseForum/app/models/forum/taskQueue"
	"github.com/leancodebox/GooseForum/app/models/forum/topics"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/leancodebox/GooseForum/app/service/urlconfig"
)

const (
	// TaskTypeAgentWebhook is the task_queue type prefix for inbox deliveries.
	TaskTypeAgentWebhook = "agent.webhook"

	// maxDeliveryAttempts bounds total attempts per inbox row.
	maxDeliveryAttempts = 3
	// retryDelayFirst/retryDelaySecond are the delays after attempt 1 and 2.
	retryDelayFirst  = time.Minute
	retryDelaySecond = 5 * time.Minute
	// totalTimeout bounds the whole request (dial + TLS + write + read).
	totalTimeout = 5 * time.Second
	// maxResponseBody bounds the response body read (64KB).
	maxResponseBody = 64 * 1024
	// userAgent is the stable sender identity header.
	userAgent = "yourtj-hub-webhook/1.0"

	contentTypeJSON = "application/json"
)

// TaskPayload is the durable task_queue payload; it carries only the inbox id.
type TaskPayload struct {
	InboxId uint64 `json:"inboxId"`
}

// WebhookPayload is the fixed small POST body. It carries IDs, safe topic and
// post facts, a safe actor view, a preview, URLs, and timestamps — never full
// content, tokens, hashes, or endpoint details.
type WebhookPayload struct {
	EventId    uint64     `json:"eventId"`
	EventType  string     `json:"eventType"`
	Timestamp  int64      `json:"timestamp"`
	Topic      *TopicFact `json:"topic,omitempty"`
	Post       *PostFact  `json:"post,omitempty"`
	Actor      *ActorFact `json:"actor,omitempty"`
	Preview    string     `json:"preview"`
	URL        string     `json:"url"`
	OccurredAt int64      `json:"occurredAt"`
}

type TopicFact struct {
	Id    uint64 `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type PostFact struct {
	Id     uint64 `json:"id"`
	PostNo uint64 `json:"postNo"`
	URL    string `json:"url"`
}

type ActorFact struct {
	Id        uint64 `json:"id"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarUrl string `json:"avatarUrl"`
}

// Resolver resolves host names to IP addresses (net.DefaultResolver fits).
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// DeliverySender posts one webhook payload to one endpoint. Implementations
// must return sanitized errors that never contain the endpoint URL.
type DeliverySender interface {
	Send(ctx context.Context, endpoint string, payload WebhookPayload) error
}

// senderFactory is the injection seam for deterministic delivery tests.
var senderFactory = func() DeliverySender {
	return &webhookSender{
		resolver: net.DefaultResolver,
		timeout:  totalTimeout,
	}
}

type webhookSender struct {
	resolver    Resolver
	dialContext func(ctx context.Context, network, addr string) (net.Conn, error)
	roundTrip   func(req *http.Request) (*http.Response, error)
	timeout     time.Duration
}

// RunTask is the background worker handler for agent.webhook tasks. It reads
// the current inbox row (the delivery fact source): missing rows and rows not
// pending are no-ops; disabled/deleted/no-endpoint Agents are marked skipped;
// network delivery failures schedule run_at-based retries (1m then 5m) and
// become a terminal failed state after maxDeliveryAttempts.
func RunTask(ctx context.Context, task *taskQueue.Entity) error {
	taskPayload, err := decodeTaskPayload(task.TaskJson)
	if err != nil {
		// A malformed task is not retryable: mark the task done and log.
		slog.Error("agentwebhook: malformed task payload", "id", task.Id, "err", err)
		return nil
	}
	inbox := agentInbox.GetByID(taskPayload.InboxId)
	if inbox == nil {
		return nil
	}
	if inbox.DeliveryStatus != agentInbox.DeliveryPending {
		return nil
	}
	agent := agents.GetByUserID(inbox.AgentId)
	if agent == nil || agent.Enabled != agents.StatusEnabled || strings.TrimSpace(agent.WebhookEndpoint) == "" {
		if err := agentInbox.MarkSkipped(inbox.Id, "agent unavailable or no webhook endpoint"); err != nil {
			slog.Error("agentwebhook: mark skipped failed", "inboxId", inbox.Id, "err", err)
			return err
		}
		return nil
	}

	webhookPayload := buildWebhookPayload(inbox)
	webhookPayload.EventType = outboundEventType(inbox.EventType)

	sender := senderFactory()
	sendErr := sender.Send(ctx, agent.WebhookEndpoint, webhookPayload)
	if sendErr != nil {
		sanitized := sanitizeError(sendErr)
		attempts, recErr := agentInbox.RecordFailure(inbox.Id, sanitized)
		if recErr != nil {
			slog.Error("agentwebhook: record failure failed", "inboxId", inbox.Id, "err", recErr)
			return recErr
		}
		if attempts >= maxDeliveryAttempts {
			if err := agentInbox.MarkFailed(inbox.Id, sanitized); err != nil {
				slog.Error("agentwebhook: mark failed failed", "inboxId", inbox.Id, "err", err)
				return err
			}
			return nil
		}
		delay := retryDelayFirst
		if attempts >= 2 {
			delay = retryDelaySecond
		}
		if err := taskQueue.UpdateRunAt(task.Id, time.Now().Add(delay)); err != nil {
			slog.Error("agentwebhook: schedule retry failed", "inboxId", inbox.Id)
			return errors.New("webhook retry schedule failed")
		}
		// Returning the error makes the worker mark the task retrying; the
		// run_at filter keeps it invisible until the delay has passed.
		return fmt.Errorf("%s", sanitized)
	}
	if err := agentInbox.MarkDelivered(inbox.Id); err != nil {
		slog.Error("agentwebhook: mark delivered failed", "inboxId", inbox.Id, "err", err)
		return err
	}
	return nil
}

func decodeTaskPayload(taskJSON string) (TaskPayload, error) {
	var payload TaskPayload
	if err := json.Unmarshal([]byte(taskJSON), &payload); err != nil {
		return payload, err
	}
	if payload.InboxId == 0 {
		return payload, errors.New("agentwebhook: empty inbox id")
	}
	return payload, nil
}

func outboundEventType(eventType string) string {
	switch eventType {
	case agentInbox.EventTypeTopicPublished:
		return "agent.mention.topic_published"
	case agentInbox.EventTypeTopicUpdated:
		return "agent.mention.topic_updated"
	case agentInbox.EventTypePostCreated:
		return "agent.mention.post_created"
	default:
		return "agent.mention.unknown"
	}
}

// buildWebhookPayload assembles the current-state payload. Missing topic,
// post, or actor facts degrade safely to id-only fields — never a retry loop.
func buildWebhookPayload(inbox *agentInbox.Entity) WebhookPayload {
	now := time.Now()
	payload := WebhookPayload{
		EventId:    inbox.Id,
		EventType:  outboundEventType(inbox.EventType),
		Timestamp:  now.Unix(),
		Preview:    inbox.ContentPreview,
		OccurredAt: inbox.CreatedAt.UnixMilli(),
	}

	topic := topics.Get(inbox.TopicId)
	if topic.Id != 0 {
		payload.Topic = &TopicFact{
			Id:    topic.Id,
			Title: topic.Title,
			URL:   absoluteURL(urlconfig.PostDetail(topic.Id)),
		}
		payload.URL = absoluteURL(urlconfig.PostDetail(topic.Id))
	} else {
		payload.Topic = &TopicFact{Id: inbox.TopicId}
	}

	post := posts.Get(inbox.PostId)
	if post.Id != 0 {
		payload.Post = &PostFact{
			Id:     post.Id,
			PostNo: post.PostNo,
			URL:    absoluteURL(urlconfig.PostDetail(post.TopicId) + "#post-" + strconv.FormatUint(post.Id, 10)),
		}
		if payload.URL == "" {
			payload.URL = absoluteURL(urlconfig.PostDetail(post.TopicId))
		}
	} else {
		payload.Post = &PostFact{Id: inbox.PostId}
	}

	actor, err := users.Get(inbox.ActorId)
	if err != nil || actor.Id == 0 {
		payload.Actor = &ActorFact{Id: inbox.ActorId}
	} else {
		payload.Actor = &ActorFact{
			Id:        actor.Id,
			Username:  actor.Username,
			Nickname:  actor.Nickname,
			AvatarUrl: actor.GetWebAvatarUrl(),
		}
	}
	return payload
}

func absoluteURL(path string) string {
	base := strings.TrimRight(hotdataserve.GetSiteSettingsConfigCache().SiteUrl, "/")
	if base == "" {
		return path
	}
	return base + path
}

// Send validates the endpoint, resolves and pins a public IP, and POSTs the
// payload without redirects, within totalTimeout, reading at most 64KB.
func (s *webhookSender) Send(ctx context.Context, endpoint string, payload WebhookPayload) error {
	deliveryCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	target, err := validateEndpoint(endpoint)
	if err != nil {
		return err
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.New("webhook payload marshal failed")
	}
	pinnedIP, err := resolvePinned(deliveryCtx, s.resolver, target.Hostname())
	if err != nil {
		if errors.Is(deliveryCtx.Err(), context.DeadlineExceeded) {
			return errors.New("webhook request timed out")
		}
		return err
	}
	req, err := http.NewRequestWithContext(deliveryCtx, http.MethodPost, target.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("webhook request build failed")
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("X-Yourtj-Event-Id", strconv.FormatUint(payload.EventId, 10))
	req.Header.Set("X-Yourtj-Event-Type", payload.EventType)
	req.Header.Set("User-Agent", userAgent)

	if s.roundTrip != nil {
		resp, roundErr := s.roundTrip(req)
		return s.handleResponse(resp, roundErr)
	}

	dialFn := s.dialContext
	if dialFn == nil {
		dialFn = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{Timeout: s.timeout}).DialContext(ctx, network, addr)
		}
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// addr is hostname:port; dial only the already validated IP.
			_, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, splitErr
			}
			return dialFn(ctx, network, net.JoinHostPort(pinnedIP, port))
		},
		TLSHandshakeTimeout: s.timeout,
		DisableKeepAlives:   true,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   s.timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, doErr := client.Do(req)
	return s.handleResponse(resp, doErr)
}

func (s *webhookSender) handleResponse(resp *http.Response, err error) error {
	if err != nil {
		return classifyError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody+1))
	if readErr != nil {
		return errors.New("webhook response read failed")
	}
	if len(body) > maxResponseBody {
		return errors.New("webhook response body too large")
	}
	return nil
}

func classifyError(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("webhook request timed out")
	}
	var hostnameErr x509.HostnameError
	var unknownAuthorityErr x509.UnknownAuthorityError
	var certificateErr x509.CertificateInvalidError
	if errors.As(err, &hostnameErr) || errors.As(err, &unknownAuthorityErr) || errors.As(err, &certificateErr) {
		return errors.New("webhook TLS verification failed")
	}
	return errors.New("webhook request failed")
}

// sanitizeError makes sure no URL, query, header, token, or body ever lands in
// the inbox last_error column or the retry log line.
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	// classifyError-style errors are already sanitized; anything else (e.g. a
	// DB error) is replaced with a generic message. HTTP status failures only
	// carry the numeric status, never the response body.
	switch {
	case strings.HasPrefix(message, "webhook returned HTTP "):
		return message
	case message == "webhook endpoint missing",
		message == "webhook endpoint invalid",
		message == "webhook endpoint scheme not allowed",
		message == "webhook endpoint rejected",
		message == "webhook payload marshal failed",
		message == "webhook DNS resolution failed",
		message == "webhook endpoint resolves to non-public address",
		message == "webhook request build failed",
		message == "webhook request failed",
		message == "webhook request timed out",
		message == "webhook TLS verification failed",
		message == "webhook response read failed",
		message == "webhook response body too large":
		return message
	default:
		return "webhook delivery failed"
	}
}

// blockedPrefixes are the address ranges that must never be dialed: loopback,
// private, link-local, ULA, multicast, unspecified, CGNAT, documentation,
// benchmarking, and other non-public/reserved ranges (IPv4 and IPv6).
var blockedPrefixes = func() []netip.Prefix {
	raw := []string{
		"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
		"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
		"192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24", "203.0.113.0/24",
		"224.0.0.0/4", "240.0.0.0/4", "255.255.255.255/32",
		"::/128", "::1/128", "64:ff9b::/96", "2001::/32", "2001:10::/28",
		"2001:20::/28", "2001:db8::/32", "2002::/16", "fc00::/7", "fe80::/10", "ff00::/8",
	}
	prefixes := make([]netip.Prefix, 0, len(raw))
	for _, item := range raw {
		prefixes = append(prefixes, netip.MustParsePrefix(item))
	}
	return prefixes
}()

func isPublicIP(addr netip.Addr) bool {
	// Unmap turns IPv4-mapped IPv6 forms (::ffff:10.0.0.1) back into plain
	// IPv4 so the v4 blocklist applies to them as well.
	addr = addr.Unmap()
	if !addr.IsValid() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

// validateEndpoint applies sender-time endpoint validation: http(s) only, no
// userinfo, no fragment, no localhost-ish hostnames, and non-public IP
// literals are rejected. Errors are sanitized and never include the URL.
func validateEndpoint(raw string) (*url.URL, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, errors.New("webhook endpoint missing")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return nil, errors.New("webhook endpoint invalid")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("webhook endpoint scheme not allowed")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return nil, errors.New("webhook endpoint rejected")
	}
	hostname := strings.ToLower(strings.TrimSuffix(parsed.Hostname(), "."))
	if hostname == "" {
		return nil, errors.New("webhook endpoint invalid")
	}
	if hostname == "localhost" || strings.HasSuffix(hostname, ".localhost") ||
		strings.HasSuffix(hostname, ".local") || strings.HasSuffix(hostname, ".internal") {
		return nil, errors.New("webhook endpoint rejected")
	}
	if ip := net.ParseIP(hostname); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok || !isPublicIP(addr) {
			return nil, errors.New("webhook endpoint rejected")
		}
	}
	return parsed, nil
}

// resolvePinned resolves every A/AAAA record and rejects the host if any
// address is non-public (resolve-all, reject-any). It returns the first
// validated IP string for pinned dialing.
func resolvePinned(ctx context.Context, resolver Resolver, hostname string) (string, error) {
	if ip := net.ParseIP(hostname); ip != nil {
		addr, ok := netip.AddrFromSlice(ip)
		if !ok || !isPublicIP(addr) {
			return "", errors.New("webhook endpoint rejected")
		}
		return addr.String(), nil
	}
	addrs, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return "", errors.New("webhook DNS resolution failed")
	}
	if len(addrs) == 0 {
		return "", errors.New("webhook DNS resolution failed")
	}
	for _, addr := range addrs {
		ip, ok := netip.AddrFromSlice(addr.IP)
		if !ok || !isPublicIP(ip) {
			return "", errors.New("webhook endpoint resolves to non-public address")
		}
	}
	return addrs[0].IP.String(), nil
}
