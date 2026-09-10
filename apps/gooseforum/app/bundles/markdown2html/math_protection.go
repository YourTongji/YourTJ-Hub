package markdown2html

import (
	"fmt"
	"html"
	"strings"
)

const mathPlaceholderPrefix = "@@YOURTJ_MATH_"

type mathSegment struct {
	text    string
	display bool
	start   int
	end     int
}

type mathPlaceholder struct {
	token    string
	original string
}

type mathDelimiter struct {
	open         string
	close        string
	display      bool
	singleDollar bool
}

var mathEnvironments = []string{
	"equation",
	"equation*",
	"align",
	"align*",
	"aligned",
	"gather",
	"gather*",
	"multline",
	"multline*",
	"split",
	"cases",
	"matrix",
	"pmatrix",
	"bmatrix",
	"vmatrix",
	"Vmatrix",
}

var mathDelimiters = func() []mathDelimiter {
	delimiters := []mathDelimiter{
		{open: "$$", close: "$$", display: true},
		{open: "\\[", close: "\\]", display: true},
	}
	for _, environment := range mathEnvironments {
		delimiters = append(delimiters, mathDelimiter{
			open:    "\\begin{" + environment + "}",
			close:   "\\end{" + environment + "}",
			display: true,
		})
	}
	delimiters = append(delimiters,
		mathDelimiter{open: "$", close: "$", display: false, singleDollar: true},
		mathDelimiter{open: "\\(", close: "\\)", display: false},
	)
	return delimiters
}()

func protectMathSegments(source string) (string, []mathPlaceholder) {
	segments := extractMathSegments(source)
	if len(segments) == 0 {
		return source, nil
	}

	placeholders := make([]mathPlaceholder, 0, len(segments))
	used := make(map[string]bool, len(segments))
	var builder strings.Builder
	last := 0

	for index, segment := range segments {
		token := uniqueMathToken(source, index, used)
		builder.WriteString(source[last:segment.start])
		builder.WriteString(token)
		placeholders = append(placeholders, mathPlaceholder{
			token:    token,
			original: source[segment.start:segment.end],
		})
		last = segment.end
	}
	builder.WriteString(source[last:])

	return builder.String(), placeholders
}

func restoreMathSegments(rendered string, placeholders []mathPlaceholder) string {
	for _, placeholder := range placeholders {
		rendered = strings.ReplaceAll(rendered, placeholder.token, html.EscapeString(placeholder.original))
	}
	return rendered
}

func uniqueMathToken(source string, index int, used map[string]bool) string {
	token := fmt.Sprintf("%s%d@@", mathPlaceholderPrefix, index)
	for suffix := 1; strings.Contains(source, token) || used[token]; suffix++ {
		token = fmt.Sprintf("%s%d_%d@@", mathPlaceholderPrefix, index, suffix)
	}
	used[token] = true
	return token
}

func extractMathSegments(source string) []mathSegment {
	var segments []mathSegment
	index := 0

	for index < len(source) {
		openIndex, delimiter, ok := findNextMathDelimiter(source, index)
		if !ok {
			break
		}
		openEnd := openIndex + len(delimiter.open)

		if delimiter.singleDollar {
			if openIndex+1 < len(source) && isMathWhitespace(source[openIndex+1]) {
				index = openIndex + 1
				continue
			}

			closeIndex := findMathInlineClosing(source, openEnd)
			if closeIndex != -1 {
				text := source[openEnd:closeIndex]
				if len(text) > 0 && !strings.Contains(text, "\n") {
					segments = append(segments, mathSegment{
						text:    text,
						display: delimiter.display,
						start:   openIndex,
						end:     closeIndex + 1,
					})
					index = closeIndex + 1
					continue
				}
			}
			index = openIndex + 1
			continue
		}

		closeIndex := findMathClosingDelimiter(source, delimiter.close, openEnd)
		if closeIndex != -1 {
			text := source[openEnd:closeIndex]
			if strings.TrimSpace(text) != "" {
				segments = append(segments, mathSegment{
					text:    text,
					display: delimiter.display,
					start:   openIndex,
					end:     closeIndex + len(delimiter.close),
				})
				index = closeIndex + len(delimiter.close)
				continue
			}
		}
		index = openEnd
	}

	return segments
}

func findNextMathDelimiter(source string, start int) (int, mathDelimiter, bool) {
	bestIndex := -1
	var best mathDelimiter
	for _, delimiter := range mathDelimiters {
		index := start
		for {
			relative := strings.Index(source[index:], delimiter.open)
			if relative == -1 {
				break
			}
			candidate := index + relative
			if isMathEscaped(source, candidate) {
				index = candidate + 1
				continue
			}
			if bestIndex == -1 || candidate < bestIndex {
				bestIndex = candidate
				best = delimiter
			}
			break
		}
	}
	return bestIndex, best, bestIndex != -1
}

func isMathEscaped(source string, index int) bool {
	backslashes := 0
	for i := index - 1; i >= 0 && source[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 1
}

func isMathWhitespace(char byte) bool {
	switch char {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func findMathClosingDelimiter(source, delimiter string, startIndex int) int {
	braceLevel := 0
	index := startIndex
	for index < len(source) {
		if braceLevel <= 0 && strings.HasPrefix(source[index:], delimiter) {
			return index
		}
		switch source[index] {
		case '\\':
			index++
		case '{':
			braceLevel++
		case '}':
			braceLevel--
		}
		index++
	}
	return -1
}

func findMathInlineClosing(source string, startIndex int) int {
	braceLevel := 0
	index := startIndex
	for index < len(source) {
		if braceLevel <= 0 && source[index] == '$' {
			if index > startIndex && !isMathWhitespace(source[index-1]) {
				return index
			}
			return -1
		}
		switch source[index] {
		case '\\':
			index++
		case '{':
			braceLevel++
		case '}':
			braceLevel--
		}
		index++
	}
	return -1
}
