package tokenservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// PasswordResetClaims is the JWT payload used for password reset links.
// TokenVersion binds the link to the user's current token_version at issue
// time: any password change / revocation bumps token_version, so an old reset
// link cannot be replayed after the account has been reset or recovered
// (issue #106). Email remains as the secondary ownership claim.
type PasswordResetClaims struct {
	UserId       uint64 `json:"userId"`
	Email        string `json:"email"`
	TokenVersion uint64 `json:"tokenVersion"`
	jwt.RegisteredClaims
}

// GeneratePasswordResetToken creates a signed password reset token bound to
// the user's current tokenVersion. Callers must pass users.EntityComplete.TokenVersion
// captured at issue time.
func GeneratePasswordResetToken(userId uint64, email string, tokenVersion uint64) (string, error) {
	key, err := passwordResetKey()
	if err != nil {
		return "", err
	}
	claims := PasswordResetClaims{
		UserId:       userId,
		Email:        email,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{PasswordResetAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ParsePasswordResetToken parses and validates a password reset token.
// Callers must re-check claims.TokenVersion against the user's current
// token_version; this function only verifies signature + structure.
func ParsePasswordResetToken(tokenString string) (*PasswordResetClaims, error) {
	key, err := passwordResetKey()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &PasswordResetClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithAudience(PasswordResetAudience))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*PasswordResetClaims); ok && token.Valid && claims.UserId != 0 && strings.TrimSpace(claims.Email) != "" {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// passwordResetPurpose 派生签名密钥的用途标签：与会话/激活/审核操作 JWT 隔离，
// 任何一方的 token 都不能被另一方接受。
const passwordResetPurpose = "yourtj-password-reset-token"

// PasswordResetAudience 是重置链接 JWT 必需的 aud 声明，把重置令牌与会话/激活
// 令牌在用途层面隔离；解析时缺失或不匹配都会失败关闭。
const PasswordResetAudience = "password-reset"

// DerivePasswordResetKey derives the dedicated password-reset signing key from a
// base key, so reset links cannot be consumed as session or activation tokens.
// Production passes the app signing key; tests pass an explicit base to exercise
// signature-failure paths without duplicating the derivation.
func DerivePasswordResetKey(base []byte) []byte {
	mac := hmac.New(sha256.New, base)
	mac.Write([]byte(passwordResetPurpose))
	return mac.Sum(nil)
}

func passwordResetKey() ([]byte, error) {
	key, err := signingKey()
	if err != nil {
		return nil, err
	}
	return DerivePasswordResetKey(key), nil
}
