package fileusageservice

import (
	"errors"
	"log/slog"
	"net/url"
	"path"
	"strings"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"gorm.io/gorm"
)

type Usage struct {
	FileName  string
	UsageType string
}

func ReplaceTopic(topicID uint64, userID uint64, content string) {
	ReplaceTopicWithImages(topicID, userID, content, nil)
}

func ReplaceTopicWithImages(topicID uint64, userID uint64, content string, extraImages []string) {
	urls := markdown2html.ExtractImageURLs(content)
	urls = append(urls, extraImages...)
	replace(fileUsage.TargetTopic, topicID, []string{fileUsage.UsageInlineImage}, userID, namesToUsages(urls, fileUsage.UsageInlineImage), false)
}

// MaxGalleryImagesPerTopicWrite caps how many in-site gallery candidates (see
// GalleryImageURLs) one topic write may carry. The web quick-publish picker
// enforces the same limit client-side; this server cap keeps the request and
// its usage rows bounded for non-conforming clients.
const MaxGalleryImagesPerTopicWrite = 9

// MaxImageURLLength guards the persisted image_urls text column and the
// file_usages.file_name varchar(512) column against oversized raw input.
const MaxImageURLLength = 2048

// GalleryImageURLs returns the in-site gallery candidates among urls: trimmed,
// within MaxImageURLLength, and resolvable to a file name (see fileNameFromURL),
// deduplicated by that name. Existence, readiness and ownership are not checked
// here (FilterOwnedImageURLs does that), so MaxGalleryImagesPerTopicWrite counts
// candidates — a coarse pre-filter costing no DB lookup per unbounded input URL.
// URLs whose path carries no /file/img/ name (e.g. external image hosts)
// neither count nor persist.
func GalleryImageURLs(urls []string) []string {
	kept := make([]string, 0, len(urls))
	seen := make(map[string]bool, len(urls))
	for _, value := range urls {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > MaxImageURLLength {
			continue
		}
		name := fileNameFromURL(value)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		kept = append(kept, value)
	}
	return kept
}

// FilterOwnedImageURLs returns the input URLs that resolve to a ready upload
// owned by userID. Only such files may become ACTIVE content references:
// registering a foreign or unknown /file/img/ name would let an attacker pin
// another user's file and keep it publicly served after the owner's
// delete/privacy purge (GC skips files with live references).
func FilterOwnedImageURLs(userID uint64, urls []string) []string {
	candidates := GalleryImageURLs(urls)
	kept := make([]string, 0, len(candidates))
	for _, value := range candidates {
		name := fileNameFromURL(value)
		// file_usages.file_name is varchar(512); longer names cannot be referenced.
		if len(name) > 512 {
			continue
		}
		entity := filedata.GetByName(name)
		if entity.Id == 0 || entity.StorageStatus != filedata.StorageStatusReady || entity.UserId != userID {
			continue
		}
		kept = append(kept, value)
	}
	return kept
}

// RegisterTopicInlineImagesOwned replaces a topic's inline_image usage rows
// with the ACTIVE references derived from its first-post markdown and its
// explicit gallery list, keeping only files owned by userID (see
// FilterOwnedImageURLs). It is the ownership-checked write path for user
// content; unowned markdown URLs are skipped instead of being pinned.
// pending=true registers private PENDING rows for drafts or review candidates.
// This alone does not authorize moderator preview; readers check the content.
func RegisterTopicInlineImagesOwned(topicID uint64, userID uint64, content string, gallery []string, pending bool) {
	urls := markdown2html.ExtractImageURLs(content)
	urls = append(urls, gallery...)
	replace(fileUsage.TargetTopic, topicID, []string{fileUsage.UsageInlineImage}, userID, ownedUsages(userID, urls), pending)
}

// RegisterPostInlineImagesOwned is the ownership-checked reply/post variant
// of RegisterTopicInlineImagesOwned for a single post's markdown content.
func RegisterPostInlineImagesOwned(postID uint64, userID uint64, content string, pending bool) {
	replace(fileUsage.TargetPost, postID, []string{fileUsage.UsageInlineImage}, userID, ownedUsages(userID, markdown2html.ExtractImageURLs(content)), pending)
}

