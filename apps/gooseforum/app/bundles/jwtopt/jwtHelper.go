package jwtopt

import (
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/spf13/cast"
)

var (
	once sync.Once
	std  *JWT
	// signingKey is captured once at package init (process start) and is not
	// affected by a runtime config reload — it aligns with securestore's
	// first-use capture. Rotating the key therefore requires a process restart
	// so all surfaces switch together; see docs/operations/deployment.md.
	signingKey = preferences.GetString("app.signingKey")
	validTime  = time.Duration(preferences.GetInt64("jwtopt.validTime", 86400*7)) * time.Second
)

// DefaultSigningKey is the legacy built-in fallback that was used when
// app.signingKey was missing. It is kept here ONLY as a known-bad value to
// reject: it is public in the source tree, so any deployment still using it
// lets an attacker forge JWTs and decrypt TOTP secrets. The fail-closed
// startup guard refuses to serve with this value (see SigningKeyProblem).
const DefaultSigningKey = "mq+ZeGafL+b1xdC0u9vSVg=="

// knownBadSigningKeys lists values the JWT signing key must NEVER take: the
// legacy built-in default (public in the source tree) and the deploy template
// placeholder shipped in deploy/config.toml.example. serve refuses to start
// with any of them.
var knownBadSigningKeys = map[string]string{
	DefaultSigningKey:     "built-in default signing key",
	"REPLACE_SIGNING_KEY": "deploy template placeholder signing key",
}

// SigningKeyProblem returns a human-readable reason when the configured JWT
// signing key is unsafe to use, or "" when it is acceptable. It validates the
// exact value captured at process start (the same value Std() signs with):
// empty / whitespace-only keys and known-bad values are rejected. Callers
// (the serve startup guard) must refuse to start when this returns non-empty.
func SigningKeyProblem() string {
	return SigningKeyProblemFor(signingKey)
}

// SigningKeyProblemFor applies the same weak-key policy as SigningKeyProblem
// to an arbitrary candidate value. It is exported so other packages that derive
// secrets from app.signingKey (tokenservice reset/activation tokens,
// securestore TOTP key) can share one fail-closed policy instead of each
// re-implementing the known-bad list. Returns "" when key is acceptable.
func SigningKeyProblemFor(key string) string {
	if strings.TrimSpace(key) == "" {
		return "empty signing key"
	}
	if reason, bad := knownBadSigningKeys[key]; bad {
		return reason
	}
	return ""
}

// Purpose values for scoped tokens.
const (
	// PurposeTotpChallenge marks a short-lived token issued after password
	// verification that may only be used to complete TOTP second-factor
	// verification. It never creates a session record.
	PurposeTotpChallenge = "totp_challenge"
)

// Std returns the process-wide JWT helper.
func Std() *JWT {
	once.Do(func() {
		std = NewJWT([]byte(signingKey))
	})
	return std
}

// CreateNewTokenDefault creates an access token with the configured lifetime.
func CreateNewTokenDefault(userId uint64) (string, error) {
	return CreateNewTokenWithVersion(userId, 0, validTime)
}

// CreateNewTokenDefaultWithVersion creates an access token with the configured lifetime and token version.
func CreateNewTokenDefaultWithVersion(userId, tokenVersion uint64) (string, error) {
	return CreateNewTokenWithVersion(userId, tokenVersion, validTime)
}

// CreateNewToken creates an access token with expireTime.
func CreateNewToken(userId uint64, expireTime time.Duration) (string, error) {
	return CreateNewTokenWithVersion(userId, 0, expireTime)
}

// CreateNewTokenWithVersion creates an access token with expireTime.
func CreateNewTokenWithVersion(userId, tokenVersion uint64, expireTime time.Duration) (string, error) {
	cc := CustomClaims{
		UserId:           userId,
		TokenVersion:     tokenVersion,
		RegisteredClaims: GetBaseRegisteredClaims(expireTime),
	}
	return Std().CreateToken(cc)
}

