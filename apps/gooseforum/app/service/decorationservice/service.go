// Package decorationservice owns cosmetic entitlements and equipment. Other
// domains can grant/revoke with a durable source key, but cannot bypass ownership.
package decorationservice

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/decoration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalid = errors.New("invalid decoration input")
var ErrUnavailable = errors.New("decoration unavailable or not owned")
var ErrConflict = errors.New("decoration request key reused with different input")

type Equipped struct {
	AvatarFrame   string `json:"avatarFrame"`
	NicknameStyle string `json:"nicknameStyle"`
}
type Display struct {
	AvatarFrame   *Style `json:"avatarFrame,omitempty"`
	NicknameStyle *Style `json:"nicknameStyle,omitempty"`
}
type State struct {
	Catalog    []Style  `json:"catalog"`
	OwnedCodes []string `json:"ownedCodes"`
	Equipped   Equipped `json:"equipped"`
}
type Adjustment struct {
	UserID     uint64 `json:"userId"`
	Code       string `json:"code"`
	Reason     string `json:"reason"`
	RequestKey string `json:"requestKey"`
}

func Catalog(ctx context.Context) ([]Style, error) { return catalog(db.ConnectContext(ctx)) }
func catalog(conn *gorm.DB) ([]Style, error) {
	var overrides []decoration.Definition
	if err := conn.Find(&overrides).Error; err != nil {
		return nil, err
	}
	byCode := map[string]Style{}
	for _, s := range overrides {
		// Invalid legacy/externally altered rows never reach a CSS renderer.
		if ValidateStyle(s) != nil {
			delete(byCode, s.Code)
			continue
		}
		if s.AssetScale == 0 {
			s.AssetScale = DefaultAssetScale
		}
		byCode[s.Code] = s
	}
	styles := make([]Style, 0, len(byCode))
	for _, s := range byCode {
		styles = append(styles, s)
	}
	slices.SortFunc(styles, func(a, b Style) int {
		if a.SortOrder != b.SortOrder {
			return a.SortOrder - b.SortOrder
		}
		return strings.Compare(a.Code, b.Code)
	})
	return styles, nil
}

func SaveStyle(ctx context.Context, actorID uint64, style Style) (Style, error) {
	style.Name = strings.TrimSpace(style.Name)
	if style.AssetScale == 0 {
		style.AssetScale = DefaultAssetScale
	}
	if actorID == 0 || ValidateStyle(style) != nil {
		return Style{}, ErrInvalid
	}
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error {
		current, err := catalog(tx)
		if err != nil {
			return err
		}
		for _, s := range current {
			if s.Code == style.Code && s.Kind != style.Kind {
				return ErrInvalid
			}
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "kind", "light_colors", "dark_colors", "is_enabled", "sort_order", "asset_url", "asset_scale"})}).Create(&style).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(style)
		return tx.Create(&decoration.Audit{RequestKey: "definition:" + rand.Text(), Code: style.Code, Action: "definition", Source: "manual", ActorID: actorID, Reason: string(details), CreatedAt: time.Now()}).Error
	})
	return style, err
}

func Current(ctx context.Context, userID uint64) (State, error) {
	return current(db.ConnectContext(ctx), userID)
}
func current(conn *gorm.DB, userID uint64) (State, error) {
	state := State{OwnedCodes: []string{}}
	var account users.EntityComplete
	if err := conn.Select("id").Where("id = ?", userID).First(&account).Error; err != nil {
		return state, err
	}
	var err error
	state.Catalog, err = catalog(conn)
	if err != nil {
		return state, err
	}
	var owned []decoration.Entitlement
	if err = conn.Where("user_id = ? AND revoked_at IS NULL", userID).Find(&owned).Error; err != nil {
		return state, err
	}
	for _, row := range owned {
		state.OwnedCodes = append(state.OwnedCodes, row.Code)
	}
	slices.Sort(state.OwnedCodes)
	state.OwnedCodes = slices.Compact(state.OwnedCodes)
	var equipment decoration.Equipment
	err = conn.Where("user_id = ?", userID).First(&equipment).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return state, err
	}
	for _, s := range state.Catalog {
		if !s.IsEnabled || !slices.Contains(state.OwnedCodes, s.Code) {
			continue
		}
		if s.Kind == AvatarFrame && equipment.AvatarFrame == s.Code {
			state.Equipped.AvatarFrame = s.Code
		}
		if s.Kind == NicknameStyle && equipment.NicknameStyle == s.Code {
			state.Equipped.NicknameStyle = s.Code
		}
	}
	return state, nil
}

