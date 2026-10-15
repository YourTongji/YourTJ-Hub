package routes

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/securestore"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userPasskeys"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/userSessions"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/passkeyservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/sessionservice"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/totpservice"
	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	otptotp "github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

const passkeyTestRPID = "f.yourtj.de"

func configurePasskeyAuthenticatorTest(t *testing.T) {
	t.Helper()
	values := map[string]any{
		"passkeys.enabled":                          true,
		"passkeys.rp_id":                            passkeyTestRPID,
		"passkeys.display_name":                     "YourTJ Test",
		"passkeys.web_origins":                      []string{"https://f.yourtj.de"},
		"passkeys.apple_team_id":                    "A1B2C3D4E5",
		"passkeys.ios_bundle_id":                    "tj.yourtj.forumApp",
		"passkeys.android_package_name":             "tj.yourtj.forum_app",
		"passkeys.android_sha256_cert_fingerprints": []string{strings.Repeat("a", 64)},
		"app.env": "test",
	}
	previous := make(map[string]any, len(values))
	for key := range values {
		previous[key] = preferences.GetRaw(key)
	}
	for key, value := range values {
		preferences.Set(key, value)
	}
	t.Cleanup(func() {
		for key, value := range previous {
			if value == nil {
				switch key {
				case "passkeys.enabled":
					preferences.Set(key, false)
				case "passkeys.display_name":
					preferences.Set(key, "YourTJ")
				case "passkeys.ios_bundle_id":
					preferences.Set(key, "tj.yourtj.forumApp")
				case "passkeys.android_package_name":
					preferences.Set(key, "tj.yourtj.forum_app")
				case "passkeys.web_origins", "passkeys.android_sha256_cert_fingerprints":
					preferences.Set(key, []string{})
				default:
					preferences.Set(key, "")
				}
				continue
			}
			preferences.Set(key, value)
		}
	})
}

func passkeyTestAccount(t *testing.T, conn *gorm.DB) (uint64, string) {
	t.Helper()
	user := createHTTPContractUser(t, conn, contractTestID())
	contractSessionToken(t, user)
	var session userSessions.Entity
	if err := conn.Where("user_id = ? AND user_agent = ?", user.Id, "contract-test").Order("created_at DESC").First(&session).Error; err != nil {
		t.Fatalf("find test session: %v", err)
	}
	return user.Id, session.Jti
}

type passkeyClientData struct {
	Type        string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}

type testPasskeyCredential struct {
	id      []byte
	key     *ecdsa.PrivateKey
	userID  uint64
	userHdl []byte
}

