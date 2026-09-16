package routes

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 本文件覆盖 SiteManager 权限组 stickers/sticker-save/sticker-delete/sticker-import
// 4 条管理路由 + 公开 forum/stickers 列表的契约测试（MADR 0030，issue #277 后续切片）。
// stickers（主库）与 filedata（独立 db4fileconnect 库）、file_usages 行在各子测试间
// 清删；import 的文件仅写 BLOB（本地 provider），不落盘。存储公开前缀缓存在测试
// 首尾清理，保证 url 断言走默认 /file/img 路径。

const (
	contractStickerID       uint64 = 9101
	contractStickerSecondID uint64 = 9102
)

// setupAdminStickersContractTest 在共享 harness（setupHTTPContractTest）之上注册
// SiteManager 权限组 4 条 sticker 路由与公开 stickers 列表，中间件链与 route4api.go
// 的生产注册保持一致（JWTAuthCheck + CheckWritableAccount 公共链 +
// CheckPermission(SiteManager) 子组；公开列表无鉴权中间件）。
func setupAdminStickersContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupHTTPContractTest(t)
	if err := conn.AutoMigrate(
		&rolePermissionRs.Entity{},
		&sticker.Entity{},
		&fileUsage.Entity{},
	); err != nil {
		t.Fatalf("migrate admin stickers contract tables: %v", err)
	}
	// filedata 走独立的 db4fileconnect 连接（测试模式同样各自 :memory:），
	// 需在文件库上单独迁移。
	if err := db4fileconnect.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate filedata contract table: %v", err)
	}
	conn.Where("1 = 1").Delete(&sticker.Entity{})
	conn.Where("1 = 1").Delete(&fileUsage.Entity{})
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearStorageSettingsConfigCache)

	stickersAPI := router.Group("/api/admin",
		middleware.JWTAuthCheck,
		middleware.CheckWritableAccount,
		middleware.CheckPermission(permission.SiteManager),
	)
	stickersAPI.GET("/stickers", UpButterReq(api.StickerList))
	stickersAPI.POST("/sticker-save", UpButterReq(api.SaveSticker))
	stickersAPI.POST("/sticker-delete", UpButterReq(api.DeleteSticker))
	stickersAPI.POST("/sticker-import", api.ImportStickerPack)
	router.GET("/api/forum/stickers", middleware.RateLimit(middleware.RateLimitStickerList), ginUpNP(api.PublicStickerList))
	return conn, router
}

// serveAdminStickersRaw 以 SiteManager 身份调用路由，返回原始 recorder 供结构化断言。
func serveAdminStickersRaw(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	recorder := serveAuthSecurityJSON(router, method, path, body, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	return recorder
}

// serveAdminStickersOK 以 SiteManager 身份调用路由并断言 HTTP 200 + fixture 信封。
func serveAdminStickersOK(t *testing.T, conn *gorm.DB, router *gin.Engine, method, path, body, fixture string) {
	t.Helper()
	recorder := serveAdminStickersRaw(t, conn, router, method, path, body)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

// adminStickersGuardScenarios 跑 4 条管理路由公共的中间件守卫场景：
// 未登录 401 / 冻结账号 403 / 无 SiteManager 权限 403（params.permission="站点管理"）。
func adminStickersGuardScenarios(t *testing.T, method, path string) {
	t.Run("missing session returns 401", func(t *testing.T) {
		_, router := setupAdminStickersContractTest(t)
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated status = %d, want 401: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "auth-required.json"))
	})

	t.Run("frozen account returns 403", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
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
		conn, router := setupAdminStickersContractTest(t)
		user := createHTTPContractUser(t, conn, contractTestID())
		recorder := serveAuthSecurityJSON(router, method, path, `{}`, contractSessionToken(t, user))
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("permission denied status = %d, want 403: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-ai-summary-settings-permission-denied.json"))
	})
}

// seedContractSticker 造固定 ID 表情包行（cleanup 按 ID 删除）。禁用行走显式
// map Updates：GORM Create 跳过布尔零值，直接 Create false 会被列 default:true 覆盖。
func seedContractSticker(t *testing.T, conn *gorm.DB, id uint64, name, fileName string, sort int, enabled bool) {
	t.Helper()
	if err := conn.Create(&sticker.Entity{
		Id:       id,
		Name:     name,
		FileName: fileName,
	}).Error; err != nil {
		t.Fatalf("seed contract sticker %d: %v", id, err)
	}
	if err := conn.Model(&sticker.Entity{}).Where("id = ?", id).
		Updates(map[string]any{"sort_order": sort, "is_enabled": enabled}).Error; err != nil {
		t.Fatalf("update contract sticker %d: %v", id, err)
	}
	t.Cleanup(func() {
		conn.Where("id = ?", id).Delete(&sticker.Entity{})
	})
}