// lockAccount serializes equipment and entitlement mutations on the same
// account, including revoke/equip races. SQLite serializes transaction writes.
func lockAccount(tx *gorm.DB, userID uint64) error {
	if userID == 0 {
		return ErrInvalid
	}
	var account users.EntityComplete
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", userID).First(&account).Error
}

func Equip(ctx context.Context, userID uint64, selection Equipped) (State, error) {
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error { return equipTx(tx, userID, selection) })
	if err != nil {
		return State{}, err
	}
	return Current(ctx, userID)
}
func equipTx(tx *gorm.DB, userID uint64, selection Equipped) error {
	if err := lockAccount(tx, userID); err != nil {
		return err
	}
	state, err := current(tx, userID)
	if err != nil {
		return err
	}
	for kind, code := range map[string]string{AvatarFrame: selection.AvatarFrame, NicknameStyle: selection.NicknameStyle} {
		if code == "" {
			continue
		}
		if !slices.Contains(state.OwnedCodes, code) {
			return ErrUnavailable
		}
		found := false
		for _, s := range state.Catalog {
			if s.Code == code && s.Kind == kind && s.IsEnabled {
				found = true
				break
			}
		}
		if !found {
			return ErrUnavailable
		}
	}
	row := decoration.Equipment{UserID: userID, AvatarFrame: selection.AvatarFrame, NicknameStyle: selection.NicknameStyle}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.AssignmentColumns([]string{"avatar_frame", "nickname_style"})}).Create(&row).Error; err != nil {
		return err
	}
	if state.Equipped == selection {
		return nil
	}
	return tx.Create(&decoration.Audit{RequestKey: "equip:" + rand.Text(), UserID: userID, Action: "equip", Source: "self", ActorID: userID, Reason: selection.AvatarFrame + " / " + selection.NicknameStyle, CreatedAt: time.Now()}).Error
}

func Adjust(ctx context.Context, actorID uint64, input Adjustment, revoke bool) (State, error) {
	err := db.ConnectContext(ctx).Transaction(func(tx *gorm.DB) error { return AdjustTx(tx, actorID, "manual", input, revoke) })
	if err != nil {
		return State{}, err
	}
	return Current(ctx, input.UserID)
}

