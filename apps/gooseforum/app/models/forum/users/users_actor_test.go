package users

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/algorithm"
	db "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
)

func setupUserIsolationTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	conn.Where("1 = 1").Delete(&EntityComplete{})
}

type verifySpyCall struct {
	storedHash string
	password   string
}

// installVerifySpy replaces the package-level verifyEncryptPassword seam with
// a recording wrapper around the real implementation, so tests can assert
// that every Verify path pays the equal-cost PBKDF2 against the intended
// stored hash. The timing-equalization calls discard their results and are
// otherwise unobservable, so without this seam deleting them would leave the
// suite green while silently re-opening the CWE-208 timing side channel.
func installVerifySpy(t *testing.T) *[]verifySpyCall {
	t.Helper()
	calls := &[]verifySpyCall{}
	original := verifyEncryptPassword
	verifyEncryptPassword = func(storedHash, password string) error {
		*calls = append(*calls, verifySpyCall{storedHash: storedHash, password: password})
		return original(storedHash, password)
	}
	t.Cleanup(func() { verifyEncryptPassword = original })
	return calls
}

func assertSingleVerifyCall(t *testing.T, calls *[]verifySpyCall, wantHash string) {
	t.Helper()
	if len(*calls) != 1 {
		t.Fatalf("verify calls = %d, want exactly 1", len(*calls))
	}
	if (*calls)[0].storedHash != wantHash {
		t.Fatalf("verify stored hash = %q, want %q", (*calls)[0].storedHash, wantHash)
	}
}

func TestVerifyRejectsBotAccount(t *testing.T) {
	setupUserIsolationTestDB(t)
	bot := MakeUser("bot-login-test", "correct-password-123", "")
	bot.ActorType = ActorTypeBot
	if err := Create(bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}
	calls := installVerifySpy(t)
	// Even with the right password the bot must never verify, and the error
	// is the same generic credentials error used for wrong passwords.
	if _, err := Verify("bot-login-test", "correct-password-123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bot login error = %v, want ErrInvalidCredentials", err)
	}
	assertSingleVerifyCall(t, calls, dummyHashForTiming)
}

func TestVerifyAcceptsHumanAccount(t *testing.T) {
	setupUserIsolationTestDB(t)
	human := MakeUser("human-login-test", "correct-password-123", "human@example.com")
	human.ActorType = ActorTypeHuman
	if err := Create(human); err != nil {
		t.Fatalf("create human: %v", err)
	}
	calls := installVerifySpy(t)
	user, err := Verify("human-login-test", "correct-password-123")
	if err != nil {
		t.Fatalf("human login failed: %v", err)
	}
	if user.Id == 0 || user.Username != "human-login-test" {
		t.Fatalf("human login returned wrong user: %#v", user)
	}
	assertSingleVerifyCall(t, calls, human.Password)
}

// Unknown accounts must fail with the same generic error as wrong passwords
// (and bot accounts), after an equal-cost PBKDF2 run, so neither the error
// identity nor the response time can be used to enumerate registered
// usernames/emails (CWE-208).
func TestVerifyUnknownAccountMatchesCredentialFailures(t *testing.T) {
	setupUserIsolationTestDB(t)
	human := MakeUser("human-login-enum-test", "correct-password-123", "human-enum@example.com")
	if err := Create(human); err != nil {
		t.Fatalf("create human: %v", err)
	}

	cases := []struct {
		name     string
		login    string
		password string
		wantHash string
	}{
		{"unknown username", "no-such-user", "whatever-password", dummyHashForTiming},
		{"unknown email", "no-such-user@example.com", "whatever-password", dummyHashForTiming},
		{"known username, wrong password", "human-login-enum-test", "wrong-password-456", human.Password},
		{"known email, wrong password", "human-enum@example.com", "wrong-password-456", human.Password},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := installVerifySpy(t)
			if _, err := Verify(tc.login, tc.password); !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("%s error = %v, want ErrInvalidCredentials", tc.name, err)
			}
			assertSingleVerifyCall(t, calls, tc.wantHash)
		})
	}
}

// The dummy value must stay well-formed and run the full PBKDF2 cost:
// VerifyEncryptPassword against it must reach the post-PBKDF2 comparison
// and report "incorrect password", never a fail-fast parse error. If a
// future hardening of VerifyPassword (e.g. hash/salt length checks) made
// the dummy fail fast, the timing equalization would silently vanish while
// the spy tests above stayed green.
func TestDummyHashForTimingRunsFullCost(t *testing.T) {
	if !algorithm.IsWellFormedPasswordHash(dummyHashForTiming) {
		t.Fatal("dummyHashForTiming should satisfy IsWellFormedPasswordHash")
	}
	if err := algorithm.VerifyEncryptPassword(dummyHashForTiming, "whatever-password"); err == nil || err.Error() != "incorrect password" {
		t.Fatalf("dummy verify error = %v, want post-PBKDF2 \"incorrect password\"", err)
	}
}

// Accounts whose stored hash is empty or malformed (e.g. imported users
// without a password) can never authenticate; they must still pay the
// equal-cost PBKDF2 against the dummy hash, or a fast response would
// deterministically reveal "username exists but has no usable password".
// VerifyEncryptPassword fails fast on any malformed value, so the malformed
// detection must reject every shape it would skip PBKDF2 for.
func TestVerifyMalformedStoredHashUsesDummy(t *testing.T) {
	setupUserIsolationTestDB(t)
	validHash, validSalt, _ := strings.Cut(dummyHashForTiming, ":")
	malformed := []string{
		"",
		"not-a-hash",
		"%%%:" + validSalt, // hash segment not base64
		validHash + ":%%%", // salt segment not base64
		"a:b:c",            // more than two segments
		validHash + ":" + validSalt[:len(validSalt)-4], // salt of wrong length
		validHash[:len(validHash)-4] + ":" + validSalt, // hash of wrong length
	}
	for i, stored := range malformed {
		user := &EntityComplete{Username: fmt.Sprintf("malformed-hash-user-%d", i)}
		user.Password = stored
		if err := Create(user); err != nil {
			t.Fatalf("create user with stored hash %q: %v", stored, err)
		}
		calls := installVerifySpy(t)
		if _, err := Verify(user.Username, "whatever-password"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("stored hash %q error = %v, want ErrInvalidCredentials", stored, err)
		}
		assertSingleVerifyCall(t, calls, dummyHashForTiming)
	}

	// Same handling via the email lookup route.
	emailUser := &EntityComplete{Username: "malformed-hash-email-user", Email: "malformed-hash@example.com"}
	emailUser.Password = ""
	if err := Create(emailUser); err != nil {
		t.Fatalf("create email-route user: %v", err)
	}
	calls := installVerifySpy(t)
	if _, err := Verify("malformed-hash@example.com", "whatever-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("email-route malformed hash error = %v, want ErrInvalidCredentials", err)
	}
	assertSingleVerifyCall(t, calls, dummyHashForTiming)
}

func TestIsBotFlag(t *testing.T) {
	if (&EntityComplete{ActorType: ActorTypeBot}).IsBot() != true {
		t.Fatal("ActorTypeBot should report IsBot")
	}
	if (&EntityComplete{ActorType: ActorTypeHuman}).IsBot() != false {
		t.Fatal("ActorTypeHuman should not report IsBot")
	}
	if (&EntityComplete{}).IsBot() != false {
		t.Fatal("zero ActorType should not report IsBot")
	}
}