// CreateSessionToken creates a session-scoped access token with a fresh jti.
// It returns the token and the jti so the caller can persist the session
// record (see sessionservice). It does not touch the database itself.
func CreateSessionToken(userId, tokenVersion uint64) (token string, jti string, err error) {
	jti, err = GenerateJti()
	if err != nil {
		return "", "", err
	}
	cc := CustomClaims{
		UserId:           userId,
		TokenVersion:     tokenVersion,
		Jti:              jti,
		RegisteredClaims: GetBaseRegisteredClaims(validTime),
	}
	token, err = Std().CreateToken(cc)
	if err != nil {
		return "", "", err
	}
	return token, jti, nil
}

// CreateChallengeToken creates a short-lived token scoped to a single purpose
// (e.g. TOTP second-factor verification). It carries a jti but never creates
// a session record, so regular session-checking middleware rejects it.
func CreateChallengeToken(userId, tokenVersion uint64, purpose string, ttl time.Duration) (string, error) {
	token, _, err := CreateChallengeTokenWithJti(userId, tokenVersion, purpose, ttl)
	return token, err
}

// CreateChallengeTokenWithJti returns the challenge token and its jti so the
// caller can persist a one-time consumption record.
func CreateChallengeTokenWithJti(userId, tokenVersion uint64, purpose string, ttl time.Duration) (string, string, error) {
	jti, err := GenerateJti()
	if err != nil {
		return "", "", err
	}
	cc := CustomClaims{
		UserId:           userId,
		TokenVersion:     tokenVersion,
		Jti:              jti,
		Purpose:          purpose,
		RegisteredClaims: GetBaseRegisteredClaims(ttl),
	}
	token, err := Std().CreateToken(cc)
	if err != nil {
		return "", "", err
	}
	return token, jti, nil
}

// GenerateJti returns a cryptographically random 32-hex-character jti.
func GenerateJti() (string, error) {
	b, err := algorithm.GenerateRandomBytes(16)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifyTokenWithFresh verifies tokenStr and refreshes it when it is close to expiry.
func VerifyTokenWithFresh(tokenStr string) (userId uint64, newToken string, err error) {
	claims, newToken, err := VerifyTokenWithFreshClaims(tokenStr)
	if err != nil {
		return 0, "", err
	}
	return claims.UserId, newToken, nil
}

// VerifyTokenWithFreshClaims verifies tokenStr and returns claims, refreshing close-to-expiry tokens.
// Refreshing preserves jti and purpose so the underlying session record stays stable.
func VerifyTokenWithFreshClaims(tokenStr string) (*CustomClaims, string, error) {
	claims, err := Std().ParseToken(tokenStr)
	if err != nil {
		return nil, "", err
	}
	eTime, err := claims.GetExpirationTime()
	if err == nil && time.Now().Add(time.Second*86400*1).After(eTime.Time) {
		claims.RegisteredClaims = GetBaseRegisteredClaims(validTime)
		tokenStr, err = Std().CreateToken(*claims)
	}
	return claims, tokenStr, err
}

// VerifyToken verifies tokenStr and returns the user ID.
func VerifyToken(tokenStr string) (userId uint64, err error) {
	claims, err := Std().ParseToken(tokenStr)
	if err != nil {
		return 0, err
	}
	return claims.UserId, err
}

// GetGinAccessToken returns the bearer token or access_token cookie from c.
func GetGinAccessToken(c *gin.Context) string {
	var token string
	token = c.GetHeader("Authorization")
	token = strings.ReplaceAll(token, "Bearer ", "")
	if token == "" {
		token, _ = c.Cookie("access_token")
	}
	return token
}

// TokenSetting writes the refreshed token to headers and cookies.
func TokenSetting(c *gin.Context, newToken string) {
	TokenSettingWithMaxAge(c, newToken, validTime)
}

// TokenSettingWithMaxAge writes the token with an explicit cookie max age.
func TokenSettingWithMaxAge(c *gin.Context, newToken string, maxAge time.Duration) {
	c.Header("New-Token", newToken)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"access_token",
		newToken,
		cast.ToInt(maxAge/time.Second),
		"/",
		"",
		setting.CookieSecure(),
		true,
	)
}

// TokenClean expires the access_token cookie.
func TokenClean(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"access_token",
		"",
		-1,
		"/",
		"",
		setting.CookieSecure(),
		true,
	)
}
