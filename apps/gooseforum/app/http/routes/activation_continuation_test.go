package routes

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestActivationSuccessNamesAccountWithoutSwitchingSession(t *testing.T) {
	conn, _ := setupTwoPhaseEmailTest(t)
	router := gin.New()
	router.GET("/activate", middleware.JWTAuth, controllers.ActivateAccount)
	router.GET("/test/owner", middleware.JWTAuthCheck, func(c *gin.Context) { c.String(http.StatusOK, "%d", c.GetUint64("userId")) })
	withRouteTestSigningKey(t, "activation-continuation-test-signing-key")
	viewer := createHTTPContractUser(t, conn, contractTestID())
	subject := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(subject).Updates(map[string]any{"is_activated": users.ActivationPending, "is_frozen": users.StatusFrozen}).Error; err != nil {
		t.Fatal(err)
	}
	token, err := tokenservice.GenerateActivationToken(subject.Id, subject.Email)
	if err != nil {
		t.Fatal(err)
	}
	session := contractSessionToken(t, viewer)
	req := httptest.NewRequest(http.MethodGet, "/activate?token="+token+"&redirect=%2Fpublish", nil)
	req.Header.Set("Authorization", "Bearer "+session)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	body := rec.Body.String()
	for _, want := range []string{subject.Username, `href="/publish"`, `yourtj://account/email-verified`, "current", "restrictions"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing continuation truth %q", want)
		}
	}
	if strings.Contains(body, viewer.Username) || strings.Contains(body, "use every forum feature") {
		t.Error("activation claimed a different session or bypassed restrictions")
	}
	if rec.Header().Get("New-Token") != "" || len(rec.Result().Cookies()) != 0 {
		t.Fatal("activation issued a login credential")
	}
	probe := httptest.NewRequest(http.MethodGet, "/test/owner", nil)
	probe.Header.Set("Authorization", "Bearer "+session)
	owner := httptest.NewRecorder()
	router.ServeHTTP(owner, probe)
	if owner.Code != http.StatusOK || owner.Body.String() != strconv.FormatUint(viewer.Id, 10) {
		t.Fatal("browser account was switched")
	}
	row, err := users.Get(subject.Id)
	if err != nil || row.IsActivated != users.ActivationSuccess || row.IsFrozen != users.StatusFrozen {
		t.Fatalf("activation changed account restrictions: %#v %v", row, err)
	}
	if rec.Header().Get("Cache-Control") != "private, no-store" || rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("token-bearing result may be cached or leaked")
	}
}