// AdjustTx is the trusted integration boundary for XP, trust and Mall. It must
// run inside the caller's transaction; the source key survives retries/restarts.
// No public request can choose its source or grant itself a paid entitlement.
func AdjustTx(tx *gorm.DB, actorID uint64, source string, input Adjustment, revoke bool) error {
	if !slices.Contains([]string{"manual", "xp", "trust", "credit", "migration"}, source) || (source == "manual" && actorID == 0) || !codePattern.MatchString(input.Code) || len(input.RequestKey) < 8 || len(input.RequestKey) > 128 || strings.TrimSpace(input.Reason) == "" || utf8.RuneCountInString(input.Reason) > 512 {
		return ErrInvalid
	}
	if err := lockAccount(tx, input.UserID); err != nil {
		return err
	}
	action := "grant"
	if revoke {
		action = "revoke"
	}
	var receipt decoration.Audit
	err := tx.Where("request_key = ?", input.RequestKey).First(&receipt).Error
	if err == nil {
		if receipt.UserID != input.UserID || receipt.Code != input.Code || receipt.Action != action || receipt.Source != source || receipt.ActorID != actorID || receipt.Reason != input.Reason {
			return ErrConflict
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	styles, err := catalog(tx)
	if err != nil {
		return err
	}
	found := false
	for _, s := range styles {
		if s.Code == input.Code && (revoke || s.IsEnabled) {
			found = true
			break
		}
	}
	if !found {
		return ErrUnavailable
	}
	now := time.Now()
	if revoke {
		target := tx.Model(&decoration.Entitlement{}).Where("user_id = ? AND code = ? AND revoked_at IS NULL", input.UserID, input.Code)
		if source != "manual" {
			target = target.Where("source = ?", source)
		}
		if err := target.Update("revoked_at", now).Error; err != nil {
			return err
		}
		var remaining int64
		if err := tx.Model(&decoration.Entitlement{}).Where("user_id = ? AND code = ? AND revoked_at IS NULL", input.UserID, input.Code).Count(&remaining).Error; err != nil {
			return err
		}
		if remaining == 0 {
			for _, column := range []string{"avatar_frame", "nickname_style"} {
				if err := tx.Model(&decoration.Equipment{}).Where("user_id = ? AND "+column+" = ?", input.UserID, input.Code).Update(column, "").Error; err != nil {
					return err
				}
			}
		}
	} else {
		row := decoration.Entitlement{UserID: input.UserID, Code: input.Code, Source: source, SourceKey: input.RequestKey, GrantedAt: now}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}, {Name: "code"}, {Name: "source"}}, DoUpdates: clause.AssignmentColumns([]string{"source", "source_key", "granted_at", "revoked_at"}), Where: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "user_decorations.revoked_at IS NOT NULL"}}}}).Create(&row).Error; err != nil {
			return err
		}
	}
	return tx.Create(&decoration.Audit{RequestKey: input.RequestKey, UserID: input.UserID, Code: input.Code, Action: action, Source: source, ActorID: actorID, Reason: input.Reason, CreatedAt: now}).Error
}

func Effective(ctx context.Context, userID uint64) *Display {
	return EffectiveBatch(ctx, []uint64{userID})[userID]
}
func EffectiveBatch(ctx context.Context, userIDs []uint64) map[uint64]*Display {
	return effectiveBatch(db.ConnectContext(ctx), userIDs)
}
func effectiveBatch(conn *gorm.DB, userIDs []uint64) map[uint64]*Display {
	result := map[uint64]*Display{}
	if len(userIDs) == 0 {
		return result
	}
	var equipment []decoration.Equipment
	// Closed accounts keep historical audit data but never display cosmetics.
	if err := conn.Where("user_id IN ? AND user_id IN (SELECT id FROM users WHERE deleted_at IS NULL)", userIDs).Find(&equipment).Error; err != nil {
		return result
	}
	if len(equipment) == 0 {
		return result
	}
	styles, err := catalog(conn)
	if err != nil {
		return result
	}
	byCode := map[string]Style{}
	for _, s := range styles {
		if s.IsEnabled {
			byCode[s.Code] = s
		}
	}
	var owned []decoration.Entitlement
	if err := conn.Where("user_id IN ? AND revoked_at IS NULL", userIDs).Find(&owned).Error; err != nil {
		return result
	}
	active := map[uint64]map[string]bool{}
	for _, e := range owned {
		if active[e.UserID] == nil {
			active[e.UserID] = map[string]bool{}
		}
		active[e.UserID][e.Code] = true
	}
	for _, e := range equipment {
		d := Display{}
		if s, ok := byCode[e.AvatarFrame]; ok && s.Kind == AvatarFrame && active[e.UserID][s.Code] {
			d.AvatarFrame = &s
		}
		if s, ok := byCode[e.NicknameStyle]; ok && s.Kind == NicknameStyle && active[e.UserID][s.Code] {
			d.NicknameStyle = &s
		}
		if d.AvatarFrame != nil || d.NicknameStyle != nil {
			result[e.UserID] = &d
		}
	}
	return result
}

type AdminState struct {
	State
	Audits []decoration.Audit `json:"audits"`
}

func AdminCurrent(ctx context.Context, userID uint64) (AdminState, error) {
	state, err := Current(ctx, userID)
	result := AdminState{State: state, Audits: []decoration.Audit{}}
	if err != nil {
		return result, err
	}
	err = db.ConnectContext(ctx).Where("user_id = ?", userID).Order("id DESC").Limit(100).Find(&result.Audits).Error
	return result, err
}
