package passkeyservice

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestAndroidOriginUsesConfiguredCertificateFingerprint(t *testing.T) {
	fingerprint := strings.Repeat("aA:", 31) + "aA"
	origin, err := androidOrigin(fingerprint)
	if err != nil {
		t.Fatalf("androidOrigin() error = %v", err)
	}
	want := "android:apk-key-hash:" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0xaa}, 32))
	if origin != want {
		t.Fatalf("androidOrigin() = %q, want %q", origin, want)
	}
}

func TestAndroidOriginRejectsInvalidFingerprint(t *testing.T) {
	for _, fingerprint := range []string{"", "not-a-fingerprint", strings.Repeat("0", 62)} {
		if _, err := androidOrigin(fingerprint); err == nil {
			t.Errorf("androidOrigin(%q) unexpectedly succeeded", fingerprint)
		}
	}
}

func TestValidateWebOriginEnforcesRPAndTransport(t *testing.T) {
	tests := []struct {
		name   string
		rpID   string
		origin string
		local  bool
		wantOK bool
	}{
		{name: "production origin", rpID: "f.yourtj.de", origin: "https://f.yourtj.de", wantOK: true},
		{name: "RP subdomain", rpID: "yourtj.de", origin: "https://f.yourtj.de", wantOK: true},
		{name: "wrong domain", rpID: "yourtj.de", origin: "https://yourtj.example", wantOK: false},
		{name: "production HTTP", rpID: "localhost", origin: "http://localhost:5234", wantOK: false},
		{name: "local loopback", rpID: "localhost", origin: "http://localhost:5234", local: true, wantOK: true},
		{name: "non-loopback local HTTP", rpID: "192.168.1.5", origin: "http://192.168.1.5:5234", local: true, wantOK: false},
		{name: "path not allowed", rpID: "f.yourtj.de", origin: "https://f.yourtj.de/login", wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateWebOrigin(test.rpID, test.origin, test.local)
			if (err == nil) != test.wantOK {
				t.Fatalf("validateWebOrigin() error = %v, want success=%t", err, test.wantOK)
			}
		})
	}
}

func TestNewWebAuthnAcceptsConfiguredWebAndAndroidOrigins(t *testing.T) {
	fingerprint := strings.Repeat("ab", 32)
	android, err := androidOrigin(fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Enabled:        true,
		RPID:           "f.yourtj.de",
		DisplayName:    "YourTJ",
		WebOrigins:     []string{"https://f.yourtj.de", "https://dev.f.yourtj.de"},
		AndroidOrigins: []string{android},
	}
	if _, err := cfg.NewWebAuthn(); err != nil {
		t.Fatalf("NewWebAuthn() error = %v", err)
	}
}
