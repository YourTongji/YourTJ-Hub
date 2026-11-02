package users

import (
	"errors"

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
