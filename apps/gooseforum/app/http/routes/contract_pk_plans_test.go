package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	pkcontroller "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖排课方案云端同步端点（issue #537）的契约测试：
// GET/PUT/DELETE /api/pk/plans。中间件链与 route4api.go 生产注册一致
// （JWTAuthCheck + 写操作 CheckWritableAccount + pk.plans 限流），业务响应为
// PK 信封 {code,msg,data}；401/403 由认证/可写中间件产生 forum 信封
// （auth-required / account-frozen fixtures）。
// updatedAt 是服务端动态时钟；PK 信封夹具走 assertPkPlansFixture（code/msg
// 对齐，data 按用例语义断言），401/403 的 forum 信封夹具仍走共享
// assertFixtureEnvelope。

const pkPlansPutBody = `{"plans":[{"id":"plan_a","name":"方案 1","createdAt":1725000000000,"stagedCourses":[{"courseCode":"122004","courseName":"数据结构","courseNameReserved":"","credit":4,"courseType":"必","courseNature":["专业必修"],"teacher":[{"teacherName":"张伟","teacherCode":"T001"}],"status":2,"courseDetail":[]}],"selectedCourses":["122004.01"],"customEvents":[{"id":"evt_1","label":"有事","day":6,"sections":[1,2],"weeks":[1,3,5]}]}],"activePlanId":"plan_a","majorSelected":{"calendarId":121,"grade":2024,"major":"080601","majorName":"计算机科学与技术"},"weekView":{"week":5,"useCurrent":true}}`

// pkPlansEnvelope 解码 PK 统一信封（data 保留原始 JSON 供结构断言）。
type pkPlansEnvelope struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func decodePkPlansEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) pkPlansEnvelope {
	t.Helper()
	var envelope pkPlansEnvelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode pk envelope %q: %v", recorder.Body.String(), err)
	}
	return envelope
}

