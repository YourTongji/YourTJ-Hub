package jwtopt

import (
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/cast"
)

const (
	// DefaultValidTime is the default sliding idle lifetime of a session token (30 days).
	DefaultValidTime = 30 * 24 * time.Hour
	// DefaultRefreshInterval throttles sliding renewal so an active session rotates
	// at most once per 24 hours instead of on every request or only in the final 24h.
	DefaultRefreshInterval = 24 * time.Hour
	// DefaultMaxSessionLifetime caps how long a single login session can stay alive
	// through sliding renewal (180 days). Setting jwtopt.maxSessionLifetime = 0 disables the cap.
	DefaultMaxSessionLifetime = 180 * 24 * time.Hour
)

var (
	once sync.Once
	std  *JWT
	// signingKey is captured once at package init (process start) and is not
	// affected by a runtime config reload — it aligns with securestore's
	// first-use capture. Rotating the key therefore requires a process restart
	// so all surfaces switch together; see docs/operations/deployment.md.
	signingKey = preferences.GetString("app.signingKey")
)

// ValidTime returns the configured sliding token/session lifetime (default 30 days).
func ValidTime() time.Duration {
	secs := preferences.GetInt64("jwtopt.validTime", int64(DefaultValidTime/time.Second))
	if secs <= 0 {
		return DefaultValidTime
	}
	return time.Duration(secs) * time.Second
}

// RefreshInterval returns the minimum gain/throttle interval between sliding refreshes
// (default 24 hours), clamped to at most half of ValidTime so short-TTL configurations
// can still slide before expiry.
func RefreshInterval() time.Duration {
	valid := ValidTime()
	secs := preferences.GetInt64("jwtopt.refreshInterval", int64(DefaultRefreshInterval/time.Second))
	interval := DefaultRefreshInterval
	if secs > 0 {
		interval = time.Duration(secs) * time.Second
	}
	if half := valid / 2; half > 0 && interval > half {
		return half
	}
	return interval
}

// MaxSessionLifetime returns the configured absolute lifetime cap for a single
// session (default 180 days). Zero disables the absolute cap.
func MaxSessionLifetime() time.Duration {
	secs := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(DefaultMaxSessionLifetime/time.Second))
	if secs < 0 {
		return DefaultMaxSessionLifetime
	}
	return time.Duration(secs) * time.Second
}

// InitialSessionLifetime returns the lifetime for a newly issued session,
// bounded by MaxSessionLifetime when enabled.
func InitialSessionLifetime() time.Duration {
	valid := ValidTime()
	if maxLife := MaxSessionLifetime(); maxLife > 0 && maxLife < valid {
		return maxLife
	}
	return valid
}

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
	return CreateNewTokenWithVersion(userId, 0, ValidTime())
}

