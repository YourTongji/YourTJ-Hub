package users

import (
	"errors"
	"testing"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
)

func setupUserIsolationTestDB(t *testing.T) {
	t.Helper()
	conn := db.Connect()
	if err := conn.AutoMigrate(&EntityComplete{}); err != nil {
		t.Fatalf("migrate users table: %v", err)
	}
	conn.Where("1 = 1").Delete(&EntityComplete{})
}

func TestVerifyRejectsBotAccount(t *testing.T) {
	setupUserIsolationTestDB(t)
	bot := MakeUser("bot-login-test", "correct-password-123", "")
	bot.ActorType = ActorTypeBot
	if err := Create(bot); err != nil {
		t.Fatalf("create bot: %v", err)
	}
	// Even with the right password the bot must never verify, and the error
	// is the same generic credentials error used for wrong passwords.
	if _, err := Verify("bot-login-test", "correct-password-123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bot login error = %v, want ErrInvalidCredentials", err)
	}
}

func TestVerifyAcceptsHumanAccount(t *testing.T) {
	setupUserIsolationTestDB(t)
	human := MakeUser("human-login-test", "correct-password-123", "human@example.com")
	human.ActorType = ActorTypeHuman
	if err := Create(human); err != nil {
		t.Fatalf("create human: %v", err)
	}
	user, err := Verify("human-login-test", "correct-password-123")
	if err != nil {
		t.Fatalf("human login failed: %v", err)
	}
	if user.Id == 0 || user.Username != "human-login-test" {
		t.Fatalf("human login returned wrong user: %#v", user)
	}
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

	if _, err := Verify("no-such-user", "whatever-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown username error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := Verify("no-such-user@example.com", "whatever-password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("unknown email error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := Verify("human-login-enum-test", "wrong-password-456"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password error = %v, want ErrInvalidCredentials", err)
	}
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
