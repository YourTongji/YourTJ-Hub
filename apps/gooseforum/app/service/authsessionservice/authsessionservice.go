// Package authsessionservice validates forum session JWTs (the HS256
// "access_token") against the live session table. Both the HTTP middleware
// and the OIDC login bridge share this single verification path so the
// accepted-session semantics stay identical everywhere.
package authsessionservice

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// ValidateToken verifies a forum session token and returns the user ID, jti
// and the (possibly refreshed) token when it maps to a live, non-expired
// session owned by that user with a matching TokenVersion. Challenge tokens
// (e.g. TOTP second factor) never create a session row and are rejected here.
func ValidateToken(token string) (userID uint64, jti, newToken string, ok bool) {
	if token == "" {
		return 0, "", "", false
	}
	claims, refreshedToken, err := jwtopt.VerifyTokenWithFreshClaims(token)
	if err != nil {
		return 0, "", "", false
	}
	user, userOK := userservice.GetUserInfo(claims.UserId)
	if !userOK || user.TokenVersion != claims.TokenVersion || user.ActorType == users.ActorTypeBot {
		return 0, "", "", false
	}
	if claims.Jti == "" {
		return 0, "", "", false
	}
	entity := sessionservice.GetValidByJti(claims.Jti)
	if entity == nil || entity.UserId != claims.UserId {
		return 0, "", "", false
	}
	return claims.UserId, claims.Jti, refreshedToken, true
}
