package tokenservice

import (
	"errors"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/golang-jwt/jwt/v5"
)

func TestActivationRejectsSessionAndChallengeTokens(t *testing.T) {
	const key = "activation-purpose-test-key-1619"
	withAppSigningKey(t, key)
	for _, purpose := range []string{"", "totp_challenge"} {
		t.Run("purpose="+purpose, func(t *testing.T) {
			token, err := jwtopt.NewJWT([]byte(key)).CreateToken(jwtopt.CustomClaims{
				UserId:           42,
				Purpose:          purpose,
				RegisteredClaims: jwtopt.GetBaseRegisteredClaims(time.Hour),
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseActivationToken(token); err == nil {
				t.Fatal("non-activation token accepted for email activation")
			}
		})
	}
}

func TestActivationPurposeAndClaimValidation(t *testing.T) {
	const rawKey = "activation-purpose-test-key-1619"
	withAppSigningKey(t, rawKey)
	key, err := activationKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		change  func(*ActivationClaims)
		method  jwt.SigningMethod
		raw     bool
		expired bool
	}{
		{name: "missing audience", change: func(c *ActivationClaims) { c.Audience = nil }},
		{name: "wrong audience", change: func(c *ActivationClaims) { c.Audience = jwt.ClaimStrings{"password-reset"} }},
		{name: "missing expiry", change: func(c *ActivationClaims) { c.ExpiresAt = nil }},
		{name: "expired", expired: true, change: func(c *ActivationClaims) { c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Hour)) }},
		{name: "not yet valid", change: func(c *ActivationClaims) { c.NotBefore = jwt.NewNumericDate(time.Now().Add(time.Hour)) }},
		{name: "zero user", change: func(c *ActivationClaims) { c.UserId = 0 }},
		{name: "empty email", change: func(c *ActivationClaims) { c.Email = "" }},
		{name: "blank email", change: func(c *ActivationClaims) { c.Email = " \t\n" }},
		{name: "wrong algorithm", method: jwt.SigningMethodHS384},
		{name: "shared key", raw: true},
		{name: "legacy activation", raw: true, change: func(c *ActivationClaims) { c.Audience = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := ActivationClaims{UserId: 42, Email: "owner@example.test", RegisteredClaims: jwt.RegisteredClaims{
				Audience: jwt.ClaimStrings{activationAudience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			}}
			if tc.change != nil {
				tc.change(&claims)
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			signing := key
			if tc.raw {
				signing = []byte(rawKey)
			}
			token, err := jwt.NewWithClaims(method, claims).SignedString(signing)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseActivationToken(token); err == nil || tc.expired && !errors.Is(err, jwt.ErrTokenExpired) {
				t.Fatalf("expected rejection (expired=%v), got %v", tc.expired, err)
			}
		})
	}
	for _, c := range []ActivationClaims{{UserId: 0, Email: "owner@example.test"}, {UserId: 42}, {UserId: 42, Email: " \t"}} {
		if _, err := GenerateActivationToken(c.UserId, c.Email); err == nil {
			t.Fatal("issued token with incomplete identity")
		}
	}
}

func TestActivationTokensAreIsolatedFromOtherPurposes(t *testing.T) {
	const rawKey = "activation-isolation-test-key-1619"
	withAppSigningKey(t, rawKey)
	activation, err := GenerateActivationToken(42, "owner@example.test")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseActivationToken(activation)
	if err != nil || len(parsed.Audience) != 1 || parsed.Audience[0] != activationAudience {
		t.Fatalf("activation audience missing: %v", err)
	}
	reset, err := GeneratePasswordResetToken(42, "owner@example.test", 0)
	if err != nil {
		t.Fatal(err)
	}
	moderation, err := IssueModerationAction(ModerationActionClaims{Subject: ModerationActionSubjectReport, ID: 42, Action: "dismiss"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{reset, moderation} {
		if _, err := ParseActivationToken(token); err == nil {
			t.Fatal("foreign purpose accepted as activation")
		}
	}
	if _, err := ParsePasswordResetToken(activation); err == nil {
		t.Fatal("activation accepted as password reset")
	}
	if _, err := ParseModerationAction(activation, time.Now()); err == nil {
		t.Fatal("activation accepted as moderation action")
	}
	if _, err := jwtopt.NewJWT([]byte(rawKey)).ParseToken(activation); err == nil {
		t.Fatal("activation accepted as session/challenge")
	}
	withAppSigningKey(t, "rotated-activation-test-key-1619")
	if _, err := ParseActivationToken(activation); err == nil {
		t.Fatal("old activation survived signing-key rotation")
	}
}
