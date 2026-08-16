package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/dailyStats"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/agentservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// setupAdminOpsContractTest 在共享 harness（setupHTTPContractTest）之上注册 Admin 权限组
// 7 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(Admin) 子组）。
// agents / opt_record 表在各子测试间清空，避免同进程共享库中其他测试的数据串扰。
func setupAdminOpsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&rolePermissionRs.Entity{},
		&agents.Entity{},
		&optRecord.Entity{},
	); err != nil {
		t.Fatalf("migrate admin operations contract tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&agents.Entity{})
	conn.Where("1 = 1").Delete(&optRecord.Entity{})

	adminAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.Admin),
	)
	adminAPI.POST("/agent-list", UpButterReq(api.AgentList))
	adminAPI.POST("/agent-create", UpButterReq(api.AgentCreate))
	adminAPI.POST("/agent-update", UpButterReq(api.AgentUpdate))
	adminAPI.POST("/agent-rotate-token", UpButterReq(api.AgentRotateToken))
	adminAPI.POST("/agent-disable", UpButterReq(api.AgentDisable))
	adminAPI.POST("/opt-record-page", UpButterReq(api.OptRecordPage))
	adminAPI.POST("/traffic-overview", UpButterReq(api.GetTrafficOverview))
	return conn, router
}

// createContractAdmin 创建登录用户并授予 Admin 权限
// （复用 grantContractPermission：独立角色 ID，规避 10min 权限缓存串扰）。
func createContractAdmin(t *testing.T, conn *gorm.DB) *users.EntityComplete {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, user.Id, permission.Admin)
	return user
}

// assertAdminOpsPermissionDenied 断言无 Admin 权限的登录用户被 CheckPermission
// 中间件拦截为 HTTP 403 + permission.denied（params.permission 为本地化名"管理员"）。
func assertAdminOpsPermissionDenied(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, fixture string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	recorder := serveJSON(router, path, `{}`, contractSessionToken(t, user))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// adminOpsGuardScenarios 跑 7 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 Admin 权限 403。
func adminOpsGuardScenarios(t *testing.T, path string, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminOpsContractTest(t)
		assertInteractionUnauthenticated(t, router, path, `{}`, fixturePrefix+"-unauthenticated.json")
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		assertInteractionForbidden(t, conn, router, path, `{}`, fixturePrefix+"-forbidden.json")
	})

	t.Run("user without Admin permission returns 403", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		assertAdminOpsPermissionDenied(t, conn, router, path, fixturePrefix+"-permission-denied.json")
	})
}

// serveAdminOpsOK 以 Admin 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminOpsOK(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string, fixture string) {
	t.Helper()
	admin := createContractAdmin(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, admin))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// serveAdminOpsRaw 以 Admin 身份调用路由，返回原始 recorder 供动态结果的结构化断言。
func serveAdminOpsRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, body string) *httptest.ResponseRecorder {
	t.Helper()
	admin := createContractAdmin(t, conn)
	recorder := serveJSON(router, path, body, contractSessionToken(t, admin))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// createContractAgent 通过生产 handler 创建 Agent，返回 agentId 与一次性明文 token。
func createContractAgent(t *testing.T, conn *gorm.DB, router *gin.Engine, username string) (uint64, string) {
	t.Helper()
	recorder := serveAdminOpsRaw(t, conn, router, "/api/admin/agent-create", `{"username":"`+username+`"}`)
	response := decodeContractEnvelope(t, recorder)
	if response.Code != 0 {
		t.Fatalf("create agent %s envelope = %#v, want success", username, response)
	}
	var created struct {
		Agent struct {
			AgentId uint64 `json:"agentId"`
		} `json:"agent"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Result, &created); err != nil {
		t.Fatalf("decode create agent result %s: %v", response.Result, err)
	}
	if created.Agent.AgentId == 0 || created.Token == "" {
		t.Fatalf("create agent result = %s, want agentId and token", response.Result)
	}
	return created.Agent.AgentId, created.Token
}

func TestAdminAgentListHTTPContract(t *testing.T) {
	path := "/api/admin/agent-list"

	t.Run("success with empty list", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{}`, "admin-agent-list-success.json")
	})

	t.Run("created agent appears without secret fields", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, token := createContractAgent(t, conn, router, "contract-list-bot")
		recorder := serveAdminOpsRaw(t, conn, router, path, `{}`)
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("list envelope = %#v, want success", response)
		}
		var list []struct {
			AgentId     uint64 `json:"agentId"`
			Username    string `json:"username"`
			TokenPrefix string `json:"tokenPrefix"`
			Enabled     int8   `json:"enabled"`
		}
		if err := json.Unmarshal(response.Result, &list); err != nil {
			t.Fatalf("decode list result %s: %v", response.Result, err)
		}
		if len(list) != 1 {
			t.Fatalf("list = %#v, want exactly the created agent", list)
		}
		item := list[0]
		if item.AgentId != agentID || item.Username != "contract-list-bot" {
			t.Fatalf("list item = %#v", item)
		}
		if item.Enabled != agents.StatusEnabled || !strings.HasPrefix(item.TokenPrefix, agentservice.TokenMark) {
			t.Fatalf("list item = %#v, want enabled with agt_ prefix", item)
		}
		if strings.Contains(recorder.Body.String(), "tokenHash") {
			t.Fatal("list response must not expose tokenHash")
		}
		if strings.Contains(recorder.Body.String(), token) {
			t.Fatal("list response leaks the plaintext token")
		}
	})

	adminOpsGuardScenarios(t, path, "admin-agent-list")
}

func TestAdminAgentCreateHTTPContract(t *testing.T) {
	path := "/api/admin/agent-create"

	t.Run("success returns the one-time token", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		admin := createContractAdmin(t, conn)
		body := `{"username":"contract-cr-bot","nickname":"创建机器人","webhookEndpoint":"https://hooks.example.com/agent"}`
		recorder := serveJSON(router, path, body, contractSessionToken(t, admin))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("create envelope = %#v, want success", response)
		}
		var created struct {
			Agent struct {
				AgentId         uint64 `json:"agentId"`
				Username        string `json:"username"`
				Nickname        string `json:"nickname"`
				TokenPrefix     string `json:"tokenPrefix"`
				WebhookEndpoint string `json:"webhookEndpoint"`
				Enabled         int8   `json:"enabled"`
				CreatedBy       uint64 `json:"createdBy"`
			} `json:"agent"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal(response.Result, &created); err != nil {
			t.Fatalf("decode create result %s: %v", response.Result, err)
		}
		if created.Agent.Username != "contract-cr-bot" || created.Agent.Nickname != "创建机器人" ||
			created.Agent.WebhookEndpoint != "https://hooks.example.com/agent" ||
			created.Agent.Enabled != agents.StatusEnabled || created.Agent.CreatedBy != admin.Id {
			t.Fatalf("created agent = %#v", created.Agent)
		}
		if !strings.HasPrefix(created.Token, agentservice.TokenMark) {
			t.Fatalf("token = %q, want agt_ prefix", created.Token)
		}
		if created.Agent.TokenPrefix != created.Token[:len(agentservice.TokenMark)+8] {
			t.Fatalf("tokenPrefix = %q, want prefix of the issued token", created.Agent.TokenPrefix)
		}
		stored := agents.GetByUserID(created.Agent.AgentId)
		if stored == nil || stored.TokenHash == "" || stored.TokenHash == created.Token {
			t.Fatal("stored token must be a hash, not plaintext")
		}
	})

	t.Run("invalid username format returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{"username":"bad"}`, "admin-agent-create-username-invalid.json")
	})

	t.Run("duplicate username returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		createContractAgent(t, conn, router, "contract-dup-bot")
		serveAdminOpsOK(t, conn, router, path, `{"username":"contract-dup-bot"}`, "admin-agent-create-username-exists.json")
	})

	t.Run("invalid webhook returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{"username":"contract-hook-bot","webhookEndpoint":"ftp://x"}`, "admin-agent-create-webhook-invalid.json")
	})

	t.Run("missing username stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{}`, "admin-agent-create-invalid-params.json")
	})

	adminOpsGuardScenarios(t, path, "admin-agent-create")
}

