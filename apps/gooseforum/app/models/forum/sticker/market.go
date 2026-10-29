package sticker

import (
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PackMarket marks stickers materialized from an external market pack. They
// are shared assets like official stickers but appear only through published
// packs, so the official directory and admin library never list them.
const PackMarket = "market"

// Market pack import states. A pack becomes selectable only after an
// administrator publishes it; the state reports what the server holds locally.
const (
	MarketStatusNew       = "new"
	MarketStatusQueued    = "queued"
	MarketStatusImporting = "importing"
	MarketStatusReady     = "ready"
	MarketStatusPartial   = "partial"
	MarketStatusFailed    = "failed"
)

// MarketTopic is an upstream category snapshot. It only groups packs for
// browsing and never grants visibility on its own.
type MarketTopic struct {
	Id         uint64    `gorm:"primaryKey;column:id;autoIncrement;not null;"`
	Source     string    `gorm:"column:source;type:varchar(32);not null;uniqueIndex:idx_sticker_market_topic,priority:1;"`
	UpstreamID string    `gorm:"column:upstream_id;type:varchar(64);not null;uniqueIndex:idx_sticker_market_topic,priority:2;"`
	Label      string    `gorm:"column:label;type:varchar(64);not null;default:'';"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0;"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime;"`
}

func (MarketTopic) TableName() string { return "sticker_market_topics" }

// MarketPack is one upstream sticker group plus local import and publication
// state. Upstream identifiers stay separate from the local id so a changed
// upstream catalog cannot rename or merge local packs.
type MarketPack struct {
	Id            uint64 `gorm:"primaryKey;column:id;autoIncrement;not null;" json:"id"`
	Source        string `gorm:"column:source;type:varchar(32);not null;uniqueIndex:idx_sticker_market_pack,priority:1;" json:"source"`
	UpstreamID    string `gorm:"column:upstream_id;type:varchar(128);not null;uniqueIndex:idx_sticker_market_pack,priority:2;" json:"upstreamId"`
	Name          string `gorm:"column:name;type:varchar(128);not null;default:'';" json:"name"`
	Topic         string `gorm:"column:topic;type:varchar(64);not null;default:'';index;" json:"topic"`
	UpstreamCount int    `gorm:"column:upstream_count;not null;default:0;" json:"upstreamCount"`
	UpstreamOrder int    `gorm:"column:upstream_order;not null;default:0;" json:"upstreamOrder"`
	Archived      bool   `gorm:"column:archived;not null;default:false;" json:"archived"`
	Missing       bool   `gorm:"column:missing;not null;default:false;" json:"missing"`
	Status        string `gorm:"column:status;type:varchar(16);not null;default:'new';index;" json:"status"`
	Published     bool   `gorm:"column:published;not null;default:false;index;" json:"published"`
	SortOrder     int    `gorm:"column:sort_order;not null;default:0;" json:"sortOrder"`
	ItemCount     int    `gorm:"column:item_count;not null;default:0;" json:"itemCount"`
	FailedCount   int    `gorm:"column:failed_count;not null;default:0;" json:"failedCount"`
	// ImportTotal is the item count of the latest import run; ItemCount plus
	// FailedCount reach it as the run progresses.
	ImportTotal    int    `gorm:"column:import_total;not null;default:0;" json:"importTotal"`
	LastError      string `gorm:"column:last_error;type:varchar(64);not null;default:'';" json:"lastError"`
	CoverStickerID uint64 `gorm:"column:cover_sticker_id;not null;default:0;" json:"coverStickerId"`
	// IconURL/IconText are the upstream catalog cover, shown to administrators
	// before import. Members only ever see the locally stored cover.
	IconURL    string     `gorm:"column:icon_url;type:varchar(512);not null;default:'';" json:"iconUrl"`
	IconText   string     `gorm:"column:icon_text;type:varchar(64);not null;default:'';" json:"iconText"`
	SyncedAt   time.Time  `gorm:"column:synced_at;" json:"syncedAt"`
	ImportedAt *time.Time `gorm:"column:imported_at;" json:"importedAt"`
	CreatedAt  time.Time  `gorm:"column:created_at;autoCreateTime;<-:create;" json:"createdAt"`
	UpdatedAt  time.Time  `gorm:"column:updated_at;autoUpdateTime;" json:"updatedAt"`
}

func (MarketPack) TableName() string { return "sticker_market_packs" }

// MarketItem maps one upstream sticker to the local sticker asset that holds
// its downloaded bytes. A failed item keeps StickerID 0 and a stable reason.
type MarketItem struct {
	PackID     uint64    `gorm:"primaryKey;autoIncrement:false;column:pack_id;"`
	UpstreamID string    `gorm:"primaryKey;column:upstream_id;type:varchar(128);"`
	StickerID  uint64    `gorm:"column:sticker_id;not null;default:0;index;"`
	SortOrder  int       `gorm:"column:sort_order;not null;default:0;"`
	Error      string    `gorm:"column:error;type:varchar(32);not null;default:'';"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime;"`
}

func (MarketItem) TableName() string { return "sticker_market_items" }

// PackSubscription is private account state, written under the same per-account
// library lock as personal stickers. Removing it never touches shared assets.
type PackSubscription struct {
	UserID    uint64    `gorm:"primaryKey;autoIncrement:false;column:user_id;"`
	PackID    uint64    `gorm:"primaryKey;autoIncrement:false;column:pack_id;index;"`
	SortOrder int       `gorm:"column:sort_order;not null;default:0;"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime;<-:create;"`
}

func (PackSubscription) TableName() string { return "user_sticker_packs" }

func marketDB() *gorm.DB { return db.Connect() }

// UpsertMarketTopicsTx replaces label/order for every listed topic.
func UpsertMarketTopicsTx(tx *gorm.DB, topics []MarketTopic) error {
	if len(topics) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "upstream_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"label", "sort_order", "updated_at"}),
	}).Create(&topics).Error
}

