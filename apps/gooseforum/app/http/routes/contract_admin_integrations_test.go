package routes

import (
	"net/http"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/filemigrateservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组 mail/storage/mcp 10 条集成设置路由的契约测试
// （issue #277 P3 切片六）。mail/storage/mcp 的 hotdataserve 缓存与各 pageConfig 行
// 在子测试间清理，避免同进程共享库串扰。

// 契约用固定任务行时间戳（与 fixtures/admin-*-tasks-success.json 一致）。
var contractTaskCreatedAt = time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

// setupAdminIntegrationsContractTest 在共享 harness（setupHTTPContractTest）之上注册
// mail/storage/mcp 10 条路由，中间件链与 route4api.go 的生产注册保持一致
// （JWTAuthCheck + CheckWritableAccount 公共链 + CheckPermission(SiteManager) 子组）。
func setupAdminIntegrationsContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}, &taskQueue.Entity{}); err != nil {
		t.Fatalf("migrate admin integrations contract tables: %v", err)
	}
	clearAdminIntegrationCaches()
	t.Cleanup(clearAdminIntegrationCaches)

	integrationsAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	integrationsAPI.GET("/mail-settings", UpButterReq(api.GetMailSettings))
	integrationsAPI.POST("/save-mail-settings", UpButterReq(api.SaveMailSettings))
	integrationsAPI.POST("/test-mail-connection", UpButterReq(api.TestMailConnection))
	integrationsAPI.GET("/storage-settings", UpButterReq(api.GetStorageSettings))
	integrationsAPI.POST("/save-storage-settings", UpButterReq(api.SaveStorageSettings))
	integrationsAPI.POST("/test-storage-connection", UpButterReq(api.TestStorageConnection))
	integrationsAPI.POST("/storage-migrate-task", UpButterReq(api.CreateStorageMigrateTask))
	integrationsAPI.GET("/storage-migrate-tasks", UpButterReq(api.GetStorageMigrateTasks))
	integrationsAPI.GET("/mcp-settings", UpButterReq(api.GetMCPSettings))
	integrationsAPI.POST("/save-mcp-settings", UpButterReq(api.SaveMCPSettings))
	return conn, router
}

func clearAdminIntegrationCaches() {
	hotdataserve.ClearMailSettingsConfigCache()
	hotdataserve.ClearStorageSettingsConfigCache()
	hotdataserve.ClearMCPSettingsConfigCache()
}

// adminIntegrationsGuardScenarios 跑本文件 10 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
func adminIntegrationsGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminIntegrationsContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-unauthenticated.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
			t.Fatalf("freeze contract user: %v", err)
		}
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("frozen account status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-forbidden.json"))
	})

	t.Run("user without SiteManager returns 403", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixturePrefix+"-permission-denied.json"))
	})
}

// persistContractStorageSettings 持久化存储配置并清缓存（storage 读取走 hotdataserve 缓存），
// 子测试结束后删除对应行并再次清缓存。
func persistContractStorageSettings(t *testing.T, conn *gorm.DB, config pageConfig.StorageSettings) {
	t.Helper()
	persistHTTPContractConfig(t, conn, pageConfig.StorageSettingsPage, config)
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.StorageSettingsPage).Delete(&pageConfig.Entity{})
		hotdataserve.ClearStorageSettingsConfigCache()
	})
}

// seedContractTask 插入一条固定字段的 task_queue 行并在子测试结束后删除，
// 供任务列表/下载场景产出确定性 fixture。
func seedContractTask(t *testing.T, conn *gorm.DB, entity taskQueue.Entity) {
	t.Helper()
	if err := conn.Create(&entity).Error; err != nil {
		t.Fatalf("seed task %d: %v", entity.Id, err)
	}
	t.Cleanup(func() {
		conn.Delete(&taskQueue.Entity{}, entity.Id)
	})
}

func TestAdminGetMailSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/mail-settings"

	t.Run("success returns the stored mail settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.EmailSettings, pageConfig.MailSettingsConfig{
			EnableMail:   true,
			SmtpHost:     "smtp.contract.example.test",
			SmtpPort:     465,
			UseSSL:       true,
			SmtpUsername: "mailer@contract.example.test",
			SmtpPassword: "contract-smtp-secret",
			FromName:     "契约站务",
			FromEmail:    "noreply@contract.example.test",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-mail-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-mail-settings")
}

func TestAdminSaveMailSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-mail-settings"

	t.Run("success replaces the stored mail settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.EmailSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearMailSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enableMail":true,"smtpHost":"smtp.new.example.test","smtpPort":587,"useSSL":false,"smtpUsername":"u","smtpPassword":"p","fromName":"新站务","fromEmail":"hi@new.example.test"}}`,
			"admin-save-mail-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.EmailSettings, pageConfig.MailSettingsConfig{})
		if stored.SmtpHost != "smtp.new.example.test" || stored.SmtpPort != 587 || stored.FromName != "新站务" {
			t.Fatalf("stored mail settings = %#v, want submitted values", stored)
		}
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-mail-settings")
}

func TestAdminTestMailConnectionHTTPContract(t *testing.T) {
	path := "/api/admin/test-mail-connection"

	t.Run("missing testEmail fails request validation", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"smtpHost":"smtp.contract.example.test"}}`,
			"admin-test-mail-connection-invalid-params.json")
	})

	t.Run("unreachable SMTP reports a testFailed result inside the success envelope", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		recorder := serveAdminSiteRaw(t, conn, router, http.MethodPost, path,
			`{"settings":{"enableMail":true,"smtpHost":"127.0.0.1","smtpPort":1,"useSSL":false,"fromEmail":"noreply@contract.example.test"},"testEmail":"probe@contract.example.test"}`)
		result := decodeSiteResult(t, recorder)
		if result["success"] != false {
			t.Fatalf("result.success = %#v, want false for an unreachable SMTP server", result["success"])
		}
		if result["messageCode"] != "admin.mail.testFailed" {
			t.Fatalf("result.messageCode = %#v, want admin.mail.testFailed", result["messageCode"])
		}
		params, ok := result["params"].(map[string]any)
		if !ok {
			t.Fatalf("result.params = %#v, want an error params object", result["params"])
		}
		if errText, _ := params["error"].(string); errText == "" {
			t.Fatalf("result.params.error = %#v, want the raw send error text", params["error"])
		}
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-test-mail-connection")
}

func TestAdminGetStorageSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/storage-settings"

	t.Run("success returns the stored storage settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.StorageSettingsPage, pageConfig.StorageSettings{
			Provider: "local",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-storage-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-storage-settings")
}

func TestAdminSaveStorageSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-storage-settings"

	t.Run("success replaces the stored storage settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.StorageSettingsPage).Delete(&pageConfig.Entity{})
			hotdataserve.ClearStorageSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"local","endpoint":"","bucket":"","region":"","bucketLookup":"","secure":false,"accessKey":"","secretKey":"","publicUrlPrefix":""}}`,
			"admin-save-storage-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.StorageSettingsPage, pageConfig.StorageSettings{})
		if stored.Provider != "local" {
			t.Fatalf("stored storage settings = %#v, want provider local", stored)
		}
	})

	t.Run("s3 without endpoint and bucket fails with saveFailed", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"s3","bucket":"contract-bucket"}}`,
			"admin-save-storage-settings-save-failed.json")
	})

	t.Run("unknown provider fails with invalidParams", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"ftp"}}`,
			"admin-save-storage-settings-invalid-params.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-storage-settings")
}

func TestAdminTestStorageConnectionHTTPContract(t *testing.T) {
	path := "/api/admin/test-storage-connection"

	t.Run("local provider always succeeds without a backend", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"local"}}`,
			"admin-test-storage-connection-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-test-storage-connection")
}

func TestAdminCreateStorageMigrateTaskHTTPContract(t *testing.T) {
	path := "/api/admin/storage-migrate-task"

	t.Run("local provider is rejected with migrateFailed", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		persistContractStorageSettings(t, conn, pageConfig.StorageSettings{Provider: "local"})
		// 注意：控制器用独立哨兵 errStorageProviderInvalid 判定 provider 错误，
		// 与服务返回的 filemigrateservice.ErrProviderNotS3 不匹配，因此实际暴露的
		// 是 admin.storage.migrateFailed（admin.storage.migrateInvalidProvider 不可达）。
		serveAdminSiteOK(t, conn, router, http.MethodPost, path, `{}`,
			"admin-storage-migrate-task-migrate-failed.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-storage-migrate-task")
}

func TestAdminListStorageMigrateTasksHTTPContract(t *testing.T) {
	path := "/api/admin/storage-migrate-tasks"

	t.Run("success lists the seeded migration task", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990001,
			Type:        filemigrateservice.TaskTypeFileMigrate,
			Status:      taskQueue.StatusFailed,
			TaskJson:    `{"lastId":0,"total":5,"processed":4,"failed":1,"clearAfterMigrate":true}`,
			RetryCount:  1,
			LastError:   "contract boom",
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: time.Time{},
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-storage-migrate-tasks-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-storage-migrate-tasks")
}

func TestAdminGetMcpSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/mcp-settings"

	t.Run("success returns the stored MCP settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.MCPSettings, pageConfig.MCPSettingsConfig{
			Enabled: true,
			Writes:  true,
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-mcp-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-mcp-settings")
}

func TestAdminSaveMcpSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-mcp-settings"

	t.Run("success replaces the stored MCP settings", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.MCPSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearMCPSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enabled":true,"writes":false}}`,
			"admin-save-mcp-settings-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.MCPSettings, pageConfig.MCPSettingsConfig{})
		if !stored.Enabled || stored.Writes {
			t.Fatalf("stored MCP settings = %#v, want submitted values", stored)
		}
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-mcp-settings")
}
