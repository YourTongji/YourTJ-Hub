package forum

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// homepageAgentLinks advertises the existing llms exports. describedby is the
// registered relation for a machine-readable description of the homepage.
const homepageAgentLinks = `</llms.txt>; rel="describedby"; type="text/plain", </llms-full.txt>; rel="describedby"; type="text/plain"`

func setHomepageAgentLinks(c *gin.Context) {
	c.Header("Link", homepageAgentLinks)
}

// ServesHomeMarkdown reports whether this homepage request should receive the
// markdown representation instead of HTML or the JSON page payload.
func ServesHomeMarkdown(c *gin.Context) bool {
	return !isPageRequest(c) && prefersTextMarkdown(c.GetHeader("Accept"))
}

// prefersTextMarkdown reports whether Accept explicitly prefers text/markdown
// over text/html. A missing header, an equal quality, or a wildcard such as
// */* keeps HTML, which is what ordinary browsers send.
func prefersTextMarkdown(accept string) bool {
	markdownQ := -1.0
	htmlQ := -1.0
	for _, part := range strings.Split(accept, ",") {
		media, q, ok := parseAcceptMedia(part)
		if !ok || q <= 0 {
			continue
		}
		switch media {
		case "text/markdown":
			if q > markdownQ {
				markdownQ = q
			}
		case "text/html":
			if q > htmlQ {
				htmlQ = q
			}
		}
	}
	if markdownQ < 0 {
		return false
	}
	return htmlQ < 0 || markdownQ > htmlQ
}

func parseAcceptMedia(part string) (string, float64, bool) {
	part = strings.TrimSpace(part)
	if part == "" {
		return "", 0, false
	}
	segments := strings.Split(part, ";")
	media := strings.ToLower(strings.TrimSpace(segments[0]))
	if media == "" || strings.Contains(media, " ") {
		return "", 0, false
	}
	q := 1.0
	for _, param := range segments[1:] {
		key, value, found := strings.Cut(strings.TrimSpace(param), "=")
		if !found || strings.ToLower(strings.TrimSpace(key)) != "q" {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || parsed < 0 {
			return "", 0, false
		}
		if parsed > 1 {
			parsed = 1
		}
		q = parsed
	}
	return media, q, true
}
