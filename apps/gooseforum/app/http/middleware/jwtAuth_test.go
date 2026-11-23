package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	jwt "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/jwtopt"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pointsRecord"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPoints"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userStatistics"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userTotpChallenges"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/userservice"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func setupSessionAuthTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(
		&users.EntityComplete{},
		&userPoints.Entity{},
		&pointsRecord.Entity{},
		&userStatistics.Entity{},
		&userSessions.Entity{},
		&userTotpChallenges.Entity{},
	); err != nil {
		t.Fatalf("migrate tables: %v", err)
	}
	conn.Where("1 = 1").Delete(&userTotpChallenges.Entity{})
	conn.Where("1 = 1").Delete(&userSessions.Entity{})
	conn.Where("1 = 1").Delete(&userStatistics.Entity{})
	conn.Where("1 = 1").Delete(&pointsRecord.Entity{})
	conn.Where("1 = 1").Delete(&userPoints.Entity{})
	conn.Where("1 = 1").Delete(&users.EntityComplete{})
}

func requestWithJWT(token string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(JWTAuthCheck)
	router.GET("/", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestJWTAuthRejectsTokenWithoutSessionRecord(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("sessionless", "password", "sessionless@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	// Token signed with a jti that has no user_sessions row.
	token, _, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken() error = %v", err)
	}
	recorder := requestWithJWT(token)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no session row)", recorder.Code)
	}
}

func TestJWTAuthRejectsRevokedSession(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("revoked", "password", "revoked@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	token, jti, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatalf("CreateSessionToken() error = %v", err)
	}
	if err := sessionservice.Create(user.Id, jti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatalf("sessionservice.Create() error = %v", err)
	}

	// Before revocation the token is accepted.
	if recorder := requestWithJWT(token); recorder.Code != http.StatusOK {
		t.Fatalf("pre-revoke status = %d, want 200", recorder.Code)
	}

	// After revocation the token is rejected immediately.
	if err := sessionservice.RevokeByID(user.Id, sessionservice.GetValidByJti(jti).Id); err != nil {
		t.Fatalf("RevokeByID() error = %v", err)
	}
	if recorder := requestWithJWT(token); recorder.Code != http.StatusUnauthorized {
		t.Fatalf("post-revoke status = %d, want 401", recorder.Code)
	}
}

func TestJWTAuthRejectsChallengeToken(t *testing.T) {
	setupSessionAuthTestDB(t)

	user, err := userservice.CreateUser("challenge", "password", "challenge@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	challenge, err := jwt.CreateChallengeToken(user.Id, user.TokenVersion, jwt.PurposeTotpChallenge, 5*time.Minute)
	if err != nil {
		t.Fatalf("CreateChallengeToken() error = %v", err)
	}
	recorder := requestWithJWT(challenge)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (challenge token has no session)", recorder.Code)
	}
}

