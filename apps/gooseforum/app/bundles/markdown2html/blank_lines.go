package markdown2html

import (
	"regexp"
	"strconv"
	"strings"
)

// Visible blank lines (docs/decisions/0133-visible-blank-lines.md).
//
// Markdown stays the only stored source. Between top-level blocks, every blank
// line after the first in a run renders as one empty-line spacer, at most
// MaxRenderedBlankLines per run. Code, math, list items and indented
// continuations keep CommonMark behaviour. The rule inserts a marker paragraph
// at each spacer and swaps it after rendering, so block structure never
// changes. The Web preview (resource/src/runtime/blank-lines.ts) and Flutter
// (core/lib/src/markdown/blank_lines.dart) share
// packages/api-contract/fixtures/markdown-blank-lines.json.

// MaxRenderedBlankLines caps the spacers rendered for one run of blank lines.
const MaxRenderedBlankLines = 3

// BlankLineHTML is the rendered spacer; styles live in resource/src/styles/prose.css.
const BlankLineHTML = `<p class="gf-blank-line" aria-hidden="true"></p>`

const blankLineMarkerPrefix = "GFBLANKLINE"

var (
	blankLineFenceRe    = regexp.MustCompile("^(`{3,}|~{3,})")
	blankLineListItemRe = regexp.MustCompile(`^(?:([*+-])|\d{1,9}([.)]))(?:[ \t]|$)`)
	blankLineRunRe      = regexp.MustCompile(`\n[ \t]*\r?\n[ \t]*\r?\n`)
)

// blankLineMarker returns a marker that does not occur in source.
func blankLineMarker(source string) string {
	marker := blankLineMarkerPrefix
	for suffix := 0; strings.Contains(source, marker); suffix++ {
		marker = blankLineMarkerPrefix + strconv.Itoa(suffix)
	}
	return marker
}

// markBlankLines inserts marker paragraphs at visible blank lines and returns
// source unchanged when there are none. max < 0 means no cap.
func markBlankLines(source, marker string, max int) string {
	if max == 0 || !blankLineRunRe.MatchString(source) {
		return source
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(source, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(normalized, "\n")
	// Math uses the same segments as protectMathSegments, so a marker can never
	// land inside an expression that is later swapped out as one placeholder.
	math := extractMathSegments(normalized)
	out := make([]string, 0, len(lines)+4)
	pending := make([]string, 0, 4)
	fence, list := "", ""
	seenBlock, changed := false, false
	offset := 0

	for _, line := range lines {
		start := offset
		offset += len(line) + 1
		if fence != "" {
			out = append(out, line)
			if closesBlankLineFence(line, fence) {
				fence = ""
			}
			continue
		}
		for len(math) > 0 && math[0].end <= start {
			math = math[1:]
		}
		if len(math) > 0 && math[0].start < start {
			out = append(append(out, pending...), line)
			pending = pending[:0]
			continue
		}
		if strings.Trim(line, " \t") == "" {
			pending = append(pending, line)
			continue
		}

		indent := blankLineIndent(line)
		body := strings.TrimLeft(line, " \t")
		item := ""
		if indent < 4 {
			if match := blankLineListItemRe.FindStringSubmatch(body); match != nil {
				// A different bullet or ordered delimiter starts a new list.
				item = match[1] + match[2]
				if match[2] != "" {
					item = "1" + item
				}
			}
		}
		extra := len(pending) - 1
		if max > 0 && extra > max {
			extra = max
		}
		// Only before an unindented block: indented lines may continue a list
		// item or indented code, and items of the same list share one list.
		if seenBlock && extra > 0 && indent == 0 && !(item != "" && item == list) {
			out = append(out, "")
			for range extra {
				out = append(out, marker, "")
			}
			changed = true
		} else {
			out = append(out, pending...)
		}
		if item != "" {
			list = item
		} else if len(pending) > 0 && indent == 0 {
			list = ""
		}
		pending = pending[:0]
		out = append(out, line)
		seenBlock = true

		if indent < 4 {
			if open := blankLineFenceRe.FindString(body); open != "" && !(open[0] == '`' && strings.Contains(body[len(open):], "`")) {
				fence = open
			}
		}
	}
	out = append(out, pending...)
	if !changed {
		return source
	}
	return strings.Join(out, "\n")
}

// replaceBlankLineMarkers swaps rendered marker paragraphs for spacers.
func replaceBlankLineMarkers(html, marker string) string {
	return strings.ReplaceAll(html, "<p>"+marker+"</p>", BlankLineHTML)
}

func closesBlankLineFence(line, fence string) bool {
	if blankLineIndent(line) >= 4 {
		return false
	}
	body := strings.Trim(line, " \t")
	if len(body) < len(fence) {
		return false
	}
	for i := 0; i < len(body); i++ {
		if body[i] != fence[0] {
			return false
		}
	}
	return true
}

func blankLineIndent(line string) int {
	width := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			width++
		case '\t':
			width += 4 - width%4
		default:
			return width
		}
	}
	return width
}
