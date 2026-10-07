package anonymousidentityservice

import (
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
)

func exerciseProfilePrivacy(t *testing.T, s Service) {
	t.Helper()
	state, err := s.State(1)
	if err != nil {
		t.Fatal(err)
	}
	batch, err := s.Generate(1, state.Day, "privacy-test-key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Confirm(1, batch.ID, 0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetShowContent(2, false); err == nil {
		t.Fatal("unconfigured owner changed privacy")
	}
	if err := s.DB.Model(&users.EntityComplete{}).Where("id = 1").Updates(map[string]any{"is_frozen": 1, "anonymous_governance_blocked": true}).Error; err != nil {
		t.Fatal(err)
	}
	for _, show := range []bool{false, true} {
		if err := s.SetShowContent(1, show); err != nil {
			t.Fatal("restriction prevented privacy management", err)
		}
		state, err := s.State(1)
		if err != nil || state.ShowContent != show {
			t.Fatal("preference not persisted", state, err)
		}
		if _, err := s.Resolve(1, "persona"); err == nil {
			t.Fatal("privacy management bypassed governance")
		}
	}
	if err := s.DB.Delete(&users.EntityComplete{}, 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.SetShowContent(1, false); err == nil {
		t.Fatal("closed owner changed privacy")
	}
}

func TestAnonymousProfilePrivacy(t *testing.T)           { exerciseProfilePrivacy(t, setup(t, false)) }
func TestPostgreSQLAnonymousProfilePrivacy(t *testing.T) { exerciseProfilePrivacy(t, setup(t, true)) }
