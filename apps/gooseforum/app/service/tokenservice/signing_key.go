package tokenservice

import (
	"errors"

	"github.com/leancodebox/GooseForum/app/bundles/jwtopt"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
)

// signingKey returns the HMAC key for reset/activation tokens. It is fail-closed:
// serve already refuses to boot on a weak app.signingKey, so an empty/known-bad
// value here signals a test or misconfiguration rather than a runnable deploy —
// we return an error instead of silently signing with a forgeable key (issue #106).
//
// Unlike jwtopt (captured at process start) and securestore (captured on first
// use), the key is read on every call so the fail-closed policy also covers a
// runtime-edited weak value. Rotation still requires a process restart so all
// surfaces switch together — see docs/operations/deployment.md.
func signingKey() ([]byte, error) {
	key := preferences.GetString("app.signingKey")
	if reason := jwtopt.SigningKeyProblemFor(key); reason != "" {
		return nil, errors.New("tokenservice: " + reason)
	}
	return []byte(key), nil
}
