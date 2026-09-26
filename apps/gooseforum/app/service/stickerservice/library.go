package stickerservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
)

const MaxLibraryItems = 200
const MaxPersonalUploads = 1000
const MaxResolveNames = 200

var ErrLibraryFull = errors.New("sticker library full")
var ErrUploadQuota = errors.New("personal sticker upload quota reached")
var ErrInvalidLibraryInput = errors.New("invalid sticker library input")

type LibrarySaveInput struct {
	StickerName string  `json:"stickerName"`
	FileName    string  `json:"fileName"`
	DisplayName *string `json:"displayName"`
}

// Resolve accepts explicit, bounded tokens, never an index of private assets.
// Tokens identify shared content; knowing an unguessable personal token permits
// rendering and collecting it. Private membership and labels are never returned.
func Resolve(names []string) ([]StickerItem, error) {
	if len(names) > MaxResolveNames {
		return nil, ErrInvalidLibraryInput
	}
	// Content may contain token-looking text that was never a valid asset name.
	// Ignore it like an unknown token, without poisoning valid names in the same
	// batch. Apply the total-input bound first, even when every name is invalid.
	validNames := make([]string, 0, len(names))
	for _, name := range names {
		if ValidateName(name) {
			validNames = append(validNames, name)
		}
	}
	entities, err := sticker.EnabledByNames(validNames)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]sticker.Entity, len(entities))
	for _, entity := range entities {
		byName[entity.Name] = entity
	}
	items := make([]StickerItem, 0, len(entities))
	for _, name := range names {
		if entity, ok := byName[name]; ok {
			items = append(items, itemFor(entity, ""))
			delete(byName, name)
		}
	}
	return items, nil
}

func MyLibrary(ctx context.Context, userID uint64) ([]StickerItem, error) {
	items := make([]StickerItem, 0)
	type labeledAsset struct {
		entity sticker.Entity
		label  string
	}
	var assets []labeledAsset
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		entries, err := sticker.LibraryEntriesTx(tx, userID)
		if err != nil {
			return err
		}
		ids := make([]uint64, 0, len(entries))
		for _, entry := range entries {
			ids = append(ids, entry.StickerID)
		}
		entities, err := sticker.EntitiesByIDsTx(tx, ids)
		if err != nil {
			return err
		}
		byID := make(map[uint64]sticker.Entity, len(entities))
		for _, entity := range entities {
			byID[entity.Id] = entity
		}
		for _, entry := range entries {
			if entity, ok := byID[entry.StickerID]; ok {
				assets = append(assets, labeledAsset{entity, entry.DisplayName})
			}
		}
		return nil
	})
	if err == nil {
		for _, asset := range assets {
			items = append(items, itemFor(asset.entity, asset.label))
		}
	}
	return items, err
}

func SaveToLibrary(ctx context.Context, userID uint64, input LibrarySaveInput) (StickerItem, error) {
	return saveToLibrary(db.ConnectContext(ctx), ctx, userID, input)
}

