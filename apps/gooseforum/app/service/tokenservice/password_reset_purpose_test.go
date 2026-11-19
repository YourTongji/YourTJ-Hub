package tokenservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/golang-jwt/jwt/v5"
)

func TestPasswordResetRejectsOtherTokenPurposes(t *testing.T) {
	withAppSigningKey(t, "reset-purpose-test-key")
	session := jwtopt.CustomClaims{UserId: 12, TokenVersion: 3, Jti: "test-session", RegisteredClaims: jwtopt.GetBaseRegisteredClaims(time.Hour)}
	for _, purpose := range []string{"", jwtopt.PurposeTotpChallenge} {
		session.Purpose = purpose
		token, err := jwtopt.NewJWT([]byte("reset-purpose-test-key")).CreateToken(session)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ParsePasswordResetToken(token); err == nil {
			t.Errorf("accepted session/challenge purpose %q", purpose)
		}
	}
	activation, err := GenerateActivationToken(12, "reset@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePasswordResetToken(activation); err == nil {
		t.Error("accepted activation token")
	}
}

func TestPasswordResetRequiresAccountEmail(t *testing.T) {
	withAppSigningKey(t, "reset-purpose-test-key")
	for _, tc := range []struct {
		id    uint64
		email string
	}{{12, ""}, {12, "  "}, {0, "reset@example.com"}} {
		token, err := GeneratePasswordResetToken(tc.id, tc.email, 3)
		if err != nil {
			t.Fatalf("GeneratePasswordResetToken(user=%d, email=%q): %v", tc.id, tc.email, err)
		}
		if _, err := ParsePasswordResetToken(token); err == nil {
			t.Errorf("accepted reset for user %d email %q", tc.id, tc.email)
		}
	}
}

func TestPasswordResetRejectsMissingExpiryAndWrongAudience(t *testing.T) {
	withAppSigningKey(t, "reset-purpose-test-key")
	key, err := passwordResetKey()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name         string
		audience     string
		expires      *jwt.NumericDate
		method       jwt.SigningMethod
		omitAudience bool
	}{
		{"missing expiry", PasswordResetAudience, nil, jwt.SigningMethodHS256, false},
		{"expired", PasswordResetAudience, jwt.NewNumericDate(time.Now().Add(-time.Hour)), jwt.SigningMethodHS256, false},
		{"missing audience", "", jwt.NewNumericDate(time.Now().Add(time.Hour)), jwt.SigningMethodHS256, true},
		{"wrong audience", "activation", jwt.NewNumericDate(time.Now().Add(time.Hour)), jwt.SigningMethodHS256, false},
		{"wrong algorithm", PasswordResetAudience, jwt.NewNumericDate(time.Now().Add(time.Hour)), jwt.SigningMethodHS384, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registered := jwt.RegisteredClaims{ExpiresAt: tc.expires}
			if !tc.omitAudience {
				registered.Audience = jwt.ClaimStrings{tc.audience}
			}
			claims := PasswordResetClaims{UserId: 12, Email: "test@example.com", RegisteredClaims: registered}
			token, err := jwt.NewWithClaims(tc.method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParsePasswordResetToken(token); err == nil {
				t.Fatal("accepted invalid reset")
			}
		})
	}
	reset, err := GeneratePasswordResetToken(12, "test@example.com", 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseActivationToken(reset); err == nil {
		t.Fatal("activation accepted reset token")
	}
	if _, err := jwtopt.NewJWT([]byte("reset-purpose-test-key")).ParseToken(reset); err == nil {
		t.Fatal("session accepted reset token")
	}
}
