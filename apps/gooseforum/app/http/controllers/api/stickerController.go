package api

import (
	"errors"
	"io"
	"net/http"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
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
	entities, err := sticker.All()
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(stickerAdminItems(entities))
}

// PublicStickerList returns enabled stickers with public access paths; the
// editor picker and client-side token replacement consume it.
func PublicStickerList() component.Response {
	items, err := stickerservice.EnabledList()
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(items)
}

// StickerSaveReq creates (Id == 0) or updates (Id != 0) one sticker row.
// The file itself is uploaded separately (single upload or pack import).
type StickerSaveReq struct {
	Id        uint64 `json:"id"`
	Name      string `json:"name"`
	FileName  string `json:"fileName"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
}

func stickerWriteError(err error, fallback component.MessageCode) component.Response {
	code := fallback
	switch {
	case errors.Is(err, stickerservice.ErrNameRequired):
		code = component.MessageAdminStickerNameRequired
	case errors.Is(err, stickerservice.ErrNameInvalid):
		code = component.MessageAdminStickerNameInvalid
	case errors.Is(err, stickerservice.ErrNameExists):
		code = component.MessageAdminStickerNameExists
	case errors.Is(err, stickerservice.ErrFileRequired):
		code = component.MessageUploadFileMissing
	case errors.Is(err, stickerservice.ErrNotFound):
		code = component.MessageAdminStickerNotFound
	}
	return component.FailResponseCode(code, nil)
}
func SaveSticker(req component.BetterRequest[StickerSaveReq]) component.Response {
	err := stickerservice.Save(betterRequestContext(req), req.UserId, stickerservice.SaveInput{Id: req.Params.Id, Name: req.Params.Name, FileName: req.Params.FileName, SortOrder: req.Params.SortOrder, IsEnabled: req.Params.IsEnabled})
	if err != nil {
		return stickerWriteError(err, component.MessageAdminStickerSaveFailed)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// StickerDeleteReq removes a sticker row and releases its file usage so the
// storage GC can reclaim the image when no other reference remains.
type StickerDeleteReq struct {
	Id uint64 `json:"id"`
}

func DeleteSticker(req component.BetterRequest[StickerDeleteReq]) component.Response {
	if err := stickerservice.Delete(betterRequestContext(req), req.Params.Id); err != nil {
		return stickerWriteError(err, component.MessageAdminStickerDeleteFailed)
	}
	return component.SuccessResponseCode("success", component.MessageOperationSuccess, nil)
}

// ImportStickerPack handles only multipart transport; the service owns all
// validation, naming, storage, and reference lifecycle decisions.
func ImportStickerPack(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, stickerservice.ImportMaxZipBytes+(1<<20))
	header, err := c.FormFile("file")
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		code := component.MessageUploadFileMissing
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			code = component.MessageAdminStickerImportTooLarge
		}
		c.JSON(http.StatusOK, component.FailDataCode(code, nil))
		return
	}
	if header.Size > stickerservice.ImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return
	}
	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageRequestParseFailed, nil))
		return
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, stickerservice.ImportMaxZipBytes+1))
	if err != nil || len(data) > stickerservice.ImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return
	}
	result, err := stickerservice.ImportPack(c.Request.Context(), c.GetUint64("userId"), data)
	if err != nil {
		code := component.MessageOperationFailed
		if errors.Is(err, stickerservice.ErrInvalidZip) {
			code = component.MessageAdminStickerImportInvalidZip
		}
		c.JSON(http.StatusOK, component.FailDataCode(code, nil))
		return
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(result, component.MessageOperationSuccess, nil))
}
