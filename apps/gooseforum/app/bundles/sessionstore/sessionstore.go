package sessionstore

import (
	"net/http"
	"sync"

	"github.com/gorilla/sessions"
	"github.com/leancodebox/GooseForum/app/bundles/algorithm"
	"github.com/leancodebox/GooseForum/app/bundles/preferences"
	"github.com/leancodebox/GooseForum/app/bundles/setting"
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
