// Package authsessionservice validates forum session JWTs (the HS256
// "access_token") against the live session table. Both the HTTP middleware
// and the OIDC login bridge share this single verification path so the
// accepted-session semantics stay identical everywhere.
package authsessionservice

import (
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// SessionValidation holds a verified forum session and any sliding renewal.
type SessionValidation struct {
	UserID       uint64
	Jti          string
	Token        string
	ExpiresAt    time.Time
	Refreshed    bool
	NeedsPersist bool
}

// ValidateSession verifies a forum session token against the live user and
// user_sessions row, then computes a sliding renewal (bounded by
// jwtopt.MaxSessionLifetime) when eligible. Challenge tokens never create a
// session row and are rejected here.
func ValidateSession(token string) (SessionValidation, bool) {
	if token == "" {
		return SessionValidation{}, false
	}
	claims, err := jwtopt.Std().ParseToken(token)
	if err != nil {
		return SessionValidation{}, false
	}
	user, userOK := userservice.GetUserInfo(claims.UserId)
	if !userOK || user.TokenVersion != claims.TokenVersion || user.ActorType == users.ActorTypeBot {
		return SessionValidation{}, false
	}
	if claims.Jti == "" {
		return SessionValidation{}, false
	}
	entity := sessionservice.GetValidByJti(claims.Jti)
	if entity == nil || entity.UserId != claims.UserId {
		return SessionValidation{}, false
	}
	nextToken, expiresAt, refreshed, needsPersist, err := jwtopt.RefreshSessionTokenIfNeeded(token, claims, entity.CreatedAt, entity.ExpiresAt)
	if err != nil {
		return SessionValidation{}, false
	}
	return SessionValidation{
		UserID:       claims.UserId,
		Jti:          claims.Jti,
		Token:        nextToken,
		ExpiresAt:    expiresAt,
		Refreshed:    refreshed,
		NeedsPersist: needsPersist,
	}, true
}

// ValidateToken verifies a forum session token and returns the user ID, jti
// and the (possibly refreshed) token when it maps to a live, non-expired
// session owned by that user with a matching TokenVersion. Challenge tokens
// (e.g. TOTP second factor) never create a session row and are rejected here.
func ValidateToken(token string) (userID uint64, jti, newToken string, ok bool) {
	res, ok := ValidateSession(token)
	if !ok {
		return 0, "", "", false
	}
	return res.UserID, res.Jti, res.Token, true
}
