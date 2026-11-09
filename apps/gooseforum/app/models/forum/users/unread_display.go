package users

import (
	"encoding/json"
	"errors"
	"slices"

	"gorm.io/gorm"
)

const (
	UnreadBadgeNumber = "number"
	UnreadBadgeDot    = "dot"
)

var ErrUnreadBadgeStyle = errors.New("invalid unread badge style")

// GetUnreadBadgeStyle reads an account preference independently of unread facts
// and push-delivery consent. Unknown legacy values retain the numeric default.
func GetUnreadBadgeStyle(userID uint64) (string, error) {
	var user EntityComplete
	if err := builder().Select("id", "unread_badge_style").Where("id = ? AND deleted_at IS NULL", userID).Take(&user).Error; err != nil {
		return "", err
	}
	if user.UnreadBadgeStyle == UnreadBadgeDot {
		return UnreadBadgeDot, nil
	}
	return UnreadBadgeNumber, nil
}

// SetUnreadBadgeStyle updates only the display field; a preference save cannot
// overwrite a concurrent password/session or push-consent update.
func SetUnreadBadgeStyle(userID uint64, style string) error {
	if style != UnreadBadgeNumber && style != UnreadBadgeDot {
		return ErrUnreadBadgeStyle
	}
	result := builder().Model(&EntityComplete{}).Where("id = ? AND deleted_at IS NULL", userID).Update("unread_badge_style", style)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var ErrUnreadBadgeCategories = errors.New("invalid unread badge categories")

// BadgeCategories follows the notification center taxonomy, independently of delivery groups.
func DefaultUnreadBadgeCategories() []string {
	return []string{"replies", "mentions", "likes", "following", "system"}
}

type UnreadDisplayPreferences struct {
	BadgeStyle      string   `json:"badgeStyle"`
	BadgeCategories []string `json:"badgeCategories"`
}

func GetUnreadDisplayPreferences(userID uint64) (UnreadDisplayPreferences, error) {
	var user EntityComplete
	prefs := UnreadDisplayPreferences{BadgeStyle: UnreadBadgeNumber, BadgeCategories: DefaultUnreadBadgeCategories()}
	if err := builder().Select("id", "unread_badge_style", "unread_badge_categories").Where("id = ? AND deleted_at IS NULL", userID).Take(&user).Error; err != nil {
		return prefs, err
	}
	if user.UnreadBadgeStyle == UnreadBadgeDot {
		prefs.BadgeStyle = UnreadBadgeDot
	}
	if user.UnreadBadgeCategories != nil {
		if err := json.Unmarshal([]byte(*user.UnreadBadgeCategories), &prefs.BadgeCategories); err != nil {
			return prefs, err
		}
		if prefs.BadgeCategories == nil {
			return prefs, ErrUnreadBadgeCategories
		}
	}
	return prefs, nil
}

// Patch columns directly: concurrent style/category saves cannot overwrite each other or push consent.
func PatchUnreadDisplayPreferences(userID uint64, style *string, categories *[]string) error {
	if style == nil && categories == nil {
		return ErrUnreadBadgeCategories
	}
	changes := map[string]any{}
	if style != nil {
		if *style != UnreadBadgeNumber && *style != UnreadBadgeDot {
			return ErrUnreadBadgeStyle
		}
		changes["unread_badge_style"] = *style
	}
	if categories != nil {
		if len(*categories) > 5 {
			return ErrUnreadBadgeCategories
		}
		for _, category := range *categories {
			if !slices.Contains(DefaultUnreadBadgeCategories(), category) {
				return ErrUnreadBadgeCategories
			}
		}
		ordered := make([]string, 0, len(*categories))
		for _, category := range DefaultUnreadBadgeCategories() {
			if slices.Contains(*categories, category) {
				ordered = append(ordered, category)
			}
		}
		if len(ordered) != len(*categories) {
			return ErrUnreadBadgeCategories
		}
		raw, err := json.Marshal(ordered)
		if err != nil {
			return err
		}
		changes["unread_badge_categories"] = string(raw)
	}
	result := builder().Model(&EntityComplete{}).Where("id = ? AND deleted_at IS NULL", userID).Updates(changes)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