func makeRegistrationResponse(t *testing.T, challenge, origin string, userID uint64, existingCredentialID ...[]byte) ([]byte, testPasskeyCredential) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	credentialID := make([]byte, 32)
	if len(existingCredentialID) > 0 {
		credentialID = append(credentialID[:0], existingCredentialID[0]...)
	} else if _, err := rand.Read(credentialID); err != nil {
		t.Fatal(err)
	}
	publicKey, err := cbor.Marshal(map[int]any{
		1:  2,
		3:  -7,
		-1: 1,
		-2: key.X.Bytes(),
		-3: key.Y.Bytes(),
	})
	if err != nil {
		t.Fatal(err)
	}
	rpHash := sha256.Sum256([]byte(passkeyTestRPID))
	// UP + UV + BE + BS + AT. A zero signature counter is valid for this
	// synced credential and must not be treated as a clone warning.
	authData := append([]byte(nil), rpHash[:]...)
	authData = append(authData, 0x5d)
	authData = append(authData, 0, 0, 0, 0)
	authData = append(authData, make([]byte, 16)...)
	var credentialIDLength [2]byte
	binary.BigEndian.PutUint16(credentialIDLength[:], uint16(len(credentialID)))
	authData = append(authData, credentialIDLength[:]...)
	authData = append(authData, credentialID...)
	authData = append(authData, publicKey...)
	attestationObject, err := cbor.Marshal(map[string]any{
		"fmt":      "none",
		"attStmt":  map[string]any{},
		"authData": authData,
	})
	if err != nil {
		t.Fatal(err)
	}
	clientDataJSON, err := json.Marshal(passkeyClientData{Type: "webauthn.create", Challenge: challenge, Origin: origin})
	if err != nil {
		t.Fatal(err)
	}
	encodedID := base64.RawURLEncoding.EncodeToString(credentialID)
	response, err := json.Marshal(map[string]any{
		"id": encodedID, "rawId": encodedID, "type": "public-key",
		"response": map[string]any{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientDataJSON),
			"attestationObject": base64.RawURLEncoding.EncodeToString(attestationObject),
			"transports":        []string{"internal"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	userHandle := passkeyTestUserHandle(t, userID)
	return response, testPasskeyCredential{id: credentialID, key: key, userID: userID, userHdl: userHandle}
}

func makeAssertionResponse(t *testing.T, challenge, origin string, credential testPasskeyCredential, counter uint32, validUV, validSignature bool) []byte {
	t.Helper()
	rpHash := sha256.Sum256([]byte(passkeyTestRPID))
	authData := append([]byte(nil), rpHash[:]...)
	flags := byte(0x01 | 0x08 | 0x10) // UP + BE + BS
	if validUV {
		flags |= 0x04
	}
	authData = append(authData, flags)
	var counterBytes [4]byte
	binary.BigEndian.PutUint32(counterBytes[:], counter)
	authData = append(authData, counterBytes[:]...)
	clientDataJSON, err := json.Marshal(passkeyClientData{Type: "webauthn.get", Challenge: challenge, Origin: origin})
	if err != nil {
		t.Fatal(err)
	}
	clientDataHash := sha256.Sum256(clientDataJSON)
	signed := append(append([]byte(nil), authData...), clientDataHash[:]...)
	signedHash := sha256.Sum256(signed)
	signature, err := ecdsa.SignASN1(rand.Reader, credential.key, signedHash[:])
	if err != nil {
		t.Fatal(err)
	}
	if !validSignature {
		signature[len(signature)-1] ^= 0xff
	}
	encodedID := base64.RawURLEncoding.EncodeToString(credential.id)
	body, err := json.Marshal(map[string]any{
		"id": encodedID, "rawId": encodedID, "type": "public-key",
		"response": map[string]any{
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientDataJSON),
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData),
			"signature":         base64.RawURLEncoding.EncodeToString(signature),
			"userHandle":        base64.RawURLEncoding.EncodeToString(credential.userHdl),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func passkeyTestUserHandle(t *testing.T, userID uint64) []byte {
	t.Helper()
	pepper, err := securestore.Pepper("yourtj-webauthn-user-handle-v1")
	if err != nil {
		t.Fatal(err)
	}
	var encodedID [8]byte
	binary.BigEndian.PutUint64(encodedID[:], userID)
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write(encodedID[:])
	return mac.Sum(nil)
}

func beginAndRegister(t *testing.T, userID uint64, jti string) testPasskeyCredential {
	t.Helper()
	proof, err := passkeyservice.Reauthenticate(userID, jti, "secret123", "")
	if err != nil {
		t.Fatalf("reauthenticate for registration: %v", err)
	}
	begin, err := passkeyservice.BeginRegistration(userID, jti, proof)
	if err != nil {
		t.Fatalf("begin registration: %v", err)
	}
	if begin.PublicKey.User.ID == nil || fmt.Sprint(begin.PublicKey.User.ID) == "" || begin.PublicKey.AuthenticatorSelection.UserVerification != "required" {
		t.Fatalf("registration options omitted user handle or required UV: %+v", begin.PublicKey)
	}
	response, credential := makeRegistrationResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, userID)
	request := httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(userID, jti, begin.CeremonyID, "Test synced passkey", request); err != nil {
		t.Fatalf("finish registration with standard WebAuthn data: %v", err)
	}
	return credential
}

func TestPasskeyRegistrationAndDiscoverableLoginIssueExistingSession(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	userID, jti := passkeyTestAccount(t, conn)
	credential := beginAndRegister(t, userID, jti)
	setup, err := totpservice.Setup(userID)
	if err != nil {
		t.Fatal(err)
	}
	code, err := otptotp.GenerateCode(setup.Secret, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := totpservice.Enable(userID, code); err != nil {
		t.Fatalf("enable TOTP for independent passkey login: %v", err)
	}
	begin, err := passkeyservice.BeginAuthentication()
	if err != nil {
		t.Fatalf("begin discoverable authentication: %v", err)
	}
	assertion := makeAssertionResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, credential, 0, true, true)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(assertion)))
	session, err := passkeyservice.FinishAuthentication(begin.CeremonyID, request, "passkey-test", "127.0.0.1")
	if err != nil {
		t.Fatalf("finish discoverable login for zero-counter passkey with TOTP enabled: %v", err)
	}
	if session.UserID != userID || session.Token == "" || session.Jti == "" {
		t.Fatalf("passkey login returned incomplete standard session: %+v", session)
	}
	storedSession := sessionservice.GetValidByJti(session.Jti)
	if storedSession == nil || storedSession.UserId != userID {
		t.Fatal("passkey login did not create a valid existing user_sessions record")
	}

	replayRequest := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(assertion)))
	if _, err := passkeyservice.FinishAuthentication(begin.CeremonyID, replayRequest, "passkey-test", "127.0.0.1"); err == nil {
		t.Fatal("replayed authentication challenge unexpectedly succeeded")
	}
	begin, err = passkeyservice.BeginAuthentication()
	if err != nil {
		t.Fatalf("begin signature-counter authentication: %v", err)
	}
	assertion = makeAssertionResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, credential, 1, true, true)
	request = httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(assertion)))
	if _, err := passkeyservice.FinishAuthentication(begin.CeremonyID, request, "passkey-test", "127.0.0.1"); err != nil {
		t.Fatalf("passkey with an advancing signature counter failed: %v", err)
	}
	var storedCredential userPasskeys.CredentialEntity
	if err := conn.Where("credential_id = ?", credential.id).First(&storedCredential).Error; err != nil {
		t.Fatalf("load credential after counter update: %v", err)
	}
	storedCredentialJSON, err := securestore.DecryptPurpose(storedCredential.CredentialData, securestore.PasskeyCredentialPurpose)
	if err != nil {
		t.Fatalf("decrypt credential after counter update: %v", err)
	}
	var persistedCredential webauthn.Credential
	if err := json.Unmarshal([]byte(storedCredentialJSON), &persistedCredential); err != nil {
		t.Fatalf("decode credential after counter update: %v", err)
	}
	if persistedCredential.Authenticator.SignCount != 1 {
		t.Fatalf("signature counter = %d, want 1", persistedCredential.Authenticator.SignCount)
	}
	proof, err := passkeyservice.Reauthenticate(userID, jti, "secret123", code)
	if err != nil {
		t.Fatalf("reauthenticate before revocation: %v", err)
	}
	credentialID := base64.RawURLEncoding.EncodeToString(credential.id)
	if err := passkeyservice.RevokeCredential(userID, jti, proof, credentialID); err != nil {
		t.Fatalf("revoke credential: %v", err)
	}
	begin, err = passkeyservice.BeginAuthentication()
	if err != nil {
		t.Fatal(err)
	}
	assertion = makeAssertionResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, credential, 0, true, true)
	request = httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(assertion)))
	if _, err := passkeyservice.FinishAuthentication(begin.CeremonyID, request, "passkey-test", "127.0.0.1"); err == nil {
		t.Fatal("revoked passkey unexpectedly authenticated")
	}
}

