package algorithm

import (
	"strings"
	"testing"
)

func TestPasswordLifecycle(t *testing.T) {
	stored, err := MakePassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("MakePassword failed: %v", err)
	}
	if parts := strings.Split(stored, ":"); len(parts) != 2 {
		t.Fatalf("stored password should have hash and salt, got %q", stored)
	}

	if err := VerifyEncryptPassword(stored, "correct horse battery staple"); err != nil {
		t.Fatalf("VerifyEncryptPassword failed: %v", err)
	}
	if err := VerifyEncryptPassword(stored, "wrong password"); err == nil {
		t.Fatalf("expected wrong password error")
	}
}

func TestVerifyEncryptPasswordRejectsMalformedValue(t *testing.T) {
	if err := VerifyEncryptPassword("not-a-hash", "password"); err == nil {
		t.Fatalf("expected malformed stored password error")
	}
}

func TestIsWellFormedPasswordHash(t *testing.T) {
	stored, err := MakePassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("MakePassword failed: %v", err)
	}
	if !IsWellFormedPasswordHash(stored) {
		t.Fatalf("MakePassword output %q should be well-formed", stored)
	}
	validHash, validSalt, _ := strings.Cut(stored, ":")
	malformed := []string{
		"",
		"not-a-hash",
		"%%%:" + validSalt,
		validHash + ":%%%",
		"a:b:c",
		validHash + ":" + validSalt[:len(validSalt)-4],
		validHash[:len(validHash)-4] + ":" + validSalt,
	}
	for _, value := range malformed {
		if IsWellFormedPasswordHash(value) {
			t.Fatalf("%q should not be well-formed", value)
		}
	}
}

func TestVerifyPasswordRejectsInvalidBase64(t *testing.T) {
	if err := VerifyPassword("not base64", "also not base64", "password"); err == nil {
		t.Fatalf("expected invalid hash error")
	}

	hash, _, err := EncryptPassword("password")
	if err != nil {
		t.Fatalf("EncryptPassword failed: %v", err)
	}
	if err := VerifyPassword(hash, "not base64", "password"); err == nil {
		t.Fatalf("expected invalid salt error")
	}
}

func TestEqualHashes(t *testing.T) {
	if !equalHashes([]byte{1, 2, 3}, []byte{1, 2, 3}) {
		t.Fatalf("equal hashes should match")
	}
	if equalHashes([]byte{1, 2, 3}, []byte{1, 2, 4}) {
		t.Fatalf("different hashes should not match")
	}
	if equalHashes([]byte{1, 2}, []byte{1, 2, 3}) {
		t.Fatalf("different length hashes should not match")
	}
}