// ownedUsages converts owned, ready image URLs into inline_image usage rows,
// deduplicating by resolved file name.
func ownedUsages(userID uint64, urls []string) []Usage {
	kept := FilterOwnedImageURLs(userID, urls)
	usages := make([]Usage, 0, len(kept))
	seen := make(map[string]bool, len(kept))
	for _, value := range kept {
		name := fileNameFromURL(value)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		usages = append(usages, Usage{FileName: name, UsageType: fileUsage.UsageInlineImage})
	}
	return usages
}

func ReplaceAvatar(userId uint64, fileNames []string) {
	replace(fileUsage.TargetUser, userId, []string{fileUsage.UsageAvatar}, userId, namesToUsages(fileNames, fileUsage.UsageAvatar), false)
}

// CoverRef 是封面 URL 解析后的结果：对象名 + 归属判定。解析必须在事务外完成，事务里
// 只允许通过 tx 读写 file_usages/users。
type CoverRef struct {
	Name  string
	Owned bool
}

// ResolveCover 解析封面 URL 并判定归属（issue #1365）。它读取存储配置缓存与文件库，必须
// 在任何事务之外调用：这些读会借第二条连接，在事务内执行会与持连接的外层事务互锁（SQLite
// 单连接测试直接死锁，生产是连接池/锁序风险）。查询失败返回错误，不能把「读不到归属」
// 当成「没有归属」——那会静默吊销封面引用。
func ResolveCover(userId uint64, coverURL string, ownerIDs ...uint64) (CoverRef, error) {
	name := FileNameFromURL(coverURL)
	if name == "" || len(name) > 512 {
		return CoverRef{}, nil
	}
	owned, err := coveredByOwner(userId, ownerIDs, name)
	if err != nil {
		return CoverRef{}, err
	}
	return CoverRef{Name: name, Owned: owned}, nil
}

// ReplaceCoverTx 用 ResolveCover 的结果替换封面的 ACTIVE 引用，必须与写入
// users.profile_cover_url 同一事务（否则 URL 可能落库而引用没写成）。空引用清空旧行
// （换下的旧封面随即可回收）；封面未变化时保留现有引用、不做归属校验（代管管理员 A
// 上传的封面，B 再保存其他字段时不能被吊销）。
func ReplaceCoverTx(tx *gorm.DB, userId uint64, ref CoverRef) error {
	if ref.Name == "" {
		return replaceTx(tx, fileUsage.TargetUser, userId, []string{fileUsage.UsageCover}, userId, nil, false)
	}
	existing, err := fileUsage.ListTargetUsagesTx(tx, fileUsage.TargetUser, userId, fileUsage.UsageCover)
	if err != nil {
		return err
	}
	if len(existing) == 1 && existing[0].FileName == ref.Name && existing[0].Status == fileUsage.UsageStatusActive {
		return nil
	}
	var usages []Usage
	if ref.Owned {
		usages = []Usage{{FileName: ref.Name, UsageType: fileUsage.UsageCover}}
	}
	return replaceTx(tx, fileUsage.TargetUser, userId, []string{fileUsage.UsageCover}, userId, usages, false)
}

// coveredByOwner 校验封面文件属于目标用户本人或代管操作者。查询失败必须返回错误：
// 读不到归属就把引用当成不存在，会让封面被静默吊销。
func coveredByOwner(userId uint64, ownerIDs []uint64, name string) (bool, error) {
	owned, err := ownedReadyFile(userId, name)
	if err != nil || owned {
		return owned, err
	}
	for _, ownerID := range ownerIDs {
		if ownerID == 0 || ownerID == userId {
			continue
		}
		if owned, err := ownedReadyFile(ownerID, name); err != nil || owned {
			return owned, err
		}
	}
	return false, nil
}

// ownedReadyFile 校验对象名对应本人上传的 ready 文件：只有这样的文件才能成为内容引用，
// 否则可以借引用固定、或让回收链动到别人的文件（口径同 FilterOwnedImageURLs）。只读元数据，
// 不拉 content BLOB。
func ownedReadyFile(userID uint64, name string) (bool, error) {
	file, err := filedata.GetFileMetadataByName(name)
	if err != nil || file == nil {
		return false, err
	}
	return file.StorageStatus == filedata.StorageStatusReady && file.UserId == userID, nil
}

