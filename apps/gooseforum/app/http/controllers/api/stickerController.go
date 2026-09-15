package api

import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/gin-gonic/gin"
)

// StickerAdminItem is the admin-facing sticker row (superset of the public item).
type StickerAdminItem struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name"`
	FileName  string `json:"fileName"`
	URL       string `json:"url"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
	CreatedBy uint64 `json:"createdBy"`
}

func stickerAdminItems(entities []sticker.Entity) []StickerAdminItem {
	items := make([]StickerAdminItem, 0, len(entities))
	for _, entity := range entities {
		items = append(items, StickerAdminItem{
			Id:        entity.Id,
			Name:      entity.Name,
			FileName:  entity.FileName,
			URL:       stickerservice.ResolveURLFor(entity),
			SortOrder: entity.SortOrder,
			IsEnabled: entity.IsEnabled,
			CreatedBy: entity.CreatedBy,
		})
	}
	return items
}

// StickerList returns every sticker row for the admin console.
func StickerList(req component.BetterRequest[component.Null]) component.Response {
	return component.SuccessResponse(stickerAdminItems(sticker.All()))
}

// PublicStickerList returns enabled stickers with public access paths; the
// editor picker and client-side token replacement consume it.
func PublicStickerList() component.Response {
	return component.SuccessResponse(stickerservice.EnabledList())
}

// StickerSaveReq creates (Id == 0) or updates (Id != 0) one sticker row.
// The file itself is uploaded separately (single upload or pack import).
type StickerSaveReq struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
}

func SaveSticker(req component.BetterRequest[StickerSaveReq]) component.Response {
	name := strings.TrimSpace(req.Params.Name)
	if name == "" {
		return component.FailResponseCode(component.MessageAdminStickerNameRequired, nil)
	}
	if !stickerservice.ValidateName(name) {
		return component.FailResponseCode(component.MessageAdminStickerNameInvalid, nil)
	}
	if req.Params.Id == 0 {
		if sticker.GetByName(name).Id != 0 {
			return component.FailResponseCode(component.MessageAdminStickerNameExists, nil)
		}
		entity := sticker.Entity{
			Name:      name,
			SortOrder: req.Params.SortOrder,
			IsEnabled: req.Params.IsEnabled,
			CreatedBy: req.UserId,
		}
		if err := sticker.Save(&entity); err != nil {
			return component.FailResponseCode(component.MessageAdminStickerSaveFailed, nil)
		}
		return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
	}
	entity := sticker.GetById(req.Params.Id)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminStickerNotFound, nil)
	}
	if existing := sticker.GetByName(name); existing.Id != 0 && existing.Id != entity.Id {
		return component.FailResponseCode(component.MessageAdminStickerNameExists, nil)
	}
	entity.Name = name
	entity.SortOrder = req.Params.SortOrder
	entity.IsEnabled = req.Params.IsEnabled
	if err := sticker.Save(&entity); err != nil {
		return component.FailResponseCode(component.MessageAdminStickerSaveFailed, nil)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// StickerDeleteReq removes a sticker row and releases its file usage so the
// storage GC can reclaim the image when no other reference remains.
type StickerDeleteReq struct {
	Id uint64 `json:"id"`
}

func DeleteSticker(req component.BetterRequest[StickerDeleteReq]) component.Response {
	entity := sticker.GetById(req.Params.Id)
	if entity.Id == 0 {
		return component.FailResponseCode(component.MessageAdminStickerNotFound, nil)
	}
	if err := sticker.DeleteById(entity.Id); err != nil {
		return component.FailResponseCode(component.MessageAdminStickerDeleteFailed, nil)
	}
	if err := fileusageservice.RemoveStickerUsages(entity.Id); err != nil {
		// The row is gone; a stranded usage row only delays GC, never leaks access.
		slog.Error("remove sticker usage failed", "stickerId", entity.Id, "err", err)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

const (
	// stickerImportMaxZipBytes caps the uploaded archive (pre-uncompression).
	stickerImportMaxZipBytes = 32 << 20
	// stickerImportMaxFiles caps entries per import to bound request time.
	stickerImportMaxFiles = 500
	// stickerImportStickerPath mirrors preset storage for easy identification.
	stickerImportStickerPath = "stickers"
)

// StickerImportResult reports one pack import outcome per entry bucket.
type StickerImportResult struct {
	Imported int                  `json:"imported"`
	Skipped  int                  `json:"skipped"`
	Failed   []StickerImportIssue `json:"failed"`
}

type StickerImportIssue struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ImportStickerPack accepts a multipart zip (form field "file") of images.
// Each image becomes one sticker named after its file stem (sanitized and
// uniquified server-side). Entry-level failures are reported, not fatal:
// one bad image must not block the rest of the pack.
func ImportStickerPack(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageUploadFileMissing, nil))
		return
	}
	if fileHeader.Size > stickerImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageRequestParseFailed, nil))
		return
	}
	defer file.Close()
	archive, err := io.ReadAll(io.LimitReader(file, stickerImportMaxZipBytes+1))
	if err != nil || len(archive) > stickerImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return
	}
	zipReader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportInvalidZip, nil))
		return
	}

	adminUserId := c.GetUint64("userId")
	result := StickerImportResult{Failed: []StickerImportIssue{}}
	for _, entry := range zipReader.File {
		if result.Imported+result.Skipped+len(result.Failed) >= stickerImportMaxFiles {
			result.Failed = append(result.Failed, StickerImportIssue{Name: "...", Reason: "tooManyFiles"})
			break
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		// 忽略 macOS 资源目录与 dotfiles。
		base := path.Base(entry.Name)
		if base == "" || strings.HasPrefix(base, ".") || strings.Contains(entry.Name, "__MACOSX") {
			continue
		}
		contentType, ok := imagepolicy.ContentTypeForExt(path.Ext(base))
		if !ok {
			result.Skipped++
			continue
		}
		// 归档条目不落盘，无 zip-slip 风险；仍限制解压体积防解压炸弹。
		reader, err := entry.Open()
		if err != nil {
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "entryOpenFailed"})
			continue
		}
		data, err := io.ReadAll(io.LimitReader(reader, filedata.MaxFileSize+1))
		reader.Close()
		if err != nil || len(data) > filedata.MaxFileSize {
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "tooLarge"})
			continue
		}
		if err := validateUploadedImage(bytes.NewReader(data), contentType); err != nil {
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "invalidImage"})
			continue
		}
		name, ok := stickerservice.UniqueName(stickerservice.StemName(base))
		if !ok {
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "unusableName"})
			continue
		}
		fileEntity, err := filedata.SaveFileFromUpload(adminUserId, data, base, stickerImportStickerPath)
		if err != nil {
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "saveFailed"})
			continue
		}
		row := sticker.Entity{
			Name:      name,
			FileName:  fileEntity.Name,
			IsEnabled: true,
			CreatedBy: adminUserId,
		}
		if err := sticker.Save(&row); err != nil {
			_ = filedata.DeleteByName(fileEntity.Name)
			result.Failed = append(result.Failed, StickerImportIssue{Name: base, Reason: "saveFailed"})
			continue
		}
		if err := fileusageservice.AddStickerUsage(adminUserId, fileEntity.Name, row.Id); err != nil {
			slog.Error("register imported sticker usage failed", "stickerId", row.Id, "err", err)
		}
		result.Imported++
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(result, component.MessageOperationSuccess, nil))
}
