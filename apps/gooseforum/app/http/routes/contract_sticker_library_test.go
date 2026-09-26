package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupStickerLibraryContractTest(t *testing.T) (*gorm.DB, *gin.Engine) {
	t.Helper()
	conn, router := setupAdminStickersContractTest(t)
	conn.Where("1 = 1").Delete(&sticker.LibraryEntry{})
	conn.Where("1 = 1").Delete(&sticker.LibraryOwner{})
	router.POST("/api/forum/stickers/resolve", middleware.RateLimit(middleware.RateLimitStickerList), UpLimitedJsonReq(64<<10, api.ResolveStickers))
	group := router.Group("/api/forum", middleware.CSRFProtection, middleware.JWTAuthCheck)
	group.GET("my-stickers", middleware.NoUpdateUserActivity, UpButterReq(api.MyStickers))
	group.POST("my-sticker-save", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(64<<10, api.SaveMySticker))
	group.POST("my-sticker-delete", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(64<<10, api.DeleteMySticker))
	group.POST("my-stickers-order", middleware.CheckWritableAccount, middleware.RateLimit(middleware.RateLimitInteract), UpLimitedJsonReq(64<<10, api.OrderMyStickers))
	return conn, router
}

func TestPersonalStickerHTTPContract(t *testing.T) {
	conn, router := setupStickerLibraryContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	otherToken := contractSessionToken(t, other)
	seedContractSticker(t, conn, contractStickerID, "smile", "stickers/9f1c2d3e-0000-4000-8000-000000000001.png", 1, true)
	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"smile"}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-save-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-stickers", "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "forum-sticker-list-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-stickers", "", otherToken)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "admin-sticker-list-empty.json"))
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/stickers/resolve", `{"names":["smile","missing","a.b","🙂","smile"]}`, "")
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "forum-sticker-list-success.json"))

	// Repeated collection renames only this membership and never duplicates it.
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"smile","displayName":"我的备注"}`, token)
	if !strings.Contains(response.Body.String(), "我的备注") {
		t.Fatalf("rename: %s", response.Body.String())
	}
	var count int64
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", owner.Id).Count(&count)
	if count != 1 {
		t.Fatalf("duplicate collect created %d entries", count)
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/stickers/resolve", `{"names":["smile"]}`, "")
	if strings.Contains(response.Body.String(), "我的备注") {
		t.Fatal("resolve leaked a private label")
	}

	seedContractSticker(t, conn, contractStickerSecondID, "wave", "stickers/wave.png", 2, true)
	serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"wave"}`, token)
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-stickers-order", `{"names":["wave","smile"]}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-stickers", "", token)
	if strings.Index(response.Body.String(), `"name":"wave"`) > strings.Index(response.Body.String(), `"name":"smile"`) {
		t.Fatal("saved order not retained")
	}
	for _, body := range []string{`{"names":["smile"]}`, `{"names":["smile","smile"]}`, `{"names":["smile","foreign"]}`} {
		response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-stickers-order", body, token)
		if !strings.Contains(response.Body.String(), "common.request.invalidParams") {
			t.Fatalf("bad order accepted: %s", response.Body.String())
		}
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-delete", `{"name":"smile"}`, otherToken)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", owner.Id).Count(&count)
	if count != 2 {
		t.Fatal("another account removed the owner's entry")
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-delete", `{"name":"smile"}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	if entity, err := sticker.GetByName("smile"); err != nil || entity.Id == 0 {
		t.Fatal("membership removal deleted an asset")
	}
}