// CreateNewTokenDefaultWithVersion creates an access token with the configured lifetime and token version.
func CreateNewTokenDefaultWithVersion(userId, tokenVersion uint64) (string, error) {
	return CreateNewTokenWithVersion(userId, tokenVersion, ValidTime())
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
		RegisteredClaims: GetBaseRegisteredClaims(InitialSessionLifetime()),
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

// RefreshTokenIfNeeded evaluates whether an already-parsed, valid token should
// slide forward. When sessionCreatedAt is non-zero and MaxSessionLifetime() > 0,
// the refreshed expiry is capped at sessionCreatedAt + MaxSessionLifetime().
// Refreshing preserves jti and purpose and does not mutate claims.
func RefreshTokenIfNeeded(tokenStr string, claims *CustomClaims, sessionCreatedAt time.Time) (newToken string, expiresAt time.Time, refreshed bool, err error) {
	nextToken, exp, didRefresh, _, err := RefreshSessionTokenIfNeeded(tokenStr, claims, sessionCreatedAt, time.Time{})
	return nextToken, exp, didRefresh, err
}

// RefreshSessionTokenIfNeeded evaluates whether a validated session token should
// slide forward, distinguishing a server-side DB extension (needsPersist=true)
// from an older client catching up to an already-persisted session expiry
// (needsPersist=false). Refreshing preserves jti and purpose and never mutates claims.
func RefreshSessionTokenIfNeeded(tokenStr string, claims *CustomClaims, sessionCreatedAt, persistedExpiresAt time.Time) (newToken string, expiresAt time.Time, refreshed bool, needsPersist bool, err error) {
	if claims == nil {
		return "", time.Time{}, false, false, ErrTokenInvalid
	}
	eTime, err := claims.GetExpirationTime()
	if err != nil || eTime == nil {
		return tokenStr, time.Time{}, false, false, err
	}
	now := time.Now()
	currentExp := eTime.Time
	valid := ValidTime()
	interval := RefreshInterval()

	targetExp := now.Add(valid)
	var hardDeadline time.Time
	if maxLife := MaxSessionLifetime(); maxLife > 0 && !sessionCreatedAt.IsZero() {
		hardDeadline = sessionCreatedAt.Add(maxLife)
		if !hardDeadline.After(now) {
			return "", time.Time{}, false, false, ErrTokenInvalid
		}
		if targetExp.After(hardDeadline) {
			targetExp = hardDeadline
		}
	}
	// JWT NumericDate has second precision; truncate before comparing so
	// sub-second differences at hardDeadline never trigger repeated refreshes.
	targetExp = targetExp.Truncate(time.Second)
	minFinalGain := min(time.Minute, interval/2)
	if minFinalGain < time.Second {
		minFinalGain = time.Second
	}

	persistedExp := currentExp
	if !persistedExpiresAt.IsZero() {
		persistedExp = persistedExpiresAt.Truncate(time.Second)
		if !hardDeadline.IsZero() && persistedExp.After(hardDeadline) {
			persistedExp = hardDeadline.Truncate(time.Second)
		}
	}

	dbGain := targetExp.Sub(persistedExp)
	needDBExtend := dbGain >= interval || (persistedExp.Sub(now) <= interval && dbGain >= minFinalGain)

	desiredExp := targetExp
	if needDBExtend {
		needsPersist = true
	} else {
		// Persisted session expiry is already within the throttle window: only
		// issue a catch-up JWT when this client's token lags behind the DB row.
		if persistedExp.Sub(currentExp) < minFinalGain {
			return tokenStr, currentExp, false, false, nil
		}
		desiredExp = persistedExp
		needsPersist = false
	}
	if !desiredExp.After(now) {
		return tokenStr, currentExp, false, false, nil
	}
	nextClaims := *claims
	nextClaims.RegisteredClaims = jwt.RegisteredClaims{
		NotBefore: jwt.NewNumericDate(now.Add(-10 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(desiredExp),
		Issuer:    "GooseForum",
	}
	signed, err := Std().CreateToken(nextClaims)
	if err != nil {
		return "", time.Time{}, false, false, err
	}
	return signed, desiredExp, true, needsPersist, nil
}

// VerifyTokenWithFresh verifies tokenStr and refreshes it when it is eligible to slide.
func VerifyTokenWithFresh(tokenStr string) (userId uint64, newToken string, err error) {
	claims, newToken, err := VerifyTokenWithFreshClaims(tokenStr)
	if err != nil {
		return 0, "", err
	}
	return claims.UserId, newToken, nil
}

// VerifyTokenWithFreshClaims verifies tokenStr and returns claims, sliding tokens
// once at least RefreshInterval() has elapsed relative to ValidTime().
// Refreshing preserves jti and purpose so the underlying session record stays stable.
func VerifyTokenWithFreshClaims(tokenStr string) (*CustomClaims, string, error) {
	claims, err := Std().ParseToken(tokenStr)
	if err != nil {
		return nil, "", err
	}
	nextToken, expiresAt, refreshed, err := RefreshTokenIfNeeded(tokenStr, claims, time.Time{})
	if err != nil {
		return nil, "", err
	}
	if refreshed {
		claims.RegisteredClaims = jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Now().Add(-10 * time.Second)),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "GooseForum",
		}
	}
	return claims, nextToken, nil
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
	TokenSettingWithMaxAge(c, newToken, InitialSessionLifetime())
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
