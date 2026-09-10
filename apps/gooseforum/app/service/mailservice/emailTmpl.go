package mailservice

import (
	"bytes"
	_ "embed"
	"html/template"
	"net/url"
	"strings"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/i18n"
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
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

func init() {
	emailTmpl = template.Must(template.New("activation").Parse(emailTemplate))
	passwordResetTmpl = template.Must(template.New("passwordReset").Parse(passwordResetTemplate))
	emailChangedTmpl = template.Must(template.New("emailChanged").Parse(emailChangedTemplate))
}

func generateActivationEmailBody(username, token string, locale ...string) (string, error) {
	siteConfig := hotdataserve.GetSiteSettingsConfigCache()
	lang := emailBodyLang(locale...)
	var buf bytes.Buffer
	err := emailTmpl.Execute(&buf, map[string]any{
		"SiteName":       siteConfig.SiteName,
		"Username":       username,
		"ActivationLink": buildEmailActionURL(emailSiteBaseURL(siteConfig.SiteUrl), urlconfig.Activate(), token, locale...),
		"Lang":           lang,
		"T":              i18n.Func(lang),
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func generatePasswordResetEmailBody(username, token string, locale ...string) (string, error) {
	siteConfig := hotdataserve.GetSiteSettingsConfigCache()
	lang := emailBodyLang(locale...)
	var buf bytes.Buffer
	err := passwordResetTmpl.Execute(&buf, map[string]any{
		"SiteName":  siteConfig.SiteName,
		"Username":  username,
		"ResetLink": buildEmailActionURL(emailSiteBaseURL(siteConfig.SiteUrl), urlconfig.ResetPassword(), token, locale...),
		"Lang":      lang,
		"T":         i18n.Func(lang),
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func generateEmailChangedEmailBody(username, newEmail string, locale ...string) (string, error) {
	siteConfig := hotdataserve.GetSiteSettingsConfigCache()
	lang := emailBodyLang(locale...)
	var buf bytes.Buffer
	err := emailChangedTmpl.Execute(&buf, map[string]any{
		"SiteName": siteConfig.SiteName,
		"Username": username,
		"NewEmail": newEmail,
		"Lang":     lang,
		"T":        i18n.Func(lang),
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
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
