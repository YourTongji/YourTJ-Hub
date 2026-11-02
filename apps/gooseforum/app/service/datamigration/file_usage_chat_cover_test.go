package datamigration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	fileDB "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/db4fileconnect"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/rolePermissionRs"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"gorm.io/gorm"
)

// prepareChatCoverBackfillDB 迁移模块库/文件库，并把存储配置缓存置空（公开前缀按库内配置
// 现读，与迁移进程保持一致）。
func prepareChatCoverBackfillDB(t *testing.T) *gorm.DB {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &messages.Entity{}, &fileUsage.Entity{}, &pageConfig.Entity{}, &rolePermissionRs.Entity{}); err != nil {
		t.Fatalf("migrate module tables: %v", err)
	}
	if err := fileDB.Connect().AutoMigrate(&filedata.Entity{}); err != nil {
		t.Fatalf("migrate file_data: %v", err)
	}
	hotdataserve.ClearStorageSettingsConfigCache()
	t.Cleanup(hotdataserve.ClearStorageSettingsConfigCache)
	return conn
}

// seedLegacyUpload 造一个「上传后从未被引用」的历史文件：ready 行 + 归属行，创建时间回溯到
// TTL 之外，只有内容引用能让它免于回收。
func seedLegacyUpload(t *testing.T, ownerID uint64, name string) {
	t.Helper()
	if _, err := filedata.SaveFile(ownerID, name, "image/png", []byte("png")); err != nil {
		t.Fatalf("seed file %s: %v", name, err)
	}
	if err := fileusageservice.AddUploadOwner(ownerID, name); err != nil {
		t.Fatalf("add upload owner %s: %v", name, err)
	}
	if err := fileDB.Connect().Exec("UPDATE file_data SET created_at = ? WHERE name = ?", time.Now().Add(-8*24*time.Hour), name).Error; err != nil {
		t.Fatalf("backdate %s: %v", name, err)
	}
	t.Cleanup(func() {
		_ = filedata.DeleteByName(name)
		db.Connect().Where("file_name = ?", name).Delete(&fileUsage.Entity{})
	})
}

func assertActiveUsage(t *testing.T, conn *gorm.DB, fileName, targetType, usageType string, targetID, userID uint64) {
	t.Helper()
	var rows []fileUsage.Entity
	if err := conn.Where("file_name = ? AND target_type = ? AND usage_type = ?", fileName, targetType, usageType).Find(&rows).Error; err != nil {
		t.Fatalf("load usage %s: %v", fileName, err)
	}
	if len(rows) != 1 {
		t.Fatalf("usage rows for %s = %d, want 1", fileName, len(rows))
	}
	if rows[0].Status != fileUsage.UsageStatusActive || rows[0].TargetId != targetID || rows[0].UserId != userID {
		t.Fatalf("usage for %s = %+v, want ACTIVE target %d owner %d", fileName, rows[0], targetID, userID)
	}
}

