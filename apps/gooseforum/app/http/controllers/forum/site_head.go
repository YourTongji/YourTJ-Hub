package forum

import (
	"html/template"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// siteHeadHTML filters legacy full HTML documents and page-owned metadata from
// the administrator's head resource slot. Incomplete raw-text tags are dropped
// so that a bad setting cannot consume the template's closing </head>.
func siteHeadHTML(input string) template.HTML {
	z := html.NewTokenizer(strings.NewReader(input))
	var out strings.Builder
	var rawTag, rawStart string
	var body strings.Builder
	for {
		kind := z.Next()
		if kind == html.ErrorToken {
			if z.Err() != io.EOF {
				return ""
			}
			return template.HTML(out.String())
		}
		t := z.Token()
		if rawTag != "" {
			switch {
			case kind == html.TextToken:
				body.Write(z.Raw())
			case kind == html.EndTagToken && t.Data == rawTag:
				out.WriteString(rawStart)
				out.WriteString(body.String())
				out.Write(z.Raw())
				rawTag, rawStart = "", ""
				body.Reset()
			}
			continue
		}
		if kind != html.StartTagToken && kind != html.SelfClosingTagToken {
			continue
		}
		switch t.Data {
		case "script", "style":
			if kind == html.StartTagToken {
				rawTag, rawStart = t.Data, string(z.Raw())
				body.Reset()
			}
		case "link":
			if headLinkAllowed(t.Attr) {
				out.Write(z.Raw())
			}
		case "meta":
			if headMetaAllowed(t.Attr) {
				out.Write(z.Raw())
			}
		}
	}
}

func headMetaAllowed(attrs []html.Attribute) bool {
	for _, attr := range attrs {
		value := strings.ToLower(strings.TrimSpace(attr.Val))
		switch strings.ToLower(attr.Key) {
		case "charset", "http-equiv":
			return false
		case "name":
			switch value {
			case "viewport", "description", "robots", "author", "keywords", "theme-color", "color-scheme":
				return false
			}
			if strings.HasPrefix(value, "twitter:") {
				return false
			}
		case "property":
			if strings.HasPrefix(value, "og:") || strings.HasPrefix(value, "article:") {
				return false
			}
		}
	}
	return true
}

func headLinkAllowed(attrs []html.Attribute) bool {
	for _, attr := range attrs {
		if strings.ToLower(attr.Key) == "rel" {
			for _, rel := range strings.Fields(strings.ToLower(attr.Val)) {
				switch rel {
				case "canonical", "prev", "next", "manifest", "icon", "apple-touch-icon":
					return false
				}
			}
		}
	}
	return true
}
