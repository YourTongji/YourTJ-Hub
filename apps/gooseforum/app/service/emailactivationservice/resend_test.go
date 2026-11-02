package emailactivationservice

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/kvstore"
)

var testUserID atomic.Uint64

func TestMain(m *testing.M) {
	path, err := os.MkdirTemp("", "resend-quota-test-")
	if err != nil {
		panic(err)
	}
	preferences.Set("badger.path", path)
	code := m.Run()
	kvstore.Close()
	_ = os.RemoveAll(path)
	os.Exit(code)
}

func TestReserveCooldownAndDailyLimit(t *testing.T) {
	userID := testUserID.Add(1)
	now := time.Date(2026, 6, 2, 10, 0, 0, 0, time.Local)
	first, _, err := reserveQuota(userID, now)
	if err != nil {
		t.Fatalf("reserveQuota() first error = %v", err)
	}
	if first.RemainingToday != 2 {
		t.Fatalf("reserveQuota() first remaining = %d, want 2", first.RemainingToday)
	}

	cooldown, _, err := reserveQuota(userID, now.Add(30*time.Second))
	if !errors.Is(err, ErrCooldown) {
		t.Fatalf("reserveQuota() cooldown error = %v, want ErrCooldown", err)
	}
	if cooldown.RetryAfterSeconds <= 0 {
		t.Fatalf("reserveQuota() retryAfterSeconds = %d, want positive", cooldown.RetryAfterSeconds)
	}

	second, _, err := reserveQuota(userID, now.Add(61*time.Second))
	if err != nil {
		t.Fatalf("reserveQuota() second send error = %v", err)
	}
	if second.RemainingToday != 1 {
		t.Fatalf("reserveQuota() second remaining = %d, want 1", second.RemainingToday)
	}
	third, _, err := reserveQuota(userID, now.Add(122*time.Second))
	if err != nil {
		t.Fatalf("reserveQuota() third send error = %v", err)
	}
	if third.RemainingToday != 0 {
		t.Fatalf("reserveQuota() third remaining = %d, want 0", third.RemainingToday)
	}
	if _, _, err = reserveQuota(userID, now.Add(183*time.Second)); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("reserveQuota() daily limit error = %v, want ErrDailyLimit", err)
	}
}
