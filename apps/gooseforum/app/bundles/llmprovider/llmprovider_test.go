package llmprovider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestListModelsSuccess 验证 /models 正常返回模型列表。
func TestListModelsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("path = %q, want /models", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-4o","owned_by":"openai"},{"id":"Qwen/Qwen2.5-7B-Instruct","owned_by":"siliconflow"}]}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, APIKey: "test-key"}
	models, err := cfg.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 || models[0].ID != "gpt-4o" || models[1].OwnedBy != "siliconflow" {
		t.Fatalf("models = %#v, want 2 entries", models)
	}
}

// TestListModelsUnsupported 验证 404（未实现 /models 的服务）返回 ErrModelsUnsupported。
func TestListModelsUnsupported(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL}
	_, err := cfg.ListModels(context.Background())
	if !errors.Is(err, ErrModelsUnsupported) {
		t.Fatalf("err = %v, want ErrModelsUnsupported", err)
	}
}

// TestListModelsEmptyData 验证空 data 数组同样视为不支持自动获取。
func TestListModelsEmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL}
	_, err := cfg.ListModels(context.Background())
	if !errors.Is(err, ErrModelsUnsupported) {
		t.Fatalf("err = %v, want ErrModelsUnsupported", err)
	}
}

// TestListModelsNotConfigured 验证未配置端点时报 ErrNotConfigured，且不发请求。
func TestListModelsNotConfigured(t *testing.T) {
	cfg := Config{}
	_, err := cfg.ListModels(context.Background())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// TestListModelsErrorNotLeaked 验证错误不携带响应原文（防泄漏）。
func TestListModelsErrorNotLeaked(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"error":{"message":"secret-internal-detail"}}`))
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL}
	_, err := cfg.ListModels(context.Background())
	if err == nil {
		t.Fatal("err = nil, want error")
	}
	if strings.Contains(err.Error(), "secret-internal-detail") {
		t.Fatalf("err leaks response body: %v", err)
	}
}
