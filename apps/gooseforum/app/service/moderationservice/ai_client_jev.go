package moderationservice

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"
)

// Jev Decisions API 客户端（issue #975）。TypeSafe POST /v1/systemone 与
// OpenRouter POST /api/alpha/decisions 请求/响应同构：
// {model, state, questions} → {model, answers{name:{type, noul|score…}}, usage}。
// 协议与 OpenAI /chat/completions 不同，故独立实现，不塞进 llmprovider。

const jevMaxResponseBytes = 1 << 20

// jevFailure Jev 调用失败分类。retryable 仅对 402/429/5xx/超时/网络错误成立；
// 400/401/403 是配置错误、2xx 但 schema 无效不换着“刷答案”，都不重试。
type jevFailure struct {
	Kind       string // timeout | rate_limited | upstream | config | malformed | error
	HTTPStatus int    // provider HTTP 状态码（无响应时为 0）
	retryable  bool
}

func (f jevFailure) Error() string { return "jev " + f.Kind }

type jevQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type jevRequest struct {
	Model     string                 `json:"model"`
	State     any                    `json:"state"`
	Questions map[string]jevQuestion `json:"questions"`
}

type jevAnswer struct {
	Type  string   `json:"type"`
	Noul  *float64 `json:"noul"`
	Score *float64 `json:"score"`
}

type jevResponse struct {
	Model    string               `json:"model"`
	Provider string               `json:"provider"`
	Answers  map[string]jevAnswer `json:"answers"`
	Usage    struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
}

// jevHTTPClient 可在测试中替换。
var jevHTTPClient = &http.Client{}

// callJev 调用 Decisions API；retries 次（≤1）仅用于可重试失败。
func callJev(ctx context.Context, endpoint, apiKey string, timeout time.Duration, retries int, request jevRequest) (jevResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return jevResponse{}, jevFailure{Kind: "error"}
	}
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		response, err := callJevOnce(ctx, endpoint, apiKey, timeout, body)
		if err == nil {
			return response, nil
		}
		lastErr = err
		var failure jevFailure
		if !errors.As(err, &failure) || !failure.retryable || ctx.Err() != nil {
			break
		}
	}
	return jevResponse{}, lastErr
}

func callJevOnce(parent context.Context, endpoint, apiKey string, timeout time.Duration, body []byte) (jevResponse, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return jevResponse{}, jevFailure{Kind: "config"}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := jevHTTPClient.Do(httpReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return jevResponse{}, jevFailure{Kind: "timeout", retryable: true}
		}
		return jevResponse{}, jevFailure{Kind: "error", retryable: true}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, jevMaxResponseBytes))
	if err != nil {
		return jevResponse{}, jevFailure{Kind: "error", retryable: true}
	}
	switch {
	case resp.StatusCode == http.StatusPaymentRequired || resp.StatusCode == http.StatusTooManyRequests:
		return jevResponse{}, jevFailure{Kind: "rate_limited", HTTPStatus: resp.StatusCode, retryable: true}
	case resp.StatusCode >= 500:
		return jevResponse{}, jevFailure{Kind: "upstream", HTTPStatus: resp.StatusCode, retryable: true}
	case resp.StatusCode != http.StatusOK:
		return jevResponse{}, jevFailure{Kind: "config", HTTPStatus: resp.StatusCode}
	}
	var decoded jevResponse
	if err := json.Unmarshal(raw, &decoded); err != nil || len(decoded.Answers) == 0 {
		return jevResponse{}, jevFailure{Kind: "malformed"}
	}
	return decoded, nil
}