func TestPersonalStickerUploadLifecycleHTTPContract(t *testing.T) {
	conn, router := setupStickerLibraryContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	file, err := filedata.SaveFileFromUpload(owner.Id, contractTinyPNG, "personal.png", "stickers")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filedata.DeleteByName(file.Name) })
	body := fmt.Sprintf(`{"fileName":%q,"displayName":"喜欢的表情"}`, "https://forum.example.test/file/img/"+file.Name)
	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, contractSessionToken(t, other))
	if !strings.Contains(response.Body.String(), "sticker.imageRequired") {
		t.Fatalf("foreign upload accepted: %s", response.Body.String())
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, token)
	var envelope struct {
		Code   int                        `json:"code"`
		Result stickerservice.StickerItem `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	item := envelope.Result
	if envelope.Code != 0 || item.IsOfficial || len(item.Name) != 50 || !strings.HasPrefix(item.Name, "u_") {
		t.Fatalf("upload result: %s", response.Body.String())
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, token)
	if !strings.Contains(response.Body.String(), item.Name) {
		t.Fatal("upload retry created a new token")
	}
	for _, endpoint := range []string{"/api/forum/stickers", "/api/admin/stickers"} {
		response = serveAdminStickersRaw(t, conn, router, http.MethodGet, endpoint, "")
		if strings.Contains(response.Body.String(), item.Name) {
			t.Fatal("directory enumerated a private asset")
		}
	}
	for _, mutation := range []struct{ path, body string }{
		{"/api/admin/sticker-save", fmt.Sprintf(`{"id":%d,"name":"renamed","isEnabled":true}`, item.ID)},
		{"/api/admin/sticker-delete", fmt.Sprintf(`{"id":%d}`, item.ID)},
	} {
		response = serveAdminStickersRaw(t, conn, router, http.MethodPost, mutation.path, mutation.body)
		if !strings.Contains(response.Body.String(), "admin.sticker.notFound") {
			t.Fatalf("admin changed personal asset: %s", response.Body.String())
		}
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", fmt.Sprintf(`{"stickerName":%q}`, item.Name), contractSessionToken(t, other))
	if !strings.Contains(response.Body.String(), item.Name) {
		t.Fatal("shared token could not be collected")
	}
	if err := stickerservice.CloseLibrary(context.Background(), owner.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := stickerservice.SaveToLibrary(context.Background(), owner.Id, stickerservice.LibrarySaveInput{StickerName: item.Name}); !errors.Is(err, sticker.ErrLibraryClosed) {
		t.Fatalf("closed library accepted stale write: %v", err)
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/stickers/resolve", fmt.Sprintf(`{"names":[%q]}`, item.Name), "")
	if !strings.Contains(response.Body.String(), item.Name) {
		t.Fatal("account closure broke shared history")
	}
	var usages int64
	conn.Model(&fileUsage.Entity{}).Where("target_type = ? AND target_id = ?", fileUsage.TargetSticker, item.ID).Count(&usages)
	if usages != 1 {
		t.Fatalf("asset lost its file reference: %d", usages)
	}
	var memberships int64
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", owner.Id).Count(&memberships)
	if memberships != 0 {
		t.Fatal("closed account retains private library")
	}
}

func TestPersonalStickerGuardsAndValidationHTTPContract(t *testing.T) {
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodGet, "/api/forum/my-stickers"},
		{http.MethodPost, "/api/forum/my-sticker-save"},
		{http.MethodPost, "/api/forum/my-sticker-delete"},
		{http.MethodPost, "/api/forum/my-stickers-order"},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			conn, router := setupStickerLibraryContractTest(t)
			response := serveAuthSecurityJSON(router, endpoint.method, endpoint.path, `{}`, "")
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("unauthenticated status %d", response.Code)
			}
			if endpoint.method == http.MethodGet {
				return
			}
			user := createHTTPContractUser(t, conn, contractTestID())
			conn.Model(user).Update("is_frozen", users.StatusFrozen)
			response = serveAuthSecurityJSON(router, endpoint.method, endpoint.path, `{}`, contractSessionToken(t, user))
			if response.Code != http.StatusForbidden {
				t.Fatalf("frozen status %d", response.Code)
			}
		})
	}
	conn, router := setupStickerLibraryContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, user)
	for _, body := range []string{`{}`, `{"stickerName":"a","fileName":"a.png"}`, `{"stickerName":"a b"}`, `{"stickerName":"smile","displayName":"` + strings.Repeat("长", 65) + `"}`} {
		response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, token)
		if !strings.Contains(response.Body.String(), "common.request.invalidParams") {
			t.Fatalf("invalid save accepted: %s", response.Body.String())
		}
	}
	for _, body := range []string{`{"names":["a"` + strings.Repeat(`,"a"`, 200) + `]}`, `{"names":["bad token"` + strings.Repeat(`,"bad token"`, 200) + `]}`} {
		response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/stickers/resolve", body, "")
		if !strings.Contains(response.Body.String(), "common.request.invalidParams") {
			t.Fatalf("invalid resolve accepted: %s", response.Body.String())
		}
	}
	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/stickers/resolve", `{"names":["`+strings.Repeat("x", 70<<10)+`"]}`, "")
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized body status = %d", response.Code)
	}
	file := &filedata.Entity{Name: "pending-sticker-test.png", Type: "image/png", Size: 1, UserId: user.Id, StorageStatus: filedata.StorageStatusPending}
	if err := db4fileconnect.Connect().Create(file).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db4fileconnect.Connect().Delete(file) })
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"fileName":"pending-sticker-test.png"}`, token)
	if !strings.Contains(response.Body.String(), "sticker.imageRequired") {
		t.Fatal("pending file accepted")
	}
}

