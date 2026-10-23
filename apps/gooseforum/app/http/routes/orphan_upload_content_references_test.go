package routes

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// seedUploadedImage 写入一张属于 ownerID 的 ready 图片并登记归属行，返回访问地址与对象名。
// 文件创建时间回溯到 TTL 之外，这样只有真正的内容引用才能让它免于回收。
func seedUploadedImage(t *testing.T, conn *gorm.DB, ownerID uint64) (image, name string) {
	t.Helper()
	image, name = saveContractImage(t, ownerID)
	if err := fileusageservice.AddUploadOwner(ownerID, name); err != nil {
		t.Fatalf("add upload owner: %v", err)
	}
	if err := fileDB.Connect().Exec("UPDATE file_data SET created_at = ? WHERE name = ?", time.Now().Add(-8*24*time.Hour), name).Error; err != nil {
		t.Fatalf("backdate %s: %v", name, err)
	}
	t.Cleanup(func() { conn.Where("file_name = ?", name).Delete(&fileUsage.Entity{}) })
	return image, name
}

func countActiveUsages(t *testing.T, conn *gorm.DB, name, targetType, usageType string) int64 {
	t.Helper()
	var count int64
	if err := conn.Model(&fileUsage.Entity{}).
		Where("file_name = ? AND target_type = ? AND usage_type = ? AND status = ?",
			name, targetType, usageType, fileUsage.UsageStatusActive).
		Count(&count).Error; err != nil {
		t.Fatalf("count usages: %v", err)
	}
	return count
}

func assertCleanupKeepsFile(t *testing.T, name string) {
	t.Helper()
	if _, err := fileusageservice.CleanupOrphanUploads(context.Background(), time.Now().Add(-fileusageservice.OrphanUploadTTL), 200); err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	if filedata.GetByName(name).Id == 0 {
		t.Fatalf("file %s with a content reference was reclaimed", name)
	}
}

// TestChatImageMessageRegistersMediaReference 覆盖 P0：私信图片只把 URL 存进
// messages.content，必须同时登记 ACTIVE 内容引用，否则收件人读不到、文件也会被回收。
func TestChatImageMessageRegistersMediaReference(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	registerImageRoute(router)
	sender := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	image, name := seedUploadedImage(t, conn, sender.Id)

	body := fmt.Sprintf(`{"peerId":%d,"content":%q,"msgType":2}`, peer.Id, image)
	if rec := serveJSON(router, "/api/forum/chat/send", body, contractSessionToken(t, sender)); rec.Code != http.StatusOK {
		t.Fatalf("send: %d %s", rec.Code, rec.Body)
	}
	var message messages.Entity
	if err := conn.Where("sender_id = ?", sender.Id).First(&message).Error; err != nil {
		t.Fatalf("load message: %v", err)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia); count != 1 {
		t.Fatalf("chat media reference rows = %d, want 1", count)
	}
	var usage fileUsage.Entity
	if err := conn.Where("file_name = ? AND usage_type = ?", name, fileUsage.UsageChatMedia).First(&usage).Error; err != nil {
		t.Fatalf("load chat media usage: %v", err)
	}
	if usage.TargetId != message.Id || usage.UserId != sender.Id {
		t.Fatalf("chat media usage = %+v, want target %d owner %d", usage, message.Id, sender.Id)
	}
	for label, token := range map[string]string{"recipient": contractSessionToken(t, peer), "anonymous": ""} {
		if res := getImage(router, image, token); res.Code != http.StatusOK {
			t.Fatalf("%s read of chat image = %d, want 200", label, res.Code)
		}
	}
	assertCleanupKeepsFile(t, name)
}

// TestProfileCoverRegistersReference 覆盖 P0：个人主页封面同样只存 URL（users.profile_cover_url）。
func TestProfileCoverRegistersReference(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	registerImageRoute(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	visitor := createHTTPContractUser(t, conn, contractTestID())
	image, name := seedUploadedImage(t, conn, owner.Id)

	body := fmt.Sprintf(`{"profileCoverUrl":%q}`, image)
	if rec := serveJSON(router, "/api/set-user-profile-cover", body, contractSessionToken(t, owner)); rec.Code != http.StatusOK {
		t.Fatalf("set profile cover: %d %s", rec.Code, rec.Body)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetUser, fileUsage.UsageCover); count != 1 {
		t.Fatalf("profile cover reference rows = %d, want 1", count)
	}
	if res := getImage(router, image, contractSessionToken(t, visitor)); res.Code != http.StatusOK {
		t.Fatalf("visitor read of profile cover = %d, want 200", res.Code)
	}
	assertCleanupKeepsFile(t, name)

	// 换掉封面后旧文件的引用消失，重新变成可回收的孤儿。
	if rec := serveJSON(router, "/api/set-user-profile-cover", `{"profileCoverUrl":""}`, contractSessionToken(t, owner)); rec.Code != http.StatusOK {
		t.Fatalf("clear profile cover: %d %s", rec.Code, rec.Body)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetUser, fileUsage.UsageCover); count != 0 {
		t.Fatalf("cleared cover kept %d reference rows", count)
	}
}

func registerImageRoute(router *gin.Engine) {
	router.GET("/file/img/*filename", api.GetFileByFileName)
}

// hideFileDataTable 让文件库查询失败，模拟一次瞬时故障；返回的清理函数在用例结束时恢复。
func hideFileDataTable(t *testing.T) {
	t.Helper()
	conn := fileDB.Connect()
	if err := conn.Migrator().RenameTable("file_data", "hotfix_file_data_unavailable"); err != nil {
		t.Fatalf("hide file_data: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Migrator().RenameTable("hotfix_file_data_unavailable", "file_data"); err != nil {
			t.Fatalf("restore file_data: %v", err)
		}
	})
}

