// Package totpservice implements optional TOTP two-factor authentication
// (RFC 6238) for the password login flow.
package totpservice

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/securestore"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotp"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpRecoveryCodes"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
	"github.com/leancodebox/GooseForum/app/models/hotdataserve"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

var (
	// ErrAlreadyEnabled 两步验证已启用。
	ErrAlreadyEnabled = errors.New("两步验证已启用")
	// ErrNotEnabled 两步验证未启用。
	ErrNotEnabled = errors.New("两步验证未启用")
	// ErrInvalidCode 两步验证码错误。
	ErrInvalidCode = errors.New("两步验证码错误")
	// ErrRateLimited 两步验证尝试过于频繁。
	ErrRateLimited = errors.New("两步验证尝试过于频繁")
)

const (
	recoveryCodeCount = 10
	recoveryCodeGroup = 4
	failWindow        = 15 * time.Minute
	failLimit         = 10
)

// recoveryCodeAlphabet 排除易混淆字符（0/O/1/I/L/U）。
const recoveryCodeAlphabet = "ABCDEFGHJKMNPQRSTVWXYZ23456789"

// SetupResult 携带生成的一次性密钥与 otpauth 链接。
type SetupResult struct {
	Secret     string
	OtpauthURL string
}

// Setup 为用户生成新的 TOTP 密钥（未启用状态），已启用时返回错误。
func Setup(userID uint64) (SetupResult, error) {
	if IsEnabled(userID) {
		return SetupResult{}, ErrAlreadyEnabled
	}
	secretBytes, err := algorithm.GenerateRandomBytes(20)
	if err != nil {
		return SetupResult{}, err
	}
	// 20 字节 base32 编码正好 32 字符、无填充。
	secret := base32.StdEncoding.EncodeToString(secretBytes)
	encrypted, err := securestore.Encrypt(secret)
	if err != nil {
		return SetupResult{}, err
	}
	entity := userTotp.GetByUserID(userID)
	if entity == nil {
		entity = &userTotp.Entity{UserId: userID, SecretEncrypted: encrypted, Enabled: 0}
		if err = userTotp.Create(entity); err != nil {
			return SetupResult{}, err
		}
	} else {
		entity.SecretEncrypted = encrypted
		entity.Enabled = 0
		if err = userTotp.Save(entity); err != nil {
			return SetupResult{}, err
		}
	}
	return SetupResult{Secret: secret, OtpauthURL: buildOtpauthURL(userID, secret)}, nil
}

// Enable 校验 TOTP 码后启用两步验证，并生成恢复码（明文只返回一次）。
func Enable(userID uint64, code string) ([]string, error) {
	if IsEnabled(userID) {
		return nil, ErrAlreadyEnabled
	}
	entity := userTotp.GetByUserID(userID)
	if entity == nil || entity.SecretEncrypted == "" {
		return nil, ErrNotEnabled
	}
	secret, err := securestore.Decrypt(entity.SecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("totpservice: 解密密钥失败: %w", err)
	}
	valid, err := totp.ValidateCustom(strings.TrimSpace(code), secret, time.Now().UTC(), validateOpts())
	if err != nil {
		// 非 6 位数字输入按无效验证码处理。
		return nil, ErrInvalidCode
	}
	if !valid {
		return nil, ErrInvalidCode
	}
	entity.Enabled = 1
	if err = userTotp.Save(entity); err != nil {
		return nil, err
	}
	codes, err := generateRecoveryCodes()
	if err != nil {
		entity.Enabled = 0
		_ = userTotp.Save(entity)
		return nil, err
	}
	for _, recoveryCode := range codes {
		if err = userTotpRecoveryCodes.Create(&userTotpRecoveryCodes.Entity{
			UserId:   userID,
			CodeHash: hashRecoveryCode(recoveryCode),
		}); err != nil {
			_ = userTotpRecoveryCodes.DeleteByUserID(userID)
			entity.Enabled = 0
			_ = userTotp.Save(entity)
			return nil, err
		}
	}
	return codes, nil
}

// Disable 校验当前 TOTP 码或登录密码后关闭两步验证并清空恢复码。
func Disable(userID uint64, codeOrPassword string) error {
	if !IsEnabled(userID) {
		return ErrNotEnabled
	}
	if !validateCodeOrPassword(userID, strings.TrimSpace(codeOrPassword)) {
		return ErrInvalidCode
	}
	entity := userTotp.GetByUserID(userID)
	if entity == nil {
		return ErrNotEnabled
	}
	entity.Enabled = 0
	if err := userTotp.Save(entity); err != nil {
		return err
	}
	if err := userTotpRecoveryCodes.DeleteByUserID(userID); err != nil {
		return err
	}
	clearFailures(userID)
	return nil
}

