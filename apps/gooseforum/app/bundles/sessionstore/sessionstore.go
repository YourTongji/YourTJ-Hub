package sessionstore

import (
	"net/http"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/setting"
)

var store *sessions.CookieStore
var once sync.Once

func GetSession() *sessions.CookieStore {
	once.Do(func() {
		store = sessions.NewCookieStore([]byte(sessionSigningKey()))
		configureSessionStore(store)
	})
	return store
}

// sessionSigningKey returns the gorilla-session cookie signing key. serve
// refuses to boot on an empty/weak app.signingKey (issue #106), so the random
// fallback below is reachable only from tests or a non-serve entrypoint — it is
// defensive, not a runnable-deploy path. A per-process random fallback also
// means cookie sessions do not survive a process restart, which is acceptable
// for that non-serve scope.
func sessionSigningKey() string {
	if signingKey := preferences.GetString("app.signingKey"); signingKey != "" {
		return signingKey
	}
	return algorithm.SafeGenerateSigningKey(32)
}

func configureSessionStore(store *sessions.CookieStore) {
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteLaxMode
	store.Options.Secure = setting.CookieSecure()
}
