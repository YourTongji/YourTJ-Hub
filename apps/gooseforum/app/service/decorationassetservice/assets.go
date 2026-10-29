package decorationassetservice

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalid = errors.New("invalid decoration asset")

// ValidateArtwork requires a centered transparent circle spanning 30% of the shorter side in every frame.
func ValidateArtwork(data []byte, contentType string) error {
	return validateRaster(data, contentType, true)
}

// UploadAsset gives an unsaved draft 24 hours to use its preview URL. A saved
// definition clears the expiry; CleanupAssets reclaims abandoned/replaced files.
func UploadAsset(ctx context.Context, actorID uint64, data []byte, filename string) (string, error) {
	return uploadAsset(ctx, actorID, data, filename, "decorations")
}

func UploadBadgeAsset(ctx context.Context, actorID uint64, data []byte, filename string) (string, error) {
	return uploadAsset(ctx, actorID, data, filename, "badges")
}

func uploadAsset(ctx context.Context, actorID uint64, data []byte, filename, prefix string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	data, filename, err := prepareArtwork(ctx, data, filename, prefix == "decorations")
	if err != nil {
		return "", err
	}
	storagePrefix := prefix
	if prefix == "badges" {
		storagePrefix = "badges/normalized"
	}
	contentType, ok := ContentType(filename)
	if !ok {
		return "", ErrInvalid
	}
	file, err := filedata.SaveFileFromUploadWithLimit(actorID, data, contentType, storagePrefix, MaxArtworkBytes)
	if err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		_ = filedata.DeleteByName(file.Name)
		return "", err
	}
	expires := time.Now().Add(24 * time.Hour)
	err = db.ConnectContext(ctx).Create(&fileUsage.Entity{FileName: file.Name, TargetType: assetTarget(prefix), TargetId: file.Id, UsageType: assetTarget(prefix), UserId: actorID, Status: fileUsage.UsageStatusActive, ExpiresAt: &expires}).Error
	if err != nil {
		_ = filedata.DeleteByName(file.Name)
		return "", err
	}
	return "/static/" + file.Name, nil
}

// ReadAsset serves only generated names with a live reference. Embedded assets
// are resolved separately by the static mount.
func ReadAsset(name string) ([]byte, time.Time, error) {
	if !fileUsage.IsManagedAssetName(name) {
		return nil, time.Time{}, fs.ErrNotExist
	}
	var usage fileUsage.Entity
	err := db.Connect().Where("file_name = ? AND target_type = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", name, assetTarget(strings.SplitN(name, "/", 2)[0]), fileUsage.UsageStatusActive, time.Now()).First(&usage).Error
	if err != nil {
		return nil, time.Time{}, fs.ErrNotExist
	}
	file, err := filedata.GetFileByName(name)
	if err != nil {
		return nil, time.Time{}, fs.ErrNotExist
	}
	return file.Data, file.CreatedAt, nil
}

// CleanupAssets is safe to retry after an interrupted file deletion. It uses
// the existing reference row as the GC queue and never deletes a live definition.
func CleanupAssets(ctx context.Context, before time.Time, limit int) (int, error) {
	var expired []fileUsage.Entity
	if err := db.ConnectContext(ctx).Where("target_type IN ? AND expires_at IS NOT NULL AND expires_at <= ?", []string{fileUsage.TargetDecorationAsset, fileUsage.TargetBadgeAsset}, before).Limit(limit).Find(&expired).Error; err != nil {
		return 0, err
	}
	removed := 0
	for _, candidate := range expired {
		name := ""
		err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			var usage fileUsage.Entity
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND expires_at IS NOT NULL AND expires_at <= ?", candidate.Id, before).First(&usage).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			} else if err != nil {
				return err
			}
			var count int64
			if err := fileUsage.ManagedAssetReferenceCount(tx, usage.FileName, &count); err != nil {
				return err
			}
			if count != 0 {
				return tx.Model(&usage).Updates(map[string]any{"status": fileUsage.UsageStatusActive, "expires_at": nil}).Error
			}
			name = usage.FileName
			return tx.Model(&usage).Update("status", fileUsage.UsageStatusPurged).Error
		})
		if err != nil {
			return removed, err
		}
		if name == "" {
			continue
		}
		if err := filedata.DeleteByName(name); err != nil {
			return removed, err
		}
		if err := db.ConnectContext(ctx).Where("id = ? AND status = ? AND expires_at IS NOT NULL", candidate.Id, fileUsage.UsageStatusPurged).Delete(&fileUsage.Entity{}).Error; err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func assetTarget(prefix string) string {
	if prefix == "badges" {
		return fileUsage.TargetBadgeAsset
	}
	return fileUsage.TargetDecorationAsset
}
