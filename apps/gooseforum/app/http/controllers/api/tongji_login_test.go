package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/component"
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
	router.GET("/api/auth/tongji/registration", TongjiRegistrationStatus)
	router.POST("/api/auth/tongji/registration", TongjiRegister)
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
	if success.Code != 303 || success.Header().Get("Location") != "/register/tongji" || hasAccessTokenCookie(success) {
		t.Fatal("login did not complete")
	}
	if strings.Contains(success.Body.String(), "school-code") || strings.Contains(success.Body.String(), "not-exposed") {
		t.Fatal("credentials in output")
	}
	finishTestTongjiRegistration(t, router, success.Result().Cookies())
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
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/register/tongji" || hasAccessTokenCookie(rec) {
		t.Fatal("registration bypassed")
	}
	registration := finishTestTongjiRegistration(t, router, rec.Result().Cookies())
	for _, c := range registration.Result().Cookies() {
		cookies[c.Name] = c
	}
	var registrationResponse struct {
		Result struct {
			Redirect string `json:"redirect"`
		} `json:"result"`
	}
	if err := json.Unmarshal(registration.Body.Bytes(), &registrationResponse); err != nil || registrationResponse.Result.Redirect != continuation {
		t.Fatal("registration lost OIDC continuation")
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
	status, statusErr := service.RegistrationStatus(result.Registration)
	if statusErr != nil {
		t.Fatal(statusErr)
	}
	result, err = service.CompleteRegistration(context.Background(), result.Registration, status.CSRFToken, campusservice.Registration{Username: "retained_user", PasswordHash: "test-hash"}, pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1})
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
		want := "/register/tongji"
		if attempt == 1 {
			want = "/campus"
		}
		if result.Code != http.StatusSeeOther || result.Header().Get("Location") != want || hasAccessTokenCookie(result) != (attempt == 1) {
			t.Fatalf("attempt %d: code %d location %s", attempt, result.Code, result.Header().Get("Location"))
		}
		if attempt == 0 {
			finishTestTongjiRegistration(t, router, result.Result().Cookies())
		}
	}
}

func finishTestTongjiRegistration(t *testing.T, router *gin.Engine, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	statusReq := httptest.NewRequest(http.MethodGet, "/api/auth/tongji/registration", nil)
	for _, cookie := range cookies {
		statusReq.AddCookie(cookie)
	}
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	var status struct {
		Result campusservice.RegistrationStatus `json:"result"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &status); err != nil || status.Result.CSRFToken == "" {
		t.Fatalf("missing registration proof: %s", statusRec.Body.String())
	}
	data, err := json.Marshal(map[string]string{"username": "chosen_student", "password": "Password123", "csrfToken": status.Result.CSRFToken})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/tongji/registration", strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !hasAccessTokenCookie(rec) {
		t.Fatalf("registration failed: %d %s", rec.Code, rec.Body.String())
	}
	return rec
}

func TestTongjiRegistrationValidatesProofAndCredentials(t *testing.T) {
	router, _ := setupTongjiLogin(t)
	service, err := tongjiLoginService()
	if err != nil {
		t.Fatal(err)
	}
	start, browser, err := service.StartLogin("/campus", "en")
	if err != nil {
		t.Fatal(err)
	}
	address, _ := url.Parse(start)
	result, err := service.Login(context.Background(), browser, address.Query().Get("state"), "code", pageConfig.SecurityAndRegistration{EnableSignup: true, MaxDailySignups: -1})
	if err != nil {
		t.Fatal(err)
	}
	status, err := service.RegistrationStatus(result.Registration)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		cookie, csrf, username, password string
		code                             int
	}{
		{"", status.CSRFToken, "chosen_student", "Password123", http.StatusGone},
		{"foreign", status.CSRFToken, "chosen_student", "Password123", http.StatusGone},
		{result.Registration, "", "chosen_student", "Password123", http.StatusForbidden},
		{result.Registration, "wrong", "chosen_student", "Password123", http.StatusForbidden},
		{result.Registration, status.CSRFToken, "bad", "Password123", http.StatusBadRequest},
		{result.Registration, status.CSRFToken, "chosen_student", "123", http.StatusBadRequest},
		{result.Registration, status.CSRFToken, "chosen_student", "abcdefgh", http.StatusBadRequest},
	} {
		body, err := json.Marshal(map[string]string{"username": tc.username, "password": tc.password, "csrfToken": tc.csrf})
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/auth/tongji/registration", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: tongjiRegistrationCookieName, Value: tc.cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != tc.code || hasAccessTokenCookie(rec) {
			t.Fatalf("guard: %d %s", rec.Code, rec.Body.String())
		}
	}
	cookies := []*http.Cookie{{Name: tongjiRegistrationCookieName, Value: result.Registration}}
	rec := finishTestTongjiRegistration(t, router, cookies)
	if rec.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("registration is cacheable")
	}
	var user users.EntityComplete
	if err := db.Connect().Where("username = ?", "chosen_student").First(&user).Error; err != nil {
		t.Fatal(err)
	}
	if user.Password == "Password123" || user.Password == "" || user.IsActivated != users.ActivationSuccess {
		t.Fatal("password/activation not set")
	}
	if err := algorithm.VerifyEncryptPassword(user.Password, "Password123"); err != nil {
		t.Fatal(err)
	}
}

// tongjiIdentityKey mirrors campusservice.fingerprint for the test setup's
// IdentityKey and the mock provider's student ID.
func tongjiIdentityKey(t *testing.T) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(strings.Repeat("i", 32)))
	if _, err := mac.Write([]byte(campusservice.Issuer + "\x00" + "2356789")); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(mac.Sum(nil))
}

func startTongjiRegistrationProof(t *testing.T, router *gin.Engine) []*http.Cookie {
	t.Helper()
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
	callback := httptest.NewRecorder()
	router.ServeHTTP(callback, req)
	if callback.Code != http.StatusSeeOther || callback.Header().Get("Location") != "/register/tongji" {
		t.Fatalf("no registration proof: %d %s", callback.Code, callback.Header().Get("Location"))
	}
	return callback.Result().Cookies()
}

func postTongjiRegistration(t *testing.T, router *gin.Engine, cookies []*http.Cookie, username, password, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"username": username, "password": password, "csrfToken": csrf})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/tongji/registration", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func tongjiRegistrationCSRF(t *testing.T, router *gin.Engine, cookies []*http.Cookie) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/tongji/registration", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var status struct {
		Result campusservice.RegistrationStatus `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &status); err != nil || status.Result.CSRFToken == "" {
		t.Fatalf("registration status unavailable: %d %s", rec.Code, rec.Body.String())
	}
	return status.Result.CSRFToken
}

func TestTongjiRegistrationFailureMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		code   component.MessageCode
	}{
		{"proof consumed or expired", campusservice.ErrFlow, http.StatusGone, component.MessageAuthRequired},
		{"wrapped proof expiry", fmt.Errorf("tx: %w", campusservice.ErrFlow), http.StatusGone, component.MessageAuthRequired},
		{"identity used", campus.ErrIdentityUsed, http.StatusConflict, component.MessageAuthTongjiAccountExists},
		{"email occupied", users.ErrEmailOccupied, http.StatusConflict, component.MessageAuthTongjiAccountExists},
		{"signup disabled", campusservice.ErrSignupDisabled, http.StatusConflict, component.MessageAuthSignupDisabled},
		{"daily quota", users.ErrSignupQuota, http.StatusConflict, component.MessageAuthRegisterDailyQuota},
		{"unexpected failure", errors.New("db down"), http.StatusConflict, component.MessageAuthRegisterFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, code := tongjiRegistrationFailure(tc.err)
			if status != tc.status || code != tc.code {
				t.Fatalf("map %v: got (%d, %s), want (%d, %s)", tc.err, status, code, tc.status, tc.code)
			}
		})
	}
}

func TestTongjiRegisterAccountCollisionGuidesRecovery(t *testing.T) {
	t.Run("identityBoundBetweenCallbackAndSubmit", func(t *testing.T) {
		router, _ := setupTongjiLogin(t)
		conn := db.Connect()
		proof := startTongjiRegistrationProof(t, router)
		owner := users.EntityComplete{Username: "owner_identity", Email: "owner@example.test"}
		if err := conn.Create(&owner).Error; err != nil {
			t.Fatal(err)
		}
		if err := conn.Create(&campus.Binding{UserID: owner.Id, IdentityKey: tongjiIdentityKey(t), Revision: "rev", Sealed: "sealed"}).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			conn.Where("identity_key = ?", tongjiIdentityKey(t)).Delete(&campus.Binding{})
			conn.Unscoped().Where("username = ?", "owner_identity").Delete(&users.EntityComplete{})
		})
		rec := postTongjiRegistration(t, router, proof, "chosen_student", "Password123", tongjiRegistrationCSRF(t, router, proof))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "auth.tongji.accountExists") {
			t.Fatalf("expected account-exists guidance: %d %s", rec.Code, rec.Body.String())
		}
		var count int64
		if err := conn.Model(&users.EntityComplete{}).Where("username = ?", "chosen_student").Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("partial account created: %d %v", count, err)
		}
	})
	t.Run("studentEmailClaimedBetweenCallbackAndSubmit", func(t *testing.T) {
		router, _ := setupTongjiLogin(t)
		conn := db.Connect()
		proof := startTongjiRegistrationProof(t, router)
		owner := users.EntityComplete{Username: "owner_email", Email: "2356789@tongji.edu.cn"}
		if err := conn.Create(&owner).Error; err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			conn.Unscoped().Where("username = ?", "owner_email").Delete(&users.EntityComplete{})
		})
		rec := postTongjiRegistration(t, router, proof, "chosen_student", "Password123", tongjiRegistrationCSRF(t, router, proof))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "auth.tongji.accountExists") {
			t.Fatalf("expected account-exists guidance: %d %s", rec.Code, rec.Body.String())
		}
		var count int64
		if err := conn.Model(&users.EntityComplete{}).Where("username = ?", "chosen_student").Count(&count).Error; err != nil || count != 0 {
			t.Fatalf("partial account created: %d %v", count, err)
		}
	})
}