func TestPasskeyRegistrationAndLoginRejectOriginUVAndSignatureFailures(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	userID, jti := passkeyTestAccount(t, conn)
	credential := beginAndRegister(t, userID, jti)

	tests := []struct {
		name     string
		origin   string
		verified bool
		validSig bool
	}{
		{name: "wrong origin", origin: "https://attacker.example", verified: true, validSig: true},
		{name: "missing UV", origin: "https://" + passkeyTestRPID, verified: false, validSig: true},
		{name: "invalid signature", origin: "https://" + passkeyTestRPID, verified: true, validSig: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			begin, err := passkeyservice.BeginAuthentication()
			if err != nil {
				t.Fatal(err)
			}
			response := makeAssertionResponse(t, begin.PublicKey.Challenge.String(), test.origin, credential, 0, test.verified, test.validSig)
			request := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(response)))
			if _, err := passkeyservice.FinishAuthentication(begin.CeremonyID, request, "test", "127.0.0.1"); err == nil {
				t.Fatal("invalid authenticator assertion unexpectedly succeeded")
			}
		})
	}
}

func TestPasskeyRegistrationRejectsCrossAccountAndExpiredChallenge(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	firstUserID, firstJti := passkeyTestAccount(t, conn)
	secondUserID, secondJti := passkeyTestAccount(t, conn)

	proof, err := passkeyservice.Reauthenticate(firstUserID, firstJti, "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := passkeyservice.BeginRegistration(firstUserID, firstJti, proof)
	if err != nil {
		t.Fatal(err)
	}
	response, _ := makeRegistrationResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, firstUserID)
	request := httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(secondUserID, secondJti, begin.CeremonyID, "cross-account", request); err == nil {
		t.Fatal("registration challenge was accepted for another account and session")
	}

	proof, err = passkeyservice.Reauthenticate(firstUserID, firstJti, "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	expired, err := passkeyservice.BeginRegistration(firstUserID, firstJti, proof)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(expired.CeremonyID))
	if err := conn.Model(&userPasskeys.ChallengeEntity{}).
		Where("id = ? AND kind = ?", fmt.Sprintf("%x", digest[:]), "registration").
		Update("expires_at", time.Now().Add(-time.Second)).Error; err != nil {
		t.Fatal(err)
	}
	response, _ = makeRegistrationResponse(t, expired.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, firstUserID)
	request = httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(firstUserID, firstJti, expired.CeremonyID, "expired", request); err == nil {
		t.Fatal("expired registration challenge unexpectedly succeeded")
	}
}

func TestPasskeyRegistrationRejectsCredentialAlreadyBoundToAnotherAccount(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	firstUserID, firstJti := passkeyTestAccount(t, conn)
	secondUserID, secondJti := passkeyTestAccount(t, conn)
	credential := beginAndRegister(t, firstUserID, firstJti)

	proof, err := passkeyservice.Reauthenticate(secondUserID, secondJti, "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := passkeyservice.BeginRegistration(secondUserID, secondJti, proof)
	if err != nil {
		t.Fatal(err)
	}
	response, _ := makeRegistrationResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, secondUserID, credential.id)
	request := httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(secondUserID, secondJti, begin.CeremonyID, "duplicate", request); err == nil {
		t.Fatal("credential already bound to another account was registered")
	}
	var stored userPasskeys.CredentialEntity
	if err := conn.Where("credential_id = ?", credential.id).First(&stored).Error; err != nil {
		t.Fatalf("load original credential: %v", err)
	}
	if stored.UserID != firstUserID || stored.RevokedAt != nil {
		t.Fatalf("duplicate registration changed original ownership: %+v", stored)
	}
	var count int64
	if err := conn.Model(&userPasskeys.CredentialEntity{}).Where("user_id = ?", secondUserID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("duplicate registration created %d credentials for the second user", count)
	}
}

func TestPasskeyConcurrentAuthenticationFinishConsumesOnce(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	userID, jti := passkeyTestAccount(t, conn)
	credential := beginAndRegister(t, userID, jti)
	begin, err := passkeyservice.BeginAuthentication()
	if err != nil {
		t.Fatal(err)
	}
	body := makeAssertionResponse(t, begin.PublicKey.Challenge.String(), "https://"+passkeyTestRPID, credential, 0, true, true)

	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/auth/passkey/login/finish", strings.NewReader(string(body)))
			_, err := passkeyservice.FinishAuthentication(begin.CeremonyID, request, "test", "127.0.0.1")
			results <- err
		}()
	}
	close(start)
	var successes int
	for range 2 {
		if err := <-results; err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent finish accepted %d completions, want exactly one", successes)
	}
}

