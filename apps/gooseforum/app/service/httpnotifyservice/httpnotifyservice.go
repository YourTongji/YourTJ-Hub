package httpnotifyservice

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jsonopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

const (
	EventTopicPublished = "topic.published"
	EventTopicUpdated   = "topic.updated"
	EventCommentCreated = "comment.created"
	EventUserSignup     = "user.signup"
	// EventReportCreated “全部举报”的聚合兼容事件：保持既有 payload 形状，
	// 覆盖 topic/post/chat_message/course_review 四类举报（issue #1049）。
	EventReportCreated = "moderation.report.created"

	// 版主审批事件（issue #1049）：payload 为统一的审批安全摘要（ApprovalPayload）。
	EventReviewTopicRequested      = "moderation.review.topic.requested"
	EventReviewPostRequested       = "moderation.review.post.requested"
	EventReportTopicCreated        = "moderation.report.topic.created"
	EventReportPostCreated         = "moderation.report.post.created"
	EventReportChatMessageCreated  = "moderation.report.chat_message.created"
	EventReportCourseReviewCreated = "moderation.report.course_review.created"
)

const (
	defaultTimeoutSeconds = 2
	maxTimeoutSeconds     = 15
	contentTypeJSON       = "application/json"
	disableAfterFailures  = 3
)

var updateMu sync.Mutex

var sendRequest = func(req *http.Request, timeout time.Duration) (*http.Response, error) {
	return (&http.Client{Timeout: timeout}).Do(req)
}

type Envelope struct {
	Event     string `json:"event"`
	Timestamp int64  `json:"timestamp"`
	Data      any    `json:"data"`
}

// Alternative 同一领域事件的一种订阅名及其 generic 负载。
type Alternative struct {
	Event string
	Data  any
}

// Message 一次领域事件的投递（issue #1049）。
//
// Alternatives 按优先级排列同一事件的订阅名（具体事件在前、聚合兼容事件在后）：
// 每个 Endpoint 只按它订阅的首个事件名投递一次，同时订阅“全部举报”与具体举报
// 类型的 Endpoint 不会收到两份。Approval 非空表示待人工审批事件，飞书等卡片通道
// 据此渲染；为空时卡片通道跳过。DedupeKey 非空时，同一 Endpoint 在 dedupeTTL 内对
// 同一审批只投递一次（重复发布、相同版本重提不重复提醒）。
type Message struct {
	Alternatives []Alternative
	Approval     *ApprovalPayload
	DedupeKey    string
}

// Notify 投递单一事件名的通知（既有事件的兼容入口）。
func Notify(eventName string, data any) {
	Publish(Message{Alternatives: []Alternative{{Event: eventName, Data: data}}})
}

// Publish 异步投递：每个 Endpoint 独立 goroutine 发送，单一通道失败或超时不影响
// 其他通道，也不阻塞发帖/举报主流程（best-effort）。
func Publish(msg Message) {
	config := hotdataserve.GetHttpNotifyConfigCache()
	if !config.Enabled {
		return
	}
	now := time.Now()
	for _, endpoint := range config.Endpoints {
		alt, ok := selectAlternative(endpoint, msg)
		if !ok {
			continue
		}
		dedupeKey := ""
		if msg.DedupeKey != "" {
			dedupeKey = endpointKey(endpoint) + "\x00" + msg.DedupeKey
			if !recentDeliveries.claim(dedupeKey, now) {
				continue
			}
		}
		// 失败（编码、建请求、发送或对方拒收）时归还去重名额：登记不等于已送达。
		releaseClaim := func() {
			if dedupeKey != "" {
				recentDeliveries.release(dedupeKey, now)
			}
		}
		channel := channelFor(endpoint)
		body, err := channel.encode(endpoint, alt, msg.Approval, now)
		if err != nil {
			slog.Error("httpnotify: encode payload failed", "endpoint", endpoint.Name, "event", alt.Event, "err", err)
			releaseClaim()
			continue
		}
		go func() {
			if !deliver(endpoint, channel, alt.Event, now.Unix(), body) {
				releaseClaim()
			}
		}()
	}
}

