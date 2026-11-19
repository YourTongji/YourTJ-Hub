package routes

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/eventNotification"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/unreadservice"
	"github.com/gin-gonic/gin"
)

const notificationPreferencesPath = "/api/forum/notification-preferences"

func TestNotificationBadgeCategoriesHTTP(t *testing.T) {
	conn, _ := setupNotificationChatContractTest(t)
	router := gin.New()
	apiRoute(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	like := eventNotification.Entity{UserId: owner.Id, EventType: eventNotification.EventTypeLike}
	if err := conn.Create(&like).Error; err != nil {
		t.Fatal(err)
	}
	unreadservice.Invalidate(owner.Id)
	if before := unreadservice.GetStatus(owner.Id); !before.Notifications || *before.NotificationCount != 1 {
		t.Fatal(before)
	}
	response := serveAuthSecurityJSON(router, http.MethodPut, notificationPreferencesPath, `{"badgeCategories":["replies"]}`, token)
	envelope := decodeContractEnvelope(t, response)
	if envelope.Code != 0 {
		t.Fatalf("category save failed: %s", response.Body.String())
	}
	var prefs struct {
		BadgeStyle      string
		BadgeCategories []string
	}
	if err := json.Unmarshal(envelope.Result, &prefs); err != nil {
		t.Fatal(err)
	}
	if prefs.BadgeStyle != "number" || !reflect.DeepEqual(prefs.BadgeCategories, []string{"replies"}) {
		t.Fatalf("preferences: %+v", prefs)
	}
	filtered := unreadservice.GetStatus(owner.Id)
	if filtered.Notifications || filtered.NotificationCount == nil || *filtered.NotificationCount != 0 || filtered.LatestUnreadId != like.Id {
		t.Fatalf("filtered badge or delivery signal: %+v", filtered)
	}
	// Old clients saving the style cannot reset the category selection.
	response = serveAuthSecurityJSON(router, http.MethodPut, notificationPreferencesPath, `{"badgeStyle":"dot"}`, token)
	if decodeContractEnvelope(t, response).Code != 0 {
		t.Fatal(response.Body.String())
	}
	filtered = unreadservice.GetStatus(owner.Id)
	if filtered.Notifications || filtered.BadgeStyle != "dot" {
		t.Fatalf("style reset categories: %+v", filtered)
	}
	// Selection does not acknowledge or hide events from the notification list.
	if count, err := eventNotification.GetUnreadCount(owner.Id); err != nil || count != 1 {
		t.Fatalf("read facts: %d %v", count, err)
	}
	reply := eventNotification.Entity{UserId: owner.Id, EventType: eventNotification.EventTypeComment}
	if err := conn.Create(&reply).Error; err != nil {
		t.Fatal(err)
	}
	unreadservice.Invalidate(owner.Id)
	if got := unreadservice.GetStatus(owner.Id); !got.Notifications || *got.NotificationCount != 1 {
		t.Fatalf("reply badge: %+v", got)
	}
	response = serveAuthSecurityJSON(router, http.MethodPut, notificationPreferencesPath, `{"badgeCategories":[]}`, token)
	if decodeContractEnvelope(t, response).Code != 0 {
		t.Fatal(response.Body.String())
	}
	if got := unreadservice.GetStatus(owner.Id); got.Notifications || *got.NotificationCount != 0 {
		t.Fatalf("empty selection: %+v", got)
	}
}

func TestNotificationPreferencesHTTPContract(t *testing.T) {
	conn, _ := setupNotificationChatContractTest(t)
	router := gin.New()
	apiRoute(router)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	token := contractSessionToken(t, owner)
	before := unreadservice.GetStatus(owner.Id)
	response := serveAuthSecurityJSON(router, http.MethodGet, notificationPreferencesPath, "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "notification-preferences-number.json"))
	response = serveAuthSecurityJSON(router, http.MethodPut, notificationPreferencesPath, `{"badgeStyle":"dot"}`, token)
	if response.Code != http.StatusOK {
		t.Fatal(response.Body.String())
	}
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "notification-preferences-dot.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, notificationPreferencesPath, "", token)
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "notification-preferences-dot.json"))
	response = serveAuthSecurityJSON(router, http.MethodGet, notificationPreferencesPath, "", contractSessionToken(t, other))
	assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, "notification-preferences-number.json"))
	after := unreadservice.GetStatus(owner.Id)
	if after.BadgeStyle != users.UnreadBadgeDot || before.Notifications != after.Notifications || before.Messages != after.Messages || *before.NotificationCount != *after.NotificationCount || *before.MessageCount != *after.MessageCount {
		t.Fatal("preference changed unread facts or left the cached style stale")
	}
}

