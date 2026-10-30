package stickerservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/gorm"
)

// MaxPackSubscriptions bounds one account's market packs, independent of the
// 200-entry personal library.
const MaxPackSubscriptions = 50

// A queued or importing pack is re-queueable after this long without progress,
// so a worker lost mid-import cannot pin it forever.
const marketImportStaleAfter = 30 * time.Minute

var (
	ErrMarketPackUnavailable  = errors.New("sticker market pack unavailable")
	ErrMarketPackNotReady     = errors.New("sticker market pack has no imported stickers")
	ErrMarketSubscriptionFull = errors.New("sticker market subscription limit reached")
)

var marketSyncMu sync.Mutex

// MarketSyncResult summarizes one catalog refresh for the administrator.
type MarketSyncResult struct {
	Topics  int `json:"topics"`
	Packs   int `json:"packs"`
	Missing int `json:"missing"`
}

// SyncMarketCatalog refreshes topic and pack metadata from the upstream index.
// It reads every page before writing anything, so a partial or failed fetch
// keeps the last complete snapshot. It never downloads images or publishes.
func SyncMarketCatalog(ctx context.Context) (MarketSyncResult, error) {
	var result MarketSyncResult
	if !marketSyncMu.TryLock() {
		return result, ErrMarketBusy
	}
	defer marketSyncMu.Unlock()
	config, err := loadMarketConfig()
	if err != nil {
		return result, err
	}
	if !config.Enabled {
		return result, ErrMarketDisabled
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	upstreamTopics, err := fetchMarketTopics(ctx, config)
	if err != nil {
		return result, errors.Join(ErrMarketUnavailable, err)
	}
	topics := make([]sticker.MarketTopic, 0, len(upstreamTopics))
	pages := 1
	for index, topic := range upstreamTopics {
		id := cleanUpstreamText(topic.ID, 64)
		if id == "" {
			continue
		}
		if id == "all" {
			pages = topic.TotalPages
			continue
		}
		label := cleanUpstreamText(topic.Label, 64)
		if label == "" {
			label = id
		}
		topics = append(topics, sticker.MarketTopic{Source: MarketSourceID, UpstreamID: id, Label: label, SortOrder: index})
	}
	pages = min(max(pages, 1), maxMarketCatalogPages)
	now := time.Now()
	packs := make([]sticker.MarketPack, 0, pages*48)
	seen := map[string]bool{}
	for page := 1; page <= pages; page++ {
		groups, totalPages, err := fetchMarketPage(ctx, config, page)
		if err != nil {
			return result, errors.Join(ErrMarketUnavailable, err)
		}
		if page == 1 && totalPages > pages {
			pages = min(totalPages, maxMarketCatalogPages)
		}
		for _, group := range groups {
			id := group.ID
			if !upstreamIDRe.MatchString(id) || seen[id] {
				continue
			}
			seen[id] = true
			name := cleanUpstreamText(group.Name, 128)
			if name == "" {
				name = id
			}
			iconURL, iconText := config.marketIcon(group.Icon)
			packs = append(packs, sticker.MarketPack{
				Source: MarketSourceID, UpstreamID: id, Name: name, Topic: cleanUpstreamText(group.Topic, 64),
				IconURL: iconURL, IconText: iconText,
				UpstreamCount: max(group.EmojiCount, 0), UpstreamOrder: len(packs), Archived: group.IsArchived,
				Status: sticker.MarketStatusNew, SyncedAt: now,
			})
		}
	}
	if len(packs) == 0 {
		return result, errors.Join(ErrMarketUnavailable, errMarketInvalidData)
	}
	var missing int64
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.UpsertMarketTopicsTx(tx, topics); err != nil {
			return err
		}
		if err := sticker.UpsertMarketPacksTx(tx, packs); err != nil {
			return err
		}
		missing, err = sticker.MarkMissingMarketPacksTx(tx, MarketSourceID, now)
		return err
	})
	if err != nil {
		return result, err
	}
	return MarketSyncResult{Topics: len(topics), Packs: len(packs), Missing: int(missing)}, nil
}

