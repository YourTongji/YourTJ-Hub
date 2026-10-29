package stickerservice

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"log/slog"
	"net/http"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/filemodel/filedata"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
)

// TaskTypeMarketImport materializes one administrator-selected market pack.
const TaskTypeMarketImport = "sticker-market-import"

// MarketAssetPrefix holds content-addressed market images: the object name is
// the SHA-256 of the downloaded bytes, so one image is stored once no matter
// how many packs, URLs or sticker tokens refer to it.
const MarketAssetPrefix = "stickers/market/"

const marketImportWorkers = 4

var ErrMarketPackNotFound = errors.New("sticker market pack not found")

type marketImportTask struct {
	PackID uint64 `json:"packId"`
}

// QueueMarketImport schedules a download of the pack's current upstream items.
// Items that already have a local sticker are kept as they are; retries only
// fetch what is missing. Publication is a separate administrator decision.
func QueueMarketImport(ctx context.Context, packID uint64) (sticker.MarketPack, error) {
	var pack sticker.MarketPack
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		pack, err = sticker.MarketPackByIDTx(tx, packID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMarketPackNotFound
		}
		if err != nil {
			return err
		}
		if marketImportBusy(pack) {
			return ErrMarketBusy
		}
		payload, err := json.Marshal(marketImportTask{PackID: pack.Id})
		if err != nil {
			return err
		}
		pack.Status = sticker.MarketStatusQueued
		pack.LastError = ""
		if err := sticker.SaveMarketPackTx(tx, &pack); err != nil {
			return err
		}
		return taskQueue.CreateTx(tx, &taskQueue.Entity{Type: TaskTypeMarketImport, Status: taskQueue.StatusPending, TaskJson: string(payload)})
	})
	return pack, err
}

// RunMarketImportTask downloads a pack in a background worker. A catalog or
// network failure leaves the previously imported items untouched and returns
// an error so the queue retries; per-item failures are recorded and reported.
func RunMarketImportTask(ctx context.Context, task *taskQueue.Entity) error {
	if task == nil || task.Type != TaskTypeMarketImport {
		return errors.New("invalid sticker market import task")
	}
	var payload marketImportTask
	if err := json.Unmarshal([]byte(task.TaskJson), &payload); err != nil || payload.PackID == 0 {
		return errors.New("invalid sticker market import payload")
	}
	pack, err := sticker.MarketPackByID(payload.PackID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := setMarketPackStatus(ctx, pack.Id, sticker.MarketStatusImporting, ""); err != nil {
		return err
	}
	config, err := loadMarketConfig()
	if err == nil && !config.Enabled {
		err = ErrMarketDisabled
	}
	var emojis []upstreamEmoji
	if err == nil {
		emojis, err = fetchMarketGroup(ctx, config, pack.UpstreamID)
	}
	if err != nil {
		reason := marketErrorReason(err)
		if errors.Is(err, ErrMarketDisabled) {
			reason = "disabled"
		}
		if statusErr := finishMarketImport(ctx, pack.Id, reason); statusErr != nil {
			return errors.Join(err, statusErr)
		}
		return err
	}
	if err := startMarketImportRun(ctx, pack.Id, emojis); err != nil {
		return err
	}
	importMarketItems(ctx, config, pack, emojis)
	if err := ctx.Err(); err != nil {
		// Lease loss or shutdown: the queue owns the retry; keep partial progress.
		return err
	}
	return finishMarketImport(ctx, pack.Id, "")
}

// startMarketImportRun resets the run's progress: failed rows are retried from
// scratch, so only stickers that already exist count before the run begins.
func startMarketImportRun(ctx context.Context, packID uint64, emojis []upstreamEmoji) error {
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("pack_id = ? AND sticker_id = 0", packID).Delete(&sticker.MarketItem{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&sticker.MarketPack{}).Where("id = ?", packID).Update("import_total", len(emojis)).Error; err != nil {
			return err
		}
		return updateMarketPackCountsTx(tx, packID)
	})
}

// updateMarketPackCountsTx recomputes the ready/failed counters from item rows
// so concurrent workers never lose an increment.
func updateMarketPackCountsTx(tx *gorm.DB, packID uint64) error {
	ready := tx.Model(&sticker.MarketItem{}).Select("count(*)").Where("pack_id = ? AND sticker_id <> 0", packID)
	failed := tx.Model(&sticker.MarketItem{}).Select("count(*)").Where("pack_id = ? AND sticker_id = 0", packID)
	return tx.Model(&sticker.MarketPack{}).Where("id = ?", packID).
		Updates(map[string]any{"item_count": ready, "failed_count": failed}).Error
}

