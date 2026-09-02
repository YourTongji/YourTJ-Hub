package routes

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
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
	integrationsAPI.GET("/schedule-settings", UpButterReq(api.GetScheduleSettings))
	integrationsAPI.POST("/save-schedule-settings", UpButterReq(api.SaveScheduleSettings))
	return conn, router
}

func clearAdminIntegrationCaches() {
	hotdataserve.ClearMailSettingsConfigCache()
	hotdataserve.ClearStorageSettingsConfigCache()
	hotdataserve.ClearMCPSettingsConfigCache()
	hotdataserve.ClearScheduleSettingsConfigCache()
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
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
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "account-frozen.json"))
	})

	t.Run("user without SiteManager returns 403", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-ai-summary-settings-permission-denied.json"))
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

	t.Run("success returns the stored mail settings with the password configured state only", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		sealed, err := securestore.EncryptPurpose("contract-smtp-secret", securestore.MailSmtpPasswordPurpose)
		if err != nil {
			t.Fatalf("encrypt contract smtp password: %v", err)
		}
		persistContractPageConfig(t, conn, pageConfig.EmailSettings, pageConfig.MailSettingsStorage{
			EnableMail:            true,
			SmtpHost:              "smtp.contract.example.test",
			SmtpPort:              465,
			UseSSL:                true,
			SmtpUsername:          "mailer@contract.example.test",
			SmtpPasswordEncrypted: sealed,
			FromName:              "契约站务",
			FromEmail:             "noreply@contract.example.test",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-mail-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-mail-settings")
}

func TestAdminSaveMailSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-mail-settings"

	t.Run("success replaces the stored mail settings and encrypts the password", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.EmailSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearMailSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enableMail":true,"smtpHost":"smtp.new.example.test","smtpPort":587,"useSSL":false,"smtpUsername":"u","smtpPassword":"p","fromName":"新站务","fromEmail":"hi@new.example.test"}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.EmailSettings, pageConfig.MailSettingsStorage{})
		if stored.SmtpHost != "smtp.new.example.test" || stored.SmtpPort != 587 || stored.FromName != "新站务" {
			t.Fatalf("stored mail settings = %#v, want submitted values", stored)
		}
		if stored.SmtpPasswordEncrypted == "" {
			t.Fatalf("stored mail settings = %#v, want an encrypted smtp password", stored)
		}
		if plain, err := securestore.DecryptPurpose(stored.SmtpPasswordEncrypted, securestore.MailSmtpPasswordPurpose); err != nil || plain != "p" {
			t.Fatalf("stored smtp password decrypt = %q, err %v; want %q", plain, err, "p")
		}
	})

	t.Run("blank smtpPassword keeps the stored password", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		sealed, err := securestore.EncryptPurpose("keep-me", securestore.MailSmtpPasswordPurpose)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		persistContractPageConfig(t, conn, pageConfig.EmailSettings, pageConfig.MailSettingsStorage{
			EnableMail:            true,
			SmtpHost:              "smtp.contract.example.test",
			SmtpPort:              465,
			UseSSL:                true,
			SmtpUsername:          "mailer@contract.example.test",
			SmtpPasswordEncrypted: sealed,
			FromName:              "契约站务",
			FromEmail:             "noreply@contract.example.test",
		})
		hotdataserve.ClearMailSettingsConfigCache()
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.EmailSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearMailSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"enableMail":true,"smtpHost":"smtp.contract.example.test","smtpPort":465,"useSSL":true,"smtpUsername":"mailer@contract.example.test","smtpPassword":"","fromName":"契约站务","fromEmail":"noreply@contract.example.test"}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.EmailSettings, pageConfig.MailSettingsStorage{})
		if plain, err := securestore.DecryptPurpose(stored.SmtpPasswordEncrypted, securestore.MailSmtpPasswordPurpose); err != nil || plain != "keep-me" {
			t.Fatalf("stored smtp password decrypt = %q, err %v; want kept %q", plain, err, "keep-me")
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
			"invalid-params.json")
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

	t.Run("success returns the stored storage settings with the credential configured state only", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		akSealed, err := securestore.EncryptPurpose("contract-ak", securestore.StorageAccessKeyPurpose)
		if err != nil {
			t.Fatalf("encrypt contract access key: %v", err)
		}
		skSealed, err := securestore.EncryptPurpose("contract-sk", securestore.StorageSecretKeyPurpose)
		if err != nil {
			t.Fatalf("encrypt contract secret key: %v", err)
		}
		persistContractPageConfig(t, conn, pageConfig.StorageSettingsPage, pageConfig.StorageSettingsStorage{
			Provider:           "s3",
			Endpoint:           "https://s3.contract.example.test",
			Bucket:             "contract-bucket",
			Region:             "contract-region",
			BucketLookup:       "auto",
			Secure:             true,
			AccessKeyEncrypted: akSealed,
			SecretKeyEncrypted: skSealed,
			PublicUrlPrefix:    "",
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-storage-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-storage-settings")
}

func TestAdminSaveStorageSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-storage-settings"

	t.Run("success replaces the stored storage settings and encrypts credentials", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.StorageSettingsPage).Delete(&pageConfig.Entity{})
			hotdataserve.ClearStorageSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"s3","endpoint":"https://s3.new.example.test","internalEndpoint":"https://s3-internal.new.example.test","bucket":"new-bucket","region":"r","bucketLookup":"auto","secure":true,"accessKey":"new-ak","secretKey":"new-sk","publicUrlPrefix":""}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.StorageSettingsPage, pageConfig.StorageSettingsStorage{})
		if stored.Provider != "s3" || stored.Endpoint != "https://s3.new.example.test" || stored.InternalEndpoint != "https://s3-internal.new.example.test" || stored.Bucket != "new-bucket" {
			t.Fatalf("stored storage settings = %#v, want submitted values", stored)
		}
		if plain, err := securestore.DecryptPurpose(stored.AccessKeyEncrypted, securestore.StorageAccessKeyPurpose); err != nil || plain != "new-ak" {
			t.Fatalf("stored accessKey decrypt = %q, err %v; want %q", plain, err, "new-ak")
		}
		if plain, err := securestore.DecryptPurpose(stored.SecretKeyEncrypted, securestore.StorageSecretKeyPurpose); err != nil || plain != "new-sk" {
			t.Fatalf("stored secretKey decrypt = %q, err %v; want %q", plain, err, "new-sk")
		}
		if strings.Contains(storedAsJSON(t, conn, pageConfig.StorageSettingsPage), "new-ak") {
			t.Fatal("plaintext access key leaked into stored config")
		}
	})

	t.Run("blank credentials keep the stored keys", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		akSealed, _ := securestore.EncryptPurpose("keep-ak", securestore.StorageAccessKeyPurpose)
		skSealed, _ := securestore.EncryptPurpose("keep-sk", securestore.StorageSecretKeyPurpose)
		persistContractPageConfig(t, conn, pageConfig.StorageSettingsPage, pageConfig.StorageSettingsStorage{
			Provider:           "s3",
			Endpoint:           "https://s3.contract.example.test",
			Bucket:             "contract-bucket",
			AccessKeyEncrypted: akSealed,
			SecretKeyEncrypted: skSealed,
		})
		hotdataserve.ClearStorageSettingsConfigCache()
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.StorageSettingsPage).Delete(&pageConfig.Entity{})
			hotdataserve.ClearStorageSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"provider":"s3","endpoint":"https://s3.contract.example.test","bucket":"contract-bucket","region":"","bucketLookup":"auto","secure":true,"accessKey":"","secretKey":"","publicUrlPrefix":""}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.StorageSettingsPage, pageConfig.StorageSettingsStorage{})
		if plain, err := securestore.DecryptPurpose(stored.AccessKeyEncrypted, securestore.StorageAccessKeyPurpose); err != nil || plain != "keep-ak" {
			t.Fatalf("stored accessKey decrypt = %q, err %v; want kept %q", plain, err, "keep-ak")
		}
		if plain, err := securestore.DecryptPurpose(stored.SecretKeyEncrypted, securestore.StorageSecretKeyPurpose); err != nil || plain != "keep-sk" {
			t.Fatalf("stored secretKey decrypt = %q, err %v; want kept %q", plain, err, "keep-sk")
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
			"invalid-params.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-storage-settings")
}