func TestNotificationPreferencesRequestBoundaries(t *testing.T) {
	for _, tc := range []struct {
		body, fixture string
		status        int
	}{
		{`{"badgeStyle":"unknown"}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{"badgeCategories":["all"]}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{"badgeCategories":["replies","replies"]}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{"badgeCategories":null}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{"badgeStyle":"dot","badgeCategories":["invalid"]}`, "notification-preferences-invalid.json", http.StatusOK},
		{`{"badgeCategories":"replies"}`, "parse-failed.json", http.StatusBadRequest},
		{`{"badgeCategories":[1]}`, "parse-failed.json", http.StatusBadRequest},
		{`{"badgeStyle":false}`, "parse-failed.json", http.StatusBadRequest},
		{`{"badgeStyle":"dot","userId":999}`, "parse-failed.json", http.StatusBadRequest},
		{`null`, "parse-failed.json", http.StatusBadRequest},
		{`{"badgeStyle":"dot"} {}`, "parse-failed.json", http.StatusBadRequest},
		{`{"badgeStyle":"` + strings.Repeat("a", 5<<10) + `"}`, "parse-failed.json", http.StatusBadRequest},
	} {
		t.Run(tc.body[:min(len(tc.body), 45)], func(t *testing.T) {
			conn, _ := setupNotificationChatContractTest(t)
			router := gin.New()
			apiRoute(router)
			owner := createHTTPContractUser(t, conn, contractTestID())
			response := serveAuthSecurityJSON(router, http.MethodPut, notificationPreferencesPath, tc.body, contractSessionToken(t, owner))
			if response.Code != tc.status {
				t.Fatalf("status %d: %s", response.Code, response.Body.String())
			}
			assertFixtureEnvelope(t, decodeContractEnvelope(t, response), contractFixture(t, tc.fixture))
			style, err := users.GetUnreadBadgeStyle(owner.Id)
			if err != nil || style != users.UnreadBadgeNumber {
				t.Fatal("rejected update changed storage")
			}
		})
	}
}

func TestNotificationPreferencesMiddleware(t *testing.T) {
	for _, guard := range []string{"get-session", "put-session", "frozen", "csrf", "pending-allowed", "rate"} {
		t.Run(guard, func(t *testing.T) {
			conn, _ := setupNotificationChatContractTest(t)
			router := gin.New()
			apiRoute(router)
			owner := createHTTPContractUser(t, conn, contractTestID())
			if guard == "frozen" {
				if err := conn.Model(owner).Update("is_frozen", users.StatusFrozen).Error; err != nil {
					t.Fatal(err)
				}
			}
			if guard == "pending-allowed" {
				security := emptySecurityConfig()
				security.EnableEmailVerification = true
				persistSecurityPolicyConfig(t, conn, security)
				if err := conn.Model(owner).Update("is_activated", users.ActivationPending).Error; err != nil {
					t.Fatal(err)
				}
			}
			if guard == "rate" {
				restrictContractRateLimit(t, conn, middleware.RateLimitInteract)
			}
			token := contractSessionToken(t, owner)
			method := http.MethodPut
			if strings.HasSuffix(guard, "session") {
				token = ""
			}
			if guard == "get-session" {
				method = http.MethodGet
			}
			if guard == "rate" {
				for i := 0; i < 5; i++ {
					response := serveAuthSecurityJSON(router, method, notificationPreferencesPath, `{"badgeStyle":"dot"}`, token)
					if response.Code != http.StatusOK || decodeContractEnvelope(t, response).Code != 0 {
						t.Fatal(response.Body.String())
					}
				}
			}
			serve := serveAuthSecurityJSON
			if guard == "csrf" {
				serve = serveAuthSecurityJSONWithCookie
			}
			response := serve(router, method, notificationPreferencesPath, `{"badgeStyle":"dot"}`, token)
			status, fixture := http.StatusUnauthorized, "auth-required.json"
			switch guard {
			case "frozen":
				status, fixture = http.StatusForbidden, "account-frozen.json"
			case "csrf":
				status, fixture = http.StatusForbidden, "csrf-rejected.json"
			case "pending-allowed":
				status, fixture = http.StatusOK, "notification-preferences-dot.json"
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
