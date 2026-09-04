package fileusageservice

import (
	"log/slog"
	"net/url"
	"path"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/markdown2html"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
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
	replace(fileUsage.TargetTopic, topicID, []string{fileUsage.UsageInlineImage}, userID, namesToUsages(urls, fileUsage.UsageInlineImage))
}

// MaxGalleryImagesPerTopicWrite caps how many explicit gallery images one
// topic write may carry. The web quick-publish picker enforces the same limit
// client-side; this server cap keeps the request and its usage rows bounded
// for non-conforming clients.
const MaxGalleryImagesPerTopicWrite = 9

// MaxImageURLLength guards the persisted image_urls text column and the
// file_usages.file_name varchar(512) column against oversized raw input.
const MaxImageURLLength = 2048

// FilterOwnedImageURLs returns the input URLs that resolve to a ready upload
// owned by userID. Only such files may become ACTIVE content references:
// registering a foreign or unknown /file/img/ name would let an attacker pin
// another user's file and keep it publicly served after the owner's
// delete/privacy purge (GC skips files with live references).
func FilterOwnedImageURLs(userID uint64, urls []string) []string {
	kept := make([]string, 0, len(urls))
	seen := make(map[string]bool, len(urls))
	for _, value := range urls {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > MaxImageURLLength {
			continue
		}
		name := fileNameFromURL(value)
		if name == "" || len(name) > 512 || seen[name] {
			continue
		}
		entity := filedata.GetByName(name)
		if entity.Id == 0 || entity.StorageStatus != filedata.StorageStatusReady || entity.UserId != userID {
			continue
		}
		seen[name] = true
		kept = append(kept, value)
	}
	return kept
}

// RegisterTopicInlineImagesOwned replaces a topic's inline_image usage rows
// with the ACTIVE references derived from its first-post markdown and its
// explicit gallery list, keeping only files owned by userID (see
// FilterOwnedImageURLs). It is the ownership-checked write path for user
// content; unowned markdown URLs are skipped instead of being pinned.
func RegisterTopicInlineImagesOwned(topicID uint64, userID uint64, content string, gallery []string) {
	urls := markdown2html.ExtractImageURLs(content)
	urls = append(urls, gallery...)
	replace(fileUsage.TargetTopic, topicID, []string{fileUsage.UsageInlineImage}, userID, ownedUsages(userID, urls))
}

// RegisterPostInlineImagesOwned is the ownership-checked reply/post variant
// of RegisterTopicInlineImagesOwned for a single post's markdown content.
func RegisterPostInlineImagesOwned(postID uint64, userID uint64, content string) {
	replace(fileUsage.TargetPost, postID, []string{fileUsage.UsageInlineImage}, userID, ownedUsages(userID, markdown2html.ExtractImageURLs(content)))
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
	replace(fileUsage.TargetUser, userId, []string{fileUsage.UsageAvatar}, userId, namesToUsages(fileNames, fileUsage.UsageAvatar))
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

func replace(targetType string, targetId uint64, usageTypes []string, userId uint64, usages []Usage) {
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
		})
	}
	if err := fileUsage.ReplaceTargetUsages(targetType, targetId, usageTypes, rows); err != nil {
		slog.Error("replace file usages failed", "targetType", targetType, "targetId", targetId, "err", err)
	}
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
