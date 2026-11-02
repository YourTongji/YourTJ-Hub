package setting

import (
	"net/url"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
)

const on = "on"
const off = "off"
const cookieSecureAuto = "auto"

// UseMigration reports whether database migrations are enabled.
func UseMigration() bool {
	openMigration := preferences.GetString("db.migration", on)
	return openMigration == on
}

// IsProduction reports whether the application runs in production mode.
func IsProduction() bool {
	return preferences.Get("app.env", "production") == "production"
}

// IsLocal reports whether the application runs in local development mode.
func IsLocal() bool {
	return preferences.Get("app.env", "production") == "local"
}

// IsDebug reports whether debug mode is enabled.
func IsDebug() bool {
	if preferences.IsSet("app.debug") {
		return preferences.GetBool("app.debug")
	}
	return IsLocal()
}

// GetCDNURL returns the configured CDN base URL.
func GetCDNURL() string {
	return preferences.GetString("app.cdn_url", "")
}

// URL joins path to the configured site URL.
func URL(path string) string {
	return preferences.Get("server.url") + path
}

// IsLoopbackHost reports whether host is a loopback host name or address.
// An empty host is not loopback; callers decide what a missing host means.
func IsLoopbackHost(host string) bool {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// CookieSecure reports whether session cookies should carry the Secure flag.
//
// `server.cookieSecure` overrides the resolution: "on" forces Secure, "off"
// disables it, and "auto" (the default, also used for unrecognized values)
// derives it from `server.url`. Auto keeps the CWE-614 bottom line — anything
// it cannot resolve to a plain-http non-loopback origin stays Secure — but such
// an origin drops the flag so browsers still send the cookie and web login
// works on LAN/IP deployments (issue #1277). Loopback over http stays Secure
// (#113), and local keeps it off so 0.0.0.0 / LAN-IP dev access works.
func CookieSecure() bool {
	switch strings.ToLower(strings.TrimSpace(preferences.GetString("server.cookieSecure", cookieSecureAuto))) {
	case on:
		return true
	case off:
		return false
	}
	if IsLocal() {
		return false
	}
	u, err := url.Parse(strings.TrimSpace(preferences.GetString("server.url", "")))
	if err != nil || u.Hostname() == "" { // unparsable or host-less URL: fail closed
		return true
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return true
	case "http":
		return IsLoopbackHost(u.Hostname())
	default: // no scheme (e.g. //example.com) or non-http(s): fail closed
		return true
	}
}
