package controllers

import (
	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/service/urlconfig"
	"net/url"
	"regexp"
	"strings"
)

var activationReadingPath = regexp.MustCompile(`^/(?:p/post/[1-9][0-9]*(?:/[1-9][0-9]*)?|courses/[1-9][0-9]*)$`)

// A continuation is a user-clicked reading/form destination. API, auth and
// activation endpoints are never allowed, even when they share our origin.
func activationContinuation(raw string) string {
	const fallback = "/settings?tab=profile"
	if len(raw) > 2048 || !urlconfig.IsSafeRedirect(raw) {
		return fallback
	}
	target, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	if strings.ContainsAny(target.Path, "\\%") {
		return fallback
	}
	switch target.Path {
	case "/", "/publish", "/campus", "/settings", "/drafts", "/messages", "/notifications", "/schedule", "/courses", "/search":
		return target.String()
	}
	if activationReadingPath.MatchString(target.Path) {
		return target.String()
	}
	return fallback
}
func activationLoginURL(raw string) string {
	return "/login?redirect=" + url.QueryEscape(activationContinuation(raw))
}