// pkPlansFixture PK 信封形状的静态夹具（{code,msg,data}）。PK 域业务响应
// 与共享 helper 断言的 forum 信封（{result,code,messageCode,params}）形状
// 不同，不能走 contractFixture/assertFixtureEnvelope 路径（其对空 result
// unmarshal 必然失败）。
type pkPlansFixture struct {
	Code int             `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

func pkPlansFixtureOf(t *testing.T, filename string) pkPlansFixture {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract fixture path")
	}
	root := filepath.Join(filepath.Dir(testFile), "..", "..", "..", "..", "..")
	contents, err := os.ReadFile(filepath.Join(root, "packages", "api-contract", "fixtures", filename))
	if err != nil {
		t.Fatalf("read fixture %s: %v", filename, err)
	}
	var fixture pkPlansFixture
	if err := json.Unmarshal(contents, &fixture); err != nil {
		t.Fatalf("decode fixture %s: %v", filename, err)
	}
	return fixture
}

// assertPkPlansFixture 对齐 PK 信封 code/msg；data 语义由调用方按用例断言
// （null 字面 / updatedAt 结构 / msg contains）。
func assertPkPlansFixture(t *testing.T, actual pkPlansEnvelope, fixture pkPlansFixture) {
	t.Helper()
	if actual.Code != fixture.Code {
		t.Fatalf("pk envelope code = %d, want fixture code %d", actual.Code, fixture.Code)
	}
	if actual.Msg != fixture.Msg {
		t.Fatalf("pk envelope msg = %q, want fixture msg %q", actual.Msg, fixture.Msg)
	}
}

func setupPkPlansContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&pk.ScheduleSnapshotEntity{}); err != nil {
		t.Fatalf("migrate pk schedule snapshot: %v", err)
	}
	cleanup := func() {
		if err := conn.Where("1 = 1").Delete(&pk.ScheduleSnapshotEntity{}).Error; err != nil {
			t.Errorf("cleanup pk schedule snapshot: %v", err)
		}
	}
	cleanup()
	t.Cleanup(cleanup)

	// 中间件链与 route4api.go 生产注册一致（pkApi 组尾三行；CSRFProtection
	// 前置于认证，issue #406 契约，review blocker 修复后同链）。
	pkApi := router.Group("/api/pk")
	pkLoginApi := pkApi.Group("", middleware.CSRFProtection, middleware.JWTAuthCheck)
	pkLoginApi.GET("plans", middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthNoReq(pkcontroller.GetPlans))
	pkLoginApi.PUT("plans", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthJsonReq(pkcontroller.PutPlans))
	pkLoginApi.DELETE("plans", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitPkPlans), pkAuthNoReq(pkcontroller.DeletePlans))
	return conn, router
}

// pkPlansBodyWithNPlans 构造含 n 套合法方案（id/name 齐全）的 PUT 请求体。
func pkPlansBodyWithNPlans(n int) string {
	plans := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		plans = append(plans, fmt.Sprintf(
			`{"id":"plan_%d","name":"方案 %d","createdAt":1725000000000,"stagedCourses":[],"selectedCourses":[],"customEvents":[]}`, i, i))
	}
	return fmt.Sprintf(`{"plans":[%s],"activePlanId":"plan_1","majorSelected":{},"weekView":{}}`, strings.Join(plans, ","))
}

// review: 云端为空时 GET 返回字面 null（前端据此判定首登自动上传）。
func TestPkPlansGetEmptyHTTPContract(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plans", "", contractSessionToken(t, user))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertPkPlansFixture(t, decodePkPlansEnvelope(t, recorder), pkPlansFixtureOf(t, "pk-plans-get-empty.json"))
	if got := string(decodePkPlansEnvelope(t, recorder).Data); got != "null" {
		t.Fatalf("empty cloud data = %s, want literal null", got)
	}
}

// review: PUT→GET→DELETE 全链路——快照四字段逐字段回读、updatedAt 与 PUT
// 响应一致（同一行服务端时钟）、DELETE 后回到 data:null。
func TestPkPlansPutGetDeleteRoundtripHTTPContract(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/pk/plans", pkPlansPutBody, token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("put status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	putEnvelope := decodePkPlansEnvelope(t, recorder)
	if putEnvelope.Code != 0 {
		t.Fatalf("put code = %d, want 0: %s", putEnvelope.Code, recorder.Body.String())
	}
	var putData struct {
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.Unmarshal(putEnvelope.Data, &putData); err != nil || putData.UpdatedAt == "" {
		t.Fatalf("put data = %s, want non-empty updatedAt (err=%v)", putEnvelope.Data, err)
	}

	// GET 回读。
	recorder = serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plans", "", token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	var getData struct {
		Plans []struct {
			Id            string `json:"id"`
			StagedCourses []struct {
				CourseCode string `json:"courseCode"`
				Status     int    `json:"status"`
			} `json:"stagedCourses"`
			SelectedCourses []string `json:"selectedCourses"`
			CustomEvents    []struct {
				Label string `json:"label"`
			} `json:"customEvents"`
		} `json:"plans"`
		ActivePlanId  string `json:"activePlanId"`
		MajorSelected struct {
			Major *string `json:"major"`
		} `json:"majorSelected"`
		WeekView struct {
			Week *int `json:"week"`
		} `json:"weekView"`
		UpdatedAt string `json:"updatedAt"`
	}
	getEnvelope := decodePkPlansEnvelope(t, recorder)
	if err := json.Unmarshal(getEnvelope.Data, &getData); err != nil {
		t.Fatalf("decode get data %s: %v", getEnvelope.Data, err)
	}
	if len(getData.Plans) != 1 || getData.Plans[0].Id != "plan_a" || len(getData.Plans[0].StagedCourses) != 1 ||
		getData.Plans[0].StagedCourses[0].CourseCode != "122004" || getData.Plans[0].StagedCourses[0].Status != 2 ||
		len(getData.Plans[0].SelectedCourses) != 1 || getData.Plans[0].SelectedCourses[0] != "122004.01" ||
		len(getData.Plans[0].CustomEvents) != 1 || getData.Plans[0].CustomEvents[0].Label != "有事" {
		t.Fatalf("roundtrip plans mismatch: %s", getEnvelope.Data)
	}
	if getData.ActivePlanId != "plan_a" || getData.MajorSelected.Major == nil || *getData.MajorSelected.Major != "080601" ||
		getData.WeekView.Week == nil || *getData.WeekView.Week != 5 {
		t.Fatalf("roundtrip scalar fields mismatch: %s", getEnvelope.Data)
	}
	if getData.UpdatedAt != putData.UpdatedAt {
		t.Fatalf("get updatedAt %q != put updatedAt %q", getData.UpdatedAt, putData.UpdatedAt)
	}

	// DELETE：清除后回到 data:null。
	recorder = serveAuthSecurityJSON(router, http.MethodDelete, "/api/pk/plans", "", token)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertPkPlansFixture(t, decodePkPlansEnvelope(t, recorder), pkPlansFixtureOf(t, "pk-plans-delete-success.json"))
	recorder = serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plans", "", token)
	if got := string(decodePkPlansEnvelope(t, recorder).Data); got != "null" {
		t.Fatalf("data after delete = %s, want literal null", got)
	}
}

// review: 快照按 user_id 锚定——A 上传后 B 的 GET 必须仍是 null（绝不串户）。
func TestPkPlansPerUserIsolationHTTPContract(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	userA := createHTTPContractUser(t, conn, contractTestID())
	userB := createHTTPContractUser(t, conn, contractTestID())
	tokenA := contractSessionToken(t, userA)
	tokenB := contractSessionToken(t, userB)

	recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/pk/plans", pkPlansPutBody, tokenA)
	if recorder.Code != http.StatusOK {
		t.Fatalf("put status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	recorder = serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plans", "", tokenB)
	if got := string(decodePkPlansEnvelope(t, recorder).Data); got != "null" {
		t.Fatalf("user B data = %s, want literal null (cross-user leak)", got)
	}
}

func TestPkPlansValidationFailuresHTTPContract(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)

	t.Run("dangling activePlanId returns 400", func(t *testing.T) {
		body := strings.Replace(pkPlansBodyWithNPlans(2), `"activePlanId":"plan_1"`, `"activePlanId":"plan_missing"`, 1)
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/pk/plans", body, token)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertPkPlansFixture(t, decodePkPlansEnvelope(t, recorder), pkPlansFixtureOf(t, "pk-plans-bad-request.json"))
		if env := decodePkPlansEnvelope(t, recorder); !strings.Contains(env.Msg, "activePlanId") {
			t.Fatalf("msg = %q, want dangling-activePlanId explanation", env.Msg)
		}
	})

	t.Run("over-limit plans returns 400", func(t *testing.T) {
		recorder := serveAuthSecurityJSON(router, http.MethodPut, "/api/pk/plans", pkPlansBodyWithNPlans(11), token)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", recorder.Code, recorder.Body.String())
		}
		assertPkPlansFixture(t, decodePkPlansEnvelope(t, recorder), pkPlansFixtureOf(t, "pk-plans-too-many.json"))
		if env := decodePkPlansEnvelope(t, recorder); !strings.Contains(env.Msg, "上限") {
			t.Fatalf("msg = %q, want plan-count limit explanation", env.Msg)
		}
	})
}

// review: guard 场景——未登录三方法 401；冻结账号 PUT/DELETE 403 但 GET
// 放行（对齐 myContentList「冻结可读」先例）。
func TestPkPlansGuardsHTTPContract(t *testing.T) {
	t.Run("unauthenticated get/put/delete return 401", func(t *testing.T) {
		_, router := setupPkPlansContractTest(t)
		for _, tc := range []struct{ method, body string }{
			{http.MethodGet, ""},
			{http.MethodPut, pkPlansPutBody},
			{http.MethodDelete, ""},
		} {
			recorder := serveAuthSecurityJSON(router, tc.method, "/api/pk/plans", tc.body, "")
			if recorder.Code != http.StatusUnauthorized {
				t.Fatalf("%s unauthenticated status = %d, want 401: %s", tc.method, recorder.Code, recorder.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
		}
	})

	t.Run("frozen account put/delete return 403 but get succeeds", func(t *testing.T) {
		conn, router := setupPkPlansContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		token := contractSessionToken(t, user)

		for _, tc := range []struct{ method, body string }{
			{http.MethodPut, pkPlansPutBody},
			{http.MethodDelete, ""},
		} {
			recorder := serveAuthSecurityJSON(router, tc.method, "/api/pk/plans", tc.body, token)
			if recorder.Code != http.StatusForbidden {
				t.Fatalf("%s frozen status = %d, want 403: %s", tc.method, recorder.Code, recorder.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
		}

		recorder := serveAuthSecurityJSON(router, http.MethodGet, "/api/pk/plans", "", token)
		if recorder.Code != http.StatusOK {
			t.Fatalf("frozen get status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		if got := string(decodePkPlansEnvelope(t, recorder).Data); got != "null" {
			t.Fatalf("frozen get data = %s, want literal null", got)
		}
	})
}

// review: CSRF 门（issue #406 契约，#557 review blocker 修复证据）——
// 带 access_token cookie 的跨站 PUT 在认证之前被 403 拒绝（auth.csrf.rejected，
// 不触发 JWT 续期/会话延长）；同源 cookie PUT 正常通过全链（契约声明的
// accessTokenCookie 一等认证路径）。Bearer 客户端对 CSRF 自豁免（既有
// 全部用例走 Authorization 头即其证明）。
func TestPkPlansCsrfGateHTTPContract(t *testing.T) {
	t.Run("cross-site cookie put rejected 403 before authentication", func(t *testing.T) {
		conn, router := setupPkPlansContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		request := httptest.NewRequest(http.MethodPut, "http://forum.example.test/api/pk/plans", strings.NewReader(pkPlansPutBody))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://evil.example.test")
		request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("cross-site cookie put status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "csrf-rejected.json"))
		// 与其他 cookie 写组同语义：拒绝发生在认证之前，不得产生任何数据。
		var count int64
		if err := conn.Model(&pk.ScheduleSnapshotEntity{}).Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("rejected cross-site put must not write (count=%d err=%v)", count, err)
		}
	})

	t.Run("same-origin cookie put passes gate and reaches handler", func(t *testing.T) {
		conn, router := setupPkPlansContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		token := contractSessionToken(t, user)

		request := httptest.NewRequest(http.MethodPut, "http://forum.example.test/api/pk/plans", strings.NewReader(pkPlansPutBody))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Origin", "http://forum.example.test")
		request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("same-origin cookie put status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		if env := decodePkPlansEnvelope(t, recorder); env.Code != 0 {
			t.Fatalf("same-origin cookie put code = %d, want 0: %s", env.Code, recorder.Body.String())
		}
		var count int64
		if err := conn.Model(&pk.ScheduleSnapshotEntity{}).Count(&count).Error; err != nil || count != 1 {
			t.Fatalf("same-origin cookie put must persist snapshot (count=%d err=%v)", count, err)
		}
	})
}

// Two devices must not silently replace each other's edits or recreate a deleted revision.
func TestPkPlansRejectsStaleRevision(t *testing.T) {
	conn, router := setupPkPlansContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	put := func(base string) *httptest.ResponseRecorder {
		body := strings.TrimSuffix(pkPlansPutBody, "}") + `,"baseUpdatedAt":"` + base + `"}`
		return serveAuthSecurityJSON(router, http.MethodPut, "/api/pk/plans", body, token)
	}
	first := put("")
	if first.Code != 200 {
		t.Fatal(first.Body.String())
	}
	var data struct {
		UpdatedAt string `json:"updatedAt"`
	}
	if err := json.Unmarshal(decodePkPlansEnvelope(t, first).Data, &data); err != nil {
		t.Fatal(err)
	}
	if got := put(""); got.Code != 409 {
		t.Fatalf("concurrent creation = %d, want 409", got.Code)
	}
	second := put(data.UpdatedAt)
	if second.Code != 200 {
		t.Fatal(second.Body.String())
	}
	stale := put(data.UpdatedAt)
	if stale.Code != 409 {
		t.Fatalf("stale update = %d, want 409", stale.Code)
	}
	assertPkPlansFixture(t, decodePkPlansEnvelope(t, stale), pkPlansFixtureOf(t, "pk-plans-conflict.json"))
	serveAuthSecurityJSON(router, http.MethodDelete, "/api/pk/plans", "", token)
	if got := put(data.UpdatedAt); got.Code != 409 {
		t.Fatalf("deleted base = %d, want 409", got.Code)
	}
}
