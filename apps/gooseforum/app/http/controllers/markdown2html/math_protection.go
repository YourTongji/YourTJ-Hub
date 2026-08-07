package markdown2html

import (
	"fmt"
	"html"
	"strings"
)

const (
	mathPlaceholderPrefix = "@@YOURTJ_MATH_"
	mathBlockDelimiter    = "$$"
	mathInlineDelimiter   = "$"
)

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
		relative := strings.Index(source[index:], mathInlineDelimiter)
		if relative == -1 {
			break
		}
		dollarIndex := index + relative

		if isMathEscaped(source, dollarIndex) {
			index = dollarIndex + 1
			continue
		}

		if strings.HasPrefix(source[dollarIndex:], mathBlockDelimiter) {
			closeIndex := findMathClosingDelimiter(source, mathBlockDelimiter, dollarIndex+len(mathBlockDelimiter))
			if closeIndex != -1 {
				text := source[dollarIndex+len(mathBlockDelimiter) : closeIndex]
				if strings.TrimSpace(text) != "" {
					segments = append(segments, mathSegment{
						text:    text,
						display: true,
						start:   dollarIndex,
						end:     closeIndex + len(mathBlockDelimiter),
					})
					index = closeIndex + len(mathBlockDelimiter)
					continue
				}
			}
			index = dollarIndex + 1
			continue
		}

		if dollarIndex+1 < len(source) && isMathWhitespace(source[dollarIndex+1]) {
			index = dollarIndex + 1
			continue
		}

		closeIndex := findMathInlineClosing(source, dollarIndex+1)
		if closeIndex != -1 {
			text := source[dollarIndex+1 : closeIndex]
			if len(text) > 0 && !strings.Contains(text, "\n") {
				segments = append(segments, mathSegment{
					text:    text,
					display: false,
					start:   dollarIndex,
					end:     closeIndex + 1,
				})
				index = closeIndex + 1
				continue
			}
		}
		index = dollarIndex + 1
	}

	return segments
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
		if braceLevel <= 0 && source[index] == mathInlineDelimiter[0] {
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
