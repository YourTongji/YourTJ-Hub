package eventhandlers

import (
	"context"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/connect/dbconnect"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/users"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/httpnotifyservice"
)

func TestSignupReusesHTTPNotificationWithoutEmailOrCredentials(t *testing.T) {
	conn := dbconnect.Connect()
	if err := conn.AutoMigrate(&users.EntityComplete{}); err != nil {
		t.Fatal(err)
	}
	u := users.MakeUser("signup-public-name", "private-password-123", "private-signup@example.com")
	if err := conn.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Unscoped().Delete(u) })
	received := captureHttpNotify(t, httpnotifyservice.EventUserSignup, httpnotifyservice.EventReportCreated)
	if err := handleHttpNotifyUserSignUp(context.Background(), &UserSignUpEvent{UserId: u.Id, Username: u.Username}); err != nil {
		t.Fatal(err)
	}
	body := string(waitHttpNotify(t, received))
	if !strings.Contains(body, `"event":"user.signup"`) || !strings.Contains(body, u.Username) {
		t.Fatalf("missing existing signup event: %s", body)
	}
	for _, secret := range []string{u.Email, u.Password, "private-password-123"} {
		if strings.Contains(body, secret) {
			t.Fatal("signup webhook contains private account data")
		}
	}
	expectNoMoreHttpNotify(t, received)
}

func TestPostURL(t *testing.T) {
	if got := postURL(123, 456); got != "/p/post/123#post-456" {
		t.Fatalf("postURL() = %q", got)
	}
	if got := postURL(123, 0); got != "/p/post/123" {
		t.Fatalf("postURL without post = %q", got)
	}
	if got := postURL(0, 456); got != "" {
		t.Fatalf("postURL without topic = %q", got)
	}
}
