package stickerservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/localcache"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/fileUsage"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/sticker"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/fileusageservice"
	"gorm.io/gorm"
)

// MaxBatchStickers bounds one administrator batch action.
const MaxBatchStickers = 500

const maxPackNameLen = 64

var (
	ErrPackNameRequired = errors.New("sticker pack name required")
	ErrPackInvalid      = errors.New("invalid sticker pack")
)

// packKeyRe matches the keys stickers store in Entity.Pack. New packs get a
// generated key, so administrators only ever edit the display name.
var packKeyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// presetPackDefaults name and order the seeded groups until an administrator
// edits them; other keys sort after them.
var presetPackDefaults = map[string]struct {
	name  string
	order int
}{
	sticker.PackOfficial: {"默认", 0},
	"emojipackage":       {"EmojiPackage", 10},
	"flowerhd":           {"花字", 20},
	"wxmeme":             {"程序员梗", 30},
}

const customPackOrder = 1000

var officialPackCache = &localcache.Cache[[]OfficialPackItem]{MaxEntries: 1}

// OfficialPackItem is the public view of one official group with enabled stickers.
type OfficialPackItem struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	CoverURL     string `json:"coverUrl"`
	StickerCount int    `json:"stickerCount"`
}

// AdminOfficialPack is one official group with all of its stickers counted.
type AdminOfficialPack struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	SortOrder      int    `json:"sortOrder"`
	CoverStickerID uint64 `json:"coverStickerId"`
	CoverURL       string `json:"coverUrl"`
	StickerCount   int    `json:"stickerCount"`
	EnabledCount   int    `json:"enabledCount"`
}

func validPackKey(key string) bool { return packKeyRe.MatchString(key) }

func cleanPackName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", ErrPackNameRequired
	}
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxPackNameLen || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", ErrPackInvalid
	}
	return name, nil
}

// packView merges stored metadata with the defaults for one key.
func packView(key string, rows map[string]sticker.OfficialPack) (name string, order int, cover uint64) {
	if row, ok := rows[key]; ok {
		return row.Name, row.SortOrder, row.CoverStickerID
	}
	if preset, ok := presetPackDefaults[key]; ok {
		return preset.name, preset.order, 0
	}
	return key, customPackOrder, 0
}

func officialPackRows(tx *gorm.DB) (map[string]sticker.OfficialPack, error) {
	rows, err := sticker.OfficialPacksTx(tx)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]sticker.OfficialPack, len(rows))
	for _, row := range rows {
		byKey[row.Key] = row
	}
	return byKey, nil
}

func sortPacks[T any](packs []T, order func(T) (int, string)) {
	sort.SliceStable(packs, func(i, j int) bool {
		oi, ki := order(packs[i])
		oj, kj := order(packs[j])
		if oi != oj {
			return oi < oj
		}
		return ki < kj
	})
}

// OfficialPackList lists the official groups members can pick from, in
// administrator order. The cover is the chosen sticker when it is still
// enabled in the group, otherwise the group's first sticker.
func OfficialPackList() ([]OfficialPackItem, error) {
	return officialPackCache.GetOrLoadE("all", func() ([]OfficialPackItem, error) {
		stickers, err := EnabledList()
		if err != nil {
			return nil, err
		}
		rows, err := officialPackRows(db.Connect())
		if err != nil {
			return nil, err
		}
		byKey := map[string]*OfficialPackItem{}
		orders := map[string]int{}
		var packs []*OfficialPackItem
		for _, item := range stickers {
			key := item.Pack
			if key == "" {
				key = sticker.PackOfficial
			}
			pack := byKey[key]
			if pack == nil {
				name, order, cover := packView(key, rows)
				pack = &OfficialPackItem{Key: key, Name: name}
				byKey[key], orders[key] = pack, order
				packs = append(packs, pack)
				if cover != 0 {
					if index := slices.IndexFunc(stickers, func(s StickerItem) bool { return s.ID == cover && s.Pack == key }); index >= 0 {
						pack.CoverURL = stickers[index].URL
					}
				}
			}
			if pack.CoverURL == "" {
				pack.CoverURL = item.URL
			}
			pack.StickerCount++
		}
		sortPacks(packs, func(p *OfficialPackItem) (int, string) { return orders[p.Key], p.Key })
		items := make([]OfficialPackItem, 0, len(packs))
		for _, pack := range packs {
			items = append(items, *pack)
		}
		return items, nil
	}, stickerCacheTTL)
}

