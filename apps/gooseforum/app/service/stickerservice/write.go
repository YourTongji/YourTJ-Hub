package stickerservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
	"gorm.io/gorm"
)

var (
	ErrNameRequired = errors.New("sticker name required")
	ErrNameInvalid  = errors.New("invalid sticker name")
	ErrNameExists   = errors.New("sticker name exists")
	// ErrNameReserved: the name is another sticker's previous name and existing
	// content still resolves it to that sticker (issue #1428).
	ErrNameReserved = errors.New("sticker name reserved by a previous name")
	ErrFileRequired = errors.New("an owned image upload is required")
	ErrNotFound     = errors.New("sticker not found")
)

// SaveInput creates (Id 0) or edits one official sticker. An empty Pack keeps
// the current group, or the default group for a new sticker.
type SaveInput struct {
	Id             uint64
	Name, FileName string
	Pack           string
	SortOrder      int
	IsEnabled      bool
}

// checkStickerNameTx reports whether the sticker rowID (0 for a new one) may
// take name: another sticker's current or previous name is unavailable,
// because sent content resolves it to that sticker (issue #1428).
func checkStickerNameTx(tx *gorm.DB, rowID uint64, name string) error {
	existing, err := sticker.GetByNameTx(tx, name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing.Id != 0 && existing.Id != rowID {
		return ErrNameExists
	}
	alias, err := sticker.AliasByNameTx(tx, name)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil && (rowID == 0 || alias.StickerID != rowID) {
		return ErrNameReserved
	}
	return nil
}

// saveRenamedStickerTx stores an existing sticker whose name may have changed
// and keeps previousName resolving to it.
func saveRenamedStickerTx(tx *gorm.DB, row *sticker.Entity, previousName string) error {
	// Taking back a previous name releases it before the row claims it.
	if err := sticker.ReleaseAliasTx(tx, row.Name, row.Id); err != nil {
		return err
	}
	if err := sticker.SaveTx(tx, row); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrNameExists
		}
		return err
	}
	// Sent tokens keep the old name; record it so they still resolve here.
	if previousName != row.Name {
		return sticker.RecordAliasTx(tx, previousName, row.Id)
	}
	return nil
}

func Save(ctx context.Context, userID uint64, input SaveInput) error {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return ErrNameRequired
	}
	if !ValidateName(input.Name) {
		return ErrNameInvalid
	}
	err := db.Connect().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := sticker.Entity{CreatedBy: userID, IsOfficial: true, Pack: sticker.PackOfficial}
		if input.Id != 0 {
			existing, err := sticker.GetByIDTx(tx, input.Id)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			if err != nil {
				return err
			}
			if !existing.IsOfficial {
				return ErrNotFound
			}
			row = existing
		}
		// Keep duplicate-name errors independent of whether a new upload was supplied.
		if err := checkStickerNameTx(tx, row.Id, input.Name); err != nil {
			return err
		}
		previousName := row.Name
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
		if pack := strings.TrimSpace(input.Pack); pack != "" && pack != row.Pack {
			if known, err := knownPackTx(tx, pack); err != nil {
				return err
			} else if !known {
				return ErrPackInvalid
			}
			row.Pack = pack
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
		} else if err := saveRenamedStickerTx(tx, &row, previousName); err != nil {
			return err
		}
		return fileusageservice.SetStickerUsageTx(tx, row.Id, row.CreatedBy, row.FileName, fileUsage.UsageStatusActive)
	})
	if err == nil {
		InvalidateCache()
	}
	return err
}

func Delete(ctx context.Context, id uint64) error {
	err := db.Connect().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row, err := sticker.GetByIDTx(tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if !row.IsOfficial {
			return ErrNotFound
		}
		if err := fileusageservice.SetStickerUsageTx(tx, id, 0, "", fileUsage.UsageStatusActive); err != nil {
			return err
		}
		return sticker.DeleteTx(tx, id)
	})
	if err == nil {
		InvalidateCache()
	}
	return err
}

// ImportImage is the shared ZIP/preset path. File storage is a separate database:
// compensate its write when the atomic sticker/reference transaction fails.
// skipExisting makes preset seeding idempotent even across concurrent runs.
func ImportImage(ctx context.Context, userID uint64, data []byte, fileName, rawName, pack string, sortOrder int, skipExisting bool) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if len(data) > filedata.MaxFileSize {
		return false, errors.New("tooLarge")
	}
	// 声明扩展名仍先过支持集（危险扩展在读内容前拒绝）；真实类型只由文件头决定，
	// 入库名按真实类型取扩展名（issue #1393）。
	if _, ok := imagepolicy.ContentTypeForFilename(fileName); !ok {
		return false, errors.New("invalidImage")
	}
	contentType, contentErr := imagepolicy.DetectContentType(bytes.NewReader(data))
	if contentErr != nil {
		return false, errors.New("invalidImage")
	}
	base := SanitizeName(rawName)
	if base == "" {
		return false, errors.New("unusableName")
	}
	file, err := filedata.SaveFileFromUpload(userID, data, contentType, "stickers")
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
			// A previous name still resolves to its renamed sticker: treat it as taken.
			if _, err := sticker.AliasByNameTx(tx, name); err == nil {
				if skipExisting {
					skipped = true
					return nil
				}
				continue
			} else if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			row := sticker.Entity{Name: name, FileName: file.Name, SortOrder: sortOrder, IsEnabled: true, IsOfficial: true, Pack: pack, CreatedBy: userID}
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
			return fileusageservice.SetStickerUsageTx(tx, row.Id, userID, file.Name, fileUsage.UsageStatusActive)
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
	if !skipped {
		InvalidateCache()
	}
	return skipped, nil
}
