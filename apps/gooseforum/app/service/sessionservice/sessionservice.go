package sessionservice

import (
	"fmt"
	"net"
	"strings"
	"time"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/models/forum/userSessions"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/service/userservice"
	"gorm.io/gorm"
)

// sessionLifetime mirrors jwtopt.validTime so session records stay in sync
// with token expiry (default 7 days).
func sessionLifetime() time.Duration {
	return time.Duration(preferences.GetInt64("jwtopt.validTime", 86400*7)) * time.Second
}

// Create persists a new session record for a freshly issued jti.
func Create(userID uint64, jti string, userAgent string, ip string) error {
	entity := &userSessions.Entity{
		UserId:    userID,
		Jti:       jti,
		UserAgent: truncateUA(userAgent),
		Ip:        ip,
		ExpiresAt: time.Now().Add(sessionLifetime()),
	}
	return userSessions.Create(entity)
}

// GetValidByJti returns a non-expired session for jti, or nil.
// Used by the auth middleware to reject revoked or challenge-only tokens.
func GetValidByJti(jti string) *userSessions.Entity {
	entity := userSessions.GetByJti(jti)
	if entity == nil {
		return nil
	}
	if entity.ExpiresAt.Before(time.Now()) {
		return nil
	}
	return entity
}

// TouchExpiry extends the session record expiry to match a refreshed token.
func TouchExpiry(jti string, expiresAt time.Time) {
	entity := userSessions.GetByJti(jti)
	if entity == nil {
		return
	}
	_ = userSessions.UpdateExpiresAtByJti(jti, expiresAt)
}

// List returns the user's sessions, newest first.
func List(userID uint64) ([]userSessions.Entity, error) {
	return userSessions.ListByUserID(userID)
}

// RevokeByJti deletes one session owned by the user, keyed by its jti.
func RevokeByJti(userID uint64, jti string) error {
	return userSessions.DeleteByJtiAndUserID(userID, jti)
}

// RevokeByID deletes one session owned by the user.
func RevokeByID(userID uint64, id uint64) error {
	return userSessions.DeleteByID(userID, id)
}

// RevokeAll deletes every session of the user.
func RevokeAll(userID uint64) error {
	return userSessions.DeleteAllByUserID(userID)
}

// RevokeAllAndInvalidate atomically deletes every session and invalidates every
// previously issued token for the user. The user-info cache is cleared only
// after the database transaction commits.
func RevokeAllAndInvalidate(userID uint64) error {
	if err := db.Connect().Transaction(func(tx *gorm.DB) error {
		if err := userSessions.DeleteAllByUserIDWithDB(tx, userID); err != nil {
			return err
		}
		return users.IncrementTokenVersionWithDB(tx, userID)
	}); err != nil {
		return err
	}
	userservice.InvalidateUserInfoCache(userID)
	return nil
}

// CleanupExpired removes expired session rows (cheap housekeeping on issue).
func CleanupExpired() {
	_ = userSessions.DeleteExpired()
}

// MaskIP hides the last octet of an IPv4 address and the interface ID of IPv6.
func MaskIP(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		// Strip port if present.
		if host, _, err := net.SplitHostPort(ip); err == nil {
			parsed = net.ParseIP(host)
			ip = host
		}
	}
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.*", v4[0], v4[1], v4[2])
	}
	if v6 := parsed.To16(); v6 != nil {
		// Keep the network half, mask the interface half.
		group := func(b []byte) uint16 { return uint16(b[0])<<8 | uint16(b[1]) }
		return fmt.Sprintf("%x:%x:%x:%x:*", group(v6[0:2]), group(v6[2:4]), group(v6[4:6]), group(v6[6:8]))
	}
	return ip
}

func truncateUA(ua string) string {
	if len(ua) > 512 {
		return ua[:512]
	}
	return ua
}