// contractStickerZip 在内存中构造导入用 zip 压缩包（不落盘）。
func contractStickerZip(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for name, content := range entries {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buffer.Bytes()
}

// serveStickerImport 以 multipart/form-data 提交导入压缩包
// （ImportStickerPack 为直接 gin handler，不经 UpButterReq）；
// archive 为 nil 时不携带 file 字段，覆盖 upload.file.missing 分支。
func serveStickerImport(router http.Handler, path string, archive []byte, token string) *httptest.ResponseRecorder {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if archive != nil {
		part, err := writer.CreateFormFile("file", "pack.zip")
		if err != nil {
			return httptest.NewRecorder()
		}
		_, _ = part.Write(archive)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, path, body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

// serveAdminStickersImportOK 以 SiteManager 身份提交导入压缩包并断言 fixture 信封
// （导入结果桶不含动态字段，可整体精确匹配）。
func serveAdminStickersImportOK(t *testing.T, conn *gorm.DB, router *gin.Engine, archive []byte, fixture string) {
	t.Helper()
	manager := createContractSiteManager(t, conn)
	recorder := serveStickerImport(router, "/api/admin/sticker-import", archive, contractSessionToken(t, manager))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, fixture))
}

func TestForumStickerListHTTPContract(t *testing.T) {
	path := "/api/forum/stickers"

	t.Run("success returns enabled stickers only", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "hidden", "stickers/9f1c2d3e-0000-4000-8000-000000000002.png", 2, false)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, path, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "forum-sticker-list-success.json"))
	})

	t.Run("empty library returns an empty array", func(t *testing.T) {
		_, router := setupAdminStickersContractTest(t)
		recorder := serveAuthSecurityJSON(router, http.MethodGet, path, "", "")
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-sticker-list-empty.json"))
	})
}

func TestAdminStickerListHTTPContract(t *testing.T) {
	path := "/api/admin/stickers"

	t.Run("success returns every sticker including disabled rows", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
		seedContractSticker(t, conn, contractStickerSecondID, "梗图-冲", "", 2, false)
		serveAdminStickersOK(t, conn, router, http.MethodGet, path, "", "admin-sticker-list-success.json")
	})

	t.Run("empty library returns an empty array", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		serveAdminStickersOK(t, conn, router, http.MethodGet, path, "", "admin-sticker-list-empty.json")
	})

	adminStickersGuardScenarios(t, http.MethodGet, path)
}

func TestAdminStickerSaveHTTPContract(t *testing.T) {
	path := "/api/admin/sticker-save"

	t.Run("success creates the sticker", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		manager := createContractSiteManager(t, conn)
		file, err := filedata.SaveFileFromUpload(manager.Id, contractTinyPNG, "single.png", "test")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = filedata.DeleteByName(file.Name) })
		recorder := serveAuthSecurityJSON(router, http.MethodPost, path,
			fmt.Sprintf(`{"id":0,"name":"contract_new","fileName":"/file/img/%s","sortOrder":7,"isEnabled":true}`, file.Name), contractSessionToken(t, manager))
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-sticker-action-success.json"))
		var references int64
		conn.Model(&fileUsage.Entity{}).Where("file_name = ? AND target_type = ? AND status = ?", file.Name, fileUsage.TargetSticker, fileUsage.UsageStatusActive).Count(&references)
		if references != 1 {
			t.Fatalf("active references = %d", references)
		}
		created := sticker.GetByName("contract_new")
		if created.Id == 0 || created.SortOrder != 7 || !created.IsEnabled {
			t.Fatalf("created sticker = %#v, want contract_new enabled sortOrder 7", created)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", created.Id).Delete(&sticker.Entity{})
		})
	})

	t.Run("success updates an existing sticker", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "", 1, true)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path,
			`{"id":9101,"name":"smile","sortOrder":9,"isEnabled":false}`,
			"admin-sticker-action-success.json")
		updated := sticker.GetById(contractStickerID)
		if updated.SortOrder != 9 || updated.IsEnabled {
			t.Fatalf("updated sticker = %#v, want sortOrder 9 disabled", updated)
		}
	})

	t.Run("duplicate name returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "", 1, true)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path,
			`{"id":0,"name":"smile","sortOrder":0,"isEnabled":true}`,
			"admin-sticker-name-exists.json")
	})

	t.Run("whitespace-only name returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path,
			`{"id":0,"name":"   ","sortOrder":0,"isEnabled":true}`,
			"admin-sticker-name-required.json")
	})

	t.Run("unknown id returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path,
			`{"id":987654321,"name":"contract_new","sortOrder":0,"isEnabled":true}`,
			"admin-sticker-not-found.json")
	})

	adminStickersGuardScenarios(t, http.MethodPost, path)
}

