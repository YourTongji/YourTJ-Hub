package mailservice

import (
	"bytes"
	"html"
	"io"
	"mime"
	"mime/multipart"
	stdmail "net/mail"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
)

// Serialize the production message, then parse MIME as a receiving client would.
// No SMTP client, provider, account or persistent database is involved.
func TestAccountEmailMIMEAlternatives(t *testing.T) {
	kinds := []struct{ kind, key string }{
		{"activation", "activation"}, {"reset_password", "passwordReset"},
		{"email_changed", "changed"}, {"email_change_pending", "changePending"},
	}
	for _, kind := range kinds {
		for _, lang := range i18n.SupportedLocales {
			t.Run(kind.kind+"/"+lang, func(t *testing.T) {
				task := EmailTask{Type: kind.kind, To: "recipient@example.test", Username: "A <B> & C", Token: "a/b+c=", NewEmail: "new@example.test", Locale: lang}
				message, err := buildAccountEmailMessage(pageConfig.MailSettingsConfig{FromName: "Mail sender", FromEmail: "sender@example.test"}, pageConfig.SiteSettingsConfig{SiteName: "Study <Hub> & 学苑", SiteUrl: "https://forum.example.test/"}, task)
				if err != nil {
					t.Fatal(err)
				}
				var wire bytes.Buffer
				if _, err := message.WriteTo(&wire); err != nil {
					t.Fatal(err)
				}
				received, err := stdmail.ReadMessage(&wire)
				if err != nil {
					t.Fatal(err)
				}
				subject, err := new(mime.WordDecoder).DecodeHeader(received.Header.Get("Subject"))
				if err != nil {
					t.Fatal(err)
				}
				wantSubject := "[Study <Hub> & 学苑] " + i18n.T(lang, "email."+kind.key+".subject")
				if subject != wantSubject {
					t.Errorf("subject = %q, want %q", subject, wantSubject)
				}
				to, err := received.Header.AddressList("To")
				if err != nil || len(to) != 1 || to[0].Address != task.To {
					t.Fatalf("unexpected To: %v, %v", to, err)
				}
				from, err := received.Header.AddressList("From")
				if err != nil || len(from) != 1 || from[0].Address != "sender@example.test" {
					t.Fatalf("unexpected From: %v, %v", from, err)
				}
				mediaType, params, err := mime.ParseMediaType(received.Header.Get("Content-Type"))
				if err != nil {
					t.Fatal(err)
				}
				if mediaType != "multipart/alternative" {
					t.Fatalf("content type = %q, want multipart/alternative", mediaType)
				}
				reader := multipart.NewReader(received.Body, params["boundary"])
				for _, expectedType := range []string{"text/plain", "text/html"} {
					part, err := reader.NextPart()
					if err != nil {
						t.Fatal(err)
					}
					partType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
					if err != nil || partType != expectedType || !strings.EqualFold(partParams["charset"], "utf-8") {
						t.Fatalf("part content type = %q (%v)", part.Header.Get("Content-Type"), err)
					}
					bodyBytes, err := io.ReadAll(part)
					if err != nil {
						t.Fatal(err)
					}
					body := string(bodyBytes)
					username := task.Username
					if expectedType == "text/html" {
						username = html.EscapeString(username)
					}
					if !strings.Contains(body, username) {
						t.Errorf("%s missing correctly escaped username", expectedType)
					}
					readable := body
					if expectedType == "text/html" {
						readable = html.UnescapeString(body)
					}
					for _, key := range []string{"email." + kind.key + ".heading", "email." + kind.key + ".instruction"} {
						translated := i18n.T(lang, key, "site", "Study <Hub> & 学苑", "newEmail", task.NewEmail)
						if translated == key || !strings.Contains(readable, translated) {
							t.Errorf("%s missing localized %s", expectedType, key)
						}
					}
					securityKeys := map[string][]string{
						"reset_password":       {"email.passwordReset.expires", "email.passwordReset.ignore", "email.passwordReset.keepPrivate", "email.passwordReset.contact"},
						"email_changed":        {"email.changed.securityNote", "email.changed.contact"},
						"email_change_pending": {"email.changePending.securityNote", "email.changed.contact"},
					}
					for _, key := range securityKeys[kind.kind] {
						if !strings.Contains(readable, i18n.T(lang, key)) {
							t.Errorf("%s missing security guidance %s", expectedType, key)
						}
					}
					if kind.kind == "activation" || kind.kind == "reset_password" {
						action := "/activate"
						if kind.kind == "reset_password" {
							action = "/reset-password"
						}
						if !strings.Contains(readable, "https://forum.example.test"+action+"?lang="+lang+"&token=a%2Fb%2Bc%3D") {
							t.Errorf("%s missing unchanged token/locale URL", expectedType)
						}
					}
					if kind.kind == "activation" {
						for _, key := range []string{"email.activation.ignore", "email.activation.expires", "email.activation.recovery", "email.activation.manualRecovery"} {
							translated := i18n.T(lang, key)
							if translated == key || !strings.Contains(readable, translated) {
								t.Errorf("%s missing localized guidance %s", expectedType, key)
							}
						}
						if !strings.Contains(readable, "https://forum.example.test/settings?lang="+lang+"&tab=profile") {
							t.Errorf("%s missing existing settings recovery URL", expectedType)
						}
					}
					if expectedType == "text/plain" {
						for _, word := range []string{"按钮", "button", "ボタン", "Schaltfläche"} {
							if strings.Contains(body, word) {
								t.Errorf("plain text incorrectly refers to a button: %q", word)
							}
						}
					}
					if expectedType == "text/plain" && (strings.Contains(body, "<html") || strings.Contains(body, "<style") || strings.Contains(body, "&amp;")) {
						t.Error("plain alternative contains HTML markup/escaping")
					}
				}
				if _, err := reader.NextPart(); err != io.EOF {
					t.Fatalf("extra MIME part: %v", err)
				}
			})
		}
	}
}

