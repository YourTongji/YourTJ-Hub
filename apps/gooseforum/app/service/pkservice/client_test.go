package pkservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient 构造指向本地 httptest 服务的客户端，关闭退避/睡眠加速测试。
func newTestClient(t *testing.T, handler http.Handler) *onesystemClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := newOnesystemClient()
	c.baseURL = srv.URL
	c.maxAttempts = 3
	c.sleep = func(time.Duration) {}
	c.backoff = func(int) time.Duration { return 0 }
	return c
}

// pageResponse 构造一系统分页响应。
func pageResponse(total int, list []CourseRaw) string {
	b, _ := json.Marshal(map[string]any{
		"data": map[string]any{"total_": total, "list": list},
	})
	return string(b)
}

func rawCourse(id int, code string) CourseRaw {
	uid := uint64(id)
	credit := 5.0
	return CourseRaw{
		Id:          &uid,
		Code:        code,
		Name:        "课程" + code,
		CourseCode:  code[:3],
		CourseName:  "课程" + code,
		Credits:     &credit,
		TeacherList: []TeacherRaw{},
	}
}

func TestFetchPageParsesResponse(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.Contains(r.Header.Get("Cookie"), "JWTUser=abc") {
			t.Errorf("cookie missing: %q", r.Header.Get("Cookie"))
		}
		if r.Header.Get("Referer") != onesystemReferer {
			t.Errorf("referer = %q", r.Header.Get("Referer"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageResponse(250, []CourseRaw{rawCourse(1, "A001")})))
	}))

	page, err := c.fetchPage(context.Background(), "JWTUser=abc", 121, 1, 200)
	if err != nil {
		t.Fatalf("fetchPage: %v", err)
	}
	if page.Data.Total_ != 250 {
		t.Errorf("total = %d, want 250", page.Data.Total_)
	}
	if len(page.Data.List) != 1 {
		t.Errorf("list len = %d, want 1", len(page.Data.List))
	}
}

func TestFetchPageRetriesOnRetryableThenSucceeds(t *testing.T) {
	attempts := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("boom"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(pageResponse(1, nil)))
	}))
	if _, err := c.fetchPage(context.Background(), "cookie", 121, 1, 200); err != nil {
		t.Fatalf("fetchPage after retry: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2 (one retry)", attempts)
	}
}

func TestFetchPageNonRetryableStatusReportsHTTPCode(t *testing.T) {
	// AC2：cookie 失效等非可重试状态必须给出 HTTP 状态 + 提示，且不触发重试。
	calls := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"未登录"}`))
	}))
	_, err := c.fetchPage(context.Background(), "bad-cookie", 121, 1, 200)
	if err == nil {
		t.Fatal("expected error on 401")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should contain HTTP status 401, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("non-retryable status must not retry, calls = %d", calls)
	}
}

func TestFetchPageRetryExhaustion(t *testing.T) {
	calls := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream down"))
	}))
	_, err := c.fetchPage(context.Background(), "cookie", 121, 1, 200)
	if err == nil {
		t.Fatal("expected error after retry exhaustion")
	}
	if calls != c.maxAttempts {
		t.Errorf("calls = %d, want %d", calls, c.maxAttempts)
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error should mention status, got: %v", err)
	}
}

func TestRedactCredentials(t *testing.T) {
	// 上游错误响应若回带会话凭证，不得原样持久化到 fetchlog。
	got := redactCredentials(`{"message":"未登录","url":"/login?JWTUser=secret&sessionid=abc"}`)
	if strings.Contains(got, "secret") || strings.Contains(got, "abc") {
		t.Errorf("credentials not redacted: %q", got)
	}
	if !strings.Contains(got, "JWTUser=***") {
		t.Errorf("expected JWTUser redacted marker, got: %q", got)
	}
	// 普通错误提示应原样保留。
	if plain := redactCredentials(`{"message":"系统繁忙"}`); !strings.Contains(plain, "系统繁忙") {
		t.Errorf("benign hint altered: %q", plain)
	}
}

func TestFetchPageRejectsBusinessFailureCode(t *testing.T) {
	// HTTP 200 但业务/鉴权失败（code!=0）：不得当作成功空页，且不可重试（review HIGH）。
	calls := 0
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":1,"msg":"未登录或会话失效","data":null}`))
	}))
	_, err := c.fetchPage(context.Background(), "bad-cookie", 121, 1, 200)
	if err == nil {
		t.Fatal("expected error on HTTP 200 + code!=0")
	}
	if !strings.Contains(err.Error(), "code=1") {
		t.Errorf("error should mention code=1, got: %v", err)
	}
	if calls != 1 {
		t.Errorf("business failure must not retry, calls = %d", calls)
	}
}

func TestFetchPageCodeZeroIsSuccess(t *testing.T) {
	// code=0 或缺失时按成功页处理，不误判。
	for name, body := range map[string]string{
		"code-zero":  `{"code":0,"msg":"success","data":{"total_":1,"list":[]}}`,
		"no-code":    pageResponse(1, nil),
		"code-null":  `{"code":null,"data":{"total_":1,"list":[]}}`,
		"code-text0": `{"code":"0","data":{"total_":1,"list":[]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(body))
			}))
			if _, err := c.fetchPage(context.Background(), "cookie", 121, 1, 200); err != nil {
				t.Fatalf("code-zero/none should succeed: %v", err)
			}
		})
	}
}

func TestFetchPageRejectsStringBusinessCode(t *testing.T) {
	// 字符串形式的 code="1" 同样应拦截（勿因 JSON 字符串引号误判为成功）。
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"1","msg":"未登录","data":null}`))
	}))
	_, err := c.fetchPage(context.Background(), "bad-cookie", 121, 1, 200)
	if err == nil {
		t.Fatal("expected error on string code=\"1\"")
	}
	if !strings.Contains(err.Error(), "code=1") {
		t.Errorf("error should mention code=1, got: %v", err)
	}
}

func TestFetchPageCodeBoundary(t *testing.T) {
	// 负数 code 应拦截；非数字字符串 code 应 fail-closed（报错而非当成功）。
	negative := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":-1,"msg":"fail","data":null}`))
	}))
	if _, err := negative.fetchPage(context.Background(), "cookie", 121, 1, 200); err == nil {
		t.Fatal("expected error on negative code")
	}

	nonNumeric := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":"success","msg":"ok","data":{"total_":1,"list":[]}}`))
	}))
	if _, err := nonNumeric.fetchPage(context.Background(), "cookie", 121, 1, 200); err == nil {
		t.Fatal("expected error on non-numeric code (fail-closed)")
	}
}