func AddAdminUpload(userId uint64, fileName string) {
	name := fileNameFromURL(fileName)
	if name == "" {
		return
	}
	if err := fileUsage.Create(&fileUsage.Entity{
		FileName:   name,
		TargetType: fileUsage.TargetAdminUpload,
		TargetId:   userId,
		UsageType:  fileUsage.UsageAdminUpload,
		UserId:     userId,
	}); err != nil {
		slog.Error("create admin upload usage failed", "userId", userId, "fileName", name, "err", err)
	}
}

// AddStickerUsage pins a sticker image file to its sticker row with an ACTIVE
// sticker reference. The file stays publicly served and GC-exempt while the
// sticker row exists; RemoveStickerUsages on delete lets storage GC reclaim it.
func AddStickerUsage(adminUserID uint64, fileName string, stickerId uint64) error {
	name := fileNameFromURL(fileName)
	if name == "" {
		return errors.New("sticker file name is empty")
	}
	return fileUsage.CreateIfAbsent(&fileUsage.Entity{
		FileName:   name,
		TargetType: fileUsage.TargetSticker,
		TargetId:   stickerId,
		UsageType:  fileUsage.UsageSticker,
		UserId:     adminUserID,
	})
}

// RemoveStickerUsages releases a deleted sticker's file reference.
func RemoveStickerUsages(stickerId uint64) error {
	return fileUsage.DeleteTargetUsages(fileUsage.TargetSticker, stickerId, fileUsage.UsageSticker)
}

// AddUploadOwner records that userId owns the uploaded file. The row is
// idempotent: a duplicate complete request must not create a second usage row.
func AddUploadOwner(userID uint64, fileName string) error {
	name := fileNameFromURL(fileName)
	if name == "" {
		return nil
	}
	return fileUsage.CreateIfAbsent(&fileUsage.Entity{
		FileName:   name,
		TargetType: fileUsage.TargetUploadOwner,
		TargetId:   userID,
		UsageType:  fileUsage.UsageUploadOwner,
		UserId:     userID,
	})
}

func namesToUsages(values []string, usageType string) []Usage {
	usages := make([]Usage, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		name := fileNameFromURL(value)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		usages = append(usages, Usage{FileName: name, UsageType: usageType})
	}
	return usages
}

// replace 在独立事务里替换引用；错误只记日志（内容写入侧不因附件登记失败而失败）。
func replace(targetType string, targetId uint64, usageTypes []string, userId uint64, usages []Usage, pending bool) {
	err := db.Connect().Transaction(func(tx *gorm.DB) error {
		return replaceTx(tx, targetType, targetId, usageTypes, userId, usages, pending)
	})
	if err != nil {
		slog.Error("replace file usages failed", "targetType", targetType, "targetId", targetId, "err", err)
	}
}

// replaceTx 与 replace 相同，但在调用方事务里执行（失败随事务回滚）。
func replaceTx(tx *gorm.DB, targetType string, targetId uint64, usageTypes []string, userId uint64, usages []Usage, pending bool) error {
	status := fileUsage.UsageStatusActive
	if pending {
		status = fileUsage.UsageStatusPending
	}
	rows := make([]fileUsage.Entity, 0, len(usages))
	for _, usage := range usages {
		if usage.FileName == "" || usage.UsageType == "" {
			continue
		}
		rows = append(rows, fileUsage.Entity{
			FileName:   usage.FileName,
			TargetType: targetType,
			TargetId:   targetId,
			UsageType:  usage.UsageType,
			UserId:     userId,
			Status:     status,
		})
	}
	return fileUsage.ReplaceTargetUsagesTx(tx, targetType, targetId, usageTypes, rows)
}

func fileNameFromURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if !parsed.IsAbs() && !strings.HasPrefix(parsed.Path, "/") {
		name := path.Clean(parsed.Path)
		if name == "." || strings.HasPrefix(name, "..") {
			return ""
		}
		return name
	}
	if parsed.IsAbs() && parsed.Host != "" && !strings.HasPrefix(parsed.Path, "/file/img/") {
		return ""
	}
	if !strings.HasPrefix(parsed.Path, "/file/img/") {
		return ""
	}
	name := strings.TrimPrefix(parsed.Path, "/file/img/")
	name = path.Clean("/" + name)
	return strings.TrimPrefix(name, "/")
}

