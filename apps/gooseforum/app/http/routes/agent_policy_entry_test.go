package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/agents"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

// TestAgentCreateRejectsReservedUsername Agent（机器人）建号同样执行保留/禁用
// 名单检查：命中返回 auth.username.reserved/banned，且不创建 users/agents 行。
func TestAgentCreateRejectsReservedUsername(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	conn := db.Connect()
	// setupAgentAdminTestDB 未迁移 page_config；策略检查（hotdataserve 缓存读
	// page_config 行）依赖该表存在。补迁移保证名单可持久化并命中。
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config for agent policy test: %v", err)
	}
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.ReservedUsernames = []string{"administrator", "moderator"}
		cfg.BannedUsernames = []string{"spammer"}
		return cfg
	}())

	tests := []struct {
		name     string
		username string
		wantCode string
	}{
		{name: "reserved", username: "administrator", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "reserved case-insensitive", username: "Administrator", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "reserved leet", username: "m0derat0r", wantCode: string(component.MessageAuthUsernameReserved)},
		{name: "banned", username: "spammer", wantCode: string(component.MessageAuthUsernameBanned)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, resp := postAgent(t, router, "/api/admin/agent-create",
				`{"username":"`+tt.username+`","webhookEndpoint":"https://example.com/hook"}`)
			if code != http.StatusOK {
				t.Fatalf("status = %d, want 200 envelope: %s", code, mustJSON(resp))
			}
			if got := resp["messageCode"]; got != tt.wantCode {
				t.Fatalf("messageCode = %v, want %s (body %s)", got, tt.wantCode, mustJSON(resp))
			}
			// 不得创建任何 agent/users 行
			var agentCount int64
			conn.Model(&agents.Entity{}).Count(&agentCount)
			if agentCount != 0 {
				t.Fatalf("agents count = %d, want 0 after rejected create", agentCount)
			}
			var userCount int64
			conn.Model(&users.EntityComplete{}).Count(&userCount)
			if userCount != 0 {
				t.Fatalf("users count = %d, want 0 after rejected create", userCount)
			}
		})
	}
}

// TestAgentCreateAllowsUnreservedUsername 名单之外的名字可正常创建（防误伤回归）。
func TestAgentCreateAllowsUnreservedUsername(t *testing.T) {
	setupAgentAdminTestDB(t)
	router := agentAdminRouter(1)
	conn := db.Connect()
	// 同上：补迁移 page_config 表。
	if err := conn.AutoMigrate(&pageConfig.Entity{}); err != nil {
		t.Fatalf("migrate page_config for agent policy test: %v", err)
	}
	persistSecurityPolicyConfig(t, conn, func() pageConfig.SecurityAndRegistration {
		cfg := emptySecurityConfig()
		cfg.ReservedUsernames = []string{"administrator"}
		return cfg
	}())

	_, resp := postAgent(t, router, "/api/admin/agent-create",
		`{"username":"clean-bot-name","webhookEndpoint":"https://example.com/hook"}`)
	if resp["code"] == nil || resp["messageCode"] != nil {
		// 成功响应的 messageCode 为空；失败时 messageCode 非空。
		// 这里直接断言信封 code==0（成功）更稳：成功响应 result 含 agent。
		if got := resp["code"].(float64); got != 0 {
			t.Fatalf("clean agent create failed: %s", mustJSON(resp))
		}
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result missing on clean create: %s", mustJSON(resp))
	}
	if result["token"] == "" {
		t.Fatalf("clean create should return token: %s", mustJSON(resp))
	}
}

var _ = json.Marshal
