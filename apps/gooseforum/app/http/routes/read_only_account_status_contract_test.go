package routes

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/migration"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
)

func TestReadOnlyAccountGETStatusContract(t *testing.T) {
	conn, _ := setupAccountContractTest(t)
	if err := conn.AutoMigrate(migration.SchemaModels()...); err != nil {
		t.Fatal(err)
	}
	enableContractEmailVerification(t, conn)
	router := gin.New()
	RegisterByGin(router)

	for _, state := range []string{"active", "frozen", "pending"} {
		t.Run(state, func(t *testing.T) {
			owner := createHTTPContractUser(t, conn, contractTestID())
			switch state {
			case "frozen":
				if err := conn.Model(owner).Update("is_frozen", users.StatusFrozen).Error; err != nil {
					t.Fatal(err)
				}
			case "pending":
				if err := conn.Model(owner).Update("is_activated", users.ActivationPending).Error; err != nil {
					t.Fatal(err)
				}
			}
			userservice.InvalidateUserInfoCache(owner.Id)
			token := contractSessionToken(t, owner)
			topicID := contractTestID()
			createContractPublishedTopic(t, conn, topicID, contractTestID(), owner.Id)
			paths := []string{
				"/api/user/decorations",
				"/api/forum/anonymous/state",
				fmt.Sprintf("/api/forum/content/appeals?kind=topic&id=%d", topicID),
			}
			for _, path := range paths {
				// GET bypasses cookie CSRF; account write eligibility is not a read guard.
				req := httptest.NewRequest(http.MethodGet, path, nil)
				req.AddCookie(&http.Cookie{Name: "access_token", Value: token})
				req.Header.Set("Origin", "https://cross-origin.example")
				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)
				if rec.Code != http.StatusOK || decodeContractEnvelope(t, rec).Code != 0 {
					t.Fatalf("%s GET %s: %d %s", state, path, rec.Code, rec.Body.String())
				}
				if unauth := serveAuthSecurityJSON(router, http.MethodGet, path, "", ""); unauth.Code != http.StatusUnauthorized {
					t.Fatalf("unauthenticated GET %s: %d %s", path, unauth.Code, unauth.Body.String())
				}
			}
			if admin := serveAuthSecurityJSON(router, http.MethodGet, "/api/admin/decorations", "", token); admin.Code != http.StatusForbidden {
				t.Fatalf("%s admin GET lost its guard: %d %s", state, admin.Code, admin.Body.String())
			}
		})
	}
}