func setMarketPackStatus(ctx context.Context, packID uint64, status, reason string) error {
	return db.ConnectContext(ctx).Model(&sticker.MarketPack{}).Where("id = ?", packID).
		Updates(map[string]any{"status": status, "last_error": reason}).Error
}

// importMarketItems fetches every item without a local sticker using a small
// worker pool. Already materialized items are immutable: a changed upstream
// image never alters what earlier messages show.
func importMarketItems(ctx context.Context, config marketConfig, pack sticker.MarketPack, emojis []upstreamEmoji) {
	existing, err := sticker.MarketItems(pack.Id)
	if err != nil {
		slog.Warn("sticker market: load pack items failed", "pack", pack.Id, "err", err)
	}
	done := make(map[string]bool, len(existing))
	for _, item := range existing {
		done[item.UpstreamID] = item.StickerID != 0
	}
	type job struct {
		index int
		emoji upstreamEmoji
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for range marketImportWorkers {
		wg.Go(func() {
			for next := range jobs {
				if ctx.Err() != nil {
					continue
				}
				importMarketItem(ctx, config, pack, next.index, next.emoji)
				if err := updateMarketPackCountsTx(db.ConnectContext(ctx), pack.Id); err != nil && ctx.Err() == nil {
					slog.Warn("sticker market: update import progress failed", "pack", pack.Id, "err", err)
				}
			}
		})
	}
	for index, emoji := range emojis {
		if done[emoji.ID] {
			// Keep the upstream order current without downloading again.
			_ = db.ConnectContext(ctx).Model(&sticker.MarketItem{}).
				Where("pack_id = ? AND upstream_id = ?", pack.Id, emoji.ID).Update("sort_order", index).Error
			continue
		}
		jobs <- job{index, emoji}
	}
	close(jobs)
	wg.Wait()
}

func importMarketItem(ctx context.Context, config marketConfig, pack sticker.MarketPack, index int, emoji upstreamEmoji) {
	item := sticker.MarketItem{PackID: pack.Id, UpstreamID: emoji.ID, SortOrder: index}
	fileName, err := downloadMarketImage(ctx, config, emoji.URL)
	if err == nil {
		err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
			row := sticker.Entity{
				Name:        marketStickerName(pack.Source, pack.UpstreamID, emoji.ID),
				DisplayName: marketStickerLabel(pack.Name, index),
				FileName:    fileName,
				SortOrder:   index,
				IsEnabled:   true,
			}
			inserted, err := sticker.InsertMarketStickerTx(tx, &row)
			if err != nil {
				return err
			}
			if !inserted {
				// The token exists from an earlier, interrupted attempt; reuse it.
				if row, err = sticker.GetByNameTx(tx, row.Name); err != nil {
					return err
				}
				if row.Pack != sticker.PackMarket {
					return ErrNameExists
				}
			} else if err := fileusageservice.SetStickerUsageTx(tx, row.Id, 0, row.FileName, fileUsage.UsageStatusActive); err != nil {
				return err
			}
			item.StickerID = row.Id
			return sticker.UpsertMarketItemTx(tx, &item)
		})
	}
	if err == nil {
		return
	}
	if ctx.Err() != nil {
		return
	}
	item.StickerID = 0
	item.Error = marketErrorReason(err)
	if writeErr := sticker.UpsertMarketItemTx(db.ConnectContext(ctx), &item); writeErr != nil {
		slog.Warn("sticker market: record item failure failed", "pack", pack.Id, "err", writeErr)
	}
}

// downloadMarketImage returns the stored object name for the image at rawURL.
func downloadMarketImage(ctx context.Context, config marketConfig, rawURL string) (string, error) {
	if !config.allowedImageURL(rawURL) {
		return "", errMarketHostBlocked
	}
	result, err := marketClient.FetchImage(ctx, rawURL)
	if err != nil {
		return "", err
	}
	if result.StatusCode != http.StatusOK {
		return "", errMarketStatus
	}
	if result.FinalURL != nil && !config.allowedImageURL(result.FinalURL.String()) {
		return "", errMarketHostBlocked
	}
	data := result.Body
	if len(data) == 0 || len(data) > filedata.MaxFileSize {
		return "", errMarketImageTooLarge
	}
	// The bytes decide the type, exactly as for member uploads (issue #1393).
	contentType, err := imagepolicy.DetectContentType(bytes.NewReader(data))
	if errors.Is(err, imagepolicy.ErrUnsupportedImageFormat) {
		// AVIF is common upstream: store a PNG/GIF copy every client can render,
		// then validate that output like any other image.
		if data, err = transcodeMarketAVIF(data); err != nil {
			return "", err
		}
		contentType, err = imagepolicy.DetectContentType(bytes.NewReader(data))
	}
	if errors.Is(err, imagepolicy.ErrUnsupportedImageFormat) {
		return "", errMarketUnsupportedImage
	}
	if err != nil {
		return "", errMarketInvalidImage
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", errMarketInvalidImage
	}
	if imageConfig.Width > maxMarketImageSide || imageConfig.Height > maxMarketImageSide {
		return "", errMarketImageTooLarge
	}
	return storeMarketAsset(ctx, data, contentType)
}

