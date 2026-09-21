package api

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/campusservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/oidcservice"
	"github.com/gin-gonic/gin"
)

type loginSchoolProvider struct {
	campusservice.Provider
	exchanges int
}

func (p *loginSchoolProvider) Authorize(state, nonce, verifier, mode string) string {
	return "https://school.example/auth?state=" + url.QueryEscape(state)
}
func (p *loginSchoolProvider) Exchange(context.Context, string, string) (campusservice.Credentials, error) {
	p.exchanges++
	return campusservice.Credentials{Access: "not-exposed", Refresh: "not-exposed"}, nil
}
func (p *loginSchoolProvider) Identity(_ context.Context, c campusservice.Credentials, _ string) (campusservice.Credentials, error) {
	c.StudentID = "2356789"
	c.Subject = "official-subject"
	return c, nil
}

func setupTongjiLogin(t *testing.T) (*gin.Engine, *loginSchoolProvider) {
	t.Helper()
	setupOIDCProviderTestDB(t)
	setupOAuthCallbackTestDB(t)
	conn := db.Connect()
	if err := conn.AutoMigrate(&campus.Binding{}, &campus.IdentityReservation{}, &pointsRecord.Entity{}, &userPoints.Entity{}, &userStatistics.Entity{}); err != nil {
		t.Fatal(err)
	}
	conn.Where("1=1").Delete(&campus.Binding{})
	conn.Where("1=1").Delete(&campus.IdentityReservation{})
	conn.Unscoped().Where("email = ?", "2356789@tongji.edu.cn").Delete(&users.EntityComplete{})
	setSecurityConfigForCallbackTest(t, pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1, EnableEmailVerification: true, AllowedDomains: []string{"tongji.edu.cn"}})
	p := &loginSchoolProvider{}
	service := campusservice.New(campusservice.Config{EncryptionKey: strings.Repeat("e", 32), IdentityKey: strings.Repeat("i", 32)}, campus.Store{DB: conn}, p)
	old := tongjiLoginService
	tongjiLoginService = func() (*campusservice.Service, error) { return service, nil }
	t.Cleanup(func() { tongjiLoginService = old })
	router := gin.New()
	router.GET("/api/auth/:provider", ProviderLogin)
	router.GET("/api/campus/tongji/callback", func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-store")
		c.Header("Referrer-Policy", "no-referrer")
	}, CampusCallback)
	oidc, err := oidcservice.Router()
	if err != nil {
		t.Fatal(err)
	}
	router.GET("/api/oauth/*path", gin.WrapH(oidc))
	return router, p
}

func TestTongjiLoginCallbackRequiresBrowserAndIgnoresCallbackRedirect(t *testing.T) {
	router, p := setupTongjiLogin(t)
	start := httptest.NewRecorder()
	router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/auth/tongji?redirect=%2Fcampus", nil))
	location, _ := url.Parse(start.Header().Get("Location"))
	var cookie *http.Cookie
	for _, c := range start.Result().Cookies() {
		if c.Name == tongjiLoginCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/api/campus/tongji/callback" || cookie.Domain != "" {
		t.Fatal("missing browser proof")
	}
	callback := "/api/campus/tongji/callback?state=" + url.QueryEscape(location.Query().Get("state")) + "&code=school-code&redirect=https://evil.test"
	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, callback, nil))
	if p.exchanges != 0 || hasAccessTokenCookie(denied) || denied.Header().Get("Location") != "/login?tongjiNotice=failed" {
		t.Fatal("cookie-less callback accepted")
	}
	req := httptest.NewRequest(http.MethodGet, callback, nil)
	req.AddCookie(cookie)
	success := httptest.NewRecorder()
	router.ServeHTTP(success, req)
	if success.Code != 303 || success.Header().Get("Location") != "/settings?onboarding=tongji&returnTo=%2Fcampus" || !hasAccessTokenCookie(success) {
		t.Fatal("login did not complete")
	}
	if strings.Contains(success.Body.String(), "school-code") || strings.Contains(success.Body.String(), "not-exposed") {
		t.Fatal("credentials in output")
	}
	replay := httptest.NewRecorder()
	router.ServeHTTP(replay, req)
	if p.exchanges != 1 || hasAccessTokenCookie(replay) {
		t.Fatal("callback replay accepted")
	}
}