func TestAdminAgentUpdateHTTPContract(t *testing.T) {
	path := "/api/admin/agent-update"

	t.Run("success applies only the present fields", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, _ := createContractAgent(t, conn, router, "contract-upd-bot")
		recorder := serveAdminOpsRaw(t, conn, router, path,
			`{"agentId":`+strconv.FormatUint(agentID, 10)+`,"nickname":"改名后","webhookEndpoint":"https://hooks2.example.com/agent"}`)
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("update envelope = %#v, want success", response)
		}
		var item struct {
			AgentId         uint64 `json:"agentId"`
			Username        string `json:"username"`
			Nickname        string `json:"nickname"`
			WebhookEndpoint string `json:"webhookEndpoint"`
			Enabled         int8   `json:"enabled"`
		}
		if err := json.Unmarshal(response.Result, &item); err != nil {
			t.Fatalf("decode update result %s: %v", response.Result, err)
		}
		if item.AgentId != agentID || item.Username != "contract-upd-bot" ||
			item.Nickname != "改名后" || item.WebhookEndpoint != "https://hooks2.example.com/agent" ||
			item.Enabled != agents.StatusEnabled {
			t.Fatalf("updated agent = %#v", item)
		}
	})

	t.Run("unknown agent returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{"agentId":987654321,"nickname":"X"}`, "admin-agent-update-not-found.json")
	})

	t.Run("re-enabling a disabled agent requires rotation first", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, _ := createContractAgent(t, conn, router, "contract-need-bot")
		disableRecorder := serveAdminOpsRaw(t, conn, router, path, `{"agentId":`+strconv.FormatUint(agentID, 10)+`,"enabled":0}`)
		if decodeContractEnvelope(t, disableRecorder).Code != 0 {
			t.Fatalf("disable via update failed: %s", disableRecorder.Body.String())
		}
		serveAdminOpsOK(t, conn, router, path, `{"agentId":`+strconv.FormatUint(agentID, 10)+`,"enabled":1}`, "admin-agent-update-needs-rotate.json")
	})

	t.Run("enabled outside 0-1 stays a legacy HTTP 200 validation failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, _ := createContractAgent(t, conn, router, "contract-enb-bot")
		serveAdminOpsOK(t, conn, router, path, `{"agentId":`+strconv.FormatUint(agentID, 10)+`,"enabled":5}`, "admin-agent-update-invalid-params.json")
	})

	adminOpsGuardScenarios(t, path, "admin-agent-update")
}

func TestAdminAgentRotateTokenHTTPContract(t *testing.T) {
	path := "/api/admin/agent-rotate-token"

	t.Run("success invalidates the old token immediately", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, oldToken := createContractAgent(t, conn, router, "contract-rot-bot")
		recorder := serveAdminOpsRaw(t, conn, router, path, `{"agentId":`+strconv.FormatUint(agentID, 10)+`}`)
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 {
			t.Fatalf("rotate envelope = %#v, want success", response)
		}
		var rotated struct {
			AgentId uint64 `json:"agentId"`
			Token   string `json:"token"`
		}
		if err := json.Unmarshal(response.Result, &rotated); err != nil {
			t.Fatalf("decode rotate result %s: %v", response.Result, err)
		}
		if rotated.AgentId != agentID || !strings.HasPrefix(rotated.Token, agentservice.TokenMark) || rotated.Token == oldToken {
			t.Fatalf("rotated = %#v, want new agt_ token for agent %d", rotated, agentID)
		}
		if _, _, err := agentservice.ResolveByToken(oldToken); err == nil {
			t.Fatal("old token must stop resolving immediately after rotation")
		}
		if _, _, err := agentservice.ResolveByToken(rotated.Token); err != nil {
			t.Fatalf("new token should resolve: %v", err)
		}
	})

	t.Run("unknown agent returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{"agentId":987654321}`, "admin-agent-rotate-token-not-found.json")
	})

	adminOpsGuardScenarios(t, path, "admin-agent-rotate-token")
}

