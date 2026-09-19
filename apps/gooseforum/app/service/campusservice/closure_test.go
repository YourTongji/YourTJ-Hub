package campusservice

import (
	"errors"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/campus"
	"gorm.io/gorm"
)

func TestAccountClosureRevokesReadableCredentialsAfterLocalDeletion(t *testing.T) {
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
				if _, err := s.store.Get(id); !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Fatal("account closed before credential deletion")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
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
