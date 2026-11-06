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
			Audience:  jwt.ClaimStrings{"password-reset"},
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
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithAudience("password-reset"))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*PasswordResetClaims); ok && token.Valid && claims.UserId != 0 && strings.TrimSpace(claims.Email) != "" {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// Derive a dedicated key so reset links cannot be consumed as session or activation tokens.
func passwordResetKey() ([]byte, error) {
	key, err := signingKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte("yourtj-password-reset-token"))
	return mac.Sum(nil), nil
}
