package stickerservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"strconv"
	"sync"
	"time"

	"github.com/gen2brain/avif"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/imagepolicy"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/storageservice"
)

// Administrators review packs before importing them, so the admin catalog
// needs a cover for packs the server does not hold yet. The browser never
// loads the upstream host: the server fetches the upstream icon (or the pack's
// first sticker when upstream lists none) through the market client, stores a
// small still thumbnail once and serves that from local storage.
const (
	marketCoverSide = 160
	// marketCoverWorkers bounds upstream fetches while a catalog page loads
	// many covers at once.
	marketCoverWorkers    = 4
	marketCoverTimeout    = 30 * time.Second
	marketCoverRetryAfter = 10 * time.Minute
)

// MarketCoverPreviewPath is the admin endpoint that materializes a cover on
// first request; the catalog links to it until the thumbnail exists.
const MarketCoverPreviewPath = "/api/admin/sticker-market-cover"

var errMarketCoverUnavailable = errors.New("sticker market cover unavailable")

var (
	marketCoverFlights singleflight.Group
	marketCoverSlots   = make(chan struct{}, marketCoverWorkers)
	// marketCoverMisses remembers recent failures so a broken upstream icon is
	// not refetched on every catalog load.
	marketCoverMisses sync.Map
)

// marketPackIconURL is the admin cover for a pack without an imported cover.
// Packs whose upstream cover is text still get an image of their first sticker;
// the client falls back to the text when no image can be made.
func marketPackIconURL(pack sticker.MarketPack) string {
	if pack.IconFile != "" {
		return storageservice.PublicAccessPath(pack.IconFile)
	}
	if pack.Source != MarketSourceID {
		// Uploaded packs only ever show the cover of a stored sticker.
		return ""
	}
	return MarketCoverPreviewPath + "?id=" + strconv.FormatUint(pack.Id, 10)
}

// MarketCoverPreview returns the stored thumbnail of a pack's upstream cover,
// fetching and storing it on first use.
func MarketCoverPreview(ctx context.Context, packID uint64) (string, error) {
	pack, err := sticker.MarketPackByID(packID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrMarketPackNotFound
	}
	if err != nil {
		return "", err
	}
	if pack.IconFile != "" {
		return pack.IconFile, nil
	}
	if pack.Source != MarketSourceID {
		return "", errMarketCoverUnavailable
	}
	if failed, ok := marketCoverMisses.Load(packID); ok && time.Since(failed.(time.Time)) < marketCoverRetryAfter {
		return "", errMarketCoverUnavailable
	}
	name, err, _ := marketCoverFlights.Do(strconv.FormatUint(packID, 10), func() (any, error) {
		// Waiting requests share this fetch, so it must outlive the first one.
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), marketCoverTimeout)
		defer cancel()
		select {
		case marketCoverSlots <- struct{}{}:
			defer func() { <-marketCoverSlots }()
		case <-fetchCtx.Done():
			return "", fetchCtx.Err()
		}
		name, err := storeMarketCover(fetchCtx, pack)
		if err != nil {
			marketCoverMisses.Store(packID, time.Now())
			return "", err
		}
		marketCoverMisses.Delete(packID)
		return name, db.ConnectContext(fetchCtx).Model(&sticker.MarketPack{}).
			Where("id = ?", packID).UpdateColumn("icon_file", name).Error
	})
	if err != nil {
		return "", err
	}
	return name.(string), nil
}

func storeMarketCover(ctx context.Context, pack sticker.MarketPack) (string, error) {
	config, err := loadMarketConfig()
	if err != nil {
		return "", err
	}
	if !config.Enabled {
		return "", ErrMarketDisabled
	}
	source := pack.IconURL
	if source == "" {
		emojis, err := fetchMarketGroup(ctx, config, pack.UpstreamID)
		if err != nil {
			return "", err
		}
		if len(emojis) == 0 {
			return "", errMarketCoverUnavailable
		}
		source = emojis[0].URL
	}
	data, err := fetchMarketImageBytes(ctx, config, source)
	if err != nil {
		return "", err
	}
	thumbnail, err := marketCoverThumbnail(data)
	if err != nil {
		return "", err
	}
	return storeMarketAsset(ctx, thumbnail, "image/png")
}

// marketCoverThumbnail renders the first frame of any accepted image, AVIF
// included, as a PNG whose long side is at most marketCoverSide.
func marketCoverThumbnail(data []byte) ([]byte, error) {
	var config image.Config
	var err error
	_, isAVIF := avifBrand(data)
	if isAVIF {
		config, err = avif.DecodeConfig(bytes.NewReader(data))
	} else {
		if _, err = imagepolicy.DetectContentType(bytes.NewReader(data)); err != nil {
			return nil, errMarketUnsupportedImage
		}
		config, _, err = image.DecodeConfig(bytes.NewReader(data))
	}
	if err != nil || config.Width < 1 || config.Height < 1 {
		return nil, errMarketInvalidImage
	}
	if config.Width > maxMarketImageSide || config.Height > maxMarketImageSide {
		return nil, errMarketImageTooLarge
	}
	var frame image.Image
	if isAVIF {
		marketTranscodeMu.Lock()
		frame, err = decodeMarketAVIFStill(data)
		marketTranscodeMu.Unlock()
	} else {
		frame, _, err = image.Decode(bytes.NewReader(data))
	}
	if err != nil {
		return nil, errMarketInvalidImage
	}
	var out bytes.Buffer
	if err := png.Encode(&out, scaleToFit(toNRGBA(frame), marketCoverSide)); err != nil {
		return nil, errMarketInvalidImage
	}
	return out.Bytes(), nil
}

// decodeMarketAVIFStill decodes only the first frame of a still or sequence.
func decodeMarketAVIFStill(data []byte) (frame image.Image, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			frame, err = nil, fmt.Errorf("avif decode panic: %v", recovered)
		}
	}()
	return avif.Decode(bytes.NewReader(data))
}
