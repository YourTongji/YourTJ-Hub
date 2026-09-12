package pkservice

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pk"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// envOnesystemCookie 一系统本科 Cookie 环境变量名（运维 cron 用）。
const envOnesystemCookie = "ONESYSTEM_COOKIE"

const (
	envOnesystemUndergraduateCookie = "ONESYSTEM_UNDERGRADUATE_COOKIE"
	envOnesystemGraduateXToken      = "ONESYSTEM_GRADUATE_X_TOKEN"
	envOnesystemXToken              = "ONESYSTEM_X_TOKEN"
	envOnesystemGraduateCookie      = "ONESYSTEM_GRADUATE_COOKIE" // 旧版兼容
)

// Audience 是一系统登录受众，也是 PK 数据域的隔离维度。
type Audience = pk.Audience

const (
	AudienceUndergraduate = pk.AudienceUndergraduate
	AudienceGraduate      = pk.AudienceGraduate
)

// ParseAudience 将外部受众参数规范化；空值保持历史本科默认行为。
func ParseAudience(value string) (Audience, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "undergraduate", "ug", "本科":
		return AudienceUndergraduate, nil
	case "graduate", "grad", "研究生":
		return AudienceGraduate, nil
	default:
		return "", fmt.Errorf("不支持的同步数据来源 %q：请使用 undergraduate 或 graduate", value)
	}
}

// ResolveCookie 按优先级解析本科一系统 Cookie header。
//  1. --onesystem-cookie 参数（显式覆盖，运维临时用）
//  2. ONESYSTEM_COOKIE 环境变量（运维 cron）
//  3. 管理端设置（securestore 加密落库，读取时解密）
//
// 不落库明文；参数/环境变量不经数据库。
func ResolveCookie(flagValue string) (string, error) {
	return ResolveCredentialForAudience(flagValue, AudienceUndergraduate)
}

// ResolveCookieForAudience 是历史兼容名称；研究生调用方得到的是 X-Token。
func ResolveCookieForAudience(flagValue string, audience Audience) (string, error) {
	return ResolveCredentialForAudience(flagValue, audience)
}

// ResolveCredentialForAudience 按受众解析一系统凭证。研究生凭证是
// EnquiryOfCourses 使用的 sessionStorage.sessionid（请求头 X-Token），不是本科
// manualArrange 使用的 Cookie header；保留旧环境变量/落库字段以兼容已有部署。
func ResolveCredentialForAudience(flagValue string, audience Audience) (string, error) {
	if !audience.Valid() {
		return "", fmt.Errorf("无效的一系统数据来源 %q", audience)
	}
	if v := strings.TrimSpace(flagValue); v != "" {
		return v, nil
	}

	if audience == AudienceGraduate {
		for _, envName := range []string{envOnesystemGraduateXToken, envOnesystemXToken, envOnesystemGraduateCookie} {
			if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
				return v, nil
			}
		}
		cfg := hotdataserve.GetOnesystemSettingsConfigCache()
		v, err := decryptCredential(cfg.GraduateXTokenEncrypted, securestore.OneSystemXTokenPurpose)
		if err != nil {
			return "", errors.New("解密管理端保存的研究生 X-Token 失败（app.signingKey 可能已轮换，请到管理端重新保存）")
		}
		if v != "" {
			return v, nil
		}
		v, err = decryptCredential(cfg.GraduateCookieEncrypted, securestore.OneSystemCookiePurpose)
		if err != nil {
			return "", errors.New("解密管理端保存的研究生旧版凭证失败（app.signingKey 可能已轮换，请到管理端重新保存）")
		}
		if v != "" {
			return v, nil
		}
		return "", fmt.Errorf("缺少研究生一系统 X-Token：请通过 --onesystem-x-token 参数、%s 或 %s 环境变量，或管理端「一系统同步」设置提供", envOnesystemGraduateXToken, envOnesystemXToken)
	}

	if v := strings.TrimSpace(os.Getenv(envOnesystemUndergraduateCookie)); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv(envOnesystemCookie)); v != "" {
		return v, nil
	}
	cfg := hotdataserve.GetOnesystemSettingsConfigCache()
	v, err := decryptCredential(cfg.CookieEncrypted, securestore.OneSystemCookiePurpose)
	if err != nil {
		return "", errors.New("解密管理端保存的一系统 Cookie 失败（app.signingKey 可能已轮换，请到管理端重新保存）")
	}
	if v != "" {
		return v, nil
	}
	return "", fmt.Errorf("缺少本科一系统 Cookie：请通过 --onesystem-cookie 参数、%s 环境变量或管理端「一系统同步」设置提供", envOnesystemUndergraduateCookie)
}

func decryptCredential(encrypted, purpose string) (string, error) {
	if strings.TrimSpace(encrypted) == "" {
		return "", nil
	}
	plain, err := securestore.DecryptPurpose(encrypted, purpose)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(plain), nil
}

func audienceLabel(audience Audience) string {
	if audience == AudienceGraduate {
		return "研究生"
	}
	return "本科"
}
