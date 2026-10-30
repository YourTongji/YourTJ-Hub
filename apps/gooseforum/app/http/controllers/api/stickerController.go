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
	Pack      string `json:"pack"`
	SortOrder int    `json:"sortOrder"`
	IsEnabled bool   `json:"isEnabled"`
	CreatedBy uint64 `json:"createdBy"`
	// Aliases are previous names that sent content still resolves to this row.
	Aliases []string `json:"aliases"`
}

func stickerAdminItems(entities []sticker.Entity, aliases map[uint64][]string) []StickerAdminItem {
	items := make([]StickerAdminItem, 0, len(entities))
	for _, entity := range entities {
		names := aliases[entity.Id]
		if names == nil {
			names = []string{}
		}
		items = append(items, StickerAdminItem{
			Id:        entity.Id,
			Name:      entity.Name,
			FileName:  entity.FileName,
			URL:       stickerservice.ResolveURLFor(entity),
			Pack:      entity.Pack,
			SortOrder: entity.SortOrder,
			IsEnabled: entity.IsEnabled,
			CreatedBy: entity.CreatedBy,
			Aliases:   names,
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
	ids := make([]uint64, 0, len(entities))
	for _, entity := range entities {
		ids = append(ids, entity.Id)
	}
	aliases, err := sticker.AliasNamesBySticker(ids)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(stickerAdminItems(entities, aliases))
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
	Id       uint64 `json:"id"`
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	// Pack moves the sticker to another official pack; empty keeps it.
	Pack      string `json:"pack"`
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
	case errors.Is(err, stickerservice.ErrNameReserved):
		code = component.MessageAdminStickerNameReserved
	case errors.Is(err, stickerservice.ErrFileRequired):
		code = component.MessageUploadFileMissing
	case errors.Is(err, stickerservice.ErrNotFound):
		code = component.MessageAdminStickerNotFound
	case errors.Is(err, stickerservice.ErrPackNameRequired):
		code = component.MessageAdminStickerNameRequired
	case errors.Is(err, stickerservice.ErrPackInvalid):
		code = component.MessageRequestInvalidParams
	}
	return component.FailResponseCode(code, nil)
}
func SaveSticker(req component.BetterRequest[StickerSaveReq]) component.Response {
	err := stickerservice.Save(betterRequestContext(req), req.UserId, stickerservice.SaveInput{Id: req.Params.Id, Name: req.Params.Name, FileName: req.Params.FileName, Pack: req.Params.Pack, SortOrder: req.Params.SortOrder, IsEnabled: req.Params.IsEnabled})
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

// readStickerUpload reads the multipart `file` field of a sticker upload, a
// ZIP archive or one image. On failure it has already written the response.
func readStickerUpload(c *gin.Context) ([]byte, string, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, stickerservice.ImportMaxZipBytes+(1<<20))
	header, err := c.FormFile("file")
	if c.Request.MultipartForm != nil {
		// The bytes are read into memory before this returns.
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	if err != nil {
		code := component.MessageUploadFileMissing
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			code = component.MessageAdminStickerImportTooLarge
		}
		c.JSON(http.StatusOK, component.FailDataCode(code, nil))
		return nil, "", false
	}
	if header.Size > stickerservice.ImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return nil, "", false
	}
	file, err := header.Open()
	if err != nil {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageRequestParseFailed, nil))
		return nil, "", false
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, stickerservice.ImportMaxZipBytes+1))
	if err != nil || len(data) > stickerservice.ImportMaxZipBytes {
		c.JSON(http.StatusOK, component.FailDataCode(component.MessageAdminStickerImportTooLarge, nil))
		return nil, "", false
	}
	return data, header.Filename, true
}

func stickerImportFailure(c *gin.Context, err error) {
	code := component.MessageOperationFailed
	switch {
	case errors.Is(err, stickerservice.ErrInvalidZip):
		code = component.MessageAdminStickerImportInvalidZip
	case errors.Is(err, stickerservice.ErrPackInvalid):
		code = component.MessageRequestInvalidParams
	case errors.Is(err, stickerservice.ErrMarketPackNotFound):
		code = component.MessageAdminStickerNotFound
	}
	c.JSON(http.StatusOK, component.FailDataCode(code, nil))
}

// ImportStickerPack handles only multipart transport; the service owns all
// validation, naming, storage, and reference lifecycle decisions. The
// optional `pack` field picks the official pack that receives the images.
func ImportStickerPack(c *gin.Context) {
	data, fileName, ok := readStickerUpload(c)
	if !ok {
		return
	}
	result, err := stickerservice.ImportOfficial(c.Request.Context(), c.GetUint64("userId"), c.PostForm("pack"), fileName, data)
	if err != nil {
		stickerImportFailure(c, err)
		return
	}
	c.JSON(http.StatusOK, component.SuccessDataCode(result, component.MessageOperationSuccess, nil))
}

// PublicStickerPacks lists the official packs members can pick from.
func PublicStickerPacks() component.Response {
	packs, err := stickerservice.OfficialPackList()
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(packs)
}

// AdminStickerPacks lists every official pack with its counts.
func AdminStickerPacks(req component.BetterRequest[component.Null]) component.Response {
	packs, err := stickerservice.AdminOfficialPacks(betterRequestContext(req))
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(packs)
}

// AdminStickerPackSave creates a pack (empty key) or edits its name, order and cover.
func AdminStickerPackSave(req component.BetterRequest[stickerservice.SaveOfficialPackInput]) component.Response {
	key, err := stickerservice.SaveOfficialPack(betterRequestContext(req), req.Params)
	if err != nil {
		return stickerWriteError(err, component.MessageAdminStickerSaveFailed)
	}
	return component.SuccessResponse(map[string]string{"key": key})
}

// AdminStickerPackDeleteReq removes a pack after moving its stickers to MoveTo.
type AdminStickerPackDeleteReq struct {
	Key    string `json:"key"`
	MoveTo string `json:"moveTo"`
}

func AdminStickerPackDelete(req component.BetterRequest[AdminStickerPackDeleteReq]) component.Response {
	moved, err := stickerservice.DeleteOfficialPack(betterRequestContext(req), req.Params.Key, req.Params.MoveTo)
	if err != nil {
		return stickerWriteError(err, component.MessageAdminStickerDeleteFailed)
	}
	return component.SuccessResponse(map[string]int64{"moved": moved})
}

// AdminStickerBatch enables, disables, moves or deletes selected official stickers.
func AdminStickerBatch(req component.BetterRequest[stickerservice.BatchInput]) component.Response {
	changed, err := stickerservice.BatchStickers(betterRequestContext(req), req.Params)
	if err != nil {
		return stickerWriteError(err, component.MessageAdminStickerSaveFailed)
	}
	return component.SuccessResponse(map[string]int{"changed": changed})
}
