package routes

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/middleware"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/tokenservice"
	"github.com/gin-gonic/gin"
)

// The empty current address is a synthetic account fixture. Staging then uses
// the real password-protected email-change route, not a production account.
func TestActivationRejectsRevokedSessionWithFreshPendingEmail(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(user).Update("email", "").Error; err != nil {
		t.Fatal(err)
	}
	user.Email = ""
	stageTwoPhaseEmailChange(t, router, user, fmt.Sprintf("pending-%d@example.test", user.Id))
	before, err := users.Get(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if before.FreshPendingEmail(time.Now()) == "" {
		t.Fatal("fixture has no fresh pending email")
	}

	// Match the process-wide issuer key, as in a running server; do not use a
	// token signed under a key that the session middleware cannot accept.
	withRouteTestSigningKey(t, string(jwtopt.Std().SigningKey))
	token, jti, err := jwtopt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, jti, "activation-test", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.RevokeByJti(user.Id, jti); err != nil {
		t.Fatal(err)
	}
	if sessionservice.GetValidByJti(jti) != nil {
		t.Fatal("fixture session is still live")
	}

	assertActivationRejectedWithoutMutation(t, router, token, before)
}

func assertActivationRejectedWithoutMutation(t *testing.T, router http.Handler, token string, before users.EntityComplete) {
	t.Helper()
	response := getActivatePage(t, router, token)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `class="result fail"`) {
		t.Error("non-activation token did not render a failed activation result")
	}
	if len(response.Result().Cookies()) != 0 || response.Header().Get("New-Token") != "" {
		t.Error("activation issued login credentials")
	}
	after, err := users.Get(before.Id)
	if err != nil {
		t.Fatal(err)
	}
	// Includes pending/activation timestamps, password, version, freeze and roles.
	if !reflect.DeepEqual(before, after) {
		t.Error("rejected activation changed account state")
	}
}

func TestActivationRejectsCrossPurposeAcrossEmailStates(t *testing.T) {
	for _, state := range []string{"fresh pending", "no pending", "expired pending", "nonempty current email"} {
		for _, purpose := range []string{"session", "totp_challenge", "stale session version"} {
			t.Run(state+"/"+purpose, func(t *testing.T) {
				conn, router := setupTwoPhaseEmailTest(t)
				withRouteTestSigningKey(t, string(jwtopt.Std().SigningKey))
				user := createHTTPContractUser(t, conn, contractTestID())
				email := ""
				if state == "nonempty current email" {
					email = user.Email
				}
				var pendingAt *time.Time
				pending := ""
				if state != "no pending" {
					at := time.Now().Add(-time.Hour)
					if state == "expired pending" {
						at = time.Now().Add(-8 * 24 * time.Hour)
					}
					pendingAt = &at
					pending = fmt.Sprintf("matrix-%d@example.test", user.Id)
				}
				if err := conn.Model(user).Updates(map[string]any{
					"email": email, "pending_email": pending, "pending_email_at": pendingAt, "token_version": 7,
				}).Error; err != nil {
					t.Fatal(err)
				}
				before, err := users.Get(user.Id)
				if err != nil {
					t.Fatal(err)
				}
				var token string
				if purpose == "totp_challenge" {
					token, err = jwtopt.CreateChallengeToken(user.Id, 7, purpose, 5*time.Minute)
				} else {
					version := uint64(7)
					if purpose == "stale session version" {
						version = 6
					}
					token, _, err = jwtopt.CreateSessionToken(user.Id, version)
				}
				if err != nil {
					t.Fatal(err)
				}
				assertActivationRejectedWithoutMutation(t, router, token, before)
			})
		}
	}
}

// This deliberately synthetic unactivated, empty-email fixture establishes only
// the gate consequence, not prevalence or reachability through registration.
func TestActivationMisuseDoesNotLiftEmailOrFreezeGate(t *testing.T) {
	for _, frozen := range []bool{false, true} {
		t.Run(fmt.Sprintf("frozen=%v", frozen), func(t *testing.T) {
			conn, router := setupTwoPhaseEmailTest(t)
			withRouteTestSigningKey(t, string(jwtopt.Std().SigningKey))
			user := createHTTPContractUser(t, conn, contractTestID())
			now := time.Now()
			updates := map[string]any{"email": "", "is_activated": users.ActivationPending,
				"pending_email": fmt.Sprintf("gate-%d@example.test", user.Id), "pending_email_at": now}
			if frozen {
				updates["is_frozen"] = users.StatusFrozen
			}
			if err := conn.Model(user).Updates(updates).Error; err != nil {
				t.Fatal(err)
			}
			router.POST("/test/write", middleware.JWTAuthCheck, middleware.CheckWritableAccount,
				func(c *gin.Context) { c.Status(http.StatusNoContent) })
			session := contractSessionToken(t, user)
			want := "permission.emailRequired"
			if frozen {
				want = "permission.userFrozen"
			}
			checkGate := func() {
				t.Helper()
				response := serveJSON(router, "/test/write", `{}`, session)
				if response.Code != http.StatusForbidden || decodeContractEnvelope(t, response).MessageCode != want {
					t.Fatalf("write gate changed: status=%d body=%s", response.Code, response.Body.String())
				}
			}
			checkGate()
			before, err := users.Get(user.Id)
			if err != nil {
				t.Fatal(err)
			}
			challenge, err := jwtopt.CreateChallengeToken(user.Id, user.TokenVersion, "totp_challenge", 5*time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			assertActivationRejectedWithoutMutation(t, router, challenge, before)
			checkGate()
		})
	}
}

func TestActivationCanConfirmPendingEmailForEmptyEmailAccount(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupTwoPhaseEmailTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	if err := conn.Model(user).Update("email", "").Error; err != nil {
		t.Fatal(err)
	}
	user.Email = ""
	pending := fmt.Sprintf("legitimate-%d@example.test", user.Id)
	stageTwoPhaseEmailChange(t, router, user, pending)
	token, err := tokenservice.GenerateActivationToken(user.Id, pending)
	if err != nil {
		t.Fatal(err)
	}
	response := getActivatePage(t, router, token)
	after, err := users.Get(user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(response.Body.String(), `class="result success"`) || after.Email != pending ||
		after.PendingEmail != "" || after.PendingEmailAt != nil || after.IsActivated != users.ActivationSuccess {
		t.Fatal("legitimate pending-email confirmation failed")
	}
	if len(response.Result().Cookies()) != 0 || response.Header().Get("New-Token") != "" {
		t.Fatal("email confirmation issued login credentials")
	}
}