// ShouldNotify 任一事件名存在可投递的 Endpoint 时返回 true（调用方据此跳过负载构建）。
func ShouldNotify(eventNames ...string) bool {
	return shouldNotify(hotdataserve.GetHttpNotifyConfigCache(), eventNames...)
}

func shouldNotify(config pageConfig.HttpNotifyConfig, eventNames ...string) bool {
	if !config.Enabled {
		return false
	}
	for _, endpoint := range config.Endpoints {
		for _, eventName := range eventNames {
			if endpointAccepts(endpoint, eventName) {
				return true
			}
		}
	}
	return false
}

func endpointAccepts(endpoint pageConfig.HttpNotifyEndpoint, eventName string) bool {
	if !endpoint.Enabled || endpoint.AbnormalTerminated || strings.TrimSpace(endpoint.URL) == "" {
		return false
	}
	return channelFor(endpoint).supportsEvent(eventName) && slices.Contains(endpoint.Events, eventName)
}

func selectAlternative(endpoint pageConfig.HttpNotifyEndpoint, msg Message) (Alternative, bool) {
	if channelFor(endpoint).requiresApproval() && msg.Approval == nil {
		return Alternative{}, false
	}
	for _, alt := range msg.Alternatives {
		if endpointAccepts(endpoint, alt.Event) {
			return alt, true
		}
	}
	return Alternative{}, false
}

func endpointKey(endpoint pageConfig.HttpNotifyEndpoint) string {
	if endpoint.Id != "" {
		return "id:" + endpoint.Id
	}
	return "url:" + endpoint.URL
}

// deliver 发送一次投递并记录结果，返回是否成功。
func deliver(endpoint pageConfig.HttpNotifyEndpoint, channel notifyChannel, eventName string, timestamp int64, body []byte) bool {
	req, err := channel.buildRequest(endpoint, eventName, deliveryID(), timestamp, body)
	if err != nil {
		message := deliveryErrorMessage(err)
		slog.Error("httpnotify: build request failed", "endpoint", endpoint.Name, "event", eventName, "err", message)
		recordDeliveryResult(endpoint, false, message)
		return false
	}
	resp, err := sendRequest(req, endpointTimeout(endpoint))
	if err != nil {
		message := deliveryErrorMessage(err)
		slog.Error("httpnotify: request failed", "endpoint", endpoint.Name, "event", eventName, "err", message)
		recordDeliveryResult(endpoint, false, message)
		return false
	}
	defer resp.Body.Close()
	if err := channel.checkResponse(resp); err != nil {
		message := deliveryErrorMessage(err)
		slog.Warn("httpnotify: delivery rejected", "endpoint", endpoint.Name, "event", eventName, "status", resp.StatusCode, "err", message)
		recordDeliveryResult(endpoint, false, message)
		return false
	}
	recordDeliveryResult(endpoint, true, "")
	return true
}

// maxDeliveryErrorRunes lastError 的最大长度（管理端回显与日志）。
const maxDeliveryErrorRunes = 200

// deliveryErrorMessage 生成可落库/打日志的失败摘要：*url.Error 的字符串会带完整
// 请求 URL，而飞书 webhook URL 本身就是凭据（hook token），因此只保留其内层错误。
func deliveryErrorMessage(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	message := strings.TrimSpace(err.Error())
	if runes := []rune(message); len(runes) > maxDeliveryErrorRunes {
		message = string(runes[:maxDeliveryErrorRunes])
	}
	return message
}