func TestAdminStickerDeleteHTTPContract(t *testing.T) {
	path := "/api/admin/sticker-delete"

	t.Run("success hard-deletes the sticker", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path, `{"id":9101}`, "admin-sticker-action-success.json")
		if got := sticker.GetById(contractStickerID); got.Id != 0 {
			t.Fatalf("sticker %d still readable after delete", contractStickerID)
		}
	})

	t.Run("unknown sticker returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		serveAdminStickersOK(t, conn, router, http.MethodPost, path, `{"id":987654321}`, "admin-sticker-not-found.json")
	})

	adminStickersGuardScenarios(t, http.MethodPost, path)
}

func TestAdminStickerImportHTTPContract(t *testing.T) {
	path := "/api/admin/sticker-import"

	t.Run("success imports images and reports per-entry failures", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		archive := contractStickerZip(t, map[string][]byte{
			"smile.png":  contractTinyPNG,
			"notes.txt":  []byte("not an image"),
			"broken.png": []byte("PNG forgery bytes"),
		})
		serveAdminStickersImportOK(t, conn, router, archive, "admin-sticker-import-success.json")
		created := sticker.GetByName("smile")
		if created.Id == 0 || !created.IsEnabled || created.FileName == "" {
			t.Fatalf("imported sticker = %#v, want enabled row named smile", created)
		}
		t.Cleanup(func() {
			conn.Where("id = ?", created.Id).Delete(&sticker.Entity{})
			conn.Where("target_type = ?", fileUsage.TargetSticker).Delete(&fileUsage.Entity{})
			db4fileconnect.Connect().Where("name = ?", created.FileName).Delete(&filedata.Entity{})
		})
	})

	t.Run("missing file returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		manager := createContractSiteManager(t, conn)
		recorder := serveStickerImport(router, path, nil, contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-img-upload-file-missing.json"))
	})

	t.Run("non-zip content returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		serveAdminStickersImportOK(t, conn, router,
			[]byte("this is not a zip archive"), "admin-sticker-import-invalid-zip.json")
	})

	t.Run("oversized archive returns business failure", func(t *testing.T) {
		conn, router := setupAdminStickersContractTest(t)
		manager := createContractSiteManager(t, conn)
		// Store 模式（不压缩）写满超过 32MB 上限的条目，使上传分片体积真实
		// 超限触发 admin.sticker.importTooLarge（控制器在解压前按分片大小拦截）。
		buffer := &bytes.Buffer{}
		writer := zip.NewWriter(buffer)
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: "smile.png", Method: zip.Store})
		if err != nil {
			t.Fatalf("create oversized zip entry: %v", err)
		}
		if _, err := entry.Write(bytes.Repeat([]byte{0x00}, 32<<20+1)); err != nil {
			t.Fatalf("write oversized zip entry: %v", err)
		}
		if err := writer.Close(); err != nil {
			t.Fatalf("close oversized zip writer: %v", err)
		}
		recorder := serveStickerImport(router, path, buffer.Bytes(), contractSessionToken(t, manager))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
		}
		assertFixtureEnvelope(t, decodeContractEnvelope(t, recorder), contractFixture(t, "admin-sticker-import-too-large.json"))
	})

	adminStickersGuardScenarios(t, http.MethodPost, path)
}