// TestBackfillFileUsagesChatCoverProtectsHistoricalMedia 是 issue #1365 的部署回归：
// 历史私信图片与个人封面只存 URL，回填引用前会被孤儿回收任务直接删掉。
func TestBackfillFileUsagesChatCoverProtectsHistoricalMedia(t *testing.T) {
	conn := prepareChatCoverBackfillDB(t)
	const (
		senderID   = 9_800_500_001
		convID     = 9_800_500_101
		chatID     = 9_800_500_201
		inlineID   = 9_800_500_202
		chatName   = "2026/10/01/legacy-chat.png"
		inlineName = "2026/10/01/legacy-inline.png"
		coverName  = "2026/10/01/legacy-cover.png"
	)
	seedLegacyUpload(t, senderID, chatName)
	seedLegacyUpload(t, senderID, inlineName)
	seedLegacyUpload(t, senderID, coverName)
	messagesRows := []messages.Entity{
		{Id: chatID, ConvId: convID, SenderId: senderID, Content: "/file/img/" + chatName, MsgType: 2},
		{Id: inlineID, ConvId: convID, SenderId: senderID, Content: "看这个 ![](/file/img/" + inlineName + ")", MsgType: 1},
	}
	if err := conn.Create(&messagesRows).Error; err != nil {
		t.Fatalf("seed messages: %v", err)
	}
	user := users.MakeUser(fmt.Sprintf("legacy-cover-%d", senderID), "secret123", fmt.Sprintf("legacy-cover-%d@example.test", senderID))
	user.Id = senderID
	user.ProfileCoverUrl = "/file/img/" + coverName
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("id IN ?", []uint64{chatID, inlineID}).Delete(&messages.Entity{})
		conn.Where("id = ?", senderID).Delete(&users.EntityComplete{})
	})

	for _, name := range []string{chatName, inlineName, coverName} {
		if fileUsage.HasAnyReferences(name) {
			t.Fatalf("seeded legacy file %s already has a content reference", name)
		}
	}

	result := BackfillFileUsagesChatCoverWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}
	if result.Inserted != 3 {
		t.Fatalf("backfill inserted=%d, want 3 (chat image, inline image, cover)", result.Inserted)
	}
	assertActiveUsage(t, conn, chatName, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia, chatID, senderID)
	assertActiveUsage(t, conn, inlineName, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia, inlineID, senderID)
	assertActiveUsage(t, conn, coverName, fileUsage.TargetUser, fileUsage.UsageCover, senderID, senderID)

	if _, err := fileusageservice.CleanupOrphanUploads(context.Background(), time.Now().Add(-fileusageservice.OrphanUploadTTL), 200); err != nil {
		t.Fatalf("CleanupOrphanUploads() error = %v", err)
	}
	for _, name := range []string{chatName, inlineName, coverName} {
		if filedata.GetByName(name).Id == 0 {
			t.Fatalf("historical file %s was reclaimed despite the backfilled reference", name)
		}
	}

	// 幂等：第二次执行不再插入任何行。
	again := BackfillFileUsagesChatCoverWithDB(conn)
	if again.Failed > 0 || again.Inserted != 0 {
		t.Fatalf("second run inserted=%d failed=%d, want 0/0", again.Inserted, again.Failed)
	}
}

// TestBackfillFileUsagesChatCoverResolvesPublicUrlPrefix 覆盖 publicUrlPrefix 直读地址：
// 历史封面存的是 CDN 绝对 URL，回填必须解析出对象名才能登记引用。
func TestBackfillFileUsagesChatCoverResolvesPublicUrlPrefix(t *testing.T) {
	conn := prepareChatCoverBackfillDB(t)
	const (
		userID    = 9_800_500_301
		coverName = "2026/10/01/cdn-cover.png"
		prefix    = "https://cdn.example.test/media"
	)
	seedLegacyUpload(t, userID, coverName)
	storageConfig, err := json.Marshal(pageConfig.StorageSettingsStorage{Provider: "local", PublicUrlPrefix: prefix})
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Create(&pageConfig.Entity{PageType: pageConfig.StorageSettingsPage, Config: string(storageConfig)}).Error; err != nil {
		t.Fatalf("seed storage settings: %v", err)
	}
	t.Cleanup(func() {
		conn.Where("page_type = ?", pageConfig.StorageSettingsPage).Delete(&pageConfig.Entity{})
		hotdataserve.ClearStorageSettingsConfigCache()
	})
	hotdataserve.ClearStorageSettingsConfigCache()

	user := users.MakeUser(fmt.Sprintf("legacy-cdn-cover-%d", userID), "secret123", fmt.Sprintf("legacy-cdn-cover-%d@example.test", userID))
	user.Id = userID
	user.ProfileCoverUrl = prefix + "/" + coverName
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() { conn.Where("id = ?", userID).Delete(&users.EntityComplete{}) })

	// 文本消息里的 markdown 图片也可以是 CDN 直读地址：预筛必须与解析范围一致，否则这类
	// 历史图片不会被扫描、仍会被回收。
	const (
		messageID  = 9_800_500_401
		inlineName = "2026/10/01/cdn-inline.png"
	)
	seedLegacyUpload(t, userID, inlineName)
	if err := conn.Create(&messages.Entity{
		Id: messageID, ConvId: 9_800_500_402, SenderId: userID, MsgType: 1,
		Content: fmt.Sprintf("图集 ![](%s/%s)", prefix, inlineName),
	}).Error; err != nil {
		t.Fatalf("seed text message: %v", err)
	}
	t.Cleanup(func() { conn.Where("id = ?", messageID).Delete(&messages.Entity{}) })

	result := BackfillFileUsagesChatCoverWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}
	assertActiveUsage(t, conn, coverName, fileUsage.TargetUser, fileUsage.UsageCover, userID, userID)
	assertActiveUsage(t, conn, inlineName, fileUsage.TargetChatMessage, fileUsage.UsageChatMedia, messageID, userID)
}

