// Package llmprovider 提供 OpenAI 兼容的 /chat/completions 客户端（B7，issue #181）。
// qwen（DashScope compatible-mode）、OpenRouter、本地 Ollama/llama.cpp 等端点协议同构，
// 只用一个 HTTP 实现，配置（base_url/api_key/model）驱动即可切换 provider，输出 schema 不变。
package llmprovider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

// DefaultTimeout 单次 LLM 调用超时；外部 HTTP 调用必须有时限，避免请求挂死。
const DefaultTimeout = 30 * time.Second

// Config 从 config.toml [ai_summary] 段读取的提供方配置（部署级，api_key 不进 DB）。
type Config struct {
	Provider    string
	BaseURL     string
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
}

// LoadConfig 读取 [ai_summary] 配置；未配置的字段用默认值（temperature 0.3）。
func LoadConfig() Config {
	return Config{
		Provider:    preferences.GetString("ai_summary.provider"),
		BaseURL:     strings.TrimRight(preferences.GetString("ai_summary.base_url"), "/"),
		APIKey:      preferences.GetString("ai_summary.api_key"),
		Model:       preferences.GetString("ai_summary.model"),
		Temperature: preferences.GetFloat64("ai_summary.temperature", 0.3),
		MaxTokens:   preferences.GetInt("ai_summary.max_tokens", 1024),
	}
}

// Enabled 是否具备调用条件（端点与模型必须配置；api_key 本地端点可空）。
func (c Config) Enabled() bool {
	return c.BaseURL != "" && c.Model != ""
}

// ErrNotConfigured 提供方未配置（base_url/model 为空）。
var ErrNotConfigured = errors.New("llm provider not configured")

// Message chat completion 消息。
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ResponseFormat 强制 JSON 输出（OpenAI compatible 标准字段；不支持的服务端会忽略或报错，
// 报错时走失败态，不影响课程页主流程）。
type ResponseFormat struct {
	Type string `json:"type"`
}

// ChatRequest OpenAI 兼容请求体。
type ChatRequest struct {
	Model          string          `json:"model"`
	Messages       []Message       `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      *int            `json:"max_tokens,omitempty"`
	ResponseFormat *ResponseFormat `json:"response_format,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Complete 调用 {base_url}/chat/completions 并返回助手消息正文。
// 请求字段缺省时用 Config 补全；返回的错误不携带响应原文（防泄漏）。
func (c Config) Complete(ctx context.Context, req ChatRequest) (string, error) {
	if !c.Enabled() {
		return "", ErrNotConfigured
	}
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.Temperature == nil {
		t := c.Temperature
		req.Temperature = &t
	}
	if req.MaxTokens == nil && c.MaxTokens > 0 {
		m := c.MaxTokens
		req.MaxTokens = &m
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("llm marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llm build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	client := &http.Client{Timeout: DefaultTimeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("llm read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm api status %d", resp.StatusCode)
	}
	var cc chatCompletionResponse
	if err := json.Unmarshal(raw, &cc); err != nil {
		return "", fmt.Errorf("llm decode response: %w", err)
	}
	if cc.Error != nil && cc.Error.Message != "" {
		return "", fmt.Errorf("llm api error: %s", cc.Error.Message)
	}
	if len(cc.Choices) == 0 || strings.TrimSpace(cc.Choices[0].Message.Content) == "" {
		return "", errors.New("llm empty response")
	}
	return cc.Choices[0].Message.Content, nil
}
