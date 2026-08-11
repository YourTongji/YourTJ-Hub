package setting

import (
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

const on = "on"
const off = "off"

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

// CookieSecure reports whether session cookies should carry the Secure flag.
//
// Fail-closed by environment, not by site URL scheme: any environment other
// than "local" forces Secure regardless of `server.url`. This closes the
// CWE-614 gap where template defaults `app.env = "production"` with
// `server.url = "http://localhost"` dropped the Secure flag on `access_token`
// and goth session cookies. Local keeps Secure off so 0.0.0.0 / LAN-IP dev
// access over plain http keeps working (browsers accept Secure on localhost).
func CookieSecure() bool {
	return !IsLocal()
}