func saveToLibrary(conn *gorm.DB, ctx context.Context, userID uint64, input LibrarySaveInput) (StickerItem, error) {
	var item StickerItem
	var savedEntity sticker.Entity
	var savedLabel string
	if userID == 0 || (input.StickerName == "") == (input.FileName == "") {
		return item, ErrInvalidLibraryInput
	}
	if len(input.FileName) > 2048 || (input.StickerName != "" && !ValidateName(input.StickerName)) {
		return item, ErrInvalidLibraryInput
	}
	if input.DisplayName != nil {
		label := strings.TrimSpace(*input.DisplayName)
		if !utf8.ValidString(label) || utf8.RuneCountInString(label) > 64 || strings.IndexFunc(label, unicode.IsControl) >= 0 {
			return item, ErrInvalidLibraryInput
		}
		input.DisplayName = &label
	}
	fileName := ""
	if input.FileName != "" {
		fileName = fileusageservice.FileNameFromURL(input.FileName)
		file, err := filedata.GetFileMetadataByNameContext(ctx, fileName)
		_, image := imagepolicy.ContentTypeForFilename(fileName)
		if err != nil || !image || file.UserId != userID || file.Size <= 0 || file.Size > filedata.MaxFileSize {
			return item, ErrFileRequired
		}
	}
	err := conn.Transaction(func(tx *gorm.DB) error {
		if err := sticker.LockLibraryTx(tx, userID); err != nil {
			return err
		}
		entries, err := sticker.LibraryEntriesTx(tx, userID)
		if err != nil {
			return err
		}
		var entity sticker.Entity
		if input.StickerName != "" {
			entity, err = sticker.GetByNameTx(tx, input.StickerName)
			if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && !entity.IsEnabled) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
		} else {
			entity, err = sticker.PersonalUploadTx(tx, userID, fileName)
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if entity.Id != 0 && !entity.IsEnabled {
			return ErrNotFound
		}
		for _, entry := range entries {
			if entry.StickerID != entity.Id {
				continue
			}
			if input.DisplayName != nil {
				entry.DisplayName = *input.DisplayName
				if err := sticker.SaveLibraryEntryTx(tx, &entry); err != nil {
					return err
				}
			}
			savedEntity, savedLabel = entity, entry.DisplayName
			return nil
		}
		if len(entries) >= MaxLibraryItems {
			return ErrLibraryFull
		}
		if entity.Id == 0 {
			count, err := sticker.CountPersonalUploadsTx(tx, userID)
			if err != nil {
				return err
			}
			if count >= MaxPersonalUploads {
				return ErrUploadQuota
			}
			var random [24]byte
			if _, err := rand.Read(random[:]); err != nil {
				return err
			}
			entity = sticker.Entity{Name: "u_" + hex.EncodeToString(random[:]), FileName: fileName, CreatedBy: userID, IsEnabled: true, Pack: "personal", DisplayName: ""}
			// The user's chosen label stays only on their membership. Public
			// resolution never leaks labels belonging to a different account.
			if err := sticker.InsertPersonalTx(tx, &entity); err != nil {
				return err
			}
			if err := fileusageservice.SetStickerUsageTx(tx, entity.Id, userID, fileName); err != nil {
				return err
			}
		}
		order := 0
		for _, entry := range entries {
			if entry.SortOrder >= order {
				order = entry.SortOrder + 1
			}
		}
		entry := sticker.LibraryEntry{UserID: userID, StickerID: entity.Id, SortOrder: order}
		if input.DisplayName != nil {
			entry.DisplayName = *input.DisplayName
		}
		if err := sticker.SaveLibraryEntryTx(tx, &entry); err != nil {
			return err
		}
		savedEntity, savedLabel = entity, entry.DisplayName
		return nil
	})
	if err == nil {
		item = itemFor(savedEntity, savedLabel)
	}
	return item, err
}

func RemoveFromLibrary(ctx context.Context, userID uint64, name string) error {
	if userID == 0 || !ValidateName(name) {
		return ErrInvalidLibraryInput
	}
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.LockLibraryTx(tx, userID); err != nil {
			return err
		}
		entity, err := sticker.GetByNameTx(tx, name)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		return sticker.RemoveLibraryEntryTx(tx, userID, entity.Id)
	})
}

// OrderLibrary requires the exact current membership once each. Stale requests
// fail rather than silently removing or appending a concurrent collection.
func OrderLibrary(ctx context.Context, userID uint64, names []string) error {
	if userID == 0 || len(names) > MaxLibraryItems {
		return ErrInvalidLibraryInput
	}
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.LockLibraryTx(tx, userID); err != nil {
			return err
		}
		entries, err := sticker.LibraryEntriesTx(tx, userID)
		if err != nil {
			return err
		}
		if len(entries) != len(names) {
			return ErrInvalidLibraryInput
		}
		ids := make([]uint64, 0, len(entries))
		byID := make(map[uint64]sticker.LibraryEntry, len(entries))
		for _, entry := range entries {
			ids = append(ids, entry.StickerID)
			byID[entry.StickerID] = entry
		}
		entities, err := sticker.EntitiesByIDsTx(tx, ids)
		if err != nil {
			return err
		}
		byName := make(map[string]sticker.LibraryEntry, len(entities))
		for _, entity := range entities {
			byName[entity.Name] = byID[entity.Id]
		}
		for position, name := range names {
			entry, ok := byName[name]
			if !ok {
				return ErrInvalidLibraryInput
			}
			delete(byName, name)
			entry.SortOrder = position
			if err := sticker.SaveLibraryEntryTx(tx, &entry); err != nil {
				return err
			}
		}
		return nil
	})
}

func CloseLibrary(ctx context.Context, userID uint64) error {
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error { return sticker.CloseLibraryTx(tx, userID) })
}
