package totpservice

import (
	"encoding/base32"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotp"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpChallenges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpRecoveryCodes"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

func setupTotpTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&userTotp.Entity{}, &userTotpRecoveryCodes.Entity{}, &userTotpChallenges.Entity{}, &users.EntityComplete{}); err != nil {
		t.Fatalf("migrate totp tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userTotpChallenges.Entity{})
	conn.Where("1 = 1").Delete(&userTotpRecoveryCodes.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func currentCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCodeCustom(secret, time.Now().UTC(), validateOpts())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	return code
}

// TestRFC6238SHA1Vectors 使用 RFC 6238 附录 B 官方向量验证 SHA1 生成，
// 覆盖多个时间点；8 位向量为官方值，6 位向量为同 counter 截断值。
func TestRFC6238SHA1Vectors(t *testing.T) {
	secret := base32.StdEncoding.EncodeToString([]byte("12345678901234567890"))
	cases := []struct {
		ts        int64
		wantEight string
		wantSix   string
	}{
		{59, "94287082", "287082"},
		{1111111109, "07081804", "081804"},
		{1234567890, "89005924", "005924"},
		{2000000000, "69279037", "279037"},
	}
	for _, tc := range cases {
		at := time.Unix(tc.ts, 0).UTC()
		eight, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{Period: 30, Digits: otp.DigitsEight, Algorithm: otp.AlgorithmSHA1})
		if err != nil {
			t.Fatalf("ts=%d generate 8-digit: %v", tc.ts, err)
		}
		if eight != tc.wantEight {
			t.Errorf("ts=%d 8-digit = %s, want %s", tc.ts, eight, tc.wantEight)
		}
		six, err := totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1})
		if err != nil {
			t.Fatalf("ts=%d generate 6-digit: %v", tc.ts, err)
		}
		if six != tc.wantSix {
			t.Errorf("ts=%d 6-digit = %s, want %s", tc.ts, six, tc.wantSix)
		}
	}
}

func TestSetupReturnsSecretAndOtpauthURL(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7001)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if len(result.Secret) != 32 {
		t.Fatalf("secret length = %d, want 32", len(result.Secret))
	}
	if !strings.HasPrefix(result.OtpauthURL, "otpauth://totp/") ||
		!strings.Contains(result.OtpauthURL, "secret="+result.Secret) ||
		!strings.Contains(result.OtpauthURL, "algorithm=SHA1&digits=6&period=30") {
		t.Fatalf("otpauth URL malformed: %s", result.OtpauthURL)
	}
	if IsEnabled(userID) {
		t.Fatal("IsEnabled() = true before enable")
	}

	entity := userTotp.GetByUserID(userID)
	if entity == nil || entity.SecretEncrypted == "" || entity.SecretEncrypted == result.Secret {
		t.Fatalf("stored secret not encrypted: %#v", entity)
	}
}

func TestEnablePersistsAndReturnsRecoveryCodes(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7002)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	codes, err := Enable(userID, currentCode(t, result.Secret))
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("recovery codes count = %d, want 10", len(codes))
	}
	pattern := regexp.MustCompile(`^[A-HJ-NP-Z2-9]{4}-[A-HJ-NP-Z2-9]{4}$`)
	seen := map[string]bool{}
	for _, code := range codes {
		if !pattern.MatchString(code) {
			t.Fatalf("recovery code %q does not match XXXX-XXXX alphabet", code)
		}
		if seen[code] {
			t.Fatalf("duplicate recovery code %q", code)
		}
		seen[code] = true
	}
	if !IsEnabled(userID) {
		t.Fatal("IsEnabled() = false after enable")
	}
}

func TestEnableRejectsWrongCodeAndAlreadyEnabled(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7003)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err = Enable(userID, "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Enable(wrong code) error = %v, want ErrInvalidCode", err)
	}
	if IsEnabled(userID) {
		t.Fatal("IsEnabled() = true after failed enable")
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); !errors.Is(err, ErrAlreadyEnabled) {
		t.Fatalf("Enable(twice) error = %v, want ErrAlreadyEnabled", err)
	}
}

func TestVerifyTotpCode(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7004)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	ok, err := Verify(userID, currentCode(t, result.Secret))
	if err != nil || !ok {
		t.Fatalf("Verify(valid code) = %v, %v; want true, nil", ok, err)
	}
	ok, err = Verify(userID, "000000")
	if err == nil || ok {
		t.Fatalf("Verify(wrong code) = %v, %v; want false, ErrInvalidCode", ok, err)
	}
}

