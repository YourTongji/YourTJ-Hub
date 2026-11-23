package sessionservice

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"gorm.io/gorm"
)

// sessionLifetime mirrors jwtopt.InitialSessionLifetime so session records stay
// in sync with token expiry (default 30 days, bounded by maxSessionLifetime).
func sessionLifetime() time.Duration {
	return jwtopt.InitialSessionLifetime()
}

func isWithinMaxLifetime(createdAt, now time.Time) bool {
	maxLife := jwtopt.MaxSessionLifetime()
	if maxLife <= 0 || createdAt.IsZero() {
		return true
	}
	return createdAt.Add(maxLife).After(now)
}

func maxLifetimeCutoff(now time.Time) time.Time {
	if maxLife := jwtopt.MaxSessionLifetime(); maxLife > 0 {
		return now.Add(-maxLife)
	}
	return time.Time{}
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

// CreateWithDB persists a session through the caller's transaction. Ceremony
// completion uses this so consuming its one-time challenge and issuing the
// ordinary forum session commit atomically.
func CreateWithDB(tx *gorm.DB, userID uint64, jti string, userAgent string, ip string) error {
	entity := &userSessions.Entity{
		UserId:    userID,
		Jti:       jti,
		UserAgent: truncateUA(userAgent),
		Ip:        ip,
		ExpiresAt: time.Now().Add(sessionLifetime()),
	}
	return tx.Create(entity).Error
}

// GetValidByJti returns a non-expired session for jti, or nil.
// Used by the auth middleware to reject revoked, expired, or challenge-only tokens.
func GetValidByJti(jti string) *userSessions.Entity {
	entity := userSessions.GetByJti(jti)
	if entity == nil {
		return nil
	}
	now := time.Now()
	if !entity.ExpiresAt.After(now) || !isWithinMaxLifetime(entity.CreatedAt, now) {
		return nil
	}
	return entity
}

// CheckLiveContext distinguishes revocation from a transient database error.
func CheckLiveContext(ctx context.Context, userID uint64, jti string) (bool, error) {
	entity, err := userSessions.GetByJtiContext(ctx, jti)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	now := time.Now()
	return entity.UserId == userID && entity.ExpiresAt.After(now) && isWithinMaxLifetime(entity.CreatedAt, now), nil
}

// TouchExpiry monotonically extends the live session row to at least expiresAt.
// It returns (true, nil) when the update persisted or when the live row already
// covers expiresAt; it returns (false, err) on DB errors and (false, nil) when
// the session row was concurrently revoked, idle-expired, or past its cap.
func TouchExpiry(jti string, expiresAt time.Time) (bool, error) {
	now := time.Now()
	rows, err := userSessions.ExtendExpiresAtByJtiIfLive(jti, expiresAt, now)
	if err != nil {
		return false, err
	}
	if rows > 0 {
		return true, nil
	}
	entity := GetValidByJti(jti)
	if entity == nil {
		return false, nil
	}
	return !entity.ExpiresAt.Before(expiresAt), nil
}

// List returns the user's active sessions, newest first.
func List(userID uint64) ([]userSessions.Entity, error) {
	now := time.Now()
	return userSessions.ListActiveByUserID(userID, now, maxLifetimeCutoff(now))
}

// RevokeByJti deletes one session owned by the user, keyed by its jti.
func RevokeByJti(userID uint64, jti string) error {
	return userSessions.DeleteByJtiAndUserID(userID, jti)
}

// RevokeByID deletes one session owned by the user.
func RevokeByID(userID uint64, id uint64) error {
	return userSessions.DeleteByID(userID, id)
}

// RevokeAllAndInvalidate atomically deletes every session and invalidates every
// previously issued token for the user. The user-info cache is cleared only
// after the database transaction commits.
//
// The transaction runs on the process-wide db.Connect() handle: concurrent
// callers on the same user serialize on the DB, and the cache invalidation
// after commit is race-free with respect to in-flight cache loads thanks to
// the epoch-based localcache invalidation.
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

// CleanupExpired removes idle-expired and max-lifetime-expired session rows.
func CleanupExpired() {
	now := time.Now()
	_ = userSessions.DeleteExpiredBefore(now, maxLifetimeCutoff(now))
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