// UpsertMarketPacksTx refreshes upstream metadata only. Local import and
// publication state survive every catalog refresh.
func UpsertMarketPacksTx(tx *gorm.DB, packs []MarketPack) error {
	if len(packs) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "upstream_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "topic", "icon_url", "icon_text", "upstream_count", "upstream_order", "archived", "missing", "synced_at", "updated_at"}),
	}).CreateInBatches(&packs, 200).Error
}

// MarkMissingMarketPacksTx flags packs the latest complete catalog no longer lists.
func MarkMissingMarketPacksTx(tx *gorm.DB, source string, syncedBefore time.Time) (int64, error) {
	result := tx.Model(&MarketPack{}).Where("source = ? AND synced_at < ? AND missing = ?", source, syncedBefore, false).Update("missing", true)
	return result.RowsAffected, result.Error
}

func MarketTopics(source string) ([]MarketTopic, error) {
	rows := make([]MarketTopic, 0)
	err := marketDB().Where("source = ?", source).Order("sort_order ASC").Order("id ASC").Find(&rows).Error
	return rows, err
}

// MarketPackQuery filters the administrator and public catalogs.
type MarketPackQuery struct {
	Source        string
	Topic         string
	Keyword       string
	Status        string // admin filter: "", "imported", "published", or a MarketStatus*
	PublishedOnly bool
	Page          int
	PageSize      int
}

func (q MarketPackQuery) apply(conn *gorm.DB) *gorm.DB {
	conn = conn.Model(&MarketPack{}).Where("source = ?", q.Source)
	if q.PublishedOnly {
		conn = conn.Where("published = ? AND item_count > ?", true, 0)
	}
	if q.Topic != "" {
		conn = conn.Where("topic = ?", q.Topic)
	}
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(keyword))
		conn = conn.Where(`LOWER(name) LIKE ? ESCAPE '\'`, "%"+escaped+"%")
	}
	switch q.Status {
	case "":
	case "imported":
		conn = conn.Where("item_count > ?", 0)
	case "published":
		conn = conn.Where("published = ?", true)
	default:
		conn = conn.Where("status = ?", q.Status)
	}
	return conn
}

// MarketPacks pages the catalog. Published packs follow administrator order;
// the remaining catalog keeps the upstream order.
func MarketPacks(q MarketPackQuery) ([]MarketPack, int64, error) {
	rows := make([]MarketPack, 0)
	var total int64
	if err := q.apply(marketDB()).Count(&total).Error; err != nil {
		return rows, 0, err
	}
	conn := q.apply(marketDB())
	if q.PublishedOnly {
		conn = conn.Order("sort_order ASC")
	}
	err := conn.Order("upstream_order ASC").Order("id ASC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&rows).Error
	return rows, total, err
}

func MarketPackByID(id uint64) (MarketPack, error) {
	var row MarketPack
	err := marketDB().First(&row, id).Error
	return row, err
}

func MarketPackByIDTx(tx *gorm.DB, id uint64) (MarketPack, error) {
	var row MarketPack
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error
	return row, err
}

func MarketPacksByIDs(ids []uint64) ([]MarketPack, error) {
	rows := make([]MarketPack, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	err := marketDB().Where("id IN ?", ids).Find(&rows).Error
	return rows, err
}

func SaveMarketPackTx(tx *gorm.DB, pack *MarketPack) error { return tx.Save(pack).Error }

// LastMarketSync is the most recent successful catalog refresh, or zero.
func LastMarketSync(source string) (time.Time, error) {
	var pack MarketPack
	err := marketDB().Where("source = ?", source).Order("synced_at DESC").Limit(1).Find(&pack).Error
	return pack.SyncedAt, err
}

// MarketItems lists a pack's upstream items in pack order.
func MarketItems(packID uint64) ([]MarketItem, error) {
	rows := make([]MarketItem, 0)
	err := marketDB().Where("pack_id = ?", packID).Order("sort_order ASC").Order("upstream_id ASC").Find(&rows).Error
	return rows, err
}

func UpsertMarketItemTx(tx *gorm.DB, item *MarketItem) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "pack_id"}, {Name: "upstream_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"sticker_id", "sort_order", "error", "updated_at"}),
	}).Create(item).Error
}

// InsertMarketStickerTx creates a shared, approved market asset. The legacy
// is_official=true column default is cleared in the same transaction, before
// any reader can observe the row in the official directory.
func InsertMarketStickerTx(tx *gorm.DB, entity *Entity) (bool, error) {
	entity.Pack = PackMarket
	inserted, err := InsertIfNameAvailable(tx, entity)
	if err != nil || !inserted {
		return inserted, err
	}
	entity.IsOfficial = false
	return true, tx.Model(entity).Update("is_official", false).Error
}

func SubscriptionsTx(tx *gorm.DB, userID uint64) ([]PackSubscription, error) {
	rows := make([]PackSubscription, 0)
	err := tx.Where("user_id = ?", userID).Order("sort_order ASC").Order("pack_id ASC").Find(&rows).Error
	return rows, err
}

func Subscriptions(userID uint64) ([]PackSubscription, error) {
	return SubscriptionsTx(marketDB(), userID)
}

func SaveSubscriptionTx(tx *gorm.DB, row *PackSubscription) error {
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(row).Error
}

func RemoveSubscriptionTx(tx *gorm.DB, userID, packID uint64) error {
	return tx.Where("user_id = ? AND pack_id = ?", userID, packID).Delete(&PackSubscription{}).Error
}
