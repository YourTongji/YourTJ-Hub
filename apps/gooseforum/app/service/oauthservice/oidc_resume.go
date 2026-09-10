package oauthservice

import (
	"crypto/rand"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
	"github.com/markbates/goth/gothic"
)

const resumeCookie = "yourtj_oauth_resume"
const resumeLifetime = 10 * time.Minute

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// Only resume the fixed local authorization bridge. Its own browser-binding,
// expiry and single-use checks remain authoritative for the authorization ID.
func oidcResumeTarget(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", errors.New("invalid OAuth continuation")
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil || u.Scheme != "" || u.Host != "" || u.User != nil || u.Fragment != "" || u.Path != "/api/oauth/authorize/callback" || len(q) != 1 || len(q["id"]) != 1 || !requestIDPattern.MatchString(q.Get("id")) {
		return "", errors.New("invalid OAuth continuation")
	}
	return "/api/oauth/authorize/callback?id=" + url.QueryEscape(q.Get("id")), nil
}

// The continuation has a separate signed, HttpOnly cookie: Gothic replaces its
// own provider-session cookie at authorization start and deletes it on callback.
// Bind to a fresh server-generated OAuth state, provider and short expiry; never
// trust redirect/state supplied on the returning callback.
func prepareOIDCResume(w http.ResponseWriter, r *http.Request) error {
	session, _ := sessionstore.GetSession().New(r, resumeCookie)
	options := *session.Options
	session.Options = &options
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	if raw := r.URL.Query().Get("redirect"); raw != "" {
		target, err := oidcResumeTarget(raw)
		if err != nil {
			return err
		}
		provider := r.URL.Query().Get("provider")
		if provider != ProviderGitHub && provider != ProviderGoogle {
			return errors.New("invalid OAuth provider")
		}
		state := rand.Text()
		q := r.URL.Query()
		q.Set("state", state)
		r.URL.RawQuery = q.Encode()
		session.Options.MaxAge = int(resumeLifetime / time.Second)
		session.Values["state"] = state
		session.Values["provider"] = provider
		session.Values["target"] = target
		session.Values["expires"] = time.Now().Add(resumeLifetime).Unix()
	}
	return session.Save(r, w)
}

func consumeOIDCResume(w http.ResponseWriter, r *http.Request) (string, error) {
	if _, err := r.Cookie(resumeCookie); errors.Is(err, http.ErrNoCookie) {
		return "", nil
	}
	session, err := sessionstore.GetSession().New(r, resumeCookie)
	options := *session.Options
	session.Options = &options
	session.Options.MaxAge = -1
	if saveErr := session.Save(r, w); saveErr != nil {
		return "", saveErr
	}
	if err != nil || session.IsNew {
		return "", errors.New("invalid OAuth continuation cookie")
	}
	state, _ := session.Values["state"].(string)
	provider, _ := session.Values["provider"].(string)
	expires, _ := session.Values["expires"].(int64)
	if state == "" || state != gothic.GetState(r) || provider != r.URL.Query().Get("provider") || expires <= time.Now().Unix() {
		return "", errors.New("OAuth continuation expired or mismatched")
	}
	target, _ := session.Values["target"].(string)
	return oidcResumeTarget(target)
}