// MarketTopicItem is one browsable category.
type MarketTopicItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// MarketPackItem is the public view of a published pack. It never exposes
// upstream URLs; the cover is a locally stored sticker image.
type MarketPackItem struct {
	ID           uint64 `json:"id"`
	Name         string `json:"name"`
	Topic        string `json:"topic"`
	StickerCount int    `json:"stickerCount"`
	CoverURL     string `json:"coverUrl"`
}

// MarketCatalogPage is one page of published packs plus the categories that
// currently contain a published pack.
type MarketCatalogPage struct {
	Topics   []MarketTopicItem `json:"topics"`
	Packs    []MarketPackItem  `json:"packs"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

// MarketPackDetail lists a published pack's usable stickers in pack order.
type MarketPackDetail struct {
	Pack     MarketPackItem `json:"pack"`
	Stickers []StickerItem  `json:"stickers"`
}

func normalizePage(page, pageSize, fallback int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = fallback
	}
	return page, pageSize
}

// coverURLs resolves pack covers; a disabled cover sticker shows no image.
func coverURLs(packs []sticker.MarketPack) (map[uint64]string, error) {
	ids := make([]uint64, 0, len(packs))
	for _, pack := range packs {
		if pack.CoverStickerID != 0 {
			ids = append(ids, pack.CoverStickerID)
		}
	}
	urls := make(map[uint64]string, len(ids))
	if len(ids) == 0 {
		return urls, nil
	}
	entities := make([]sticker.Entity, 0, len(ids))
	if err := db.Connect().Where("id IN ? AND is_enabled = ?", ids, true).Find(&entities).Error; err != nil {
		return urls, err
	}
	for _, entity := range entities {
		urls[entity.Id] = ResolveURLFor(entity)
	}
	return urls, nil
}

func marketPackItems(packs []sticker.MarketPack) ([]MarketPackItem, error) {
	covers, err := coverURLs(packs)
	if err != nil {
		return nil, err
	}
	items := make([]MarketPackItem, 0, len(packs))
	for _, pack := range packs {
		items = append(items, MarketPackItem{ID: pack.Id, Name: pack.Name, Topic: pack.Topic, StickerCount: pack.ItemCount, CoverURL: covers[pack.CoverStickerID]})
	}
	return items, nil
}

// MarketCatalog pages the packs administrators published.
func MarketCatalog(topic, keyword string, page, pageSize int) (MarketCatalogPage, error) {
	page, pageSize = normalizePage(page, pageSize, 24)
	result := MarketCatalogPage{Topics: []MarketTopicItem{}, Packs: []MarketPackItem{}, Page: page, PageSize: pageSize}
	packs, total, err := sticker.MarketPacks(sticker.MarketPackQuery{Topic: topic, Keyword: keyword, PublishedOnly: true, Page: page, PageSize: pageSize})
	if err != nil {
		return result, err
	}
	if result.Packs, err = marketPackItems(packs); err != nil {
		return result, err
	}
	result.Total = total
	var used []string
	if err := db.Connect().Model(&sticker.MarketPack{}).Where("published = ? AND item_count > ?", true, 0).
		Distinct().Pluck("topic", &used).Error; err != nil {
		return result, err
	}
	inUse := make(map[string]bool, len(used))
	for _, id := range used {
		inUse[id] = true
	}
	topics, err := sticker.MarketTopics("")
	if err != nil {
		return result, err
	}
	for _, topic := range topics {
		if inUse[topic.UpstreamID] {
			// One id listed by two sources shows once.
			delete(inUse, topic.UpstreamID)
			result.Topics = append(result.Topics, MarketTopicItem{ID: topic.UpstreamID, Label: topic.Label})
		}
	}
	return result, nil
}

// marketStickers lists a pack's enabled, approved stickers in pack order.
func marketStickers(packID uint64) ([]StickerItem, error) {
	items, err := sticker.MarketItems(packID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(items))
	for _, item := range items {
		if item.StickerID != 0 {
			ids = append(ids, item.StickerID)
		}
	}
	stickers := make([]StickerItem, 0, len(ids))
	if len(ids) == 0 {
		return stickers, nil
	}
	entities := make([]sticker.Entity, 0, len(ids))
	if err := db.Connect().Where("id IN ? AND is_enabled = ? AND review_status = ?", ids, true, sticker.ReviewStatusApproved).Find(&entities).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]sticker.Entity, len(entities))
	for _, entity := range entities {
		byID[entity.Id] = entity
	}
	for _, item := range items {
		if entity, ok := byID[item.StickerID]; ok && entity.FileName != "" {
			stickers = append(stickers, itemFor(entity, ""))
			delete(byID, item.StickerID)
		}
	}
	return stickers, nil
}

// MarketPackStickers returns a published pack and its usable stickers.
func MarketPackStickers(packID uint64) (MarketPackDetail, error) {
	var detail MarketPackDetail
	pack, err := sticker.MarketPackByID(packID)
	if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && (!pack.Published || pack.ItemCount == 0) {
		return detail, ErrMarketPackUnavailable
	}
	if err != nil {
		return detail, err
	}
	items, err := marketPackItems([]sticker.MarketPack{pack})
	if err != nil {
		return detail, err
	}
	detail.Pack = items[0]
	detail.Stickers, err = marketStickers(pack.Id)
	return detail, err
}

// MyMarketPacks lists the account's subscribed packs that are still published,
// in subscription order. Unpublished packs stay subscribed and reappear if an
// administrator publishes them again.
func MyMarketPacks(ctx context.Context, userID uint64) ([]MarketPackItem, error) {
	subscriptions, err := sticker.SubscriptionsTx(db.ConnectContext(ctx), userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint64, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		ids = append(ids, subscription.PackID)
	}
	packs, err := sticker.MarketPacksByIDs(ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[uint64]sticker.MarketPack, len(packs))
	for _, pack := range packs {
		if pack.Published && pack.ItemCount > 0 {
			byID[pack.Id] = pack
		}
	}
	ordered := make([]sticker.MarketPack, 0, len(byID))
	for _, subscription := range subscriptions {
		if pack, ok := byID[subscription.PackID]; ok {
			ordered = append(ordered, pack)
		}
	}
	return marketPackItems(ordered)
}

// SubscribeMarketPack adds a published pack to the account. Repeating it is
// idempotent; the per-account library lock serializes the subscription limit
// and fences writes that race an account closure.
func SubscribeMarketPack(ctx context.Context, userID, packID uint64) (MarketPackItem, error) {
	var pack sticker.MarketPack
	if userID == 0 || packID == 0 {
		return MarketPackItem{}, ErrInvalidLibraryInput
	}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.LockLibraryTx(tx, userID); err != nil {
			return err
		}
		var err error
		pack, err = sticker.MarketPackByIDTx(tx, packID)
		if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && (!pack.Published || pack.ItemCount == 0) {
			return ErrMarketPackUnavailable
		}
		if err != nil {
			return err
		}
		subscriptions, err := sticker.SubscriptionsTx(tx, userID)
		if err != nil {
			return err
		}
		order := 0
		for _, subscription := range subscriptions {
			if subscription.PackID == packID {
				return nil
			}
			order = max(order, subscription.SortOrder+1)
		}
		if len(subscriptions) >= MaxPackSubscriptions {
			return ErrMarketSubscriptionFull
		}
		return sticker.SaveSubscriptionTx(tx, &sticker.PackSubscription{UserID: userID, PackID: packID, SortOrder: order})
	})
	if err != nil {
		return MarketPackItem{}, err
	}
	items, err := marketPackItems([]sticker.MarketPack{pack})
	if err != nil {
		return MarketPackItem{}, err
	}
	return items[0], nil
}

// UnsubscribeMarketPack removes only the account's subscription; stickers it
// already sent keep resolving.
func UnsubscribeMarketPack(ctx context.Context, userID, packID uint64) error {
	if userID == 0 || packID == 0 {
		return ErrInvalidLibraryInput
	}
	return db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sticker.LockLibraryTx(tx, userID); err != nil {
			return err
		}
		return sticker.RemoveSubscriptionTx(tx, userID, packID)
	})
}

// SaveMarketPackInput publishes, hides or reorders one pack. Name, Topic and
// TopicLabel rename or recategorize an uploaded pack and are ignored for
// synchronized packs, whose metadata the next sync owns.
type SaveMarketPackInput struct {
	ID         uint64  `json:"id"`
	Published  bool    `json:"published"`
	SortOrder  int     `json:"sortOrder"`
	Name       *string `json:"name"`
	Topic      *string `json:"topic"`
	TopicLabel string  `json:"topicLabel"`
}

// SaveMarketPack changes administrator-owned pack state. Hiding a pack removes
// it from the market and pickers; sent stickers keep resolving.
func SaveMarketPack(ctx context.Context, input SaveMarketPackInput) (sticker.MarketPack, error) {
	var pack sticker.MarketPack
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		pack, err = sticker.MarketPackByIDTx(tx, input.ID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMarketPackNotFound
		}
		if err != nil {
			return err
		}
		if input.Published && pack.ItemCount == 0 {
			return ErrMarketPackNotReady
		}
		if pack.Source == MarketSourceUpload {
			if input.Name != nil {
				if pack.Name, err = cleanMarketPackName(*input.Name); err != nil {
					return err
				}
			}
			if input.Topic != nil || input.TopicLabel != "" {
				topic := ""
				if input.Topic != nil {
					topic = *input.Topic
				}
				if pack.Topic, err = resolveUploadTopicTx(tx, topic, input.TopicLabel); err != nil {
					return err
				}
			}
		}
		pack.Published = input.Published
		pack.SortOrder = input.SortOrder
		return sticker.SaveMarketPackTx(tx, &pack)
	})
	return pack, err
}

// SaveMarketStickerInput withdraws, restores or renames one market sticker.
// An empty Name keeps the current name.
type SaveMarketStickerInput struct {
	StickerID uint64
	Name      string
	IsEnabled bool
}

// SaveMarketSticker applies an administrator's change to one market sticker.
// A disabled sticker cannot be inserted and its sent tokens stop resolving,
// matching a disabled official sticker. A new name replaces the generated
// token, and the previous name keeps resolving as for official stickers.
// It returns the sticker before and after the change.
func SaveMarketSticker(ctx context.Context, input SaveMarketStickerInput) (before, row sticker.Entity, err error) {
	name := strings.TrimSpace(input.Name)
	if name != "" && !ValidateName(name) {
		return before, row, ErrNameInvalid
	}
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		before, err = sticker.GetByIDTx(tx, input.StickerID)
		if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && before.Pack != sticker.PackMarket {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		row = before
		row.IsEnabled = input.IsEnabled
		if name == "" || name == row.Name {
			return tx.Model(&sticker.Entity{}).Where("id = ?", row.Id).Update("is_enabled", row.IsEnabled).Error
		}
		if err := checkStickerNameTx(tx, row.Id, name); err != nil {
			return err
		}
		previousName := row.Name
		// The chosen name is also the label pickers show, replacing "<pack> <n>".
		row.Name, row.DisplayName = name, ""
		return saveRenamedStickerTx(tx, &row, previousName)
	})
	if err == nil {
		InvalidateCache()
	}
	return before, row, err
}

// marketImportBusy reports whether a pack is already waiting for or running an import.
func marketImportBusy(pack sticker.MarketPack) bool {
	if pack.Status != sticker.MarketStatusQueued && pack.Status != sticker.MarketStatusImporting {
		return false
	}
	return time.Since(pack.UpdatedAt) < marketImportStaleAfter
}
