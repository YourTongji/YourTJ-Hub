package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/optRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/dataservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/filemigrateservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组 img-upload + data-portability 5 条路由的契约测试
// （issue #277 P3 切片六）。img-upload / data/import 为 multipart 裸 gin handler，
// data/export/download 为文件流裸 handler，均不经 UpButterReq，但守卫中间件链与
// route4api.go 的生产注册保持一致。任务行用固定字段种子保证 fixture 确定性；
// 文件内容落独立的 db4fileconnect 连接，单独迁移。

// setupAdminDataContractTest 在共享 harness（setupHTTPContractTest）之上注册
// img-upload/data 导出导入 5 条路由。
func setupAdminDataContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(&rolePermissionRs.Entity{}, &taskQueue.Entity{}, &optRecord.Entity{}); err != nil {
		t.Fatalf("migrate admin data contract tables: %v", err)
	}
	// filedata 走独立的 db4fileconnect 连接（测试模式同样各自 :memory:），
	// 需在文件库上单独迁移。
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}

	dataAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	dataAPI.POST("/img-upload", api.SaveAdminImgByGinContext)
	dataAPI.POST("/data/export", UpButterReq(api.CreateExportTask))
	dataAPI.GET("/data/export/tasks", UpButterReq(api.ListExportTasks))
	dataAPI.GET("/data/export/download/:taskId", api.DownloadExportTask)
	dataAPI.POST("/data/import", api.ImportData)
	dataAPI.GET("/data/import/tasks", UpButterReq(api.ListImportTasks))
	dataAPI.POST("/data/import/tasks/:taskId/replay", UpUriReq(api.ReplayImportTask))
	return conn, router
}

// adminDataGuardScenarios 跑本文件 5 条路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
// 守卫由中间件在绑定前拦截，统一用 JSON 请求即可（含 multipart/文件流路由）。
func adminDataGuardScenarios(t *testing.T, method, path, fixturePrefix string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminDataContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
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
		conn, router := setupAdminDataContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-ai-summary-settings-permission-denied.json"))
	})
}

// serveAdminDataRaw 以 SiteManager 身份发起请求，返回原始 recorder（不断言状态码），
// 供 400/404 业务分支与 multipart/文件流场景使用。
func serveAdminDataRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	return serveAuthSecurityJSON(router, method, path, body, contractSessionToken(t, manager))
}

// serveAdminDataMultipart 以 SiteManager 身份提交 multipart 表单。
func serveAdminDataMultipart(t *testing.T, conn *gorm.DB, router *gin.Engine, path string, files map[string][]byte) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	return serveMultipart(router, path, files, contractSessionToken(t, manager))
}

// assertAdminDataFixture 断言状态码与 fixture 信封（img-upload/import/download 的
// 业务失败是真实 HTTP 错误码 + ResultStruct 信封）。
func assertAdminDataFixture(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, fixture string) {
	t.Helper()
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

func TestAdminImgUploadHTTPContract(t *testing.T) {
	path := "/api/admin/img-upload"

	t.Run("success stores the image and returns its access url", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipart(t, conn, router, path, map[string][]byte{"file": contractTinyPNG})
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		response := decodeContractEnvelope(t, recorder)
		if response.Code != 0 || response.MessageCode != "upload.success" {
			t.Fatalf("envelope = %#v, want success with upload.success", response)
		}
		result := decodeSiteResult(t, recorder)
		if result["filename"] != "file.png" {
			t.Fatalf("result.filename = %#v, want file.png", result["filename"])
		}
		if result["size"] != float64(len(contractTinyPNG)) {
			t.Fatalf("result.size = %#v, want %d", result["size"], len(contractTinyPNG))
		}
		url, _ := result["url"].(string)
		if !strings.HasPrefix(url, "/file/img/") || !strings.HasSuffix(url, ".png") {
			t.Fatalf("result.url = %q, want a /file/img/... .png access path", url)
		}
		// 清理文件库行与 admin 上传引用（独立连接 + 主库 fileUsage）。
		storedName := strings.TrimPrefix(url, "/file/img/")
		t.Cleanup(func() {
			db4fileconnect.Connect().Where("name = ?", storedName).Delete(&filedata.Entity{})
			conn.Where("file_name = ?", storedName).Delete(&fileUsage.Entity{})
		})
	})

	t.Run("missing file field fails with 400 file.missing", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipart(t, conn, router, path, nil)
		assertAdminDataFixture(t, recorder, http.StatusBadRequest, "admin-img-upload-file-missing.json")
	})

	adminDataGuardScenarios(t, http.MethodPost, path, "admin-img-upload")
}

