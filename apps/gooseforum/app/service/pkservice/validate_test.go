package pkservice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
)

// overrideValidateClientBuilder 将 ValidateCredential 使用的客户端构造器替换为指向
// 本地 httptest 服务的实现（关闭退避/睡眠加速测试），返回恢复函数。
func overrideValidateClientBuilder(t *testing.T, srv *httptest.Server) func() {
	t.Helper()
	orig := validateClientBuilder
	validateClientBuilder = func(a Audience) *onesystemClient {
		c := newOnesystemClientForAudience(a)
		c.baseURL = srv.URL
		c.maxAttempts = 1
		c.backoff = func(int) time.Duration { return 0 }
		c.sleep = func(time.Duration) {}
		return c
	}
	return func() { validateClientBuilder = orig }
}

func seedValidationCalendar(t *testing.T, audience Audience) {
	t.Helper()
	conn := db.Connect()
	if err := conn.Create(&pk.CalendarEntity{
		CalendarId:     pk.ScopeID(audience, 121),
		Audience:       string(audience),
		CalendarIdI18n: "2025-2026-1",
		SchemaVersion:  pk.PKDataSchemaVersion,
	}).Error; err != nil {
		t.Fatalf("seed validation calendar: %v", err)
	}
}

func TestValidateCredentialValid(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceUndergraduate)

	var probePageSize int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected probe request %s %s", r.Method, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("Cookie"), "JWTUser=abc") {
			t.Errorf("cookie missing: %q", r.Header.Get("Cookie"))
		}
		var payload struct {
			PageNum_  int `json:"pageNum_"`
			PageSize_ int `json:"pageSize_"`
			Condition struct {
				Calendar int `json:"calendar"`
			} `json:"condition"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode probe request: %v", err)
		}
		probePageSize = payload.PageSize_
		if payload.Condition.Calendar != 121 {
			t.Errorf("calendar = %d, want 121", payload.Condition.Calendar)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pageResponse(1, []CourseRaw{rawCourse(1, "A001")})))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceUndergraduate, "JWTUser=abc")
	if err != nil {
		t.Fatalf("ValidateCredential: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true (message: %q)", result.Message)
	}
	if result.Message != "" {
		t.Errorf("Message = %q, want empty", result.Message)
	}
	if probePageSize != 1 {
		t.Errorf("probe pageSize = %d, want 1 (minimal probe)", probePageSize)
	}
}

func TestValidateCredentialInvalidCredential(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceUndergraduate)

	// 401 且错误体回带会话凭证：必现脱敏，防止把凭证片段带回提示。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"未登录或会话失效","url":"/login?JWTUser=secret&sessionid=abc"}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceUndergraduate, "bad-cookie")
	if err != nil {
		t.Fatalf("credential failure must not be a hard error, got: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false on 401")
	}
	if !strings.Contains(result.Message, "401") {
		t.Errorf("message should contain HTTP status 401, got: %q", result.Message)
	}
	if strings.Contains(result.Message, "secret") || strings.Contains(result.Message, "abc") {
		t.Errorf("credential not redacted: %q", result.Message)
	}
	if !strings.Contains(result.Message, "JWTUser=***") {
		t.Errorf("expected redacted marker, got: %q", result.Message)
	}
}

func TestValidateCredentialBusinessFailureCode(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceUndergraduate)

	// HTTP 200 但一系统业务/鉴权失败信封（code!=0）：同样判定凭证不可用。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":1,"msg":"未登录或会话失效","data":null}`))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceUndergraduate, "bad-cookie")
	if err != nil {
		t.Fatalf("business failure must not be a hard error, got: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false on business failure code")
	}
	if !strings.Contains(result.Message, "code=1") {
		t.Errorf("message should mention code=1, got: %q", result.Message)
	}
}

func TestValidateCredentialNoSyncedCalendar(t *testing.T) {
	migratePkTables(t) // 无任何已同步学期。

	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		t.Error("no calendar: upstream must not be probed")
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceUndergraduate, "JWTUser=abc")
	if err != nil {
		t.Fatalf("no-calendar must not be a hard error: %v", err)
	}
	if result.Valid {
		t.Error("Valid = true, want false without a synced calendar")
	}
	if !strings.Contains(result.Message, "尚无已同步学期") {
		t.Errorf("message = %q, want 尚无已同步学期 hint", result.Message)
	}
	if called {
		t.Fatal("upstream was probed despite missing calendar")
	}
}

func TestValidateCredentialGraduatePath(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceGraduate)

	var xTokens []string
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/electionservice/student/round/allArrangementCourses" {
			t.Errorf("path = %q, want allArrangementCourses", r.URL.Path)
		}
		xTokens = append(xTokens, r.Header.Get("X-Token"))
		var payload struct {
			Condition struct {
				CalendarID    string `json:"calendarId"`
				TrainingLevel string `json:"trainingLevel"`
			} `json:"condition"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode graduate probe: %v", err)
		}
		if payload.Condition.CalendarID != "121" {
			t.Errorf("calendarId = %q, want 121", payload.Condition.CalendarID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pageResponse(1, []CourseRaw{rawCourse(1, "M001")})))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceGraduate, "graduate-token")
	if err != nil {
		t.Fatalf("graduate validation: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true (message: %q)", result.Message)
	}
	// 研究生走硕士/博士两层查询（X-Token 请求头），与同步客户端一致。
	if calls != 2 {
		t.Errorf("graduate probe calls = %d, want 2 (master + doctor)", calls)
	}
	for _, token := range xTokens {
		if token != "graduate-token" {
			t.Errorf("X-Token = %q, want graduate credential", token)
		}
	}
}

func TestValidateCredentialResolvesEnvCredential(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceUndergraduate)

	t.Setenv("ONESYSTEM_UNDERGRADUATE_COOKIE", "JWTUser=env-cookie")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Cookie"), "JWTUser=env-cookie") {
			t.Errorf("env cookie not used: %q", r.Header.Get("Cookie"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(pageResponse(0, nil)))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(overrideValidateClientBuilder(t, srv))

	result, err := ValidateCredential(context.Background(), AudienceUndergraduate, "  ")
	if err != nil {
		t.Fatalf("ValidateCredential with empty credential: %v", err)
	}
	if !result.Valid {
		t.Errorf("Valid = false, want true via env-resolved credential: %q", result.Message)
	}
}

func TestValidateCredentialRejectsInvalidAudience(t *testing.T) {
	if _, err := ValidateCredential(context.Background(), Audience("bogus"), "x"); err == nil {
		t.Fatal("expected error for invalid audience")
	}
}

func TestValidateCredentialMissingCredentialSource(t *testing.T) {
	migratePkTables(t)
	seedValidationCalendar(t, AudienceUndergraduate)
	t.Setenv("ONESYSTEM_UNDERGRADUATE_COOKIE", "")
	t.Setenv("ONESYSTEM_COOKIE", "")

	if _, err := ValidateCredential(context.Background(), AudienceUndergraduate, ""); err == nil {
		t.Fatal("expected error when no credential source is configured")
	}
}