// Verify 校验 TOTP 码或恢复码。恢复码一次性：匹配未使用的哈希并标记已用。
// 失败进入内存滑动窗口限流（每用户 10 次/15 分钟）。
func Verify(userID uint64, code string) (bool, error) {
	if !IsEnabled(userID) {
		return false, ErrNotEnabled
	}
	if rateLimited(userID) {
		return false, ErrRateLimited
	}
	code = normalizeCode(code)
	if code == "" {
		recordFailure(userID)
		return false, ErrInvalidCode
	}
	valid, err := verifyTotp(userID, code)
	if err != nil {
		recordFailure(userID)
		return false, err
	}
	if valid {
		clearFailures(userID)
		return true, nil
	}
	if verifyRecoveryCode(userID, code) {
		clearFailures(userID)
		return true, nil
	}
	recordFailure(userID)
	return false, ErrInvalidCode
}

// IsEnabled 报告用户是否已启用两步验证。
func IsEnabled(userID uint64) bool {
	entity := userTotp.GetByUserID(userID)
	return entity != nil && entity.Enabled == 1
}

func validateOpts() totp.ValidateOpts {
	return totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
}

func verifyTotp(userID uint64, code string) (bool, error) {
	entity := userTotp.GetByUserID(userID)
	if entity == nil || entity.SecretEncrypted == "" {
		return false, nil
	}
	secret, err := securestore.Decrypt(entity.SecretEncrypted)
	if err != nil {
		return false, fmt.Errorf("totpservice: 解密密钥失败: %w", err)
	}
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), validateOpts())
	if err != nil {
		// 恢复码等非数字输入无法通过 TOTP 校验，交给恢复码分支处理。
		return false, nil
	}
	return valid, nil
}

func verifyRecoveryCode(userID uint64, code string) bool {
	entity := userTotpRecoveryCodes.GetUnusedByHash(userID, hashRecoveryCode(code))
	if entity == nil {
		return false
	}
	// 原子地标记已用：并发提交同一恢复码时只有一个成功，防止双花。
	return userTotpRecoveryCodes.MarkUsedIfUnused(entity.Id)
}

// validateCodeOrPassword 优先校验 TOTP 码，失败再尝试登录密码。
func validateCodeOrPassword(userID uint64, input string) bool {
	if input == "" {
		return false
	}
	if valid, err := verifyTotp(userID, input); err == nil && valid {
		return true
	}
	user, err := users.Get(userID)
	if err != nil {
		return false
	}
	return algorithm.VerifyEncryptPassword(user.Password, input) == nil
}

func buildOtpauthURL(userID uint64, secret string) string {
	issuer := strings.TrimSpace(hotdataserve.GetSiteSettingsConfigCache().SiteName)
	if issuer == "" {
		issuer = "yourtj"
	}
	account := accountName(userID)
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer), url.PathEscape(account), secret, url.QueryEscape(issuer))
}

func accountName(userID uint64) string {
	user, err := users.Get(userID)
	if err == nil && user.Username != "" {
		return user.Username
	}
	return fmt.Sprintf("user-%d", userID)
}

func generateRecoveryCodes() ([]string, error) {
	codes := make([]string, 0, recoveryCodeCount)
	for range recoveryCodeCount {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes = append(codes, code)
	}
	return codes, nil
}

func generateRecoveryCode() (string, error) {
	var sb strings.Builder
	sb.Grow(2*recoveryCodeGroup + 1)
	for i := range 2 * recoveryCodeGroup {
		if i == recoveryCodeGroup {
			sb.WriteByte('-')
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(recoveryCodeAlphabet))))
		if err != nil {
			return "", fmt.Errorf("totpservice: 生成恢复码失败: %w", err)
		}
		sb.WriteByte(recoveryCodeAlphabet[n.Int64()])
	}
	return sb.String(), nil
}

// normalizeCode 统一大小写并去除连字符/空白，使带不带连字符的恢复码都能匹配。
func normalizeCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	return strings.ReplaceAll(code, "-", "")
}

// hashRecoveryCode derives a per-purpose HMAC-SHA256 of the normalized code
// using a server-side pepper (securestore.Pepper). The pepper never enters the
// database, so a DB leak alone does not allow offline brute force of the
// 40-bit recovery-code space.
func hashRecoveryCode(code string) string {
	pepper, err := securestore.Pepper("yourtj-recovery-code")
	if err != nil {
		// 密钥派生失败意味着签名密钥配置异常，此时所有 TOTP 功能本就不可用；
		// 回退到裸 SHA-256 保持旧记录可校验（Pepper 只在配置正常时使用）。
		sum := sha256.Sum256([]byte(normalizeCode(code)))
		return hex.EncodeToString(sum[:])
	}
	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(normalizeCode(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// ---- 内存滑动窗口限流 ----

var (
	rateMu   sync.Mutex
	failures = make(map[uint64][]time.Time)
)

func rateLimited(userID uint64) bool {
	rateMu.Lock()
	defer rateMu.Unlock()
	cutoff := time.Now().Add(-failWindow)
	times := failures[userID]
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	failures[userID] = kept
	return len(kept) >= failLimit
}

func recordFailure(userID uint64) {
	rateMu.Lock()
	defer rateMu.Unlock()
	failures[userID] = append(failures[userID], time.Now())
}

func clearFailures(userID uint64) {
	rateMu.Lock()
	defer rateMu.Unlock()
	delete(failures, userID)
}