// buildRequest 生成 generic 通道请求：X-Goose-* 头与 HMAC 签名保持既有契约不变。
func buildRequest(endpoint pageConfig.HttpNotifyEndpoint, eventName string, deliveryID string, timestamp int64, body []byte) (*http.Request, error) {
	req, err := newJSONPost(endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Goose-Event", eventName)
	req.Header.Set("X-Goose-Delivery", deliveryID)
	req.Header.Set("X-Goose-Timestamp", strconv.FormatInt(timestamp, 10))
	if endpoint.Secret != "" {
		req.Header.Set("X-Goose-Signature", sign(endpoint.Secret, timestamp, body))
	}
	return req, nil
}

// newJSONPost 校验端点 URL（仅 http/https，且须带主机名）并构造 JSON POST 请求。
func newJSONPost(endpoint pageConfig.HttpNotifyEndpoint, body []byte) (*http.Request, error) {
	targetURL, err := url.Parse(strings.TrimSpace(endpoint.URL))
	if err != nil {
		return nil, err
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme: %s", targetURL.Scheme)
	}
	// http:///send、http://:9966/send 等写法能通过 url.Parse，但没有主机名，
	// net/http 只会报晦涩的 "no Host in request URL"。
	if targetURL.Hostname() == "" {
		return nil, errors.New("url is missing a host (expected http://host:port/path)")
	}
	req, err := http.NewRequest(http.MethodPost, targetURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	return req, nil
}

func sign(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(timestamp, 10)))
	mac.Write([]byte("."))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func endpointTimeout(endpoint pageConfig.HttpNotifyEndpoint) time.Duration {
	seconds := endpoint.TimeoutSeconds
	if seconds <= 0 {
		seconds = defaultTimeoutSeconds
	}
	if seconds > maxTimeoutSeconds {
		seconds = maxTimeoutSeconds
	}
	return time.Duration(seconds) * time.Second
}

func deliveryID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func recordDeliveryResult(endpoint pageConfig.HttpNotifyEndpoint, success bool, message string) {
	updateMu.Lock()
	defer updateMu.Unlock()

	entity := pageConfig.GetByPageType(pageConfig.HttpNotify)
	// 落库形状读改写：端点 secret 为密文（或 v25 迁移前的存量明文），
	// 原样保留，绝不用领域形状（json:"-" 不含密钥）覆盖（issue #324 S1）。
	storage := pageConfig.GetConfigByPageType(pageConfig.HttpNotify, pageConfig.HttpNotifyStorageConfig{Endpoints: []pageConfig.HttpNotifyStorageEndpoint{}})
	config := storage.ToConfig()
	config, changed := applyDeliveryResult(config, endpoint.Id, endpoint.URL, success, message)
	if !changed {
		return
	}
	entity.PageType = pageConfig.HttpNotify
	entity.Config = jsonopt.Encode(mergeDeliveryState(storage, config))
	pageConfig.CreateOrSave(&entity)
	hotdataserve.ClearHttpNotifyConfigCache()
}

// mergeDeliveryState 把 applyDeliveryResult 的投递状态变更合并回落库形状，
// 按 id（无 id 按 url）匹配端点，仅更新失败计数/错误/启用/熔断字段，
// 保留各端点密文/存量明文 secret 字段原样。
func mergeDeliveryState(storage pageConfig.HttpNotifyStorageConfig, applied pageConfig.HttpNotifyConfig) pageConfig.HttpNotifyStorageConfig {
	byKey := make(map[string]*pageConfig.HttpNotifyStorageEndpoint, len(storage.Endpoints))
	for i := range storage.Endpoints {
		key := storage.Endpoints[i].Id
		if key == "" {
			key = storage.Endpoints[i].URL
		}
		byKey[key] = &storage.Endpoints[i]
	}
	for _, ep := range applied.Endpoints {
		key := ep.Id
		if key == "" {
			key = ep.URL
		}
		if target := byKey[key]; target != nil {
			target.FailureCount = ep.FailureCount
			target.LastError = ep.LastError
			target.Enabled = ep.Enabled
			target.AbnormalTerminated = ep.AbnormalTerminated
		}
	}
	return storage
}

func applyDeliveryResult(config pageConfig.HttpNotifyConfig, endpointId string, endpointURL string, success bool, message string) (pageConfig.HttpNotifyConfig, bool) {
	for i := range config.Endpoints {
		endpoint := &config.Endpoints[i]
		if endpoint.Id != "" && endpointId != "" {
			if endpoint.Id != endpointId {
				continue
			}
		} else if endpoint.URL != endpointURL {
			continue
		}
		if success {
			if endpoint.FailureCount == 0 && endpoint.LastError == "" && !endpoint.AbnormalTerminated {
				return config, false
			}
			endpoint.FailureCount = 0
			endpoint.LastError = ""
			endpoint.AbnormalTerminated = false
			return config, true
		}
		endpoint.FailureCount++
		endpoint.LastError = message
		if endpoint.FailureCount >= disableAfterFailures {
			endpoint.Enabled = false
			endpoint.AbnormalTerminated = true
		}
		return config, true
	}
	return config, false
}
