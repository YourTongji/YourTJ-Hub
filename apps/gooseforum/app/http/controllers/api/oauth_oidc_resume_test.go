package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/sessionstore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userOAuth"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

// Only the upstream identity fetch is faked. Goth still persists its provider
// session, validates OAuth state and clears the session in the real callback.
type resumeProvider struct {
	goth.Provider
	user goth.User
}

func (p resumeProvider) FetchUser(goth.Session) (goth.User, error) { return p.user, nil }

func TestSocialOAuthResumesMobileOIDC(t *testing.T) {
	for _, provider := range []string{"github", "google"} {
		t.Run(provider, func(t *testing.T) {
			setupOIDCProviderTestDB(t)
			setupOAuthCallbackTestDB(t)
			user := &users.EntityComplete{Username: "resume" + provider, Email: "resume" + provider + "@example.com", IsActivated: users.ActivationSuccess}
			if err := users.Create(user); err != nil {
				t.Fatal(err)
			}
			if err := userOAuth.Create(&userOAuth.Entity{UserId: user.Id, Provider: provider, ProviderUid: "resume-" + provider}); err != nil {
				t.Fatal(err)
			}
			var upstream goth.Provider = github.New("test", "test", "https://forum.example/api/auth/github/callback")
			if provider == "google" {
				upstream = google.New("test", "test", "https://forum.example/api/auth/google/callback")
			}
			old, oldErr := goth.GetProvider(provider)
			oldStore := gothic.Store
			gothic.Store = sessionstore.GetSession()
			goth.UseProviders(resumeProvider{Provider: upstream, user: goth.User{Provider: provider, UserID: "resume-" + provider}})
			t.Cleanup(func() {
				gothic.Store = oldStore
				if oldErr == nil {
					goth.UseProviders(old)
				} else {
					delete(goth.GetProviders(), provider)
				}
			})
			oidc, err := oidcservice.Router()
			if err != nil {
				t.Fatal(err)
			}
			router := gin.New()
			router.GET("/api/oauth/*path", gin.WrapH(oidc))
			router.GET("/api/auth/:provider", ProviderLogin)
			router.GET("/api/auth/:provider/callback", ProviderCallback)
			cookies := map[string]*http.Cookie{}
			get := func(target string) *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodGet, target, nil)
				for _, c := range cookies {
					req.AddCookie(c)
				}
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				for _, c := range rec.Result().Cookies() {
					if c.MaxAge < 0 {
						delete(cookies, c.Name)
					} else {
						cookies[c.Name] = c
					}
				}
				return rec
			}
			verifier := "test-verifier-0123456789-abcdefghijklmnopqrstuvwxyz"
			sum := sha256.Sum256([]byte(verifier))
			q := url.Values{"client_id": {"yourtj-mobile"}, "redirect_uri": {"yourtj://callback"}, "response_type": {"code"}, "scope": {"openid profile email"}, "state": {"mobile-state"}, "nonce": {"mobile-nonce"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}, "login_hint": {provider}}
			rec := get(testIssuer + "/authorize?" + q.Encode())
			login, err := url.Parse(rec.Header().Get("Location"))
			if err != nil || login.Path != "/api/auth/"+provider {
				t.Fatalf("login redirect: %s", rec.Header().Get("Location"))
			}
			callback := login.Query().Get("redirect")
			rec = get(login.String())
			authorization, err := url.Parse(rec.Header().Get("Location"))
			if err != nil || authorization.Query().Get("state") == "" {
				t.Fatalf("upstream authorization: %s", rec.Body.String())
			}
			oauthCallback := "/api/auth/" + provider + "/callback?state=" + url.QueryEscape(authorization.Query().Get("state")) + "&code=upstream-code"
			// A forged state cannot create a forum session or resume authorization.
			bad := httptest.NewRequest(http.MethodGet, "/api/auth/"+provider+"/callback?state=wrong&code=x", nil)
			bad.Header.Set("X-Goose-Page", "true")
			for _, c := range cookies {
				bad.AddCookie(c)
			}
			deniedState := httptest.NewRecorder()
			router.ServeHTTP(deniedState, bad)
			if deniedState.Code < 400 || hasAccessTokenCookie(deniedState) {
				t.Fatal("forged OAuth state accepted")
			}
			rec = get(oauthCallback)
			if rec.Code != http.StatusFound || rec.Header().Get("Location") != callback {
				t.Fatalf("social callback status=%d location=%q, want %q; body=%s", rec.Code, rec.Header().Get("Location"), callback, rec.Body.String())
			}
			// Missing browser binding must not complete the request.
			binding := cookies["yourtj_oidc_binding"]
			delete(cookies, "yourtj_oidc_binding")
			denied := get(callback)
			if denied.Code == http.StatusFound {
				t.Fatal("missing binding accepted")
			}
			cookies["yourtj_oidc_binding"] = binding
			rec = get(callback)
			mobile, err := url.Parse(rec.Header().Get("Location"))
			if err != nil || mobile.Scheme != "yourtj" || mobile.Query().Get("code") == "" || mobile.Query().Get("state") != "mobile-state" {
				t.Fatalf("mobile callback: %d %s %s", rec.Code, rec.Header().Get("Location"), rec.Body.String())
			}
			body, _ := json.Marshal(map[string]string{"code": mobile.Query().Get("code"), "codeVerifier": verifier, "nonce": "mobile-nonce", "redirectUri": "yourtj://callback"})
			exchange, res := postOidcExchange(t, string(body))
			if exchange.Code != http.StatusOK {
				t.Fatalf("exchange: %s", exchange.Body.String())
			}
			token, _ := res.Result.(map[string]any)["token"].(string)
			id, _, _, ok := validateForumToken(t, token)
			if !ok || id != user.Id {
				t.Fatal("wrong forum identity")
			}
			replay, _ := postOidcExchange(t, string(body))
			if replay.Code == http.StatusOK {
				t.Fatal("authorization code replay accepted")
			}
			// Existing Web account binding must still return to settings and
			// must not issue another login session or retain an OIDC target.
			before := countSessions(t, user.Id)
			bindingStart := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(bindingStart)
			ctx.Set("userId", user.Id)
			ctx.Params = gin.Params{{Key: "provider", Value: provider}}
			ctx.Request = httptest.NewRequest(http.MethodGet, login.String(), nil)
			ProviderLogin(ctx)
			for _, c := range bindingStart.Result().Cookies() {
				if c.Name == "yourtj_oauth_resume" && c.MaxAge != -1 {
					t.Fatal("binding retained OIDC continuation")
				}
			}
			upstreamURL, _ := url.Parse(bindingStart.Header().Get("Location"))
			bindingReturn := httptest.NewRecorder()
			ctx, _ = gin.CreateTestContext(bindingReturn)
			ctx.Set("userId", user.Id)
			ctx.Params = gin.Params{{Key: "provider", Value: provider}}
			ctx.Request = httptest.NewRequest(http.MethodGet, "/api/auth/"+provider+"/callback?state="+url.QueryEscape(upstreamURL.Query().Get("state")), nil)
			for _, c := range bindingStart.Result().Cookies() {
				if c.MaxAge >= 0 {
					ctx.Request.AddCookie(c)
				}
			}
			ProviderCallback(ctx)
			if bindingReturn.Header().Get("Location") != "/settings?tab=binding" || countSessions(t, user.Id) != before {
				t.Fatalf("binding changed login behavior: %s", bindingReturn.Body.String())
			}

		})
	}
}