func TestStickerReviewRejectsImageLessCreate(t *testing.T) {
	conn, router := setupAdminStickersContractTest(t)
	serveAdminStickersOK(t, conn, router, http.MethodPost, "/api/admin/sticker-save", `{"name":"empty_image","isEnabled":true}`, "admin-img-upload-file-missing.json")
}
func TestStickerReviewUsageFailureRollsBackImport(t *testing.T) {
	conn, router := setupAdminStickersContractTest(t)
	if err := conn.Callback().Create().Before("gorm:create").Register("sticker_usage_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "file_usages" {
			_ = tx.AddError(errors.New("usage unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove("sticker_usage_failure") })
	manager := createContractSiteManager(t, conn)
	result := serveStickerImport(router, "/api/admin/sticker-import", contractStickerZip(t, map[string][]byte{"review.png": contractTinyPNG}), contractSessionToken(t, manager))
	var files int64
	db4fileconnect.Connect().Model(&filedata.Entity{}).Where("user_id = ?", manager.Id).Count(&files)
	if files != 0 {
		t.Fatalf("rollback leaked %d uploaded files", files)
	}
	if bytes.Contains(result.Body.Bytes(), []byte(`"imported":1`)) || sticker.GetByName("review").Id != 0 {
		t.Fatalf("usage failure reported success: %s", result.Body.String())
	}
}

func TestStickerReviewImportBudgetAndSkippedEntries(t *testing.T) {
	_, _ = setupAdminStickersContractTest(t)
	result, err := stickerservice.ImportPack(context.Background(), 0, contractStickerZip(t, map[string][]byte{
		"folder/": nil, ".hidden.png": contractTinyPNG, "__MACOSX/._a.png": contractTinyPNG, "readme.txt": []byte("text"),
	}))
	if err != nil || result.Skipped != 3 || len(result.Failed) != 0 {
		t.Fatalf("skipped result = %+v, %v", result, err)
	}
	entries := map[string][]byte{}
	for i := 0; i < 17; i++ {
		entries[fmt.Sprintf("%02d.png", i)] = make([]byte, filedata.MaxFileSize)
	}
	result, err = stickerservice.ImportPack(context.Background(), 0, contractStickerZip(t, entries))
	if err != nil || result.Imported != 0 || len(result.Failed) != 17 || result.Failed[16].Reason != "archiveTooLarge" {
		t.Fatalf("aggregate size result = %+v, %v", result, err)
	}
}

func TestStickerReviewConcurrentImportNames(t *testing.T) {
	conn, _ := setupAdminStickersContractTest(t)
	var group sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		group.Go(func() {
			_, err := stickerservice.ImportImage(context.Background(), 0, contractTinyPNG, "race.png", "race", 0, false)
			failures <- err
		})
	}
	group.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"race", "race-2"} {
		row := sticker.GetByName(name)
		if row.Id == 0 {
			t.Fatalf("missing %s", name)
		}
		t.Cleanup(func() { _ = filedata.DeleteByName(row.FileName) })
		var references int64
		conn.Model(&fileUsage.Entity{}).Where("target_type = ? AND target_id = ?", fileUsage.TargetSticker, row.Id).Count(&references)
		if references != 1 {
			t.Fatalf("%s references = %d", name, references)
		}
	}
}

func TestStickerReviewPublicQueryFailureIsNotEmptySuccess(t *testing.T) {
	conn, router := setupAdminStickersContractTest(t)
	if err := conn.Callback().Query().Before("gorm:query").Register("sticker_query_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "stickers" {
			_ = tx.AddError(errors.New("database unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Query().Remove("sticker_query_failure") })
	result := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/stickers", "", "")
	if bytes.Contains(result.Body.Bytes(), []byte(`"code":0`)) {
		t.Fatalf("database failure returned success: %s", result.Body.String())
	}
}

func TestStickerReviewImageOwnershipAndDisabledCreate(t *testing.T) {
	conn, router := setupAdminStickersContractTest(t)
	manager := createContractSiteManager(t, conn)
	file, err := filedata.SaveFileFromUpload(manager.Id, contractTinyPNG, "owned.png", "test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filedata.DeleteByName(file.Name) })
	other := createContractSiteManager(t, conn)
	body := fmt.Sprintf(`{"name":"owned","fileName":%q,"isEnabled":false}`, file.Name)
	rejected := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/sticker-save", body, contractSessionToken(t, other))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, rejected), contractFixture(t, "admin-img-upload-file-missing.json"))
	saved := serveAuthSecurityJSON(router, http.MethodPost, "/api/admin/sticker-save", body, contractSessionToken(t, manager))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, saved), contractFixture(t, "admin-sticker-action-success.json"))
	row := sticker.GetByName("owned")
	if row.Id == 0 || row.IsEnabled || row.FileName != file.Name {
		t.Fatalf("disabled row = %+v", row)
	}
}

func TestStickerReviewPublicRateLimit(t *testing.T) {
	conn, router := setupAdminStickersContractTest(t)
	restrictContractRateLimit(t, conn, "sticker.list")
	for range 5 {
		_ = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/stickers", "", "")
	}
	result := serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/stickers", "", "")
	if result.Code != http.StatusTooManyRequests || result.Header().Get("Retry-After") == "" {
		t.Fatalf("rate limit response = %d %s", result.Code, result.Body.String())
	}
	if !bytes.Contains(result.Body.Bytes(), []byte(`"action":"sticker.list"`)) {
		t.Fatal(result.Body.String())
	}
}
