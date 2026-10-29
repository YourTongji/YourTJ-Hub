package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/optlogger"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// StickerMarketQuery pages the published market (issue #1429).
type StickerMarketQuery struct {
	Topic    string `form:"topic"`
	Keyword  string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type StickerMarketPackQuery struct {
	ID uint64 `form:"id"`
}

type StickerPackReq struct {
	PackID uint64 `json:"packId"`
}

func stickerMarketError(err error) component.Response {
	switch {
	case errors.Is(err, stickerservice.ErrMarketPackUnavailable):
		return component.FailResponseCode(component.MessageStickerMarketPackUnavailable, nil)
	case errors.Is(err, stickerservice.ErrMarketSubscriptionFull):
		return component.FailResponseCode(component.MessageStickerMarketPackLimit, component.MessageParams{"limit": stickerservice.MaxPackSubscriptions})
	default:
		return stickerLibraryError(err)
	}
}

// StickerMarket lists published packs; anyone may browse.
func StickerMarket(req component.BetterRequest[StickerMarketQuery]) component.Response {
	page, err := stickerservice.MarketCatalog(req.Params.Topic, req.Params.Keyword, req.Params.Page, req.Params.PageSize)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(page)
}

// StickerMarketPack returns one published pack with its usable stickers.
func StickerMarketPack(req component.BetterRequest[StickerMarketPackQuery]) component.Response {
	detail, err := stickerservice.MarketPackStickers(req.Params.ID)
	if err != nil {
		return stickerMarketError(err)
	}
	return component.SuccessResponse(detail)
}

func MyStickerPacks(req component.BetterRequest[component.Null]) component.Response {
	packs, err := stickerservice.MyMarketPacks(betterRequestContext(req), req.UserId)
	if err != nil {
		return stickerMarketError(err)
	}
	return component.SuccessResponse(packs)
}

func SaveMyStickerPack(req component.BetterRequest[StickerPackReq]) component.Response {
	pack, err := stickerservice.SubscribeMarketPack(betterRequestContext(req), req.UserId, req.Params.PackID)
	if err != nil {
		return stickerMarketError(err)
	}
	return component.SuccessResponse(pack)
}

func DeleteMyStickerPack(req component.BetterRequest[StickerPackReq]) component.Response {
	if err := stickerservice.UnsubscribeMarketPack(betterRequestContext(req), req.UserId, req.Params.PackID); err != nil {
		return stickerMarketError(err)
	}
	return component.SuccessResponse(true)
}

// AdminStickerMarketQuery pages the full synchronized catalog.
type AdminStickerMarketQuery struct {
	Topic    string `form:"topic"`
	Keyword  string `form:"q"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
}

type AdminStickerMarketPackReq struct {
	ID uint64 `json:"id" form:"id"`
}

type AdminStickerMarketItemReq struct {
	StickerID uint64 `json:"stickerId"`
	// Name renames the sticker when set; its previous name keeps resolving.
	Name      string `json:"name"`
	IsEnabled bool   `json:"isEnabled"`
}

func adminStickerMarketError(err error) component.Response {
	switch {
	case errors.Is(err, stickerservice.ErrMarketBusy):
		return component.FailResponseCode(component.MessageAdminStickerMarketBusy, nil)
	case errors.Is(err, stickerservice.ErrMarketDisabled):
		return component.FailResponseCode(component.MessageAdminStickerMarketDisabled, nil)
	case errors.Is(err, stickerservice.ErrMarketUnavailable):
		return component.FailResponseCode(component.MessageAdminStickerMarketUnavailable, nil)
	case errors.Is(err, stickerservice.ErrMarketPackNotReady):
		return component.FailResponseCode(component.MessageAdminStickerMarketNotReady, nil)
	case errors.Is(err, stickerservice.ErrMarketPackNotFound), errors.Is(err, stickerservice.ErrNotFound):
		return component.FailResponseCode(component.MessageAdminStickerNotFound, nil)
	default:
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
}

func AdminStickerMarket(req component.BetterRequest[AdminStickerMarketQuery]) component.Response {
	page, err := stickerservice.AdminMarketCatalog(req.Params.Topic, req.Params.Keyword, req.Params.Status, req.Params.Page, req.Params.PageSize)
	if err != nil {
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
	return component.SuccessResponse(page)
}

// AdminStickerMarketSync refreshes catalog metadata; it never imports images.
func AdminStickerMarketSync(req component.BetterRequest[component.Null]) component.Response {
	result, err := stickerservice.SyncMarketCatalog(betterRequestContext(req))
	if err != nil {
		return adminStickerMarketError(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.ManageStickerMarket, 0, "admin.opt.stickerMarket.synced", optlogger.MessageParams{"packs": result.Packs})
	return component.SuccessResponse(result)
}

func AdminStickerMarketPack(req component.BetterRequest[AdminStickerMarketPackReq]) component.Response {
	detail, err := stickerservice.AdminMarketPackItems(req.Params.ID)
	if err != nil {
		return adminStickerMarketError(err)
	}
	return component.SuccessResponse(detail)
}

// AdminStickerMarketImport queues a server-side download of one pack.
func AdminStickerMarketImport(req component.BetterRequest[AdminStickerMarketPackReq]) component.Response {
	queued, err := stickerservice.QueueMarketImport(betterRequestContext(req), req.Params.ID)
	if err != nil {
		return adminStickerMarketError(err)
	}
	optlogger.UserOptCode(req.UserId, optlogger.ManageStickerMarket, queued.Id, "admin.opt.stickerMarket.imported", optlogger.MessageParams{"pack": queued.Name})
	pack, err := stickerservice.AdminMarketPackByID(queued.Id)
	if err != nil {
		return adminStickerMarketError(err)
	}
	return component.SuccessResponse(pack)
}

// AdminStickerMarketPackSave publishes, hides or reorders one pack.
func AdminStickerMarketPackSave(req component.BetterRequest[stickerservice.SaveMarketPackInput]) component.Response {
	saved, err := stickerservice.SaveMarketPack(betterRequestContext(req), req.Params)
	if err != nil {
		return adminStickerMarketError(err)
	}
	code := "admin.opt.stickerMarket.hidden"
	if saved.Published {
		code = "admin.opt.stickerMarket.published"
	}
	optlogger.UserOptCode(req.UserId, optlogger.ManageStickerMarket, saved.Id, code, optlogger.MessageParams{"pack": saved.Name})
	pack, err := stickerservice.AdminMarketPackByID(saved.Id)
	if err != nil {
		return adminStickerMarketError(err)
	}
	return component.SuccessResponse(pack)
}

// AdminStickerMarketItemSave withdraws, restores or renames one market sticker.
func AdminStickerMarketItemSave(req component.BetterRequest[AdminStickerMarketItemReq]) component.Response {
	before, saved, err := stickerservice.SaveMarketSticker(betterRequestContext(req), stickerservice.SaveMarketStickerInput{
		StickerID: req.Params.StickerID, Name: req.Params.Name, IsEnabled: req.Params.IsEnabled,
	})
	if errors.Is(err, stickerservice.ErrNameInvalid) || errors.Is(err, stickerservice.ErrNameExists) || errors.Is(err, stickerservice.ErrNameReserved) {
		return stickerWriteError(err, component.MessageOperationFailed)
	}
	if err != nil {
		return adminStickerMarketError(err)
	}
	if saved.Name != before.Name {
		optlogger.UserOptCode(req.UserId, optlogger.ManageStickerMarket, saved.Id, "admin.opt.stickerMarket.stickerRenamed", optlogger.MessageParams{"from": before.Name, "to": saved.Name})
	}
	if saved.IsEnabled != before.IsEnabled {
		code := "admin.opt.stickerMarket.stickerDisabled"
		if saved.IsEnabled {
			code = "admin.opt.stickerMarket.stickerEnabled"
		}
		optlogger.UserOptCode(req.UserId, optlogger.ManageStickerMarket, saved.Id, code, optlogger.MessageParams{"id": saved.Id})
	}
	return component.SuccessResponse(true)
}

// AdminStickerMarketCover redirects to the stored thumbnail of a pack's
// upstream cover, creating it on first request, so the admin catalog shows
// covers before import without the browser contacting the upstream host.
func AdminStickerMarketCover(c *gin.Context) {
	packID, _ := strconv.ParseUint(c.Query("id"), 10, 64)
	name, err := stickerservice.MarketCoverPreview(c.Request.Context(), packID)
	if err != nil {
		// Failures are remembered briefly server-side; let the browser back off too.
		c.Header("Cache-Control", "private, max-age=600")
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Redirect(http.StatusFound, storageservice.PublicAccessPath(name))
}
