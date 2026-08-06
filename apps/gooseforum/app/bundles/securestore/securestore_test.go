package securestore

import (
	"strings"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	plaintext := "JBSWY3DPEHPK3PXP"
	encoded, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encoded == plaintext {
		t.Fatal("Encrypt() returned plaintext")
	}
	got, err := Decrypt(encoded)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if got != plaintext {
		t.Fatalf("Decrypt() = %q, want %q", got, plaintext)
	}
}

func TestEncryptProducesUniqueCiphertexts(t *testing.T) {
	first, err := Encrypt("same-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	second, err := Encrypt("same-secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if first == second {
		t.Fatal("Encrypt() produced identical ciphertext for same plaintext")
	}
}

func TestDecryptRejectsCorruptedCiphertext(t *testing.T) {
	encoded, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if _, err := Decrypt(encoded + "AAAA"); err == nil {
		t.Fatal("Decrypt() accepted corrupted ciphertext")
	}
}

func TestDecryptRejectsGarbageInput(t *testing.T) {
	if _, err := Decrypt("not-base64!!!"); err == nil {
		t.Fatal("Decrypt() accepted non-base64 input")
	}
	if _, err := Decrypt(""); err == nil {
		t.Fatal("Decrypt() accepted empty input")
	}
}

func TestDecryptRejectsTamperedPayload(t *testing.T) {
	encoded, err := Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	tampered := "A" + strings.TrimPrefix(encoded, string(encoded[0]))
	if _, err := Decrypt(tampered); err == nil {
		t.Fatal("Decrypt() accepted tampered payload")
	}
}
