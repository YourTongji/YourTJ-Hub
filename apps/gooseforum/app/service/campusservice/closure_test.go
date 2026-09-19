package campusservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"gorm.io/gorm"
)

func TestAccountClosureRevokesReadableCredentialsAfterAccountClosure(t *testing.T) {
	for _, scenario := range []string{"readable", "unreadable", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			s, p := setup(t)
			db := dbconnect.Connect()
			if err := db.AutoMigrate(&campus.Binding{}); err != nil {
				t.Fatal(err)
			}
			s.store = campus.Store{DB: db}
			const id = uint64(987123)
			t.Cleanup(func() { db.Where("user_id = ?", id).Delete(&campus.Binding{}) })
			old := service.Swap(s)
			t.Cleanup(func() { service.Store(old) })
			if scenario != "absent" {
				bind(t, s, id)
			}
			if scenario == "unreadable" {
				db.Model(&campus.Binding{}).Where("user_id = ?", id).Update("sealed", "unreadable")
			}
			if scenario != "absent" {
				prepare(t, s, id, "reauthorize")
			}
			called := false
			if err := CloseForUser(id, func() error {
				called = true
				if scenario != "absent" {
					if _, err := s.store.Get(id); err != nil {
						t.Fatal("credentials deleted before account closure")
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if _, err := s.store.Get(id); !errors.Is(err, gorm.ErrRecordNotFound) {
				t.Fatal("credentials survived account closure")
			}
			if s.pending[id] != nil {
				t.Fatal("pending credentials survived closure")
			}
			if !called {
				t.Fatal("closure callback not called")
			}
			want := int32(0)
			if scenario == "readable" {
				want = 1
			}
			if p.revocations.Load() != want {
				t.Fatalf("revocations = %d, want %d", p.revocations.Load(), want)
			}
		})
	}
}

func TestAccountClosureFailureKeepsCampusCredentials(t *testing.T) {
	s, p := setup(t)
	db := dbconnect.Connect()
	if err := db.AutoMigrate(&campus.Binding{}); err != nil {
		t.Fatal(err)
	}
	s.store = campus.Store{DB: db}
	const id = uint64(987124)
	t.Cleanup(func() { db.Where("user_id = ?", id).Delete(&campus.Binding{}) })
	old := service.Swap(s)
	t.Cleanup(func() { service.Store(old) })
	bind(t, s, id)
	closeErr := errors.New("account closure failed")
	if err := CloseForUser(id, func() error { return closeErr }); !errors.Is(err, closeErr) {
		t.Fatalf("CloseForUser() error = %v, want %v", err, closeErr)
	}
	if _, err := s.store.Get(id); err != nil {
		t.Fatalf("campus credentials were deleted after failed account closure: %v", err)
	}
	if got := p.revocations.Load(); got != 0 {
		t.Fatalf("revocations = %d, want 0", got)
	}
}