// seedLegacyUser 造一个历史用户（可带封面 URL），返回其 id。
func seedLegacyUser(t *testing.T, conn *gorm.DB, userID uint64, coverURL string) {
	t.Helper()
	user := users.MakeUser(fmt.Sprintf("legacy-user-%d", userID), "secret123", fmt.Sprintf("legacy-user-%d@example.test", userID))
	user.Id = userID
	user.ProfileCoverUrl = coverURL
	if err := conn.Create(user).Error; err != nil {
		t.Fatalf("seed user %d: %v", userID, err)
	}
	t.Cleanup(func() { conn.Where("id = ?", userID).Delete(&users.EntityComplete{}) })
}

// seedLegacyMessage 造一条历史私信（显式 id / 发送者 / 正文）。
func seedLegacyMessage(t *testing.T, conn *gorm.DB, messageID, senderID uint64, content string, msgType int8) {
	t.Helper()
	if err := conn.Create(&messages.Entity{Id: messageID, ConvId: messageID, SenderId: senderID, Content: content, MsgType: msgType}).Error; err != nil {
		t.Fatalf("seed message %d: %v", messageID, err)
	}
	t.Cleanup(func() { conn.Where("id = ?", messageID).Delete(&messages.Entity{}) })
}

// TestBackfillFileUsagesChatCoverSkipsForeignFiles 是 P1 回归：历史 URL 落库时没有校验，
// 回填不能把「A 的消息/封面指向 B 的文件」变成永久的匿名读取授权 + 免回收。
func TestBackfillFileUsagesChatCoverSkipsForeignFiles(t *testing.T) {
	conn := prepareChatCoverBackfillDB(t)
	const (
		ownerID   = 9_800_600_001
		foreignID = 9_800_600_002
		messageID = 9_800_600_101
	)
	foreignChat := "2026/10/01/foreign-chat.png"
	foreignCover := "2026/10/01/foreign-cover.png"
	seedLegacyUpload(t, ownerID, foreignChat)
	seedLegacyUpload(t, ownerID, foreignCover)
	seedLegacyUser(t, conn, foreignID, "/file/img/"+foreignCover)
	// 发送者不是上传者：消息与封面都指向别人的文件。
	seedLegacyMessage(t, conn, messageID, foreignID, "/file/img/"+foreignChat, 2)

	result := BackfillFileUsagesChatCoverWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}
	if result.Inserted != 0 {
		t.Fatalf("backfill inserted=%d, want 0 for foreign files", result.Inserted)
	}
	for _, name := range []string{foreignChat, foreignCover} {
		if fileUsage.HasAnyReferences(name) {
			t.Fatalf("foreign file %s got a public reference", name)
		}
	}
}

// TestBackfillFileUsagesChatCoverAllowsManagerBotCover 保留代管场景：机器人封面由站点管理员
// 上传，引用记在机器人名下，回填必须照样登记。
func TestBackfillFileUsagesChatCoverAllowsManagerBotCover(t *testing.T) {
	conn := prepareChatCoverBackfillDB(t)
	const (
		botID     = 9_800_600_201
		managerID = 9_800_600_202
		roleID    = 9_800_600_203
	)
	const coverName = "2026/10/01/manager-bot-cover.png"
	seedLegacyUpload(t, managerID, coverName)
	seedLegacyUser(t, conn, botID, "/file/img/"+coverName)

	manager := users.MakeUser(fmt.Sprintf("manager-%d", managerID), "secret123", fmt.Sprintf("manager-%d@example.test", managerID))
	manager.Id = managerID
	manager.RoleId = roleID
	if err := conn.Create(manager).Error; err != nil {
		t.Fatalf("seed manager: %v", err)
	}
	t.Cleanup(func() { conn.Where("id = ?", managerID).Delete(&users.EntityComplete{}) })
	if err := conn.Create(&rolePermissionRs.Entity{RoleId: roleID, PermissionId: permission.SiteManager.Id()}).Error; err != nil {
		t.Fatalf("grant site manager: %v", err)
	}
	permission.InvalidateRole(roleID)
	t.Cleanup(func() {
		conn.Where("role_id = ?", roleID).Delete(&rolePermissionRs.Entity{})
		permission.InvalidateRole(roleID)
	})

	result := BackfillFileUsagesChatCoverWithDB(conn)
	if result.Failed > 0 {
		t.Fatalf("backfill failed=%d last=%s", result.Failed, result.LastFailed)
	}
	assertActiveUsage(t, conn, coverName, fileUsage.TargetUser, fileUsage.UsageCover, botID, botID)
}
