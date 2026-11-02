package stickerservice

import (
	"errors"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"gorm.io/gorm"
)

// AdminMarketPack is the administrator view of one catalog pack, including
// upstream metadata and local import state. Upstream URLs are never returned:
// IconURL is a local thumbnail or the endpoint that creates it.
type AdminMarketPack struct {
	ID uint64 `json:"id"`
	// Source is MarketSourceID for synchronized packs or MarketSourceUpload.
	Source        string     `json:"source"`
	UpstreamID    string     `json:"upstreamId"`
	Name          string     `json:"name"`
	Topic         string     `json:"topic"`
	UpstreamCount int        `json:"upstreamCount"`
	Archived      bool       `json:"archived"`
	Missing       bool       `json:"missing"`
	Status        string     `json:"status"`
	Published     bool       `json:"published"`
	SortOrder     int        `json:"sortOrder"`
	ItemCount     int        `json:"itemCount"`
	FailedCount   int        `json:"failedCount"`
	ImportTotal   int        `json:"importTotal"`
	LastError     string     `json:"lastError"`
	CoverURL      string     `json:"coverUrl"`
	IconURL       string     `json:"iconUrl"`
	IconText      string     `json:"iconText"`
	SyncedAt      time.Time  `json:"syncedAt"`
	ImportedAt    *time.Time `json:"importedAt"`
}

// AdminMarketCatalogPage is one page of the full catalog for administrators.
type AdminMarketCatalogPage struct {
	Enabled      bool              `json:"enabled"`
	Source       string            `json:"source"`
	LastSyncedAt *time.Time        `json:"lastSyncedAt"`
	Topics       []MarketTopicItem `json:"topics"`
	Packs        []AdminMarketPack `json:"packs"`
	Total        int64             `json:"total"`
	Page         int               `json:"page"`
	PageSize     int               `json:"pageSize"`
}

// AdminMarketItem is one upstream item and the local sticker holding it.
type AdminMarketItem struct {
	UpstreamID string `json:"upstreamId"`
	StickerID  uint64 `json:"stickerId"`
	Name       string `json:"name"`
	// Aliases are previous names that sent content still resolves (issue #1428).
	Aliases   []string `json:"aliases"`
	URL       string   `json:"url"`
	IsEnabled bool     `json:"isEnabled"`
	Error     string   `json:"error"`
}

// AdminMarketPackDetail lists every item an import recorded for a pack.
type AdminMarketPackDetail struct {
	Pack  AdminMarketPack   `json:"pack"`
	Items []AdminMarketItem `json:"items"`
}

func adminMarketPacks(packs []sticker.MarketPack) ([]AdminMarketPack, error) {
	covers, err := coverURLs(packs)
	if err != nil {
		return nil, err
	}
	items := make([]AdminMarketPack, 0, len(packs))
	for _, pack := range packs {
		items = append(items, AdminMarketPack{
			ID: pack.Id, Source: pack.Source, UpstreamID: pack.UpstreamID, Name: pack.Name, Topic: pack.Topic,
			UpstreamCount: pack.UpstreamCount, Archived: pack.Archived, Missing: pack.Missing,
			Status: pack.Status, Published: pack.Published, SortOrder: pack.SortOrder,
			ItemCount: pack.ItemCount, FailedCount: pack.FailedCount, ImportTotal: pack.ImportTotal, LastError: pack.LastError,
			CoverURL: covers[pack.CoverStickerID], IconURL: marketPackIconURL(pack), IconText: pack.IconText, SyncedAt: pack.SyncedAt, ImportedAt: pack.ImportedAt,
		})
	}
	return items, nil
}

// AdminMarketCatalog pages the synchronized and uploaded catalog with local
// state. status filters by "imported", "published" or one import state;
// source by MarketSourceID or MarketSourceUpload.
func AdminMarketCatalog(source, topic, keyword, status string, page, pageSize int) (AdminMarketCatalogPage, error) {
	page, pageSize = normalizePage(page, pageSize, 50)
	config, configErr := loadMarketConfig()
	result := AdminMarketCatalogPage{Enabled: configErr == nil && config.Enabled, Source: MarketSourceID, Topics: []MarketTopicItem{}, Packs: []AdminMarketPack{}, Page: page, PageSize: pageSize}
	if synced, err := sticker.LastMarketSync(MarketSourceID); err != nil {
		return result, err
	} else if !synced.IsZero() {
		result.LastSyncedAt = &synced
	}
	topics, err := sticker.MarketTopics("")
	if err != nil {
		return result, err
	}
	listed := map[string]bool{}
	for _, topic := range topics {
		if !listed[topic.UpstreamID] {
			listed[topic.UpstreamID] = true
			result.Topics = append(result.Topics, MarketTopicItem{ID: topic.UpstreamID, Label: topic.Label})
		}
	}
	if source != MarketSourceID && source != MarketSourceUpload {
		source = ""
	}
	packs, total, err := sticker.MarketPacks(sticker.MarketPackQuery{Source: source, Topic: topic, Keyword: keyword, Status: status, Page: page, PageSize: pageSize})
	if err != nil {
		return result, err
	}
	result.Total = total
	result.Packs, err = adminMarketPacks(packs)
	return result, err
}

// AdminMarketPackByID returns one pack's administrator view.
func AdminMarketPackByID(packID uint64) (AdminMarketPack, error) {
	pack, err := sticker.MarketPackByID(packID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AdminMarketPack{}, ErrMarketPackNotFound
	}
	if err != nil {
		return AdminMarketPack{}, err
	}
	items, err := adminMarketPacks([]sticker.MarketPack{pack})
	if err != nil {
		return AdminMarketPack{}, err
	}
	return items[0], nil
}

// AdminMarketPackItems shows every imported and failed item of a pack,
// including disabled stickers, so administrators can review before publishing.
func AdminMarketPackItems(packID uint64) (AdminMarketPackDetail, error) {
	var detail AdminMarketPackDetail
	pack, err := AdminMarketPackByID(packID)
	if err != nil {
		return detail, err
	}
	detail.Pack = pack
	rows, err := sticker.MarketItems(packID)
	if err != nil {
		return detail, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		if row.StickerID != 0 {
			ids = append(ids, row.StickerID)
		}
	}
	byID := map[uint64]sticker.Entity{}
	if len(ids) > 0 {
		entities := make([]sticker.Entity, 0, len(ids))
		if err := sticker.FindByIDs(ids, &entities); err != nil {
			return detail, err
		}
		for _, entity := range entities {
			byID[entity.Id] = entity
		}
	}
	aliases, err := sticker.AliasNamesBySticker(ids)
	if err != nil {
		return detail, err
	}
	detail.Items = make([]AdminMarketItem, 0, len(rows))
	for _, row := range rows {
		item := AdminMarketItem{UpstreamID: row.UpstreamID, StickerID: row.StickerID, Aliases: []string{}, Error: row.Error}
		if names := aliases[row.StickerID]; len(names) > 0 {
			item.Aliases = names
		}
		if entity, ok := byID[row.StickerID]; ok {
			item.Name, item.URL, item.IsEnabled = entity.Name, ResolveURLFor(entity), entity.IsEnabled
		}
		detail.Items = append(detail.Items, item)
	}
	return detail, nil
}
