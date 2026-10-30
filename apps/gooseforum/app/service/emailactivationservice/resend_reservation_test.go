package emailactivationservice

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
)

func mustReserve(t *testing.T, userID uint64, now time.Time) quotaReservation {
	t.Helper()
	_, reservation, err := reserveQuota(userID, now)
	if err != nil {
		t.Fatal(err)
	}
	return reservation
}

func readQuota(t *testing.T, userID uint64, now time.Time) resendState {
	t.Helper()
	key, _ := resendKey(userID, now)
	data, err := kvstore.GetBytes(key)
	if err != nil {
		t.Fatal(err)
	}
	var state resendState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	return state
}

func TestReleaseQuotaPreservesOtherSends(t *testing.T) {
	for _, releaseOldest := range []bool{false, true} {
		t.Run(map[bool]string{false: "latest", true: "oldest"}[releaseOldest], func(t *testing.T) {
			userID := testUserID.Add(1)
			now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
			first := mustReserve(t, userID, now)
			second := mustReserve(t, userID, now.Add(time.Minute))
			failed, keptAt := second, now
			if releaseOldest {
				failed, keptAt = first, now.Add(time.Minute)
			}
			for range 2 { // duplicate compensation must be idempotent
				result, err := releaseQuota(failed)
				if err != nil || result.RemainingToday != 2 {
					t.Fatalf("release = %+v, %v", result, err)
				}
			}
			state := readQuota(t, userID, now)
			if state.Count != 1 || state.LastSentAt != keptAt.Unix() || len(state.Reservations) != 1 {
				t.Fatalf("wrong surviving charge: %+v", state)
			}
			if _, _, err := reserveQuota(userID, keptAt.Add(59*time.Second)); !errors.Is(err, ErrCooldown) {
				t.Fatalf("surviving cooldown lost: %v", err)
			}
			mustReserve(t, userID, keptAt.Add(time.Minute))
			mustReserve(t, userID, keptAt.Add(2*time.Minute))
			if _, _, err := reserveQuota(userID, keptAt.Add(3*time.Minute)); !errors.Is(err, ErrDailyLimit) {
				t.Fatalf("daily limit lost: %v", err)
			}
		})
	}
}

func TestReleaseQuotaLegacyState(t *testing.T) {
	userID := testUserID.Add(1)
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	key, ttl := resendKey(userID, now)
	legacy, err := json.Marshal(struct {
		Count int   `json:"count"`
		Last  int64 `json:"lastSentAt"`
	}{2, now.Unix()})
	if err != nil {
		t.Fatal(err)
	}
	if err := kvstore.SetBytes(key, legacy, ttl); err != nil {
		t.Fatal(err)
	}
	if _, _, err := reserveQuota(userID, now.Add(59*time.Second)); !errors.Is(err, ErrCooldown) {
		t.Fatalf("legacy cooldown = %v", err)
	}
	last := mustReserve(t, userID, now.Add(time.Minute))
	if _, _, err := reserveQuota(userID, now.Add(2*time.Minute)); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("legacy daily limit = %v", err)
	}
	if _, err := releaseQuota(last); err != nil {
		t.Fatal(err)
	}
	state := readQuota(t, userID, now)
	if state.Count != 2 || state.LastSentAt != now.Unix() {
		t.Fatalf("legacy quota erased by refund: %+v", state)
	}
	mustReserve(t, userID, now.Add(time.Minute))
}

func TestReleaseQuotaDayAndMissingKeyIsolation(t *testing.T) {
	userID := testUserID.Add(1)
	now := time.Date(2026, 10, 29, 23, 59, 59, 0, time.UTC)
	old := mustReserve(t, userID, now)
	newDay := now.Add(time.Second)
	mustReserve(t, userID, newDay)
	if _, err := releaseQuota(old); err != nil {
		t.Fatal(err)
	}
	if state := readQuota(t, userID, newDay); state.Count != 1 {
		t.Fatalf("previous-day refund changed today's quota: %+v", state)
	}
	// An absent original key (as after TTL expiry) must remain absent.
	old.key += ":expired"
	if _, err := releaseQuota(old); err != nil {
		t.Fatal(err)
	}
	if _, err := kvstore.GetBytes(old.key); !errors.Is(err, kvstore.ErrNotFound) {
		t.Fatalf("refund recreated expired key: %v", err)
	}
}

func TestQuotaConcurrentReserveAndRefund(t *testing.T) {
	userID := testUserID.Add(1)
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	const requests = 24
	var wg sync.WaitGroup
	reservations := make(chan quotaReservation, requests)
	errorsCh := make(chan error, requests)
	for range requests {
		wg.Go(func() {
			_, reservation, err := reserveQuota(userID, now)
			if err == nil {
				reservations <- reservation
			} else if !errors.Is(err, ErrCooldown) {
				errorsCh <- err
			}
		})
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("concurrent reserve: %v", err)
	}
	if len(reservations) != 1 {
		t.Fatalf("concurrent winners = %d, want 1", len(reservations))
	}
	failed := <-reservations
	// A delayed failure races a new request after the interval. Neither order
	// may remove the new request or refund the old reservation more than once.
	for range requests {
		wg.Go(func() {
			if _, err := releaseQuota(failed); err != nil {
				t.Errorf("concurrent refund: %v", err)
			}
		})
	}
	wg.Go(func() {
		if _, _, err := reserveQuota(userID, now.Add(time.Minute)); err != nil {
			t.Errorf("new reservation racing refund: %v", err)
		}
	})
	wg.Wait()
	state := readQuota(t, userID, now)
	if state.Count != 1 || len(state.Reservations) != 1 || state.LastSentAt != now.Add(time.Minute).Unix() {
		t.Fatalf("concurrent refund damaged new reservation: %+v", state)
	}
}

func TestReleaseQuotaAllFailuresAllowRetry(t *testing.T) {
	userID := testUserID.Add(1)
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC)
	var reservations []quotaReservation
	for i := range resendDailyLimit {
		reservations = append(reservations, mustReserve(t, userID, now.Add(time.Duration(i)*time.Minute)))
	}
	// Complete failures out of order, including the latest one first.
	for _, i := range []int{2, 0, 1} {
		if _, err := releaseQuota(reservations[i]); err != nil {
			t.Fatal(err)
		}
	}
	if state := readQuota(t, userID, now); state.Count != 0 || state.LastSentAt != 0 || len(state.Reservations) != 0 {
		t.Fatalf("failed sends still consume quota: %+v", state)
	}
	result, _, err := reserveQuota(userID, now.Add(2*time.Minute))
	if err != nil || result.RemainingToday != 2 {
		t.Fatalf("retry after all failures = %+v, %v", result, err)
	}
}

func TestExpiredRefundReceiptDoesNotExtendQuota(t *testing.T) {
	owner := testUserID.Add(1)
	now := time.Now()
	reservation := mustReserve(t, owner, now)
	reservation.expiresAt = now.Add(-time.Second)
	if _, err := releaseQuota(reservation); err != nil {
		t.Fatal(err)
	}
	if state := readQuota(t, owner, now); state.Count != 1 {
		t.Fatal("expired compensation changed the original quota")
	}
}