// AdminOfficialPacks lists every official group, including empty ones that
// an administrator created, with enabled and total counts.
func AdminOfficialPacks(ctx context.Context) ([]AdminOfficialPack, error) {
	conn := db.ConnectContext(ctx)
	rows, err := officialPackRows(conn)
	if err != nil {
		return nil, err
	}
	counts, err := sticker.OfficialPackCounts(conn)
	if err != nil {
		return nil, err
	}
	byKey := map[string]*AdminOfficialPack{}
	add := func(key string) *AdminOfficialPack {
		if pack, ok := byKey[key]; ok {
			return pack
		}
		name, order, cover := packView(key, rows)
		pack := &AdminOfficialPack{Key: key, Name: name, SortOrder: order, CoverStickerID: cover}
		byKey[key] = pack
		return pack
	}
	add(sticker.PackOfficial)
	for key := range rows {
		add(key)
	}
	for _, count := range counts {
		key := count.Pack
		if key == "" {
			key = sticker.PackOfficial
		}
		pack := add(key)
		pack.StickerCount += count.Total
		pack.EnabledCount += count.Enabled
	}
	packs := make([]AdminOfficialPack, 0, len(byKey))
	coverIDs := make([]uint64, 0, len(byKey))
	for _, pack := range byKey {
		packs = append(packs, *pack)
		if pack.CoverStickerID != 0 {
			coverIDs = append(coverIDs, pack.CoverStickerID)
		}
	}
	sortPacks(packs, func(p AdminOfficialPack) (int, string) { return p.SortOrder, p.Key })
	covers := make([]sticker.Entity, 0, len(coverIDs))
	if len(coverIDs) > 0 {
		if err := sticker.FindByIDs(coverIDs, &covers); err != nil {
			return nil, err
		}
	}
	for index := range packs {
		for _, cover := range covers {
			if cover.Id == packs[index].CoverStickerID && cover.Pack == packs[index].Key {
				packs[index].CoverURL = ResolveURLFor(cover)
			}
		}
		if packs[index].CoverURL == "" && packs[index].StickerCount > 0 {
			var first sticker.Entity
			err := conn.Where("is_official = ? AND pack = ? AND file_name <> ''", true, packs[index].Key).
				Order("sort_order ASC").Order("id ASC").Limit(1).Find(&first).Error
			if err != nil {
				return nil, err
			}
			packs[index].CoverURL = ResolveURLFor(first)
		}
	}
	return packs, nil
}

// knownPackTx reports whether key names a group stickers can join: a stored
// pack, a preset, or a key that official stickers already use.
func knownPackTx(tx *gorm.DB, key string) (bool, error) {
	if !validPackKey(key) {
		return false, nil
	}
	if _, ok := presetPackDefaults[key]; ok {
		return true, nil
	}
	if _, err := sticker.OfficialPackTx(tx, key); err == nil {
		return true, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	var count int64
	err := tx.Model(&sticker.Entity{}).Where("is_official = ? AND pack = ?", true, key).Count(&count).Error
	return count > 0, err
}

// SaveOfficialPackInput creates a pack (empty Key) or edits one.
type SaveOfficialPackInput struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	SortOrder      int    `json:"sortOrder"`
	CoverStickerID uint64 `json:"coverStickerId"`
}

