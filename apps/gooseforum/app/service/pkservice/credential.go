package pkservice

import (
	"errors"
	"os"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// envOnesystemCookie 一系统 Cookie 环境变量名（运维 cron 用）。
const envOnesystemCookie = "ONESYSTEM_COOKIE"

// ResolveCookie 按优先级解析一系统 Cookie header：
//  1. --onesystem-cookie 参数（显式覆盖，运维临时用）
//  2. ONESYSTEM_COOKIE 环境变量（运维 cron）
//  3. 管理端设置（securestore 加密落库，读取时解密）
//
// 不落库明文；参数/环境变量不经数据库。
func ResolveCookie(flagValue string) (string, error) {
	if v := strings.TrimSpace(flagValue); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(os.Getenv(envOnesystemCookie)); v != "" {
		return v, nil
	}
	cfg := hotdataserve.GetOnesystemSettingsConfigCache()
	if v := strings.TrimSpace(cfg.CookieEncrypted); v != "" {
		plain, err := securestore.DecryptPurpose(v, securestore.OneSystemCookiePurpose)
		if err != nil {
			return "", errors.New("解密管理端保存的一系统 Cookie 失败（app.signingKey 可能已轮换，请到管理端重新保存）：" + err.Error())
		}
		if strings.TrimSpace(plain) != "" {
			return strings.TrimSpace(plain), nil
		}
	}
	return "", errors.New("缺少一系统 Cookie：请通过 --onesystem-cookie 参数、ONESYSTEM_COOKIE 环境变量或管理端「一系统同步」设置提供")
}