var marketAssetLocks = struct {
	sync.Mutex
	held map[string]*marketAssetLock
}{held: map[string]*marketAssetLock{}}

type marketAssetLock struct {
	sync.Mutex
	users int
}

func lockMarketAsset(name string) func() {
	marketAssetLocks.Lock()
	lock := marketAssetLocks.held[name]
	if lock == nil {
		lock = &marketAssetLock{}
		marketAssetLocks.held[name] = lock
	}
	lock.users++
	marketAssetLocks.Unlock()
	lock.Lock()
	return func() {
		lock.Unlock()
		marketAssetLocks.Lock()
		if lock.users--; lock.users == 0 {
			delete(marketAssetLocks.held, name)
		}
		marketAssetLocks.Unlock()
	}
}

// storeMarketAsset writes the bytes once under their content hash. The file
// name unique index makes a concurrent writer in another process lose the
// insert and reuse the published object instead of creating a second copy.
func storeMarketAsset(ctx context.Context, data []byte, contentType string) (string, error) {
	ext, ok := imagepolicy.ExtForContentType(contentType)
	if !ok {
		return "", errMarketUnsupportedImage
	}
	sum := sha256.Sum256(data)
	name := MarketAssetPrefix + hex.EncodeToString(sum[:]) + ext
	unlock := lockMarketAsset(name)
	defer unlock()
	if _, err := filedata.GetFileMetadataByNameContext(ctx, name); err == nil {
		return name, nil
	}
	if _, err := filedata.SaveFile(0, name, contentType, data); err != nil {
		if _, existing := filedata.GetFileMetadataByNameContext(ctx, name); existing == nil {
			return name, nil
		}
		return "", fmt.Errorf("%w: %w", errMarketStore, err)
	}
	return name, nil
}

// marketStickerName is a stable token per upstream item. It is derived, not
// copied: upstream names contain characters the token grammar rejects.
func marketStickerName(source, packID, itemID string) string {
	sum := sha256.Sum256([]byte(source + "\n" + packID + "\n" + itemID))
	return "m_" + hex.EncodeToString(sum[:])[:40]
}

func marketStickerLabel(packName string, index int) string {
	return cleanUpstreamText(fmt.Sprintf("%s %d", packName, index+1), sticker.MaxNameLen)
}

// finishMarketImport recomputes the pack summary from its item rows.
func finishMarketImport(ctx context.Context, packID uint64, reason string) error {
	items, err := sticker.MarketItems(packID)
	if err != nil {
		return err
	}
	ready, failed := 0, 0
	var cover uint64
	failedReasons := map[string]int{}
	for _, item := range items {
		if item.StickerID == 0 {
			failed++
			failedReasons[item.Error]++
			continue
		}
		if cover == 0 {
			cover = item.StickerID
		}
		ready++
	}
	status := sticker.MarketStatusReady
	switch {
	case reason != "" || ready == 0:
		status = sticker.MarketStatusFailed
	case failed > 0:
		status = sticker.MarketStatusPartial
	}
	if reason == "" && ready == 0 {
		reason = "empty"
		// One shared cause (for example an AVIF-only pack) is more useful than
		// "nothing imported"; mixed causes stay in the per-item list.
		if len(failedReasons) == 1 {
			for only := range failedReasons {
				if only != "" {
					reason = only
				}
			}
		}
	}
	updates := map[string]any{"status": status, "last_error": reason, "item_count": ready, "failed_count": failed, "cover_sticker_id": cover}
	if status != sticker.MarketStatusFailed {
		updates["imported_at"] = time.Now()
	}
	err = db.ConnectContext(ctx).Model(&sticker.MarketPack{}).Where("id = ?", packID).Updates(updates).Error
	if err == nil {
		InvalidateCache()
	}
	return err
}
