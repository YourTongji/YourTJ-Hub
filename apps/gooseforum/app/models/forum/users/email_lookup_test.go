package users

import (
	"errors"
	"testing"
)

func TestEmailCredentialLookupUsesRegistrationCanonicalForm(t *testing.T) {
	setupUserIsolationTestDB(t)
	u := MakeUser("MixedCaseName", "secret123", "student@example.com")
	if err := Create(u); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"student@example.com", "STUDENT@EXAMPLE.COM", " Student@Example.Com "} {
		got, err := Verify(email, "secret123")
		if err != nil || got.Id != u.Id {
			t.Errorf("Verify(%q): user=%d err=%v", email, got.Id, err)
		}
		found, err := GetByEmail(email)
		if err != nil || found.Id != u.Id {
			t.Errorf("GetByEmail(%q): user=%d err=%v", email, found.Id, err)
		}
	}
	if _, err := Verify("mixedcasename", "secret123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("username case rules changed: %v", err)
	}
	for _, email := range []string{"STUDENT@EXAMPLE.COM", "UNKNOWN@EXAMPLE.COM"} {
		if _, err := Verify(email, "wrong123"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("credential error for %q = %v", email, err)
		}
	}
}