func TestAdminCreateExportTaskHTTPContract(t *testing.T) {
	path := "/api/admin/data/export"

	t.Run("success enqueues an export task", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataRaw(t, conn, router, http.MethodPost, path, `{"tables":["users"],"format":"json"}`)
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		result := decodeSiteResult(t, recorder)
		taskID, ok := result["taskId"].(float64)
		if !ok || taskID < 1 {
			t.Fatalf("result.taskId = %#v, want a positive numeric task id", result["taskId"])
		}
		var entity taskQueue.Entity
		if err := conn.First(&entity, uint64(taskID)).Error; err != nil {
			t.Fatalf("load enqueued task: %v", err)
		}
		t.Cleanup(func() {
			conn.Delete(&taskQueue.Entity{}, entity.Id)
		})
		if entity.Type != dataservice.TaskTypeExport || entity.Status != taskQueue.StatusPending {
			t.Fatalf("task type/status = %q/%d, want export/pending", entity.Type, entity.Status)
		}
		if !strings.Contains(entity.TaskJson, `"users"`) || !strings.Contains(entity.TaskJson, `"json"`) {
			t.Fatalf("task taskJson = %q, want the submitted tables/format", entity.TaskJson)
		}
		// issue #324 S4：导出创建写操作审计（opt_record）。
		var audit optRecord.Entity
		if err := conn.Where("opt_type = ?", int(optlogger.ExportData)).Order("id desc").First(&audit).Error; err != nil {
			t.Fatalf("load export audit record: %v", err)
		}
		if !strings.Contains(audit.OptInfo, "admin.opt.data.exported") || !strings.Contains(audit.OptInfo, `"users"`) {
			t.Fatalf("export audit optInfo = %q, want admin.opt.data.exported with tables", audit.OptInfo)
		}
		t.Cleanup(func() {
			conn.Delete(&optRecord.Entity{}, audit.Id)
		})
	})

	t.Run("unsupported table fails with exportFailed", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"tables":["secrets"],"format":"json"}`, "admin-data-export-export-failed.json")
	})

	t.Run("unknown format fails request validation", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		serveAdminSiteOK(t, conn, router, http.MethodPost, path,
			`{"tables":["users"],"format":"yaml"}`, "invalid-params.json")
	})

	adminDataGuardScenarios(t, http.MethodPost, path, "admin-data-export")
}

func TestAdminListExportTasksHTTPContract(t *testing.T) {
	path := "/api/admin/data/export/tasks"

	t.Run("success lists the seeded export task", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990002,
			Type:        dataservice.TaskTypeExport,
			Status:      taskQueue.StatusSuccess,
			TaskJson:    `{"tables":["users"],"format":"json","fileName":"export-990002.json","progress":100,"errorCount":0}`,
			RetryCount:  0,
			LastError:   "",
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: contractTaskCreatedAt.Add(time.Minute),
		})
		serveAdminSiteOK(t, conn, router, http.MethodGet, path, "", "admin-data-export-tasks-success.json")
	})

	adminDataGuardScenarios(t, http.MethodGet, path, "admin-data-export-tasks")
}

func TestAdminDownloadExportTaskHTTPContract(t *testing.T) {
	path := "/api/admin/data/export/download/"

	t.Run("success streams the export file bytes", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		// 导出文件从 data/export 相对目录读取：把进程 CWD 切到临时目录，避免污染仓库。
		t.Chdir(t.TempDir())
		if err := os.MkdirAll(filepath.Join("data", "export"), 0o755); err != nil {
			t.Fatalf("mkdir export dir: %v", err)
		}
		content := []byte(`{"ok":true}`)
		if err := os.WriteFile(filepath.Join("data", "export", "contract-export.json"), content, 0o644); err != nil {
			t.Fatalf("write export file: %v", err)
		}
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990005,
			Type:        dataservice.TaskTypeExport,
			Status:      taskQueue.StatusSuccess,
			TaskJson:    `{"tables":["users"],"format":"json","fileName":"contract-export.json","progress":100,"errorCount":0}`,
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: contractTaskCreatedAt,
		})
		recorder := serveAdminDataRaw(t, conn, router, http.MethodGet, path+"990005", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		if recorder.Body.Bytes() == nil || string(recorder.Body.Bytes()) != string(content) {
			t.Fatalf("download body = %q, want %q", recorder.Body.String(), content)
		}
		if disposition := recorder.Header().Get("Content-Disposition"); disposition != `attachment; filename="contract-export.json"` {
			t.Fatalf("Content-Disposition = %q, want the attachment file name", disposition)
		}
		// issue #324 S4：导出下载也写操作审计。
		var audit optRecord.Entity
		if err := conn.Where("opt_type = ?", int(optlogger.ExportData)).Order("id desc").First(&audit).Error; err != nil {
			t.Fatalf("load export download audit record: %v", err)
		}
		if !strings.Contains(audit.OptInfo, "admin.opt.data.exported.download") || !strings.Contains(audit.OptInfo, "contract-export.json") {
			t.Fatalf("export download audit optInfo = %q, want admin.opt.data.exported.download with file name", audit.OptInfo)
		}
		t.Cleanup(func() {
			conn.Delete(&optRecord.Entity{}, audit.Id)
		})
	})

	t.Run("unknown task fails with 404 taskNotFound", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataRaw(t, conn, router, http.MethodGet, path+"88776655", "")
		assertAdminDataFixture(t, recorder, http.StatusNotFound, "admin-data-export-download-not-found.json")
	})

	t.Run("non-export task fails with 404 taskNotFound", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990006,
			Type:        filemigrateservice.TaskTypeFileMigrate,
			Status:      taskQueue.StatusSuccess,
			TaskJson:    `{"lastId":0,"total":0,"processed":0,"failed":0,"clearAfterMigrate":false}`,
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: contractTaskCreatedAt,
		})
		recorder := serveAdminDataRaw(t, conn, router, http.MethodGet, path+"990006", "")
		assertAdminDataFixture(t, recorder, http.StatusNotFound, "admin-data-export-download-not-found.json")
	})

	t.Run("pending task fails with 400 taskNotReady", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990003,
			Type:        dataservice.TaskTypeExport,
			Status:      taskQueue.StatusPending,
			TaskJson:    `{"tables":["users"],"format":"json","fileName":"","progress":0,"errorCount":0}`,
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: time.Time{},
		})
		recorder := serveAdminDataRaw(t, conn, router, http.MethodGet, path+"990003", "")
		assertAdminDataFixture(t, recorder, http.StatusBadRequest, "admin-data-export-download-not-ready.json")
	})

	t.Run("finished task without a file fails with 400 downloadDenied", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		seedContractTask(t, conn, taskQueue.Entity{
			Id:          990004,
			Type:        dataservice.TaskTypeExport,
			Status:      taskQueue.StatusSuccess,
			TaskJson:    `{"tables":["users"],"format":"json","fileName":"","progress":100,"errorCount":0}`,
			CreatedAt:   contractTaskCreatedAt,
			ProcessedAt: contractTaskCreatedAt,
		})
		recorder := serveAdminDataRaw(t, conn, router, http.MethodGet, path+"990004", "")
		assertAdminDataFixture(t, recorder, http.StatusBadRequest, "admin-data-export-download-download-denied.json")
	})

	adminDataGuardScenarios(t, http.MethodGet, path+"88776655", "admin-data-export-download")
}

func TestAdminImportDataHTTPContract(t *testing.T) {
	path := "/api/admin/data/import"

	t.Run("success imports an empty users table and reports zero rows", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		restoreImportDir := dataservice.SetImportDirForTest(t.TempDir())
		defer restoreImportDir()
		recorder := serveAdminDataMultipart(t, conn, router, path, map[string][]byte{"file": []byte(`{"users":[]}`)})
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		envelope := decodeContractEnvelope(t, recorder)
		if envelope.Code != 0 {
			t.Fatalf("code = %d, want 0", envelope.Code)
		}
		var result struct {
			TaskID uint64 `json:"taskId"`
			Status string `json:"status"`
		}
		if err := json.Unmarshal(envelope.Result, &result); err != nil {
			t.Fatalf("decode import task result: %v", err)
		}
		if result.TaskID == 0 || result.Status != "pending" {
			t.Fatalf("import result = %+v, want pending task", result)
		}
	})

	t.Run("missing file field fails with 400 importFailed", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipart(t, conn, router, path, nil)
		assertAdminDataFixture(t, recorder, http.StatusBadRequest, "admin-data-import-import-failed.json")
	})

	t.Run("non-JSON content fails with 400 importInvalidFormat", func(t *testing.T) {
		conn, router := setupAdminDataContractTest(t)
		recorder := serveAdminDataMultipart(t, conn, router, path, map[string][]byte{"file": []byte("hello")})
		assertAdminDataFixture(t, recorder, http.StatusBadRequest, "admin-data-import-invalid-format.json")
	})

	adminDataGuardScenarios(t, http.MethodPost, path, "admin-data-import")
}
