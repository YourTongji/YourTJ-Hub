package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/taskQueue"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
)

func TestActivationStatusIsOwnerOnlyAndReadOnly(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupAccountContractTest(t)
	enableContractEmailVerification(t, conn)
	owner := createHTTPContractUser(t, conn, contractTestID())
	other := createHTTPContractUser(t, conn, contractTestID())
	now := time.Now()
	if err := conn.Model(owner).Updates(map[string]any{"is_activated": users.ActivationPending, "pending_email": "staged@example.test", "pending_email_at": now}).Error; err != nil {
		t.Fatal(err)
	}
	get := func(token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/resend-activation-email?userId=%d", other.Id), nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	if rec := get(""); rec.Code != 401 {
		t.Fatalf("anonymous status=%d, want401", rec.Code)
	}
	token := contractSessionToken(t, owner)
	var before int64
	if err := conn.Model(&taskQueue.Entity{}).Count(&before).Error; err != nil {
		t.Fatal(err)
	}
	for range 2 {
		rec := get(token)
		if rec.Code != 200 || rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatalf("owner status: %d %v %s", rec.Code, rec.Header(), rec.Body.String())
		}
		var body struct {
			Result struct {
				CurrentEmail   string
				TargetEmail    string
				TargetKind     string
				State          string
				RemainingToday int
			}
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		result := body.Result
		if result.CurrentEmail != owner.Email || result.TargetEmail != "staged@example.test" || result.TargetKind != "pending" || result.State != "ready" || result.RemainingToday != 3 {
			t.Fatalf("wrong owner status: %+v", result)
		}
	}
	var count int64
	if err := conn.Model(&taskQueue.Entity{}).Count(&count).Error; err != nil || count != before {
		t.Fatalf("GET enqueued work: %d %v", count, err)
	}
	key := fmt.Sprintf("activation-email:resend:%d:%s", owner.Id, now.Format("20060102"))
	quota := []byte(fmt.Sprintf(`{"count":2,"lastSentAt":%d}`, now.Unix()))
	if err := kvstore.SetBytes(key, quota, time.Hour); err != nil {
		t.Fatal(err)
	}
	cooldown := get(token)
	var throttled struct {
		Result struct {
			State             string
			RemainingToday    int
			RetryAfterSeconds int
		}
	}
	if err := json.Unmarshal(cooldown.Body.Bytes(), &throttled); err != nil {
		t.Fatal(err)
	}
	if throttled.Result.State != "cooldown" || throttled.Result.RemainingToday != 1 || throttled.Result.RetryAfterSeconds <= 0 {
		t.Fatalf("quota read mismatch: %s", cooldown.Body.String())
	}
	after, err := kvstore.GetBytes(key)
	if err != nil || !bytes.Equal(after, quota) {
		t.Fatalf("GET changed quota: %s %v", after, err)
	}
	if err := conn.Model(owner).Updates(map[string]any{"pending_email_at": now.Add(-users.PendingEmailWindow - time.Minute), "is_activated": 1}).Error; err != nil {
		t.Fatal(err)
	}
	rec := get(token)
	var body struct {
		Result struct {
			State        string
			TargetEmail  string
			PendingEmail string
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.State != "verified" || body.Result.TargetEmail != "" || body.Result.PendingEmail != "" {
		t.Fatalf("stale pending target: %s", rec.Body.String())
	}
}

func TestActivationStatusFailureIsUnavailable(t *testing.T) {
	useContractTempKV(t)
	conn, router := setupAccountContractTest(t)
	user := createHTTPContractUser(t, conn, contractTestID())
	key := fmt.Sprintf("activation-email:resend:%d:%s", user.Id, time.Now().Format("20060102"))
	if err := kvstore.SetBytes(key, []byte("{broken"), time.Hour); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/resend-activation-email", nil)
	req.Header.Set("Authorization", "Bearer "+contractSessionToken(t, user))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != 503 {
		t.Fatalf("unavailable quota=%d %s", rec.Code, rec.Body.String())
	}
}
