package authsessionservice

import (
	"context"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
)

// CheckStreamToken performs non-refreshing, uncached validation during an SSE
// connection. Invalid credentials return false; a database failure returns an
// error so clients reconnect without treating a temporary outage as logout.
func CheckStreamToken(ctx context.Context, token string) (bool, error) {
	claims := parseStreamClaims(token)
	if claims == nil || claims.UserId == 0 || claims.Jti == "" {
		return false, nil
	}
	valid, err := sessionservice.CheckLiveContext(ctx, claims.UserId, claims.Jti)
	if err != nil || !valid {
		return valid, err
	}
	return userservice.CheckFreshSessionVersion(ctx, claims.UserId, claims.TokenVersion)
}

func parseStreamClaims(token string) *jwtopt.CustomClaims {
	claims, err := jwtopt.Std().ParseToken(token)
	if err != nil {
		return nil
	}
	return claims
}
