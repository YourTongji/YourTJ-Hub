package mailservice

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/url"
	"strings"
	texttemplate "text/template"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/defaultconfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/forum/pageConfig"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/models/hotdataserve"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
)

//go:embed activation-email.gohtml
var emailTemplate string
var emailTmpl *template.Template

//go:embed password-reset-email.gohtml
var passwordResetTemplate string
var passwordResetTmpl *template.Template

//go:embed email-changed-email.gohtml
var emailChangedTemplate string
var emailChangedTmpl *template.Template

//go:embed email-change-pending.gohtml
var emailChangePendingTemplate string
var emailChangePendingTmpl *template.Template

//go:embed account-email.gotxt
var accountTextTemplate string
var accountTextTmpl *texttemplate.Template

func init() {
	accountTextTmpl = texttemplate.Must(texttemplate.New("accountEmail").Parse(accountTextTemplate))
	emailTmpl = template.Must(template.New("activation").Parse(emailTemplate))
	passwordResetTmpl = template.Must(template.New("passwordReset").Parse(passwordResetTemplate))
	emailChangedTmpl = template.Must(template.New("emailChanged").Parse(emailChangedTemplate))
	emailChangePendingTmpl = template.Must(template.New("emailChangePending").Parse(emailChangePendingTemplate))
}

// renderAccountEmail uses one configuration snapshot for all content in a message.
func renderAccountEmail(siteConfig pageConfig.SiteSettingsConfig, task EmailTask) (string, string, string, error) {
	lang := emailBodyLang(task.Locale)
	if strings.TrimSpace(siteConfig.SiteName) == "" {
		siteConfig.SiteName = defaultconfig.GetDefaultSiteSettingsConfig().SiteName
	}
	data := map[string]any{
		"SiteName": siteConfig.SiteName,
		"Username": task.Username,
		"NewEmail": task.NewEmail,
		"Lang":     lang,
		"T":        i18n.Func(lang),
	}
	var tmpl *template.Template
	var subjectKey string
	switch task.Type {
	case "activation":
		tmpl, subjectKey = emailTmpl, "email.activation.subject"
		data["RecoveryLink"] = buildEmailSettingsURL(emailSiteBaseURL(siteConfig.SiteUrl), task.Locale)
		data["ActivationLink"] = buildEmailActionURL(emailSiteBaseURL(siteConfig.SiteUrl), urlconfig.Activate(), task.Token, task.Locale)
	case "reset_password":
		tmpl, subjectKey = passwordResetTmpl, "email.passwordReset.subject"
		data["ResetLink"] = buildEmailActionURL(emailSiteBaseURL(siteConfig.SiteUrl), urlconfig.ResetPassword(), task.Token, task.Locale)
	case "email_changed":
		tmpl, subjectKey = emailChangedTmpl, "email.changed.subject"
	case "email_change_pending":
		tmpl, subjectKey = emailChangePendingTmpl, "email.changePending.subject"
	default:
		return "", "", "", fmt.Errorf("unsupported account email type: %s", task.Type)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", "", "", err
	}
	var plain bytes.Buffer
	if err := accountTextTmpl.ExecuteTemplate(&plain, tmpl.Name(), data); err != nil {
		return "", "", "", err
	}
	// Keep administrator-configured branding on one subject line.
	siteName := strings.Join(strings.Fields(siteConfig.SiteName), " ")
	subject := fmt.Sprintf("[%s] %s", siteName, i18n.T(lang, subjectKey))
	return subject, buf.String(), strings.TrimSpace(plain.String()), nil
}

func generateActivationEmailBody(username, token string, locale ...string) (string, error) {
	_, body, _, err := renderAccountEmail(hotdataserve.GetSiteSettingsConfigCache(), EmailTask{Type: "activation", Username: username, Token: token, Locale: normalizeEmailLocale(locale...)})
	return body, err
}

func generatePasswordResetEmailBody(username, token string, locale ...string) (string, error) {
	_, body, _, err := renderAccountEmail(hotdataserve.GetSiteSettingsConfigCache(), EmailTask{Type: "reset_password", Username: username, Token: token, Locale: normalizeEmailLocale(locale...)})
	return body, err
}

func generateEmailChangedEmailBody(username, newEmail string, locale ...string) (string, error) {
	_, body, _, err := renderAccountEmail(hotdataserve.GetSiteSettingsConfigCache(), EmailTask{Type: "email_changed", Username: username, NewEmail: newEmail, Locale: normalizeEmailLocale(locale...)})
	return body, err
}

func emailSiteBaseURL(siteURL string) string {
	baseURL := strings.TrimSpace(siteURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(preferences.GetString("server.url", ""))
	}
	return strings.TrimRight(baseURL, "/")
}

func buildEmailActionURL(baseURL, actionPath, token string, locale ...string) string {
	cleanBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	cleanPath := "/" + strings.TrimLeft(actionPath, "/")
	actionURL := cleanBaseURL + cleanPath
	query := url.Values{}
	if lang := normalizeEmailLocale(locale...); lang != "" {
		query.Set("lang", lang)
	}
	query.Set("token", token)
	return actionURL + "?" + query.Encode()
}

// Recovery uses the existing owner settings page; no token or account identifier
// is carried to it, and following it never sends another message automatically.
func buildEmailSettingsURL(baseURL string, locale ...string) string {
	query := url.Values{"tab": {"profile"}}
	if lang := normalizeEmailLocale(locale...); lang != "" {
		query.Set("lang", lang)
	}
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + urlconfig.Settings() + "?" + query.Encode()
}

func emailBodyLang(locale ...string) string {
	if lang := normalizeEmailLocale(locale...); lang != "" {
		return lang
	}
	return i18n.Fallback
}

func normalizeEmailLocale(locale ...string) string {
	if len(locale) == 0 || strings.TrimSpace(locale[0]) == "" {
		return ""
	}
	return i18n.Normalize(locale[0])
}
