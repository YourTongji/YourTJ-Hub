package appleauthservice

import (
	"crypto/elliptic"
	"encoding/base64"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/golang-jwt/jwt/v5"
)

var currentClient atomic.Pointer[Client]

// Init reads server-only credentials once at startup. Missing or malformed
// configuration keeps Apple sign-in unavailable; no credential is logged.
func Init() {
	currentClient.Store(nil)
	clientID := strings.TrimSpace(preferences.GetString("apple.client_id", ""))
	teamID := strings.TrimSpace(preferences.GetString("apple.team_id", ""))
	keyID := strings.TrimSpace(preferences.GetString("apple.key_id", ""))
	encoded := strings.TrimSpace(preferences.GetString("apple.private_key_base64", ""))
	if clientID == "" || teamID == "" || keyID == "" || encoded == "" {
		return
	}
	pem, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return
	}
	key, err := jwt.ParseECPrivateKeyFromPEM(pem)
	if err != nil || key.Curve != elliptic.P256() {
		return
	}
	client := newClient(Config{ClientID: clientID, TeamID: teamID, KeyID: keyID, PrivateKey: key}, &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }})
	currentClient.Store(client)
}

func Ready() bool { return currentClient.Load() != nil }
