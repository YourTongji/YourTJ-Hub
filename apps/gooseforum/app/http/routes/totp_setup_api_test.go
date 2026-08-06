package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/http/controllers/api"
	"github.com/leancodebox/GooseForum/app/http/controllers/component"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotp"
	"github.com/leancodebox/GooseForum/app/models/forum/userTotpRecoveryCodes"
	"github.com/leancodebox/GooseForum/app/models/forum/users"
)

// setupTotpSetupTestDB migrates the tables the totp setup handler touches.
func setupTotpSetupTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}, &userTotp.Entity{}, &userTotpRecoveryCodes.Entity{}); err != nil {
		t.Fatalf("migrate totp setup tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userTotpRecoveryCodes.Entity{})
	conn.Where("1 = 1").Delete(&userTotp.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func createTotpSetupUser(t *testing.T, id uint64, username, password string) {
	t.Helper()
	user := users.MakeUser(username, password, username+"@example.com")
	user.Id = id
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

// totpSetupRouter registers the totp setup route with an authenticated user
// injected via middleware, matching the production POST contract.
func totpSetupRouter(userID uint64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userId", userID)
		c.Next()
	})
	router.POST("api/user/totp/setup", UpButterReq(api.TotpSetup))
	return router
}

type totpSetupResponse struct {
	Result struct {
		Secret     string `json:"secret"`
		OtpauthURL string `json:"otpauthUrl"`
	} `json:"result"`
	Code component.Status `json:"code"`
}

func postTotpSetup(t *testing.T, router http.Handler, password string) (int, totpSetupResponse) {
	t.Helper()
	body := strings.NewReader(`{"password":"` + password + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/totp/setup", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp totpSetupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return rec.Code, resp
}

func TestTotpSetupRouteRegisteredAsPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiRoute(router)

	registered := map[string]bool{}
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}
	if !registered["POST /api/user/totp/setup"] {
		t.Fatal("POST /api/user/totp/setup was not registered")
	}
	if registered["GET /api/user/totp/setup"] {
		t.Fatal("GET /api/user/totp/setup should not be registered")
	}
}

func TestTotpSetupPostWithCorrectPassword(t *testing.T) {
	setupTotpSetupTestDB(t)
	createTotpSetupUser(t, 9001, "totpsetup-ok", "correct-password-123")
	router := totpSetupRouter(9001)

	status, resp := postTotpSetup(t, router, "correct-password-123")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %+v)", status, resp)
	}
	if resp.Code != 0 {
		t.Fatalf("response code = %d, want 0 (success)", resp.Code)
	}
	if resp.Result.Secret == "" {
		t.Fatal("secret should not be empty")
	}
	if !strings.HasPrefix(resp.Result.OtpauthURL, "otpauth://totp/") {
		t.Fatalf("otpauthUrl = %q, want otpauth://totp/ prefix", resp.Result.OtpauthURL)
	}
}

func TestTotpSetupPostRejectsWrongPassword(t *testing.T) {
	setupTotpSetupTestDB(t)
	createTotpSetupUser(t, 9002, "totpsetup-wrong", "correct-password-123")
	router := totpSetupRouter(9002)

	status, resp := postTotpSetup(t, router, "wrong-password")
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (component errors use 200)", status)
	}
	if resp.Code == 0 {
		t.Fatal("response code should be non-zero (failure)")
	}
	if resp.Result.Secret != "" {
		t.Fatal("secret should be empty on failure")
	}
}

func TestTotpSetupPostRequiresPassword(t *testing.T) {
	setupTotpSetupTestDB(t)
	createTotpSetupUser(t, 9003, "totpsetup-nopass", "correct-password-123")
	router := totpSetupRouter(9003)

	req := httptest.NewRequest(http.MethodPost, "/api/user/totp/setup", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var resp component.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code == 0 {
		t.Fatal("setup without password should fail validation")
	}
}
