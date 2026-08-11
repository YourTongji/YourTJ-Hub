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
func signingKey() ([]byte, error) {
	key := preferences.GetString("app.signingKey")
	if reason := jwtopt.SigningKeyProblemFor(key); reason != "" {
		return nil, errors.New("tokenservice: " + reason)
	}
	return []byte(key), nil
}