func storedAsJSON(t *testing.T, conn *gorm.DB, pageType string) string {
	t.Helper()
	var entity pageConfig.Entity
	if err := conn.Where("page_type = ?", pageType).First(&entity).Error; err != nil {
		t.Fatalf("read stored %s config: %v", pageType, err)
	}
	return entity.Config
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
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.MCPSettings, pageConfig.MCPSettingsConfig{})
		if !stored.Enabled || stored.Writes {
			t.Fatalf("stored MCP settings = %#v, want submitted values", stored)
		}
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-mcp-settings")
}

func TestAdminGetScheduleSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/schedule-settings"

	t.Run("success returns the stored section times", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		persistContractPageConfig(t, conn, pageConfig.ScheduleSettings, pageConfig.ScheduleSettingsConfig{
			SectionTimes: []pageConfig.ScheduleSectionTime{
				{Section: 1, Start: "08:00", End: "08:45"},
				{Section: 2, Start: "08:50", End: "09:35"},
				{Section: 3, Start: "10:00", End: "10:45"},
				{Section: 4, Start: "10:50", End: "11:35"},
				{Section: 5, Start: "13:30", End: "14:15"},
				{Section: 6, Start: "14:20", End: "15:05"},
				{Section: 7, Start: "15:30", End: "16:15"},
				{Section: 8, Start: "16:20", End: "17:05"},
				{Section: 9, Start: "17:10", End: "17:55"},
				{Section: 10, Start: "18:30", End: "19:15"},
				{Section: 11, Start: "19:20", End: "20:05"},
				{Section: 12, Start: "20:10", End: "20:55"},
			},
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-schedule-settings-success.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodGet, path, "admin-schedule-settings")
}

func TestAdminSaveScheduleSettingsHTTPContract(t *testing.T) {
	path := "/api/admin/save-schedule-settings"

	t.Run("success replaces the stored section times", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.ScheduleSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearScheduleSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"sectionTimes":[{"section":12,"start":"20:10","end":"20:55"},{"section":1,"start":"08:00","end":"08:45"}]}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.ScheduleSettings, pageConfig.ScheduleSettingsConfig{})
		if len(stored.SectionTimes) != 2 {
			t.Fatalf("stored schedule settings = %#v, want two section times", stored)
		}
		// 输入按节次升序排序后落库（12 在前提交，存储后 1 在前）。
		if stored.SectionTimes[0].Section != 1 || stored.SectionTimes[1].Section != 12 {
			t.Fatalf("stored section times = %#v, want sorted by section ascending", stored.SectionTimes)
		}
	})

	t.Run("duplicate section entries keep the first occurrence", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		t.Cleanup(func() {
			conn.Where("page_type = ?", pageConfig.ScheduleSettings).Delete(&pageConfig.Entity{})
			hotdataserve.ClearScheduleSettingsConfigCache()
		})
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"sectionTimes":[{"section":1,"start":"08:00","end":"08:45"},{"section":1,"start":"09:00","end":"09:45"}]}}`,
			"admin-agent-disable-success.json")
		stored := pageConfig.GetConfigByPageType(pageConfig.ScheduleSettings, pageConfig.ScheduleSettingsConfig{})
		if len(stored.SectionTimes) != 1 || stored.SectionTimes[0].Start != "08:00" {
			t.Fatalf("stored section times = %#v, want the deduplicated first entry", stored.SectionTimes)
		}
	})

	t.Run("section outside 1..12 fails with invalidParams", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"sectionTimes":[{"section":13,"start":"08:00","end":"08:45"}]}}`,
			"invalid-params.json")
	})

	t.Run("malformed clock value fails with invalidParams", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"sectionTimes":[{"section":1,"start":"8:00","end":"08:45"}]}}`,
			"invalid-params.json")
	})

	t.Run("end before start fails with invalidParams", func(t *testing.T) {
		conn, router := setupAdminIntegrationsContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"settings":{"sectionTimes":[{"section":1,"start":"10:00","end":"09:00"}]}}`,
			"invalid-params.json")
	})

	adminIntegrationsGuardScenarios(t, http.MethodPost, path, "admin-save-schedule-settings")
}