func TestTongjiLoginResumesMobileOIDCAndExchangesForumSession(t *testing.T) {
	router, _ := setupTongjiLogin(t)
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
	q := url.Values{"client_id": {"yourtj-mobile"}, "redirect_uri": {"yourtj://callback"}, "response_type": {"code"}, "scope": {"openid profile email"}, "state": {"mobile-state"}, "nonce": {"mobile-nonce"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"}, "login_hint": {"tongji"}}
	rec := get(testIssuer + "/authorize?" + q.Encode())
	login, _ := url.Parse(rec.Header().Get("Location"))
	if login.Path != "/api/auth/tongji" {
		t.Fatal("wrong provider hint")
	}
	continuation := login.Query().Get("redirect")
	rec = get(login.String())
	school, _ := url.Parse(rec.Header().Get("Location"))
	rec = get("/api/campus/tongji/callback?code=school-code&state=" + url.QueryEscape(school.Query().Get("state")))
	onboarding, parseErr := url.Parse(rec.Header().Get("Location"))
	if parseErr != nil || rec.Code != 303 || onboarding.Path != "/settings" || onboarding.Query().Get("onboarding") != "tongji" || onboarding.Query().Get("returnTo") != continuation {
		t.Fatal("first login guide lost the OIDC continuation")
	}
	binding := cookies["yourtj_oidc_binding"]
	delete(cookies, "yourtj_oidc_binding")
	if get(continuation).Code == http.StatusFound {
		t.Fatal("missing OIDC browser binding accepted")
	}
	cookies["yourtj_oidc_binding"] = binding
	rec = get(continuation)
	mobile, _ := url.Parse(rec.Header().Get("Location"))
	if mobile.Scheme != "yourtj" || mobile.Query().Get("code") == "" {
		t.Fatal("did not return to mobile")
	}
	body, marshalErr := json.Marshal(map[string]string{"code": mobile.Query().Get("code"), "codeVerifier": verifier, "nonce": "mobile-nonce", "redirectUri": "yourtj://callback"})
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	exchange, res := postOidcExchange(t, string(body))
	if exchange.Code != 200 {
		t.Fatal("mobile exchange failed")
	}
	token, _ := res.Result.(map[string]any)["token"].(string)
	id, _, _, ok := validateForumToken(t, token)
	if !ok || id == 0 {
		t.Fatal("no forum session")
	}
	user, err := users.Get(id)
	if err != nil || user.Email != "2356789@tongji.edu.cn" || user.IsActivated != users.ActivationSuccess {
		t.Fatal("wrong account")
	}
	replay, _ := postOidcExchange(t, string(body))
	if replay.Code == 200 {
		t.Fatal("OIDC code reused")
	}
}

func TestTongjiLoginPreviouslyBoundIdentityUsesRecoveryNotice(t *testing.T) {
	router, _ := setupTongjiLogin(t)
	service, err := tongjiLoginService()
	if err != nil {
		t.Fatal(err)
	}
	start, browser, err := service.StartLogin("/", "en")
	if err != nil {
		t.Fatal(err)
	}
	address, _ := url.Parse(start)
	result, err := service.Login(context.Background(), browser, address.Query().Get("state"), "code", pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1})
	if err != nil {
		t.Fatal(err)
	}
	conn := db.Connect()
	if err := conn.Model(result.User).Update("email", "changed@example.test").Error; err != nil {
		t.Fatal(err)
	}
	if err := (campus.Store{DB: conn}).DeleteForUser(result.User.Id); err != nil {
		t.Fatal(err)
	}
	start, browser, err = service.StartLogin("/campus", "en")
	if err != nil {
		t.Fatal(err)
	}
	address, _ = url.Parse(start)
	req := httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?code=code&state="+url.QueryEscape(address.Query().Get("state")), nil)
	req.AddCookie(&http.Cookie{Name: tongjiLoginCookie, Value: browser})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login?tongjiNotice=accountExists&redirect=%2Fcampus" || hasAccessTokenCookie(rec) {
		t.Fatalf("unexpected recovery response: %d %s", rec.Code, rec.Header().Get("Location"))
	}
}

func TestTongjiLoginExistingAccountSkipsOnboarding(t *testing.T) {
	router, _ := setupTongjiLogin(t)
	for attempt := range 2 {
		start := httptest.NewRecorder()
		router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/auth/tongji?redirect=%2Fcampus", nil))
		location, err := url.Parse(start.Header().Get("Location"))
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/campus/tongji/callback?code=school-code&state="+url.QueryEscape(location.Query().Get("state")), nil)
		for _, cookie := range start.Result().Cookies() {
			req.AddCookie(cookie)
		}
		result := httptest.NewRecorder()
		router.ServeHTTP(result, req)
		want := "/settings?onboarding=tongji&returnTo=%2Fcampus"
		if attempt == 1 {
			want = "/campus"
		}
		if result.Code != http.StatusSeeOther || result.Header().Get("Location") != want || !hasAccessTokenCookie(result) {
			t.Fatalf("attempt %d: code %d location %s", attempt, result.Code, result.Header().Get("Location"))
		}
	}
}
