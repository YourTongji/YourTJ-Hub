package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/controllers/api"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/permission"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
)

func TestMobileWebSessionHTTPContract(t *testing.T) {
	conn, router := setupAdminRolesContractTest(t)
	router.GET("/api/auth/mobile-web-session", api.MobileWebSession)
	manager := createContractRoleManager(t, conn)
	token := contractSessionToken(t, manager)
	ordinary := createHTTPContractUser(t, conn, contractTestID())
	ordinaryToken := contractSessionToken(t, ordinary)
	moderator := createHTTPContractUser(t, conn, contractTestID())
	grantContractCategoryModerator(t, conn, moderator.Id)
	moderatorToken := contractSessionToken(t, moderator)
	courseManager := createHTTPContractUser(t, conn, contractTestID())
	grantContractPermission(t, conn, courseManager.Id, permission.CourseManager)
	courseToken := contractSessionToken(t, courseManager)
	revoked := contractSessionToken(t, manager)
	claims, err := jwtopt.Std().ParseToken(revoked)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.RevokeByJti(manager.Id, claims.Jti); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, bearer, cookie, target, destination string
		status                                    int
	}{
		{"bearer manager", token, "", "admin", "/admin", 303},
		{"cookie cannot establish session", "", token, "admin", "", 401},
		{"invalid bearer never falls back to cookie", "invalid", token, "admin", "", 401},
		{"anonymous", "", "", "admin", "", 401},
		{"ordinary user", ordinaryToken, "", "admin", "", 403},
		{"revoked", revoked, "", "admin", "", 401},
		{"category moderator workbench", moderatorToken, "", "moderation", "/moderation", 303},
		{"category moderator cannot enter admin", moderatorToken, "", "admin", "", 403},
		{"role manager is not a content moderator", token, "", "moderation", "", 403},
		{"ordinary user cannot moderate", ordinaryToken, "", "moderation", "", 403},
		{"course manager workspace", courseToken, "", "courseManagement", "/moderation/courses", 303},
		{"course reviews workspace", courseToken, "", "courseReviews", "/moderation/course-reviews", 303},
		{"forum moderator cannot manage courses", moderatorToken, "", "courseManagement", "", 403},
		{"role manager cannot review courses", token, "", "courseReviews", "", 403},
		{"course manager cannot moderate forum", courseToken, "", "moderation", "", 403},
		{"ordinary user cannot manage courses", ordinaryToken, "", "courseManagement", "", 403},
		{"unknown destination", token, "", "settings", "", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/auth/mobile-web-session?target="+tc.target+"&redirect=https://example.org", nil)
			if tc.bearer != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearer)
			}
			if tc.cookie != "" {
				req.AddCookie(&http.Cookie{Name: "access_token", Value: tc.cookie})
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status %d want %d: %s", res.Code, tc.status, res.Body.String())
			}
			if res.Header().Get("Referrer-Policy") != "no-referrer" {
				t.Fatal("session response can disclose its referrer")
			}
			if res.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("session response can be cached")
			}
			if tc.status != 303 {
				if len(res.Result().Cookies()) != 0 {
					t.Fatal("failed handoff set a cookie")
				}
				return
			}
			if res.Header().Get("Location") != tc.destination {
				t.Fatal("handoff must use fixed destination")
			}
			cookies := res.Result().Cookies()
			if len(cookies) != 1 || cookies[0].Value != tc.bearer || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/" {
				t.Fatal("missing protected session cookie")
			}
			if strings.Contains(res.Body.String(), token) || res.Header().Get("New-Token") != "" {
				t.Fatal("credential exposed outside HttpOnly cookie")
			}
		})
	}
}