func TestAndroidOpaqueOriginRequiresExplicitTrustedFingerprint(t *testing.T) {
	conn, _ := setupPasskeyContractTest(t)
	configurePasskeyAuthenticatorTest(t)
	userID, jti := passkeyTestAccount(t, conn)
	proof, err := passkeyservice.Reauthenticate(userID, jti, "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := passkeyservice.BeginRegistration(userID, jti, proof)
	if err != nil {
		t.Fatal(err)
	}
	// The configured test fingerprint is 32 bytes of 0xaa. Android's standard
	// opaque origin is admitted only after deriving it from this allowlist.
	trustedOrigin := "android:apk-key-hash:" + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("\xaa", 32)))
	response, _ := makeRegistrationResponse(t, begin.PublicKey.Challenge.String(), trustedOrigin, userID)
	request := httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(userID, jti, begin.CeremonyID, "Android test key", request); err != nil {
		t.Fatalf("explicitly configured Android opaque origin rejected: %v", err)
	}
	proof, err = passkeyservice.Reauthenticate(userID, jti, "secret123", "")
	if err != nil {
		t.Fatal(err)
	}
	begin, err = passkeyservice.BeginRegistration(userID, jti, proof)
	if err != nil {
		t.Fatal(err)
	}
	untrustedOrigin := "android:apk-key-hash:" + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("\xbb", 32)))
	response, _ = makeRegistrationResponse(t, begin.PublicKey.Challenge.String(), untrustedOrigin, userID)
	request = httptest.NewRequest(http.MethodPost, "/api/user/passkeys/registration/finish", strings.NewReader(string(response)))
	if _, err := passkeyservice.FinishRegistration(userID, jti, begin.CeremonyID, "Untrusted Android key", request); err == nil {
		t.Fatal("unconfigured Android opaque origin unexpectedly passed RP origin validation")
	}
}
