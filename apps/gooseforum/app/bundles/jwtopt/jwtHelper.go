package jwtopt

import (
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
	"github.com/spf13/cast"
)

var (
	once       sync.Once
	std        *JWT
	signingKey = preferences.Get("app.signingKey", DefaultSigningKey)
	validTime  = time.Duration(preferences.GetInt64("jwtopt.validTime", 86400*7)) * time.Second
)

// DefaultSigningKey is the built-in fallback when app.signingKey is missing.
// It must never be used in production: it is public in the source tree, so an
// attacker could forge JWTs and decrypt TOTP secrets with it.
const DefaultSigningKey = "mq+ZeGafL+b1xdC0u9vSVg=="

// IsSigningKeyDefault reports whether the configured signing key is still the
// built-in fallback. Callers should refuse to serve when this is true.
func IsSigningKeyDefault() bool {
	return signingKey == DefaultSigningKey
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