func TestAdminAgentDisableHTTPContract(t *testing.T) {
	path := "/api/admin/agent-disable"

	t.Run("success revokes the credential", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		agentID, token := createContractAgent(t, conn, router, "contract-dis-bot")
		serveAdminOpsOK(t, conn, router, path, `{"agentId":`+strconv.FormatUint(agentID, 10)+`}`, "admin-agent-disable-success.json")
		stored := agents.GetByUserID(agentID)
		if stored == nil || stored.Enabled != agents.StatusDisabled {
			t.Fatalf("agent enabled = %v, want disabled", stored.Enabled)
		}
		if stored.TokenHash != "" {
			t.Fatal("disable must revoke the stored token hash")
		}
		if _, _, err := agentservice.ResolveByToken(token); err == nil {
			t.Fatal("disabled agent token must no longer resolve (bearer auth fails 401)")
		}
	})

	t.Run("unknown agent returns business failure", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		serveAdminOpsOK(t, conn, router, path, `{"agentId":987654321}`, "admin-agent-disable-not-found.json")
	})

	adminOpsGuardScenarios(t, path, "admin-agent-disable")
}

func TestAdminOptRecordPageHTTPContract(t *testing.T) {
	path := "/api/admin/opt-record-page"

	t.Run("success returns the seeded record page", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		// optType 0 = optlogger.EditUser，targetType 1 = optlogger.User（与 agent-create 审计一致）。
		record := optRecord.Entity{
			Id:         4097,
			OptUserId:  1024,
			OptType:    0,
			TargetType: 1,
			TargetId:   "2049",
			OptInfo:    `{"messageCode":"admin.opt.agent.created","params":{"agentId":2049,"username":"campus-helper-bot"}}`,
			CreatedAt:  time.Date(2026, 8, 10, 8, 0, 0, 0, time.UTC),
		}
		if err := conn.Create(&record).Error; err != nil {
			t.Fatalf("seed opt record: %v", err)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", record.Id).Delete(&optRecord.Entity{})
		})
		// 响应 page 为 0 基（请求页码减一）：请求 page=1 回显 page=0。
		serveAdminOpsOK(t, conn, router, path, `{"page":1,"pageSize":10,"optUserId":1024}`, "admin-opt-record-page-success.json")
	})

	adminOpsGuardScenarios(t, path, "admin-opt-record-page")
}

func TestAdminTrafficOverviewHTTPContract(t *testing.T) {
	path := "/api/admin/traffic-overview"

	t.Run("success merges stat rows into the daily series", func(t *testing.T) {
		conn, router := setupAdminOpsContractTest(t)
		// 2020 年区间不会被其他测试的当日统计污染，保证分页响应确定性。
		if err := dailyStats.Increment(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), dailyStats.StatTypeRegCount, 2); err != nil {
			t.Fatalf("seed reg_count stat: %v", err)
		}
		if err := dailyStats.Increment(time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC), dailyStats.StatTypeTopicCount, 5); err != nil {
			t.Fatalf("seed topic_count stat: %v", err)
		}
		t.Cleanup(func() {
			conn.Where("stat_date >= ?", "2020-01-01").Where("stat_date <= ?", "2020-01-03").Delete(&dailyStats.Entity{})
		})
		serveAdminOpsOK(t, conn, router, path, `{"startDate":"2020-01-01","endDate":"2020-01-03"}`, "admin-traffic-overview-success.json")
	})

	adminOpsGuardScenarios(t, path, "admin-traffic-overview")
}
