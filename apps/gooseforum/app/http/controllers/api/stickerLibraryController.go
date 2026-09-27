package api

import (
	"errors"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/stickerservice"
)

type StickerNamesReq struct {
	Names []string `json:"names"`
}
type StickerNameReq struct {
	Name string `json:"name"`
}

func stickerLibraryError(err error) component.Response {
	switch {
	case errors.Is(err, stickerservice.ErrLibraryFull):
		return component.FailResponseCode(component.MessageStickerLibraryFull, component.MessageParams{"limit": stickerservice.MaxLibraryItems})
	case errors.Is(err, stickerservice.ErrUploadQuota):
		return component.FailResponseCode(component.MessageStickerUploadQuota, component.MessageParams{"limit": stickerservice.MaxPersonalUploads})
	case errors.Is(err, stickerservice.ErrInvalidLibraryInput):
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	case errors.Is(err, stickerservice.ErrFileRequired):
		return component.FailResponseCode(component.MessageStickerImageRequired, component.MessageParams{"maxSizeMb": 4})
	case errors.Is(err, stickerservice.ErrNotFound):
		return component.FailResponseCode(component.MessageStickerUnavailable, nil)
	case errors.Is(err, sticker.ErrLibraryClosed):
		return component.FailResponseCode(component.MessageAuthRequired, nil)
	default:
		return component.FailResponseCode(component.MessageOperationFailed, nil)
	}
}

func ResolveStickers(req component.BetterRequest[StickerNamesReq]) component.Response {
	if req.Params.Names == nil {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	items, err := stickerservice.Resolve(req.Params.Names)
	if err != nil {
		return stickerLibraryError(err)
	}
	return component.SuccessResponse(items)
}

func MyStickers(req component.BetterRequest[component.Null]) component.Response {
	items, err := stickerservice.MyLibrary(betterRequestContext(req), req.UserId)
	if err != nil {
		return stickerLibraryError(err)
	}
	return component.SuccessResponse(items)
}

func SaveMySticker(req component.BetterRequest[stickerservice.LibrarySaveInput]) component.Response {
	item, err := stickerservice.SaveToLibrary(betterRequestContext(req), req.UserId, req.Params)
	if err != nil {
		return stickerLibraryError(err)
	}
	return component.SuccessResponse(item)
}

func DeleteMySticker(req component.BetterRequest[StickerNameReq]) component.Response {
	if err := stickerservice.RemoveFromLibrary(betterRequestContext(req), req.UserId, req.Params.Name); err != nil {
		return stickerLibraryError(err)
	}
	return component.SuccessResponse(true)
}

func OrderMyStickers(req component.BetterRequest[StickerNamesReq]) component.Response {
	if req.Params.Names == nil {
		return component.FailResponseCode(component.MessageRequestInvalidParams, nil)
	}
	if err := stickerservice.OrderLibrary(betterRequestContext(req), req.UserId, req.Params.Names); err != nil {
		return stickerLibraryError(err)
	}
	return component.SuccessResponse(true)
}
