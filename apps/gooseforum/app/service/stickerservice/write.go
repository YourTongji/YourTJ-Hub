package stickerservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"gorm.io/gorm"
)

var (
	ErrNameRequired = errors.New("sticker name required")
	ErrNameInvalid  = errors.New("invalid sticker name")
	ErrNameExists   = errors.New("sticker name exists")
	ErrFileRequired = errors.New("an owned image upload is required")
	ErrNotFound     = errors.New("sticker not found")
)

type SaveInput struct {
	Id             uint64
	Name, FileName string
	SortOrder      int
	IsEnabled      bool
}

func Save(ctx context.Context, userID uint64, input SaveInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ErrNameRequired
	}
	if !ValidateName(input.Name) {
		return ErrNameInvalid
	}
	return db.Connect().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := sticker.Entity{CreatedBy: userID}
		if input.Id != 0 {
			existing, err := sticker.GetByIDTx(tx, input.Id)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			row = existing
		}
		// Keep duplicate-name errors independent of whether a new upload was supplied.
		existing, lookupErr := sticker.GetByNameTx(tx, input.Name)
		if lookupErr != nil && !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			return lookupErr
		}
		if existing.Id != 0 && existing.Id != row.Id {
			return ErrNameExists
		}
		fileName := row.FileName
		if input.FileName != "" {
			fileName = strings.TrimPrefix(input.FileName, storageservice.PublicAccessPath(""))
			fileName = strings.TrimPrefix(fileName, "/file/img/")
			file, err := filedata.GetFileMetadataByNameContext(ctx, fileName)
			_, image := imagepolicy.ContentTypeForFilename(fileName)
			if err != nil || file.UserId != userID || !image {
				return ErrFileRequired
			}
		}
		if row.Id == 0 && fileName == "" {
			return ErrFileRequired
		}
		row.Name = input.Name
		row.FileName = fileName
		row.SortOrder = input.SortOrder
		row.IsEnabled = input.IsEnabled
		if row.Id == 0 {
			inserted, err := sticker.InsertIfNameAvailable(tx, &row)
			if err != nil {
				return err
			}
			if !inserted {
				return ErrNameExists
			}
			// GORM's true default must not override an explicitly disabled upload.
			if !input.IsEnabled {
				row.IsEnabled = false
				if err := sticker.SaveTx(tx, &row); err != nil {
					return err
				}
			}
		} else if err := sticker.SaveTx(tx, &row); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrNameExists
			}
			return err
		}
		return fileusageservice.SetStickerUsageTx(tx, row.Id, row.CreatedBy, row.FileName)
	})
}

func Delete(ctx context.Context, id uint64) error {
	return db.Connect().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := sticker.GetByIDTx(tx, id); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if err := fileusageservice.SetStickerUsageTx(tx, id, 0, ""); err != nil {
			return err
		}
		return sticker.DeleteTx(tx, id)
	})
}

// ImportImage is the shared ZIP/preset path. File storage is a separate database:
// compensate its write when the atomic sticker/reference transaction fails.
// skipExisting makes preset seeding idempotent even across concurrent runs.
func ImportImage(ctx context.Context, userID uint64, data []byte, fileName, rawName string, sortOrder int, skipExisting bool) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if len(data) > filedata.MaxFileSize {
		return false, errors.New("tooLarge")
	}
	contentType, ok := imagepolicy.ContentTypeForFilename(fileName)
	if !ok || imagepolicy.ValidateContent(bytes.NewReader(data), contentType) != nil {
		return false, errors.New("invalidImage")
	}
	base := SanitizeName(rawName)
	if base == "" {
		return false, errors.New("unusableName")
	}
	file, err := filedata.SaveFileFromUpload(userID, data, path.Base(fileName), "stickers")
	if err != nil {
		return false, fmt.Errorf("saveFailed: %w", err)
	}
	skipped := false
	err = db.Connect().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for attempt := 1; attempt <= 1000; attempt++ {
			name := base
			if attempt > 1 {
				suffix := "-" + strconv.Itoa(attempt)
				runes := []rune(base)
				if len(runes)+len(suffix) > sticker.MaxNameLen {
					runes = runes[:sticker.MaxNameLen-len(suffix)]
				}
				name = string(runes) + suffix
			}
			row := sticker.Entity{Name: name, FileName: file.Name, SortOrder: sortOrder, IsEnabled: true, CreatedBy: userID}
			inserted, err := sticker.InsertIfNameAvailable(tx, &row)
			if err != nil {
				return err
			}
			if !inserted {
				if skipExisting {
					skipped = true
					return nil
				}
				continue
			}
			return fileusageservice.SetStickerUsageTx(tx, row.Id, userID, file.Name)
		}
		return ErrNameExists
	})
	if err != nil || skipped {
		if cleanupErr := filedata.DeleteByName(file.Name); cleanupErr != nil {
			return false, fmt.Errorf("saveFailed: %w", errors.Join(err, cleanupErr))
		}
	}
	if err != nil {
		return false, fmt.Errorf("saveFailed: %w", err)
	}
	return skipped, nil
}
