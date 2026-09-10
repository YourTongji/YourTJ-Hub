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

// envOnesystemCookie 一系统 Cookie 环境变量名（运维 cron 用）。
const envOnesystemCookie = "ONESYSTEM_COOKIE"

const (
	envOnesystemUndergraduateCookie = "ONESYSTEM_UNDERGRADUATE_COOKIE"
	envOnesystemGraduateCookie      = "ONESYSTEM_GRADUATE_COOKIE"
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

// ResolveCookie 按优先级解析一系统 Cookie header：
//  1. --onesystem-cookie 参数（显式覆盖，运维临时用）
//  2. ONESYSTEM_COOKIE 环境变量（运维 cron）
//  3. 管理端设置（securestore 加密落库，读取时解密）
//
// 不落库明文；参数/环境变量不经数据库。
func ResolveCookie(flagValue string) (string, error) {
	return ResolveCookieForAudience(flagValue, AudienceUndergraduate)
}

// ResolveCookieForAudience 按受众解析一系统 Cookie。旧的 --onesystem-cookie /
// ONESYSTEM_COOKIE 仍作为本科兼容来源；受众专用环境变量优先于旧本科来源。
func ResolveCookieForAudience(flagValue string, audience Audience) (string, error) {
	if !audience.Valid() {
		return "", fmt.Errorf("无效的一系统数据来源 %q", audience)
	}
	if v := strings.TrimSpace(flagValue); v != "" {
		return v, nil
	}
	envName := envOnesystemUndergraduateCookie
	if audience == AudienceGraduate {
		envName = envOnesystemGraduateCookie
	}
	if v := strings.TrimSpace(os.Getenv(envName)); v != "" {
		return v, nil
	}
	if audience == AudienceUndergraduate {
		if v := strings.TrimSpace(os.Getenv(envOnesystemCookie)); v != "" {
			return v, nil
		}
	}
	cfg := hotdataserve.GetOnesystemSettingsConfigCache()
	encrypted := cfg.CookieEncrypted
	if audience == AudienceGraduate {
		encrypted = cfg.GraduateCookieEncrypted
	}
	if v := strings.TrimSpace(encrypted); v != "" {
		plain, err := securestore.DecryptPurpose(v, securestore.OneSystemCookiePurpose)
		if err != nil {
			return "", errors.New("解密管理端保存的一系统 Cookie 失败（app.signingKey 可能已轮换，请到管理端重新保存）：" + err.Error())
		}
		if strings.TrimSpace(plain) != "" {
			return strings.TrimSpace(plain), nil
		}
	}
	return "", fmt.Errorf("缺少%s一系统 Cookie：请通过 --onesystem-cookie 参数、%s 环境变量或管理端「一系统同步」设置提供", audienceLabel(audience), envName)
}

func audienceLabel(audience Audience) string {
	if audience == AudienceGraduate {
		return "研究生"
	}
	return "本科"
}
