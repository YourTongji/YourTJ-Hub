package tokenservice

import (
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func withAppSigningKey(t *testing.T, key string) {
	t.Helper()
	old := preferences.GetString("app.signingKey", "")
	preferences.Set("app.signingKey", key)
	t.Cleanup(func() {
		preferences.Set("app.signingKey", old)
	})
}

func TestActivationTokenLifecycle(t *testing.T) {
	withAppSigningKey(t, "activation-test-key")

	token, err := GenerateActivationToken(12, "user@example.com")
	if err != nil {
		t.Fatalf("GenerateActivationToken failed: %v", err)
	}

	claims, err := ParseActivationToken(token)
	if err != nil {
		t.Fatalf("ParseActivationToken failed: %v", err)
	}
	if claims.UserId != 12 {
		t.Fatalf("UserId = %d, want 12", claims.UserId)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("Email = %q, want user@example.com", claims.Email)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 23*time.Hour {
		t.Fatalf("activation token expiry should be close to 24h, got %v", claims.ExpiresAt)
	}
}

func TestGenerateActivationTokenByUser(t *testing.T) {
	withAppSigningKey(t, "activation-user-key")

	token, err := GenerateActivationTokenByUser(users.EntityComplete{
		Id:    99,
		Email: "entity@example.com",
	})
	if err != nil {
		t.Fatalf("GenerateActivationTokenByUser failed: %v", err)
	}

	claims, err := ParseActivationToken(token)
	if err != nil {
		t.Fatalf("ParseActivationToken failed: %v", err)
	}
	if claims.UserId != 99 || claims.Email != "entity@example.com" {
		t.Fatalf("claims = {%d %q}, want entity user", claims.UserId, claims.Email)
	}
}

func TestPasswordResetTokenLifecycle(t *testing.T) {
	withAppSigningKey(t, "password-reset-test-key")

	const tokenVersion uint64 = 7
	token, err := GeneratePasswordResetToken(34, "reset@example.com", tokenVersion)
	if err != nil {
		t.Fatalf("GeneratePasswordResetToken failed: %v", err)
	}

	claims, err := ParsePasswordResetToken(token)
	if err != nil {
		t.Fatalf("ParsePasswordResetToken failed: %v", err)
	}
	if claims.UserId != 34 {
		t.Fatalf("UserId = %d, want 34", claims.UserId)
	}
	if claims.Email != "reset@example.com" {
		t.Fatalf("Email = %q, want reset@example.com", claims.Email)
	}
	if claims.TokenVersion != tokenVersion {
		t.Fatalf("TokenVersion = %d, want %d", claims.TokenVersion, tokenVersion)
	}
	if claims.ExpiresAt == nil || time.Until(claims.ExpiresAt.Time) <= 29*time.Minute {
		t.Fatalf("password reset token expiry should be close to 30m, got %v", claims.ExpiresAt)
	}
}

func TestTokenParsingRejectsInvalidInput(t *testing.T) {
	withAppSigningKey(t, "reject-test-key")

	if _, err := ParseActivationToken("not-a-token"); err == nil {
		t.Fatalf("expected invalid activation token error")
	}
	if _, err := ParsePasswordResetToken("not-a-token"); err == nil {
		t.Fatalf("expected invalid password reset token error")
	}
}

// TestGenerateRejectsWeakSigningKey mirrors issue #106 point 1 at the
// tokenservice layer: reset/activation tokens must never be signed with an
// empty or publicly-known signing key, even if the serve startup guard were
// bypassed (e.g. by a test harness). Generate and Parse must fail closed.
func TestGenerateRejectsWeakSigningKey(t *testing.T) {
	weakKeys := []string{
		"",
		"   ",
		"mq+ZeGafL+b1xdC0u9vSVg==", // jwtopt.DefaultSigningKey
		"REPLACE_SIGNING_KEY",      // deploy template placeholder
	}
	for _, key := range weakKeys {
		withAppSigningKey(t, key)
		if _, err := GeneratePasswordResetToken(1, "x@example.com", 0); err == nil {
			t.Fatalf("GeneratePasswordResetToken with key %q must fail closed", key)
		}
		if _, err := GenerateActivationToken(1, "x@example.com"); err == nil {
			t.Fatalf("GenerateActivationToken with key %q must fail closed", key)
		}
		if _, err := ParsePasswordResetToken("any"); err == nil {
			t.Fatalf("ParsePasswordResetToken with key %q must fail closed", key)
		}
		if _, err := ParseActivationToken("any"); err == nil {
			t.Fatalf("ParseActivationToken with key %q must fail closed", key)
		}
	}
}
