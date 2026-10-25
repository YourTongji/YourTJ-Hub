package fileusageservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"gorm.io/gorm"
)

// OwnedChatMediaNames 解析消息里的媒体，只留下发送者本人上传的 ready 文件（口径同
// FilterOwnedImageURLs）。它读取存储配置缓存与文件库，必须在发送事务之外调用：事务里再借
// 第二条连接会与外层事务互锁。归属查询失败返回错误，让发送直接失败，而不是提交一条没有
// 引用的媒体消息。
// 转发副本由转发者重新发送、并不拥有文件，它们依赖原消息的引用（原消息没有撤回路径，
// 所以该引用不会被删）。
func OwnedChatMediaNames(senderID uint64, content string, msgType int8) ([]string, error) {
	candidates := ChatMediaFileNames(content, msgType)
	names := make([]string, 0, len(candidates))
	for _, name := range candidates {
		owned, err := ownedReadyFile(senderID, name)
		if err != nil {
			return nil, err
		}
		if owned {
			names = append(names, name)
		}
	}
	return names, nil
}

// RegisterChatMediaTx 在私信事务里为 OwnedChatMediaNames 的结果登记 ACTIVE 内容引用
// （issue #1365）：messages.content 只存 URL，没有内容引用时收件人读不到（#1366 的读门禁）
// 且文件会在 TTL 后被孤儿清理删掉。幂等：CreateIfAbsent 让整个发送事务可以安全重试。
func RegisterChatMediaTx(tx *gorm.DB, messageID, senderID uint64, names []string) error {
	if err := fileUsage.GuardUploadReferencesTx(tx, names); err != nil {
		return err
	}
	for _, name := range names {
		if _, err := fileUsage.CreateIfAbsentTx(tx, &fileUsage.Entity{
			FileName:   name,
			TargetType: fileUsage.TargetChatMessage,
			TargetId:   messageID,
			UsageType:  fileUsage.UsageChatMedia,
			UserId:     senderID,
			Status:     fileUsage.UsageStatusActive,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ChatMediaFileNames 解析消息里的媒体对象名：图片/语音消息（msgType 2/3）的 content
// 整体就是一个媒体 URL，文本消息只取其中的 markdown 图片。解析认 /file/img 与配置了
// publicUrlPrefix 的对象存储直读地址；历史回填用同一份解析，避免谓词与解析再次漂移。
func ChatMediaFileNames(content string, msgType int8) []string {
	values := markdown2html.ExtractImageURLs(content)
	if msgType == 2 || msgType == 3 {
		values = append(values, content)
	}
	names := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		name := FileNameFromURL(value)
		// file_usages.file_name is varchar(512); longer names cannot be referenced.
		if name == "" || len(name) > 512 || seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
	}
	return names
}
