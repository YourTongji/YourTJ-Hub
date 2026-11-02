package routes

import (
	"net/http"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/gin-gonic/gin"
)

// Use apiRoute itself so these assertions include production CSRF, session,
// writable-account and shared interaction-limit ordering, not a copied chain.
func TestPushMiddlewareHTTPContract(t *testing.T) {
	operations := []struct{ path, body string }{
		{"/api/forum/push/subscribe", `{"subscription":{"endpoint":"https://fcm.googleapis.com/fcm/send/contract-middleware","keys":{"p256dh":"dGVzdC1wMjU2ZGg","auth":"dGVzdC1hdXRo"}},"lang":"zh"}`},
		{"/api/forum/push/unsubscribe", `{"endpoint":"https://fcm.googleapis.com/fcm/send/contract-middleware"}`},
		{"/api/forum/push/device/register", `{"platform":"ios","token":"synthetic-contract-device"}`},
		{"/api/forum/push/device/unregister", `{"token":"synthetic-contract-device"}`},
	}
	for _, operation := range operations {
		for _, guard := range []string{"session", "frozen", "pending", "csrf", "rate"} {
			t.Run(operation.path+"/"+guard, func(t *testing.T) {
				conn, _ := setupNotificationChatContractTest(t)
				router := gin.New()
				apiRoute(router)
				user := createHTTPContractUser(t, conn, contractTestID())
				switch guard {
				case "frozen":
					if err := conn.Model(user).Update("is_frozen", users.StatusFrozen).Error; err != nil {
						t.Fatal(err)
					}
				case "pending":
					security := emptySecurityConfig()
					security.EnableEmailVerification = true
					persistSecurityPolicyConfig(t, conn, security)
					if err := conn.Model(user).Update("is_activated", users.ActivationPending).Error; err != nil {
						t.Fatal(err)
					}
				case "rate":
					restrictContractRateLimit(t, conn, middleware.RateLimitInteract)
				}
				token := contractSessionToken(t, user)
				if guard == "session" {
					token = ""
				}
				if guard == "rate" {
					for attempt := 0; attempt < 5; attempt++ {
						response := serveAuthSecurityJSON(router, http.MethodPost, operation.path, operation.body, token)
						if response.Code != http.StatusOK || decodeContractEnvelope(t, response).Code != 0 {
							t.Fatalf("pre-limit attempt %d: %d %s", attempt+1, response.Code, response.Body.String())
						}
					}
				}
				serve := serveAuthSecurityJSON
				if guard == "csrf" {
					serve = serveAuthSecurityJSONWithCookie
				}
				response := serve(router, http.MethodPost, operation.path, operation.body, token)
				status, fixture := http.StatusForbidden, "account-frozen.json"
				switch guard {
				case "session":
					status, fixture = http.StatusUnauthorized, "auth-required.json"
				case "pending":
					fixture = "permission-email-required.json"
				case "csrf":
					fixture = "csrf-rejected.json"
				case "rate":
					status, fixture = http.StatusTooManyRequests, "account-close-rate-limited.json"
				}
				if response.Code != status {
					t.Fatalf("status %d, want %d: %s", response.Code, status, response.Body.String())
				}
				envelope := decodeContractEnvelope(t, response)
				assertFixtureEnvelope(t, envelope, contractFixture(t, fixture))
				if guard == "rate" {
					assertRetryAfter(t, response, envelope, middleware.RateLimitInteract)
				}
			})
		}
	}
}