// FileNameFromURL resolves a stored image URL to its file_data object name.
func FileNameFromURL(value string) string {
	if name := fileNameFromURL(value); name != "" {
		return name
	}
	return fileNameFromPublicURL(value, hotdataserve.GetStorageSettingsConfigCache().PublicUrlPrefix)
}

func fileNameFromPublicURL(value, publicPrefix string) string {
	if strings.TrimSpace(publicPrefix) == "" {
		return ""
	}
	publicURL, err := url.Parse(strings.TrimRight(publicPrefix, "/") + "/")
	if err != nil || publicURL.Scheme == "" || publicURL.Host == "" {
		return ""
	}
	imageURL, err := url.Parse(value)
	if err != nil || imageURL.User != nil || imageURL.Scheme != publicURL.Scheme || !strings.EqualFold(imageURL.Host, publicURL.Host) {
		return ""
	}
	basePath := strings.TrimRight(publicURL.Path, "/")
	name, ok := strings.CutPrefix(imageURL.Path, basePath+"/")
	if !ok || name == "" {
		return ""
	}
	for _, segment := range strings.Split(name, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return ""
		}
	}
	return name
}

// SetStickerUsageTx replaces a sticker reference in the definition transaction.
// status is the initial file_usage status: ACTIVE for official stickers,
// PENDING for personal uploads awaiting review (issue #1071).
func SetStickerUsageTx(tx *gorm.DB, stickerID, userID uint64, fileName, status string) error {
	return fileUsage.ReplaceStickerTx(tx, stickerID, userID, fileName, status)
}

// RegisterRevisionImagesTx retains private candidate images without replacing
// the currently public topic/post references. A save failure rolls back all refs.
func RegisterRevisionImagesTx(tx *gorm.DB, revisionID, userID uint64, content string, gallery []string) error {
	urls := append(markdown2html.ExtractImageURLs(content), gallery...)
	for _, usage := range ownedUsages(userID, urls) {
		if err := tx.Create(&fileUsage.Entity{FileName: usage.FileName, TargetType: fileUsage.TargetPostRevision, TargetId: revisionID, UsageType: usage.UsageType, UserId: userID, Status: fileUsage.UsageStatusPending}).Error; err != nil {
			return err
		}
	}
	return nil
}

// PublishRevisionImagesTx replaces only the effective content refs. Revision
// refs stay private, so superseded/rejected attachments never become public.
func PublishRevisionImagesTx(tx *gorm.DB, revisionID, topicID, postID uint64, first bool) error {
	target, id := fileUsage.TargetPost, postID
	if first {
		target, id = fileUsage.TargetTopic, topicID
	}
	if err := tx.Where("target_type = ? AND target_id = ? AND usage_type = ?", target, id, fileUsage.UsageInlineImage).Delete(&fileUsage.Entity{}).Error; err != nil {
		return err
	}
	// First-post edits historically also registered post refs; revoke both sets.
	if first {
		if err := tx.Where("target_type = ? AND target_id = ? AND usage_type = ?", fileUsage.TargetPost, postID, fileUsage.UsageInlineImage).Delete(&fileUsage.Entity{}).Error; err != nil {
			return err
		}
	}
	var refs []fileUsage.Entity
	if err := tx.Where("target_type = ? AND target_id = ?", fileUsage.TargetPostRevision, revisionID).Find(&refs).Error; err != nil {
		return err
	}
	for _, ref := range refs {
		ref.Id, ref.TargetType, ref.TargetId, ref.Status = 0, target, id, fileUsage.UsageStatusActive
		if err := tx.Create(&ref).Error; err != nil {
			return err
		}
	}
	return nil
}

// PrivatizeUnpublishedImagesTx closes effective references left by a draft or
// blocked public row before its next candidate is submitted.
func PrivatizeUnpublishedImagesTx(tx *gorm.DB, topicID, postID uint64, first bool) error {
	q := tx.Model(&fileUsage.Entity{}).Where("usage_type = ? AND status = ?", fileUsage.UsageInlineImage, fileUsage.UsageStatusActive)
	if first {
		q = q.Where("(target_type = ? AND target_id = ?) OR (target_type = ? AND target_id = ?)", fileUsage.TargetTopic, topicID, fileUsage.TargetPost, postID)
	} else {
		q = q.Where("target_type = ? AND target_id = ?", fileUsage.TargetPost, postID)
	}
	return q.Update("status", fileUsage.UsageStatusPending).Error
}