func TestJWTAuthSlidesActiveSessionAndEnforcesMaxLifetime(t *testing.T) {
	setupSessionAuthTestDB(t)

	prevValid := preferences.GetInt64("jwtopt.validTime", int64(jwt.DefaultValidTime/time.Second))
	prevRefresh := preferences.GetInt64("jwtopt.refreshInterval", int64(jwt.DefaultRefreshInterval/time.Second))
	prevMax := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(jwt.DefaultMaxSessionLifetime/time.Second))
	preferences.Set("jwtopt.validTime", int64(30*86400))
	preferences.Set("jwtopt.refreshInterval", int64(86400))
	preferences.Set("jwtopt.maxSessionLifetime", int64(180*86400))
	t.Cleanup(func() {
		preferences.Set("jwtopt.validTime", prevValid)
		preferences.Set("jwtopt.refreshInterval", prevRefresh)
		preferences.Set("jwtopt.maxSessionLifetime", prevMax)
	})

	user, err := userservice.CreateUser("slider", "password", "slider@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// 1. Fresh token within refreshInterval emits no New-Token or Set-Cookie.
	freshToken, freshJti, err := jwt.CreateSessionToken(user.Id, user.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, freshJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	freshRec := requestWithJWT(freshToken)
	if freshRec.Code != http.StatusOK {
		t.Fatalf("fresh status = %d, want 200", freshRec.Code)
	}
	if freshRec.Header().Get("New-Token") != "" || freshRec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("fresh token unexpectedly refreshed: New-Token=%q Set-Cookie=%q",
			freshRec.Header().Get("New-Token"), freshRec.Header().Get("Set-Cookie"))
	}

	// 2. Active token aged > 24h (28d remaining) slides forward, emits New-Token +
	// aligned cookie Max-Age, and updates user_sessions.expires_at.
	agedJti, err := jwt.GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	agedToken, err := jwt.Std().CreateToken(jwt.CustomClaims{
		UserId:           user.Id,
		TokenVersion:     user.TokenVersion,
		Jti:              agedJti,
		RegisteredClaims: jwt.GetBaseRegisteredClaims(28 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, agedJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := userSessions.UpdateExpiresAtByJti(agedJti, time.Now().Add(28*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	agedRec := requestWithJWT(agedToken)
	if agedRec.Code != http.StatusOK {
		t.Fatalf("aged status = %d, want 200", agedRec.Code)
	}
	renewed := agedRec.Header().Get("New-Token")
	if renewed == "" || renewed == agedToken {
		t.Fatalf("expected renewed New-Token header, got %q", renewed)
	}
	row := sessionservice.GetValidByJti(agedJti)
	if row == nil || time.Until(row.ExpiresAt) < 29*24*time.Hour {
		t.Fatalf("session expires_at not extended: %+v", row)
	}

	// 3. Session near maxSessionLifetime clamps cookie Max-Age and session row expiry.
	nearCapJti, err := jwt.GenerateJti()
	if err != nil {
		t.Fatal(err)
	}
	nearCapToken, err := jwt.Std().CreateToken(jwt.CustomClaims{
		UserId:           user.Id,
		TokenVersion:     user.TokenVersion,
		Jti:              nearCapJti,
		RegisteredClaims: jwt.GetBaseRegisteredClaims(5 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sessionservice.Create(user.Id, nearCapJti, "test-agent", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := db.Connect().Exec("UPDATE user_sessions SET created_at = ? WHERE jti = ?",
		time.Now().Add(-170*24*time.Hour), nearCapJti).Error; err != nil {
		t.Fatal(err)
	}
	nearCapRec := requestWithJWT(nearCapToken)
	if nearCapRec.Code != http.StatusOK {
		t.Fatalf("near-cap status = %d, want 200", nearCapRec.Code)
	}
	if nearCapRec.Header().Get("New-Token") == "" {
		t.Fatal("near-cap request should emit clamped New-Token")
	}
	var cookieMaxAge int
	for _, ck := range nearCapRec.Result().Cookies() {
		if ck.Name == "access_token" {
			cookieMaxAge = ck.MaxAge
		}
	}
	if cookieMaxAge <= 0 || cookieMaxAge > 10*86400+60 {
		t.Fatalf("near-cap cookie Max-Age = %d, want <= ~10d", cookieMaxAge)
	}

	// 4. Session past maxSessionLifetime returns 401.
	if err := db.Connect().Exec("UPDATE user_sessions SET created_at = ? WHERE jti = ?",
		time.Now().Add(-181*24*time.Hour), nearCapJti).Error; err != nil {
		t.Fatal(err)
	}
	if rec := requestWithJWT(nearCapToken); rec.Code != http.StatusUnauthorized {
		t.Fatalf("past-cap status = %d, want 401", rec.Code)
	}
}

func TestJWTAuthRenewalConsistencyOnPersistFailureMissAndCatchUp(t *testing.T) {
	setupSessionAuthTestDB(t)

	prevValid := preferences.GetInt64("jwtopt.validTime", int64(jwt.DefaultValidTime/time.Second))
	prevRefresh := preferences.GetInt64("jwtopt.refreshInterval", int64(jwt.DefaultRefreshInterval/time.Second))
	prevMax := preferences.GetInt64("jwtopt.maxSessionLifetime", int64(jwt.DefaultMaxSessionLifetime/time.Second))
	preferences.Set("jwtopt.validTime", int64(30*86400))
	preferences.Set("jwtopt.refreshInterval", int64(86400))
	preferences.Set("jwtopt.maxSessionLifetime", int64(180*86400))
	t.Cleanup(func() {
		preferences.Set("jwtopt.validTime", prevValid)
		preferences.Set("jwtopt.refreshInterval", prevRefresh)
		preferences.Set("jwtopt.maxSessionLifetime", prevMax)
	})

	user, err := userservice.CreateUser("slider-persist", "password", "slider-persist@example.com", false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	mintAgedSession := func(ttl time.Duration) (string, string) {
		jti, err := jwt.GenerateJti()
		if err != nil {
			t.Fatal(err)
		}
		tok, err := jwt.Std().CreateToken(jwt.CustomClaims{
			UserId:           user.Id,
			TokenVersion:     user.TokenVersion,
			Jti:              jti,
			RegisteredClaims: jwt.GetBaseRegisteredClaims(ttl),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := sessionservice.Create(user.Id, jti, "test-agent", "127.0.0.1"); err != nil {
			t.Fatal(err)
		}
		if err := userSessions.UpdateExpiresAtByJti(jti, time.Now().Add(ttl).Truncate(time.Second)); err != nil {
			t.Fatal(err)
		}
		return tok, jti
	}

	conn := db.Connect()
	const cbName = "test:jwt-renewal-persist-hook"
	var mode atomic.Int32 // 0=normal, 1=fail update, 2=delete row before update
	var updateCount atomic.Int64
	var targetJti atomic.Value
	targetJti.Store("")
	if err := conn.Callback().Update().Before("gorm:begin_transaction").Register(cbName, func(tx *gorm.DB) {
		if tx.Statement.Table != "user_sessions" {
			return
		}
		updateCount.Add(1)
		switch mode.Load() {
		case 1:
			_ = tx.AddError(errors.New("injected user_sessions update error"))
		case 2:
			if jti, _ := targetJti.Load().(string); jti != "" {
				_ = conn.Exec("DELETE FROM user_sessions WHERE jti = ?", jti).Error
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Callback().Update().Remove(cbName) })

	// 1. DB write failure must suppress New-Token and Set-Cookie so the client
	// keeps its old token and retries renewal later instead of outliving DB expiry.
	tok1, jti1 := mintAgedSession(6 * 24 * time.Hour)
	mode.Store(1)
	failRec := requestWithJWT(tok1)
	if failRec.Code != http.StatusOK {
		t.Fatalf("write-failure status = %d, want 200", failRec.Code)
	}
	if failRec.Header().Get("New-Token") != "" || failRec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("write-failure emitted credentials: New-Token=%q Set-Cookie=%q",
			failRec.Header().Get("New-Token"), failRec.Header().Get("Set-Cookie"))
	}
	if row := sessionservice.GetValidByJti(jti1); row == nil || time.Until(row.ExpiresAt) > 7*24*time.Hour {
		t.Fatalf("session row unexpectedly extended on write error: %+v", row)
	}

	// Once DB recovers, the same old token renews and persists cleanly.
	mode.Store(0)
	okRec := requestWithJWT(tok1)
	if okRec.Code != http.StatusOK || okRec.Header().Get("New-Token") == "" || okRec.Header().Get("Set-Cookie") == "" {
		t.Fatalf("recovery request failed to renew: status=%d New-Token=%q Set-Cookie=%q",
			okRec.Code, okRec.Header().Get("New-Token"), okRec.Header().Get("Set-Cookie"))
	}

	// 2. Old client catching up with the same stale JWT after DB already persisted
	// the 30d expiry emits New-Token without issuing another DB UPDATE.
	beforeUpdates := updateCount.Load()
	catchUpRec := requestWithJWT(tok1)
	if catchUpRec.Code != http.StatusOK || catchUpRec.Header().Get("New-Token") == "" {
		t.Fatalf("catch-up request did not return New-Token: status=%d New-Token=%q",
			catchUpRec.Code, catchUpRec.Header().Get("New-Token"))
	}
	if gotUpdates := updateCount.Load() - beforeUpdates; gotUpdates != 0 {
		t.Fatalf("catch-up triggered %d user_sessions updates, want 0", gotUpdates)
	}

	// 3. Concurrent revocation (0 rows affected + row missing) must suppress credentials.
	tok2, jti2 := mintAgedSession(6 * 24 * time.Hour)
	targetJti.Store(jti2)
	mode.Store(2)
	missRec := requestWithJWT(tok2)
	mode.Store(0)
	if missRec.Code != http.StatusOK {
		t.Fatalf("missed-row status = %d, want 200", missRec.Code)
	}
	if missRec.Header().Get("New-Token") != "" || missRec.Header().Get("Set-Cookie") != "" {
		t.Fatalf("concurrently revoked row emitted credentials: New-Token=%q Set-Cookie=%q",
			missRec.Header().Get("New-Token"), missRec.Header().Get("Set-Cookie"))
	}
}
