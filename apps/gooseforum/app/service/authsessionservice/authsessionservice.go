// Package authsessionservice validates forum session JWTs (the HS256
// "access_token") against the live session table. Both the HTTP middleware
// and the OIDC login bridge share this single verification path so the
// accepted-session semantics stay identical everywhere.
package authsessionservice

import (
	"context"

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
	userID, jti, newToken, ok, _ = ValidateTokenContext(context.Background(), token)
	return
}

// ValidateTokenContext distinguishes an invalid credential from an unavailable
// dependency. A caller must never grant access on either failure.
func ValidateTokenContext(ctx context.Context, token string) (userID uint64, jti, newToken string, ok bool, err error) {
	if token == "" {
		return 0, "", "", false, nil
	}
	claims, refreshedToken, err := jwtopt.VerifyTokenWithFreshClaims(token)
	if err != nil {
		return 0, "", "", false, nil
	}
	if claims.Jti == "" || claims.Purpose != "" {
		return 0, "", "", false, nil
	}
	user, userOK, err := userservice.LookupSessionUser(ctx, claims.UserId)
	if err != nil {
		return 0, "", "", false, err
	}
	if !userOK || user.TokenVersion != claims.TokenVersion || user.ActorType == users.ActorTypeBot {
		return 0, "", "", false, nil
	}
	live, err := sessionservice.CheckLiveContext(ctx, claims.UserId, claims.Jti)
	if err != nil || !live {
		return 0, "", "", false, err
	}
	return claims.UserId, claims.Jti, refreshedToken, true, nil
}
