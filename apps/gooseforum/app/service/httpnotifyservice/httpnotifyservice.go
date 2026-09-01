package httpnotifyservice

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	EventTopicPublished   = "topic.published"
	EventTopicUpdated     = "topic.updated"
	EventCommentCreated   = "comment.created"
	EventUserSignup       = "user.signup"
	EventReportCreated    = "moderation.report.created"
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

func Notify(eventName string, data any) {
	config := hotdataserve.GetHttpNotifyConfigCache()
	if !shouldNotify(config, eventName) {
		return
	}
	now := time.Now().Unix()
	body, err := json.Marshal(Envelope{
		Event:     eventName,
		Timestamp: now,
		Data:      data,
	})
	if err != nil {
		slog.Error("httpnotify: marshal payload failed", "event", eventName, "err", err)
		return
	}
	for _, endpoint := range config.Endpoints {
		if !endpointAccepts(endpoint, eventName) {
			continue
		}
		endpoint := endpoint
		go deliver(endpoint, eventName, now, body)
	}
}

func ShouldNotify(eventName string) bool {
	return shouldNotify(hotdataserve.GetHttpNotifyConfigCache(), eventName)
}

func shouldNotify(config pageConfig.HttpNotifyConfig, eventName string) bool {
	if !config.Enabled {
		return false
	}
	for _, endpoint := range config.Endpoints {
		if endpointAccepts(endpoint, eventName) {
			return true
		}
	}
	return false
}

func endpointAccepts(endpoint pageConfig.HttpNotifyEndpoint, eventName string) bool {
	if !endpoint.Enabled || endpoint.AbnormalTerminated || strings.TrimSpace(endpoint.URL) == "" {
		return false
	}
	return slices.Contains(endpoint.Events, eventName)
}

func deliver(endpoint pageConfig.HttpNotifyEndpoint, eventName string, timestamp int64, body []byte) {
	req, err := buildRequest(endpoint, eventName, deliveryID(), timestamp, body)
	if err != nil {
		slog.Error("httpnotify: build request failed", "endpoint", endpoint.Name, "event", eventName, "err", err)
		recordDeliveryResult(endpoint, false, err.Error())
		return
	}
	resp, err := sendRequest(req, endpointTimeout(endpoint))
	if err != nil {
		slog.Error("httpnotify: request failed", "endpoint", endpoint.Name, "event", eventName, "err", err)
		recordDeliveryResult(endpoint, false, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		slog.Warn("httpnotify: non-2xx response", "endpoint", endpoint.Name, "event", eventName, "status", resp.StatusCode)
		recordDeliveryResult(endpoint, false, resp.Status)
		return
	}
	recordDeliveryResult(endpoint, true, "")
}

func buildRequest(endpoint pageConfig.HttpNotifyEndpoint, eventName string, deliveryID string, timestamp int64, body []byte) (*http.Request, error) {
	targetURL, err := url.Parse(strings.TrimSpace(endpoint.URL))
	if err != nil {
		return nil, err
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported url scheme: %s", targetURL.Scheme)
	}
	req, err := http.NewRequest(http.MethodPost, targetURL.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("X-Goose-Event", eventName)
	req.Header.Set("X-Goose-Delivery", deliveryID)
	req.Header.Set("X-Goose-Timestamp", strconv.FormatInt(timestamp, 10))
	if endpoint.Secret != "" {
		req.Header.Set("X-Goose-Signature", sign(endpoint.Secret, timestamp, body))
	}
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
