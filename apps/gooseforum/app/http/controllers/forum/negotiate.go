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

// prefersTextMarkdown reports whether Accept prefers text/markdown over
// text/html. Quality comes from the most specific matching range
// (type/subtype, then type/*, then */*), including an explicit q=0.
// A missing header, an equal quality, or a wildcard alone keeps HTML.
func prefersTextMarkdown(accept string) bool {
	if strings.TrimSpace(accept) == "" {
		return false
	}
	markdownQ := mediaQuality(accept, "text/markdown")
	htmlQ := mediaQuality(accept, "text/html")
	return markdownQ > htmlQ && markdownQ > 0
}

func mediaQuality(accept, target string) float64 {
	bestSpecificity := 0
	bestQ := 0.0
	for _, part := range strings.Split(accept, ",") {
		media, q, ok := parseAcceptMedia(part)
		if !ok {
			continue
		}
		specificity := acceptSpecificity(media, target)
		if specificity == 0 {
			continue
		}
		if specificity > bestSpecificity || (specificity == bestSpecificity && q > bestQ) {
			bestSpecificity = specificity
			bestQ = q
		}
	}
	if bestSpecificity == 0 {
		return 0
	}
	return bestQ
}

func acceptSpecificity(media, target string) int {
	if media == target {
		return 3
	}
	mediaType, mediaSub, mediaOK := strings.Cut(media, "/")
	targetType, _, targetOK := strings.Cut(target, "/")
	if !mediaOK || !targetOK {
		return 0
	}
	if mediaSub == "*" && mediaType == targetType {
		return 2
	}
	if media == "*/*" {
		return 1
	}
	return 0
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