// SaveOfficialPack stores a pack's name, order and cover. The cover must be
// one of the pack's own stickers; zero picks the first one automatically.
func SaveOfficialPack(ctx context.Context, input SaveOfficialPackInput) (string, error) {
	name, err := cleanPackName(input.Name)
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(input.Key)
	err = db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if key == "" {
			var random [4]byte
			if _, err := rand.Read(random[:]); err != nil {
				return err
			}
			key = "p-" + hex.EncodeToString(random[:])
		} else if known, err := knownPackTx(tx, key); err != nil {
			return err
		} else if !known {
			return ErrPackInvalid
		}
		if input.CoverStickerID != 0 {
			cover, err := sticker.GetByIDTx(tx, input.CoverStickerID)
			if errors.Is(err, gorm.ErrRecordNotFound) || err == nil && (!cover.IsOfficial || cover.Pack != key) {
				return ErrPackInvalid
			}
			if err != nil {
				return err
			}
		}
		return sticker.SaveOfficialPackTx(tx, &sticker.OfficialPack{Key: key, Name: name, SortOrder: input.SortOrder, CoverStickerID: input.CoverStickerID})
	})
	if err != nil {
		return "", err
	}
	InvalidateCache()
	return key, nil
}

// DeleteOfficialPack moves a pack's stickers into another pack and removes
// its metadata. Stickers keep their names, so sent content is unaffected. The
// default pack holds new uploads and cannot be removed.
func DeleteOfficialPack(ctx context.Context, key, moveTo string) (int64, error) {
	if moveTo == "" {
		moveTo = sticker.PackOfficial
	}
	if key == sticker.PackOfficial || key == moveTo || !validPackKey(key) {
		return 0, ErrPackInvalid
	}
	var moved int64
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, candidate := range []string{key, moveTo} {
			if known, err := knownPackTx(tx, candidate); err != nil {
				return err
			} else if !known {
				return ErrPackInvalid
			}
		}
		var err error
		if moved, err = sticker.MoveOfficialStickersTx(tx, key, moveTo); err != nil {
			return err
		}
		return sticker.DeleteOfficialPackTx(tx, key)
	})
	if err != nil {
		return 0, err
	}
	InvalidateCache()
	return moved, nil
}

// Batch actions administrators apply to selected official stickers.
const (
	BatchEnable  = "enable"
	BatchDisable = "disable"
	BatchMove    = "move"
	BatchDelete  = "delete"
)

// BatchInput applies one action to up to MaxBatchStickers official stickers.
// Pack is the target of BatchMove.
type BatchInput struct {
	IDs    []uint64 `json:"ids"`
	Action string   `json:"action"`
	Pack   string   `json:"pack"`
}

// BatchStickers applies the action in one transaction and returns how many
// official stickers it changed; ids of other or missing stickers are ignored.
func BatchStickers(ctx context.Context, input BatchInput) (int, error) {
	ids := slices.Compact(slices.Sorted(slices.Values(input.IDs)))
	if len(ids) > 0 && ids[0] == 0 {
		ids = ids[1:]
	}
	if len(ids) == 0 || len(ids) > MaxBatchStickers {
		return 0, ErrPackInvalid
	}
	switch input.Action {
	case BatchEnable, BatchDisable, BatchMove, BatchDelete:
	default:
		return 0, ErrPackInvalid
	}
	changed := 0
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.Action == BatchMove {
			if known, err := knownPackTx(tx, input.Pack); err != nil {
				return err
			} else if !known {
				return ErrPackInvalid
			}
		}
		rows, err := sticker.OfficialByIDsTx(tx, ids)
		if err != nil {
			return err
		}
		for _, row := range rows {
			switch input.Action {
			case BatchEnable, BatchDisable:
				enabled := input.Action == BatchEnable
				if row.IsEnabled == enabled {
					continue
				}
				err = tx.Model(&sticker.Entity{}).Where("id = ?", row.Id).Update("is_enabled", enabled).Error
			case BatchMove:
				if row.Pack == input.Pack {
					continue
				}
				err = tx.Model(&sticker.Entity{}).Where("id = ?", row.Id).Update("pack", input.Pack).Error
			case BatchDelete:
				if err = fileusageservice.SetStickerUsageTx(tx, row.Id, 0, "", fileUsage.UsageStatusActive); err == nil {
					err = sticker.DeleteTx(tx, row.Id)
				}
			}
			if err != nil {
				return err
			}
			changed++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	InvalidateCache()
	return changed, nil
}
