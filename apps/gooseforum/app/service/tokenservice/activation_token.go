// Package tokenservice creates and parses short-lived account tokens.
package tokenservice

import (
	"crypto/hmac"
	"crypto/sha256"
	"strings"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"

	"github.com/golang-jwt/jwt/v5"
)

const activationAudience = "email-activation"
const activationKeyPurpose = "yourtj-email-activation-token"

// ActivationClaims is the JWT payload used for email activation.
type ActivationClaims struct {
	UserId uint64 `json:"userId"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateActivationTokenByUser creates an activation token from a user entity.
func GenerateActivationTokenByUser(entity users.EntityComplete) (string, error) {
	return GenerateActivationToken(entity.Id, entity.Email)
}

// GenerateActivationToken creates a signed email activation token.
func GenerateActivationToken(userId uint64, email string) (string, error) {
	if userId == 0 || strings.TrimSpace(email) == "" {
		return "", jwt.ErrTokenInvalidClaims
	}
	key, err := activationKey()
	if err != nil {
		return "", err
	}
	claims := ActivationClaims{
		UserId: userId,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Audience:  jwt.ClaimStrings{activationAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

// ParseActivationToken accepts only purpose-bound activation links. The handler
// must still match the target against the current or fresh pending email.
func ParseActivationToken(tokenString string) (*ActivationClaims, error) {
	key, err := activationKey()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(tokenString, &ActivationClaims{}, func(token *jwt.Token) (any, error) {
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(), jwt.WithAudience(activationAudience))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*ActivationClaims); ok && token.Valid {
		if claims.UserId == 0 || strings.TrimSpace(claims.Email) == "" {
			return nil, jwt.ErrTokenInvalidClaims
		}
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// activationKey separates email proof from session, challenge and recovery
// credentials even when their JSON fields overlap. No legacy-key fallback.
func activationKey() ([]byte, error) {
	key, err := signingKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(activationKeyPurpose))
	return mac.Sum(nil), nil
}