// TestChatSendFailsClosedWhenFileLookupFails 是 P1 回归：文件库读不到归属时不能当成
// 「不是本人上传」，否则消息会提交但没有引用——收件人 404，文件 7 天后被回收。
func TestChatSendFailsClosedWhenFileLookupFails(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	registerImageRoute(router)
	sender := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	image, name := seedUploadedImage(t, conn, sender.Id)
	hideFileDataTable(t)

	body := fmt.Sprintf(`{"peerId":%d,"content":%q,"msgType":2}`, peer.Id, image)
	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/forum/chat/send", body, contractSessionToken(t, sender)))
	if envelope.Code == 0 || envelope.MessageCode != "chat.send.failed" {
		t.Fatalf("media send under a file-db failure: code=%d messageCode=%q", envelope.Code, envelope.MessageCode)
	}
	var messageRows int64
	if err := conn.Model(&messages.Entity{}).Where("sender_id = ?", sender.Id).Count(&messageRows).Error; err != nil {
		t.Fatalf("count messages: %v", err)
	}
	if messageRows != 0 {
		t.Fatalf("failed media send committed %d message rows", messageRows)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia); count != 0 {
		t.Fatalf("failed media send left %d reference rows", count)
	}
}

// TestProfileCoverFailsClosedWhenFileLookupFails 是 P1 回归：文件库故障时既不能吊销旧
// 封面引用，也不能把新的 profile_cover_url 落库。
func TestProfileCoverFailsClosedWhenFileLookupFails(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	registerImageRoute(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	image, name := seedUploadedImage(t, conn, owner.Id)
	token := contractSessionToken(t, owner)
	if rec := serveJSON(router, "/api/set-user-profile-cover", fmt.Sprintf(`{"profileCoverUrl":%q}`, image), token); rec.Code != http.StatusOK {
		t.Fatalf("set profile cover: %d %s", rec.Code, rec.Body)
	}
	hideFileDataTable(t)

	envelope := decodeContractEnvelope(t, serveJSON(router, "/api/set-user-profile-cover", `{"profileCoverUrl":"/file/img/2026/10/01/next-cover.png"}`, token))
	if envelope.Code == 0 || envelope.MessageCode != "user.updateFailed" {
		t.Fatalf("cover update under a file-db failure: code=%d messageCode=%q", envelope.Code, envelope.MessageCode)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetUser, fileUsage.UsageCover); count != 1 {
		t.Fatalf("failed cover update revoked the existing reference (%d rows)", count)
	}
}

// seedCdnPublicPrefix 打开 publicUrlPrefix 并在用例结束后还原行与缓存。
func seedCdnPublicPrefix(t *testing.T, conn *gorm.DB, prefix string) {
	t.Helper()
	restorePageConfigAfterTest(t, conn, pageConfig.StorageSettingsPage)
	persistHTTPContractConfig(t, conn, pageConfig.StorageSettingsPage, pageConfig.StorageSettingsStorage{
		Provider:        "local",
		PublicUrlPrefix: prefix,
	})
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearStorageSettingsConfigCache)
}

// TestProfileCoverResolvesCdnURLOutsideTransaction 钉住「解析必须在事务外」：封面是 CDN 绝对
// 地址，解析要读存储配置（pageConfig）；若放进事务里，单连接的测试库会与外层事务互锁。
func TestProfileCoverResolvesCdnURLOutsideTransaction(t *testing.T) {
	conn, router := setupAccountContractTest(t)
	registerImageRoute(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	_, name := seedUploadedImage(t, conn, owner.Id)
	// 造文件会顺带预热存储配置缓存；必须最后再换配置并清缓存，才能让「事务内解析」暴露成互锁。
	const prefix = "https://cdn.example.test/media"
	seedCdnPublicPrefix(t, conn, prefix)

	body := fmt.Sprintf(`{"profileCoverUrl":%q}`, prefix+"/"+name)
	if rec := serveJSON(router, "/api/set-user-profile-cover", body, contractSessionToken(t, owner)); rec.Code != http.StatusOK {
		t.Fatalf("set CDN profile cover: %d %s", rec.Code, rec.Body)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetUser, fileUsage.UsageCover); count != 1 {
		t.Fatalf("CDN cover reference rows = %d, want 1", count)
	}
	assertCleanupKeepsFile(t, name)
}

// TestChatImageMessageResolvesCdnURLOutsideTransaction 同上，覆盖私信发送事务：正文里的
// CDN 绝对图片地址必须在事务外解析。
func TestChatImageMessageResolvesCdnURLOutsideTransaction(t *testing.T) {
	conn, router := setupNotificationChatContractTest(t)
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	registerImageRoute(router)
	sender := createHTTPContractUser(t, conn, contractTestID())
	peer := createHTTPContractUser(t, conn, contractTestID())
	_, name := seedUploadedImage(t, conn, sender.Id)
	const prefix = "https://cdn.example.test/media"
	seedCdnPublicPrefix(t, conn, prefix)

	body := fmt.Sprintf(`{"peerId":%d,"content":%q,"msgType":2}`, peer.Id, prefix+"/"+name)
	if rec := serveJSON(router, "/api/forum/chat/send", body, contractSessionToken(t, sender)); rec.Code != http.StatusOK {
		t.Fatalf("send CDN image message: %d %s", rec.Code, rec.Body)
	}
	if count := countActiveUsages(t, conn, name, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia); count != 1 {
		t.Fatalf("CDN chat media reference rows = %d, want 1", count)
	}
	assertCleanupKeepsFile(t, name)
}
