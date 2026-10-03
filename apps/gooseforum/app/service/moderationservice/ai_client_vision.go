package moderationservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// 视觉证据客户端（issue #975）：OpenAI-compatible /chat/completions 多模态请求。
// 不复用 llmprovider.Complete——其 Message.Content 只支持纯文本。错误只携带分类，
// 绝不回带 provider 响应原文（防止泄漏用户内容或策略细节到日志）。

// visionMaxImageBytes 送视觉模型的原图上限（base64 后约 11 MiB）；超限转人工审核。
const visionMaxImageBytes = 8 << 20

// visionMaxTokens 输出上限。推理型视觉模型（如 ling-3.0-flash-vl）的 reasoning
// 也计入该上限；过小会在 finish_reason=length 处截断 JSON 并转人工审核。
const visionMaxTokens = 4096

// visionMaxResponseBytes provider 响应体读取上限。
const visionMaxResponseBytes = 1 << 20

// visionFailure 视觉调用失败分类，落库为 ImageRecord.Status。
type visionFailure struct {
	Kind       string // refused | invalid | uncertain | error | timeout | auth | http_<status>
	HTTPStatus int    // provider HTTP 状态码（无响应时为 0）
}

func (f visionFailure) Error() string { return "vision " + f.Kind }

type visionResult struct {
	Evidence ImageEvidence
	Cost     float64
}

type visionChatRequest struct {
	Model       string              `json:"model"`
	Messages    []visionChatMessage `json:"messages"`
	Temperature float64             `json:"temperature"`
	MaxTokens   int                 `json:"max_tokens"`
}

type visionChatMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type visionContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *visionImageURL `json:"image_url,omitempty"`
}

type visionImageURL struct {
	URL string `json:"url"`
}

type visionChatResponse struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content any    `json:"content"`
			Refusal string `json:"refusal"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		Cost float64 `json:"cost"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// visionHTTPClient 可在测试中替换。
var visionHTTPClient = &http.Client{}

// callVisionEvidence 把单张已校验图片送视觉模型并严格解析证据 JSON。
func callVisionEvidence(ctx context.Context, baseURL, apiKey, model string, timeout time.Duration, mime string, data []byte) (visionResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	body, err := json.Marshal(visionChatRequest{
		Model:       model,
		Temperature: 0,
		MaxTokens:   visionMaxTokens,
		Messages: []visionChatMessage{
			{Role: "system", Content: visionEvidencePrompt},
			{Role: "user", Content: []visionContentPart{
				{Type: "text", Text: "Extract the evidence JSON for this image."},
				{Type: "image_url", ImageURL: &visionImageURL{URL: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)}},
			}},
		},
	})
	if err != nil {
		return visionResult{}, visionFailure{Kind: "error"}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return visionResult{}, visionFailure{Kind: "error"}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := visionHTTPClient.Do(httpReq)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return visionResult{}, visionFailure{Kind: "timeout"}
		}
		return visionResult{}, visionFailure{Kind: "error"}
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, visionMaxResponseBytes))
	if err != nil {
		return visionResult{}, visionFailure{Kind: "error"}
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return visionResult{}, visionFailure{Kind: "auth", HTTPStatus: resp.StatusCode}
	case resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity:
		// 多数 provider 以 400/422 拒绝其安全策略不允许的图片。
		return visionResult{}, visionFailure{Kind: "refused", HTTPStatus: resp.StatusCode}
	case resp.StatusCode != http.StatusOK:
		// 其余状态带上状态码（如 OpenRouter 对不支持图片输入的模型返回 404），便于排查。
		return visionResult{}, visionFailure{Kind: fmt.Sprintf("http_%d", resp.StatusCode), HTTPStatus: resp.StatusCode}
	}
	var decoded visionChatResponse
	if err := json.Unmarshal(raw, &decoded); err != nil || decoded.Error != nil || len(decoded.Choices) == 0 {
		return visionResult{}, visionFailure{Kind: "invalid"}
	}
	choice := decoded.Choices[0]
	if strings.TrimSpace(choice.Message.Refusal) != "" || choice.FinishReason == "content_filter" {
		return visionResult{Cost: decoded.Usage.Cost}, visionFailure{Kind: "refused"}
	}
	if choice.FinishReason == "length" {
		return visionResult{Cost: decoded.Usage.Cost}, visionFailure{Kind: "invalid"}
	}
	evidence, err := parseImageEvidence(messageText(choice.Message.Content))
	switch {
	case errors.Is(err, errEvidenceRefused):
		return visionResult{Cost: decoded.Usage.Cost}, visionFailure{Kind: "refused"}
	case errors.Is(err, errEvidenceUnsure):
		return visionResult{Cost: decoded.Usage.Cost}, visionFailure{Kind: "uncertain"}
	case err != nil:
		return visionResult{Cost: decoded.Usage.Cost}, visionFailure{Kind: "invalid"}
	}
	return visionResult{Evidence: evidence, Cost: decoded.Usage.Cost}, nil
}

// messageText 兼容 content 为字符串或 [{type:text,text}] 数组两种返回形状。
func messageText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var builder strings.Builder
		for _, part := range value {
			if item, ok := part.(map[string]any); ok {
				if text, ok := item["text"].(string); ok {
					builder.WriteString(text)
				}
			}
		}
		return builder.String()
	default:
		return fmt.Sprint(value)
	}
}
