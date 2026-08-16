package pkservice

import (
	"errors"
	"os"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
)

// 一系统凭证环境变量名（对齐 YourTJCourse-Serverless sync-onesystem-login.yml）：
// Cookie 方式（运维 cron / 管理端）与账号密码方式（SSO 登录）二选一。
const (
	envOnesystemCookie   = "ONESYSTEM_COOKIE"
	envOnesystemSno      = "ONESYSTEM_SNO"
	envOnesystemPassword = "ONESYSTEM_PASSWORD"
)

// ResolveCookie 按优先级解析一系统 Cookie header：
//  1. --onesystem-cookie 参数（显式覆盖，运维临时用）
//  2. ONESYSTEM_COOKIE 环境变量（运维 cron）
//  3. 管理端设置（securestore 加密落库，读取时解密）
//
// 前两者都不存在时，若提供了账号密码（--onesystem-sno/--onesystem-password 参数或
// ONESYSTEM_SNO/ONESYSTEM_PASSWORD 环境变量），自动执行 SSO 登录换取会话 Cookie
// （对齐 YourTJCourse-Serverless 的 Login 流程；触发「加强认证」时返回明确错误，
// 提示改用 Cookie 凭证）。
//
// 不落库明文；参数/环境变量不经数据库。
func ResolveCookie(flagValue, snoFlag, passwordFlag string) (string, error) {
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
	if sno, pwd := resolveSnoPassword(snoFlag, passwordFlag); sno != "" && pwd != "" {
		cookie, err := LoginAndGetCookie(sno, pwd)
		if err != nil {
			return "", err
		}
		return cookie, nil
	}
	return "", errors.New("缺少一系统凭证：请通过 --onesystem-cookie / ONESYSTEM_COOKIE 或 --onesystem-sno+--onesystem-password / ONESYSTEM_SNO+ONESYSTEM_PASSWORD（管理端设置仅支持 Cookie）提供")
}

// resolveSnoPassword 解析账号密码：参数优先，其次环境变量（对齐 workflow secrets 注入）。
func resolveSnoPassword(snoFlag, passwordFlag string) (sno, password string) {
	sno = strings.TrimSpace(snoFlag)
	if sno == "" {
		sno = strings.TrimSpace(os.Getenv(envOnesystemSno))
	}
	password = strings.TrimSpace(passwordFlag)
	if password == "" {
		password = strings.TrimSpace(os.Getenv(envOnesystemPassword))
	}
	return sno, password
}
