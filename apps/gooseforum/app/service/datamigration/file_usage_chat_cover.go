package datamigration

import (
	"log/slog"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/chat/messages"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"gorm.io/gorm"
)

// fileUsageChatCoverBatch 是回填每批扫描的行数：只按 id 游标分批推进，不整表加载。
const fileUsageChatCoverBatch = 500

// FileUsageChatCoverResult 汇总孤儿回收前历史私信、封面和正文附件引用的回填结果（issue #1365）。
// 这两类内容只把 URL 存在 messages.content / users.profile_cover_url 里，没有 file_usages
// 行；不补引用的话它们的文件既对读者不可见，又会被孤儿回收任务当成从未被引用的上传删除。
type FileUsageChatCoverResult struct {
	Processed  int
	Inserted   int
	Failed     int
	LastFailed string
}

func BackfillFileUsagesChatCover() FileUsageChatCoverResult {
	return BackfillFileUsagesChatCoverWithDB(db.Connect())
}

// BackfillFileUsagesChatCoverWithDB 为历史私信媒体与个人封面补登 ACTIVE 附件引用，
// 并补齐话题图集、正文和版本记录漏登的已跟踪上传（保留原有可见性）。
// 幂等：重复执行只补缺失的行（CreateIfAbsent）。归属口径与在线写入一致：只登记
// 「文件存在、ready、且由消息发送者 / 封面所属用户上传（封面另允许站点管理员代传）」
// 的对象——ACTIVE 非本人引用是永久的匿名读取授权 + 永久免回收，历史 URL 落库时没有校验，
// 回填不能把「A 把封面指向 B 的孤儿上传」变成不可撤销的公开授权。文件库查询出错按失败计数，
// 让版本化迁移卡住并在下次启动重试。
func BackfillFileUsagesChatCoverWithDB(conn *gorm.DB) FileUsageChatCoverResult {
	result := FileUsageChatCoverResult{}
	if !conn.Migrator().HasTable((&fileUsage.Entity{}).TableName()) {
		return result
	}
	backfillCoverUsages(conn, &result)
	backfillChatMediaUsages(conn, &result)
	backfillInlineUploadUsages(conn, &result)
	return result
}

func backfillCoverUsages(conn *gorm.DB, result *FileUsageChatCoverResult) {
	if !conn.Migrator().HasTable((&users.EntityComplete{}).TableName()) {
		return
	}
	var cursor uint64
	for {
		var batch []users.EntityComplete
		if err := conn.Model(&users.EntityComplete{}).
			Select("id", "profile_cover_url").
			Where("id > ? AND profile_cover_url <> ?", cursor, "").
			Order("id ASC").Limit(fileUsageChatCoverBatch).
			Find(&batch).Error; err != nil {
			failFileUsageBackfill(result, "cover_scan", err)
			return
		}
		for i := range batch {
			user := &batch[i]
			cursor = user.Id
			result.Processed++
			name := fileusageservice.FileNameFromURL(user.ProfileCoverUrl)
			if name == "" || len(name) > 512 {
				continue
			}
			// 封面允许本人上传，或站点管理员代管上传（机器人资料由管理员编辑）。
			owned, err := ownedBackfillFile(name, func(ownerID uint64) bool {
				return ownerID == user.Id || isSiteManager(ownerID)
			})
			if err != nil {
				failFileUsageBackfill(result, "cover_lookup", err)
				continue
			}
			if !owned {
				continue
			}
			insertUsageIfAbsent(conn, result, "cover_insert", fileUsage.Entity{
				FileName:   name,
				TargetType: fileUsage.TargetUser,
				TargetId:   user.Id,
				UsageType:  fileUsage.UsageCover,
				UserId:     user.Id,
				Status:     fileUsage.UsageStatusActive,
			})
		}
		if len(batch) < fileUsageChatCoverBatch {
			return
		}
	}
}

func backfillChatMediaUsages(conn *gorm.DB, result *FileUsageChatCoverResult) {
	if !conn.Migrator().HasTable((&messages.Entity{}).TableName()) {
		return
	}
	var cursor uint64
	for {
		var batch []messages.Entity
		if err := conn.Model(&messages.Entity{}).
			Select("id", "sender_id", "content", "msg_type").
			Where("id > ?", cursor).
			// 预筛要与 ChatMediaFileNames 的解析范围一致：图片/语音正文、站内
			// /file/img 链接，以及可能承载 publicUrlPrefix 直读地址的 markdown 图片。
			Where("msg_type IN ? OR content LIKE ? OR content LIKE ?", []int8{2, 3}, "%/file/img/%", "%![%").
			Order("id ASC").Limit(fileUsageChatCoverBatch).
			Find(&batch).Error; err != nil {
			failFileUsageBackfill(result, "chat_media_scan", err)
			return
		}
		if len(batch) == 0 {
			return
		}
		for i := range batch {
			message := &batch[i]
			cursor = message.Id
			result.Processed++
			for _, name := range fileusageservice.ChatMediaFileNames(message.Content, message.MsgType) {
				owned, err := ownedBackfillFile(name, func(ownerID uint64) bool { return ownerID == message.SenderId })
				if err != nil {
					failFileUsageBackfill(result, "chat_media_lookup", err)
					continue
				}
				if !owned {
					continue
				}
				insertUsageIfAbsent(conn, result, "chat_media_insert", fileUsage.Entity{
					FileName:   name,
					TargetType: fileUsage.TargetChatMessage,
					TargetId:   message.Id,
					UsageType:  fileUsage.UsageChatMedia,
					UserId:     message.SenderId,
					Status:     fileUsage.UsageStatusActive,
				})
			}
		}
	}
}

// ownedBackfillFile 校验对象名存在、ready 且归属 allowed 允许的上传者。只读元数据，不拉
// content BLOB；查询出错返回错误，不能把「读不到归属」当成「不属于」后静默跳过。
func ownedBackfillFile(name string, allowed func(ownerID uint64) bool) (bool, error) {
	file, err := filedata.GetFileMetadataByName(name)
	if err != nil {
		return false, err
	}
	if file == nil || file.StorageStatus != filedata.StorageStatusReady {
		return false, nil
	}
	return allowed(file.UserId), nil
}

// isSiteManager 判断上传者当前是否持有站点管理权限（代管管理员给机器人上传封面）。
func isSiteManager(userID uint64) bool {
	if userID == 0 {
		return false
	}
	roleID, ok := userservice.GetUserRoleId(userID)
	return ok && permission.CheckRole(roleID, permission.SiteManager)
}

func insertUsageIfAbsent(conn *gorm.DB, result *FileUsageChatCoverResult, stage string, row fileUsage.Entity) {
	inserted, err := fileUsage.CreateIfAbsentTx(conn, &row)
	if err != nil {
		result.Failed++
		result.LastFailed = stage + ":" + err.Error()
		slog.Error("file usage chat/cover backfill insert failed", "stage", stage, "fileName", row.FileName, "err", err)
		return
	}
	if inserted > 0 {
		result.Inserted++
	}
}

func failFileUsageBackfill(result *FileUsageChatCoverResult, stage string, err error) {
	result.Failed++
	result.LastFailed = stage + ":" + err.Error()
	slog.Error("file usage chat/cover backfill scan failed", "stage", stage, "err", err)
}