func TestVerifyRecoveryCodeSingleUse(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7005)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	codes, err := Enable(userID, currentCode(t, result.Secret))
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	ok, err := Verify(userID, codes[0])
	if err != nil || !ok {
		t.Fatalf("Verify(recovery code) = %v, %v; want true, nil", ok, err)
	}
	// 同一恢复码第二次使用必须失败（一次性）。
	ok, err = Verify(userID, codes[0])
	if err == nil || ok {
		t.Fatalf("Verify(reused recovery code) = %v, %v; want false, error", ok, err)
	}
	// 未使用的恢复码仍然有效。
	ok, err = Verify(userID, codes[1])
	if err != nil || !ok {
		t.Fatalf("Verify(second recovery code) = %v, %v; want true, nil", ok, err)
	}
	// 无连字符形式也应匹配同一哈希。
	if err = userTotpRecoveryCodes.DeleteByUserID(userID); err != nil {
		t.Fatalf("cleanup recovery codes: %v", err)
	}
	if err = userTotpRecoveryCodes.Create(&userTotpRecoveryCodes.Entity{UserId: userID, CodeHash: hashRecoveryCode("ABCD-EFGH")}); err != nil {
		t.Fatalf("seed recovery code: %v", err)
	}
	ok, err = Verify(userID, "ABCDEFGH")
	if err != nil || !ok {
		t.Fatalf("Verify(no-dash recovery code) = %v, %v; want true, nil", ok, err)
	}
}

func TestVerifyFailsWhenNotEnabled(t *testing.T) {
	setupTotpTestDB(t)
	_, err := Verify(7101, "123456")
	if !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("Verify(not enabled) error = %v, want ErrNotEnabled", err)
	}
}

func TestDisableWithTotpCode(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7006)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if err = Disable(userID, "000000"); !errors.Is(err, ErrInvalidCode) {
		t.Fatalf("Disable(wrong code) error = %v, want ErrInvalidCode", err)
	}
	if err = Disable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	if IsEnabled(userID) {
		t.Fatal("IsEnabled() = true after disable")
	}
	if err = Disable(userID, currentCode(t, result.Secret)); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("Disable(twice) error = %v, want ErrNotEnabled", err)
	}
}

func TestDisableWithPassword(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7007)
	password := "correct-horse-123"
	user := users.MakeUser("totp-disable-user", password, "totp-disable@example.com")
	user.Id = userID
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	if err = Disable(userID, password); err != nil {
		t.Fatalf("Disable(password) error = %v", err)
	}
	if IsEnabled(userID) {
		t.Fatal("IsEnabled() = true after disable with password")
	}
}

func TestVerifyRateLimit(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7008)

	result, err := Setup(userID)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err = Enable(userID, currentCode(t, result.Secret)); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	for range 10 {
		if _, err = Verify(userID, "000000"); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("Verify(wrong code) error = %v, want ErrInvalidCode", err)
		}
	}
	if _, err = Verify(userID, "000000"); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("Verify(over limit) error = %v, want ErrRateLimited", err)
	}
	// 正确 TOTP 码在限流窗口内同样被拒绝。
	if _, err = Verify(userID, currentCode(t, result.Secret)); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("Verify(valid code while limited) error = %v, want ErrRateLimited", err)
	}
}

func TestChallengeValidRejectsExpiredDBRow(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7201)
	jti := "expired-row-jti-0001"
	if err := SaveChallenge(userID, jti, -1*time.Minute); err != nil {
		t.Fatalf("SaveChallenge() error = %v", err)
	}
	// JWT 本身未过期，但 DB 记录已过期，中间件的 ChallengeValid 必须拒绝。
	if ChallengeValid(userID, jti) {
		t.Fatal("ChallengeValid() = true for expired DB row")
	}
	// CleanupExpiredChallenges 应删除过期行。
	CleanupExpiredChallenges()
	if entity := userTotpChallenges.GetByUserIDAndJti(userID, jti); entity != nil {
		t.Fatalf("expired challenge row not cleaned up: %#v", entity)
	}
}

func TestConsumeChallengeIsSingleUse(t *testing.T) {
	setupTotpTestDB(t)
	userID := uint64(7202)
	jti := "single-use-jti-0001"
	if err := SaveChallenge(userID, jti, 5*time.Minute); err != nil {
		t.Fatalf("SaveChallenge() error = %v", err)
	}
	if !ChallengeValid(userID, jti) {
		t.Fatal("ChallengeValid() = false for fresh challenge")
	}
	if !ConsumeChallenge(userID, jti) {
		t.Fatal("ConsumeChallenge() = false on first consume")
	}
	if ConsumeChallenge(userID, jti) {
		t.Fatal("ConsumeChallenge() = true on second consume; token must be single-use")
	}
	if ChallengeValid(userID, jti) {
		t.Fatal("ChallengeValid() = true after consumption")
	}
}
