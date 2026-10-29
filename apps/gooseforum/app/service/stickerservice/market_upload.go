package stickerservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/gorm"
)

// MarketSourceUpload marks packs administrators assemble from their own files.
// They share publication, subscriptions and per-sticker controls with synced
// packs, but never contact the upstream catalog.
const MarketSourceUpload = "upload"

// uploadTopicOrder sorts administrator topics after the upstream ones.
const uploadTopicOrder = 10000

var errMarketPackFull = errors.New("market pack holds the maximum number of stickers")

// CreateMarketPackInput names a new uploaded pack. Topic picks an existing
// topic id; TopicLabel instead finds or creates an uploaded topic by name.
type CreateMarketPackInput struct {
	Name       string `json:"name"`
	Topic      string `json:"topic"`
	TopicLabel string `json:"topicLabel"`
}

func cleanMarketPackName(raw string) (string, error) {
	name := cleanUpstreamText(raw, 128)
	if name == "" {
		return "", ErrPackNameRequired
	}
	return name, nil
}

// resolveUploadTopicTx returns the topic id for an uploaded pack.
func resolveUploadTopicTx(tx *gorm.DB, topic, label string) (string, error) {
	if label = cleanUpstreamText(label, 64); label != "" {
		existing, err := sticker.MarketTopicByLabelTx(tx, MarketSourceUpload, label)
		if err == nil {
			return existing.UpstreamID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		sum := sha256.Sum256([]byte(label))
		row := sticker.MarketTopic{Source: MarketSourceUpload, UpstreamID: "u-" + hex.EncodeToString(sum[:])[:12], Label: label, SortOrder: uploadTopicOrder}
		if err := sticker.CreateMarketTopicTx(tx, &row); err != nil {
			return "", err
		}
		return row.UpstreamID, nil
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", nil
	}
	exists, err := sticker.MarketTopicExists(tx, topic)
	if err != nil {
		return "", err
	}
	if !exists {
		return "", ErrPackInvalid
	}
	return topic, nil
}

// CreateMarketPack starts an empty, hidden uploaded pack.
func CreateMarketPack(ctx context.Context, input CreateMarketPackInput) (sticker.MarketPack, error) {
	var pack sticker.MarketPack
	name, err := cleanMarketPackName(input.Name)
	if err != nil {
		return pack, err
	}
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		return pack, err
	}
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		topic, err := resolveUploadTopicTx(tx, input.Topic, input.TopicLabel)
		if err != nil {
			return err
		}
		pack = sticker.MarketPack{
			Source: MarketSourceUpload, UpstreamID: "u" + hex.EncodeToString(random[:]), Name: name, Topic: topic,
			Status: sticker.MarketStatusNew, SyncedAt: time.Now(),
		}
		return sticker.CreateMarketPackTx(tx, &pack)
	})
	return pack, err
}

// UploadMarketPackFile adds a ZIP archive or one image to an uploaded pack.
// Images pass the same checks and AVIF conversion as imported ones; an image
// the pack already holds is skipped. Items keep upload order after the
// existing ones, and the pack summary is recomputed afterwards.
func UploadMarketPackFile(ctx context.Context, packID uint64, fileName string, data []byte) (ImportResult, error) {
	pack, err := sticker.MarketPackByID(packID)
	if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && pack.Source != MarketSourceUpload {
		return ImportResult{Failed: []ImportIssue{}}, ErrMarketPackNotFound
	}
	if err != nil {
		return ImportResult{Failed: []ImportIssue{}}, err
	}
	conn := db.ConnectContext(ctx)
	order, count, err := sticker.NextMarketItemOrder(conn, pack.Id)
	if err != nil {
		return ImportResult{Failed: []ImportIssue{}}, err
	}
	result, err := importUpload(ctx, fileName, data, func(ctx context.Context, base string, data []byte) (bool, string) {
		if count >= MaxMarketPackItems {
			return false, marketUploadReason(errMarketPackFull)
		}
		skipped, err := addUploadedMarketItem(ctx, pack, base, data, order)
		if err != nil {
			return false, marketUploadReason(err)
		}
		if !skipped {
			order++
			count++
		}
		return skipped, ""
	})
	if err != nil {
		return result, err
	}
	if result.Imported > 0 {
		err = finishMarketImport(ctx, pack.Id, "")
	}
	return result, err
}

func marketUploadReason(err error) string {
	switch {
	case errors.Is(err, errMarketPackFull):
		return "tooManyFiles"
	case errors.Is(err, errMarketImageTooLarge):
		return "tooLarge"
	case errors.Is(err, errMarketUnsupportedImage), errors.Is(err, errMarketInvalidImage):
		return "invalidImage"
	default:
		return "saveFailed"
	}
}

// addUploadedMarketItem stores one image and links it to the pack. The item
// id is the image's content hash, so uploading the same image again is a no-op.
func addUploadedMarketItem(ctx context.Context, pack sticker.MarketPack, base string, data []byte, order int) (bool, error) {
	fileName, err := storeMarketImage(ctx, data)
	if err != nil {
		return false, err
	}
	itemID := strings.TrimSuffix(path.Base(fileName), path.Ext(fileName))
	if len(itemID) > 40 {
		itemID = itemID[:40]
	}
	skipped := false
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if existing, err := sticker.MarketItemTx(tx, pack.Id, itemID); err == nil && existing.StickerID != 0 {
			skipped = true
			return nil
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		label := cleanUpstreamText(StemName(base), sticker.MaxNameLen)
		if label == "" {
			label = marketStickerLabel(pack.Name, order)
		}
		item := sticker.MarketItem{PackID: pack.Id, UpstreamID: itemID, SortOrder: order}
		return linkMarketStickerTx(tx, &item, sticker.Entity{
			Name: marketStickerName(pack.Source, pack.UpstreamID, itemID), DisplayName: label,
			FileName: fileName, SortOrder: order, IsEnabled: true,
		})
	})
	return skipped, err
}