func TestActivationEmailHasSafeRecoveryGuidance(t *testing.T) {
	for _, lang := range i18n.SupportedLocales {
		t.Run(lang, func(t *testing.T) {
			_, body, _, err := renderAccountEmail(pageConfig.SiteSettingsConfig{SiteName: "Test", SiteUrl: "https://forum.example.test"}, EmailTask{Type: "activation", Username: "testuser", Token: "token", Locale: lang})
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"email.activation.ignore", "email.activation.recovery", "email.activation.manualRecovery"} {
				value := i18n.T(lang, key)
				if value == key || !strings.Contains(html.UnescapeString(body), value) {
					t.Errorf("missing %s", key)
				}
			}
			if !strings.Contains(html.UnescapeString(body), "/settings?lang="+lang+"&tab=profile") {
				t.Error("missing fixed recovery route")
			}
		})
	}
}

func TestAccountEmailLocaleAndURLFallback(t *testing.T) {
	original := preferences.GetString("server.url", "")
	preferences.Set("server.url", " https://fallback.example.test/ ")
	t.Cleanup(func() { preferences.Set("server.url", original) })
	for _, tc := range []struct{ locale, lang, query string }{
		{"en-US", "en", "lang=en&"}, {" ja-JP ", "ja", "lang=ja&"},
		{"de_DE", "de", "lang=de&"}, {"zh-CN", "zh", "lang=zh&"},
		{"", "zh", ""}, {"unknown", "zh", "lang=zh&"},
	} {
		t.Run(tc.locale, func(t *testing.T) {
			_, body, plain, err := renderAccountEmail(pageConfig.SiteSettingsConfig{SiteName: "Study", SiteUrl: " "}, EmailTask{Type: "activation", Username: "owner", Token: "a/b+c=", Locale: tc.locale})
			if err != nil {
				t.Fatal(err)
			}
			for _, value := range []string{
				"https://fallback.example.test/activate?" + tc.query + "token=a%2Fb%2Bc%3D",
				"https://fallback.example.test/settings?" + tc.query + "tab=profile",
				i18n.T(tc.lang, "email.activation.ignore"),
			} {
				if !strings.Contains(html.UnescapeString(body), value) || !strings.Contains(plain, value) {
					t.Errorf("both alternatives must contain %q", value)
				}
			}
		})
	}
	if got := emailSiteBaseURL(" https://configured.example.test/ "); got != "https://configured.example.test" {
		t.Fatalf("configured site URL lost precedence: %q", got)
	}
}

func TestAccountEmailBrandAndAddressBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{" ", defaultconfig.GetDefaultSiteSettingsConfig().SiteName},
		{"Study\r\nBcc: extra@example.test", "Study Bcc: extra@example.test"},
	} {
		message, err := buildAccountEmailMessage(pageConfig.MailSettingsConfig{FromEmail: "sender@example.test"}, pageConfig.SiteSettingsConfig{SiteName: tc.name}, EmailTask{Type: "email_changed", To: "owner@example.test", Username: "owner", NewEmail: "new@example.test", Locale: "en"})
		if err != nil {
			t.Fatal(err)
		}
		var wire bytes.Buffer
		if _, err := message.WriteTo(&wire); err != nil {
			t.Fatal(err)
		}
		received, err := stdmail.ReadMessage(&wire)
		if err != nil {
			t.Fatal(err)
		}
		subject, err := new(mime.WordDecoder).DecodeHeader(received.Header.Get("Subject"))
		if err != nil {
			t.Fatal(err)
		}
		if subject != "["+tc.want+"] "+i18n.T("en", "email.changed.subject") || received.Header.Get("Bcc") != "" {
			t.Fatalf("unsafe or unbranded subject: %q", subject)
		}
	}
	for _, tc := range []struct{ kind, to, from string }{
		{"bogus", "owner@example.test", "sender@example.test"},
		{"activation", "not an email", "sender@example.test"},
		{"activation", "owner@example.test", "not an email"},
	} {
		message, err := buildAccountEmailMessage(pageConfig.MailSettingsConfig{FromEmail: tc.from}, pageConfig.SiteSettingsConfig{}, EmailTask{Type: tc.kind, To: tc.to})
		if err == nil || message != nil {
			t.Fatalf("invalid task unexpectedly built: %+v", tc)
		}
	}
}