func TestPersonalStickerUploadQuotaAndAtomicUsage(t *testing.T) {
	conn, router := setupStickerLibraryContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	file, err := filedata.SaveFileFromUpload(owner.Id, contractTinyPNG, "quota.png", "stickers")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = filedata.DeleteByName(file.Name) })
	body := fmt.Sprintf(`{"fileName":%q}`, file.Name)
	// Failing file usage registration must roll back both asset and membership.
	if err := conn.Callback().Create().Before("gorm:create").Register("fail_personal_usage", func(tx *gorm.DB) {
		if tx.Statement.Table == "file_usages" {
			_ = tx.AddError(errors.New("forced usage failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Create().Remove("fail_personal_usage") })
	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, token)
	if !strings.Contains(response.Body.String(), "common.operation.failed") {
		t.Fatalf("forced failure: %s", response.Body.String())
	}
	var count int64
	conn.Model(&sticker.Entity{}).Where("created_by = ? AND is_official = ?", owner.Id, false).Count(&count)
	if count != 0 {
		t.Fatal("failed usage retained asset")
	}
	conn.Model(&sticker.LibraryEntry{}).Where("user_id = ?", owner.Id).Count(&count)
	if count != 0 {
		t.Fatal("failed usage retained membership")
	}
	if err := conn.Callback().Create().Remove("fail_personal_usage"); err != nil {
		t.Fatal(err)
	}
	// Retained-asset quota remains bounded even after every membership is removed.
	assets := make([]sticker.Entity, stickerservice.MaxPersonalUploads)
	for index := range assets {
		assets[index] = sticker.Entity{Name: fmt.Sprintf("quota_asset_%d", index), FileName: fmt.Sprintf("quota-%d.png", index), CreatedBy: owner.Id, IsEnabled: true}
	}
	if err := conn.CreateInBatches(&assets, 100).Error; err != nil {
		t.Fatal(err)
	}
	if err := conn.Model(&sticker.Entity{}).Where("created_by = ?", owner.Id).Update("is_official", false).Error; err != nil {
		t.Fatal(err)
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", body, token)
	if !strings.Contains(response.Body.String(), "sticker.uploadQuota") {
		t.Fatalf("upload quota not enforced: %s", response.Body.String())
	}
	// Per-library quota has a stable, actionable contract response.
	entries := make([]sticker.LibraryEntry, stickerservice.MaxLibraryItems)
	for index := range entries {
		entries[index] = sticker.LibraryEntry{UserID: owner.Id, StickerID: assets[index].Id, SortOrder: index}
	}
	if err := conn.CreateInBatches(&entries, 100).Error; err != nil {
		t.Fatal(err)
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", fmt.Sprintf(`{"stickerName":%q}`, assets[stickerservice.MaxLibraryItems].Name), token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-library-full.json"))
}

func TestPersonalStickerDisabledMemberHTTPContract(t *testing.T) {
	conn, router := setupStickerLibraryContractTest(t)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	seedContractSticker(t, conn, contractStickerID, "smile", "stickers/smile.png", 1, true)
	response := serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"smile"}`, token)
	if !strings.Contains(response.Body.String(), `"isEnabled":true`) {
		t.Fatalf("enabled availability missing: %s", response.Body.String())
	}
	if err := stickerservice.Save(context.Background(), owner.Id, stickerservice.SaveInput{Id: contractStickerID, Name: "smile", IsEnabled: false}); err != nil {
		t.Fatal(err)
	}
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-stickers", "", token)
	if !strings.Contains(response.Body.String(), `"name":"smile"`) || !strings.Contains(response.Body.String(), `"isEnabled":false`) {
		t.Fatalf("disabled member should remain visibly unavailable: %s", response.Body.String())
	}
	for _, request := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/forum/stickers", ""},
		{http.MethodPost, "/api/forum/stickers/resolve", `{"names":["smile"]}`},
	} {
		response = serveAuthSecurityJSON(router, request.method, request.path, request.body, "")
		if strings.Contains(response.Body.String(), `"name":"smile"`) {
			t.Fatal("disabled asset was publicly available")
		}
	}
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"smile"}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-unavailable.json"))
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-stickers-order", `{"names":["smile"]}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-delete", `{"name":"smile"}`, token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "my-sticker-action-success.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, "/api/forum/my-stickers", "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "admin-sticker-list-empty.json"))
	// Re-enabling restores availability, and permanent deletion cleans memberships.
	if err := stickerservice.Save(context.Background(), owner.Id, stickerservice.SaveInput{Id: contractStickerID, Name: "smile", IsEnabled: true}); err != nil {
		t.Fatal(err)
	}
	serveAuthSecurityJSON(router, http.MethodPost, "/api/forum/my-sticker-save", `{"stickerName":"smile"}`, token)
	if err := stickerservice.Delete(context.Background(), contractStickerID); err != nil {
		t.Fatal(err)
	}
	var members int64
	if err := conn.Model(&sticker.LibraryEntry{}).Where("sticker_id = ?", contractStickerID).Count(&members).Error; err != nil || members != 0 {
		t.Fatalf("deleted asset retains members: count=%d err=%v", members, err)
	}
}
